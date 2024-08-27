package ks3

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// NodeServer driver
type NodeServer struct {
	d *Driver
}

// NewNodeServer new NodeServer
func NewNodeServer(driver *Driver) *NodeServer {
	return &NodeServer{
		d: driver,
	}
}

type ks3fsOptions struct {
	URL             string
	Bucket          string
	Path            string
	DbgLevel        string
	AdditionalArgs  string
	NotsupCompatDir bool
}

// NodePublishVolume mount the volume
func (ns *NodeServer) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.V(2).Info("NodePublishVolume:: KS3 Start")
	if err := validateNodePublishVolumeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	volID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	podUID := req.GetVolumeContext()["csi.storage.k8s.io/pod.uid"]

	options, err := parseKS3fsOptions(req.GetVolumeContext())
	if err != nil {
		klog.Errorf("parse options from VolumeAttributes for %s failed: %v", volID, err)
		return nil, status.Errorf(codes.InvalidArgument, "parse options failed: %v", err)
	}
	subPath := options.Path
	options.Path = ""
	options.NotsupCompatDir = true

	// create the tmp credential info from NodePublishSecrets
	credFilePath, err := createCredentialFile(volID, options.Bucket, req.GetSecrets())
	if err != nil {
		return nil, err
	}

	// create ks3 subPath if not exist
	ks3TmpPath := filepath.Join(tempMntPath, podUID+"_"+volID)
	if err = os.MkdirAll(ks3TmpPath, 0750); err != nil {
		klog.Errorf("create ks3TmpPath for %s failed: %v", volID, err)
		return nil, status.Errorf(codes.Internal, "create ks3TmpPath for %s failed: %v", volID, err)
	}
	notMnt, err := DefaultMounter.IsLikelyNotMountPoint(ks3TmpPath)
	if err != nil {
		klog.Errorf("check ks3TmpPath IsLikelyNotMountPoint for %s failed: %v", volID, err)
		return nil, status.Errorf(codes.Internal, "check ks3TmpPath IsLikelyNotMountPoint for %s failed: %v", volID, err)
	}
	defer func() {
		if err != nil {
			DefaultMounter.Unmount(ks3TmpPath)
			DefaultMounter.Unmount(targetPath)
		}
	}()
	if notMnt {
		if err = ks3mount(options, ks3TmpPath, credFilePath); err != nil {
			klog.Errorf("Mount %s to %s failed: %v", volID, ks3TmpPath, err)
			return nil, status.Errorf(codes.Internal, "mount failed: %v", err)
		}
	}
	err = checkKS3Mounted(ks3TmpPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check ks3TmpPath mounted fail: %s, please check ks3 bucket existed and ak/sk is correct", err)
	}

	destPath := filepath.Join(ks3TmpPath, subPath)
	if err = os.MkdirAll(destPath, 0750); err != nil {
		klog.Errorf("KS3 Create Sub Directory fail, path: %s, err: %v", destPath, err)
		return nil, status.Errorf(codes.Internal, "create subPath error: %v", err)
	}

	// umount tmp path
	if err = DefaultMounter.Unmount(ks3TmpPath); err != nil {
		klog.Errorf("Failed to umount ks3TmpPath %s for volume %s: %v", ks3TmpPath, volID, err)
		return nil, status.Errorf(codes.Internal, "umount failed: %v", err)
	}
	if err = os.Remove(ks3TmpPath); err != nil {
		klog.Errorf("Failed to remove ks3TmpPath %s for volume %s: %v", ks3TmpPath, volID, err)
		return nil, status.Errorf(codes.Internal, "remove ks3TmpPath failed: %v", err)
	}

	// mount targetPath
	options.Path = subPath
	options.NotsupCompatDir = false
	if err = os.MkdirAll(targetPath, 0750); err != nil {
		klog.Errorf("create targetPath for %s failed: %v", volID, err)
		return nil, status.Errorf(codes.Internal, "create targetPath for %s failed: %v", volID, err)
	}
	notMnt, err = DefaultMounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		klog.Errorf("isMountPoint for %s failed: %v", volID, err)
		return nil, err
	}
	if !notMnt {
		klog.Infof("Volume %s is already mounted to %s, skipping", volID, targetPath)
		return &csi.NodePublishVolumeResponse{}, nil
	}
	if err = ks3mount(options, targetPath, credFilePath); err != nil {
		klog.Errorf("Mount %s to %s failed: %v", volID, targetPath, err)
		return nil, status.Errorf(codes.Internal, "mount failed: %v", err)
	}
	err = checkKS3Mounted(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check targetPath mounted fail: %s, please check ks3 bucket existed and ak/sk is correct", err)
	}

	klog.Infof("successfully mounted volume %s to %s", volID, targetPath)

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmount the volume
func (ns *NodeServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if err := validateNodeUnpublishVolumeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	volID := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	if err := DefaultMounter.Unmount(targetPath); err != nil {
		if strings.Contains(err.Error(), "not mounted") || strings.Contains(err.Error(), "no mount point specified") {
			klog.Infof("mountpoint not mounted, skipping: %s", targetPath)
			return &csi.NodeUnpublishVolumeResponse{}, nil
		}
		klog.Errorf("failed to umount point %s for volume %s: %v", targetPath, volID, err)
		return nil, status.Errorf(codes.Internal, "umount ks3 failed: %v", err)
	}

	klog.Infof("Successfully unmounted volume %s from %s", volID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeStageVolume stage volume
func (ns *NodeServer) NodeStageVolume(_ context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeUnstageVolume unstage volume
func (ns *NodeServer) NodeUnstageVolume(_ context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeGetVolumeStats get volume stats
func (ns *NodeServer) NodeGetVolumeStats(_ context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeExpandVolume node expand volume
func (ns *NodeServer) NodeExpandVolume(_ context.Context, _ *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeGetCapabilities return the capabilities of the Node plugin
func (ns *NodeServer) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.d.NSCap,
	}, nil
}

// NodeGetInfo return info of the node on which this plugin is running
func (ns *NodeServer) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: ns.d.NodeID,
	}, nil
}
