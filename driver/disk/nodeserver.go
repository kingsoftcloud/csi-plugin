package driver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"csi-plugin/util"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	diskIDPath = "/dev/disk/by-id"
	diskPrefix = "virtio-"
)

type NodeServer struct {
	config Config

	sync.Mutex
	nodeName string
	nodeID   string
	region   string
	zone     string
	mounter  Mounter
}

func GetNodeServer(cfg *Config) *NodeServer {
	nodeName := os.Getenv("KUBE_NODE_NAME")
	if nodeName == "" {
		panic(errors.New("nodename is empty"))
	}

	// get node instance_uuid
	instanceUUID, err := util.GetSystemUUID()
	if err != nil {
		panic(err)
	}

	nodeServer := &NodeServer{
		config:   *cfg,
		nodeName: nodeName,
		nodeID:   instanceUUID,
		mounter:  newMounter(),
	}

	k8sCli := cfg.K8sClient
	node, err := k8sCli.CoreV1().Nodes().Get(context.Background(), nodeName, meta_v1.GetOptions{})
	if err != nil {
		panic(err)
	}
	nodeServer.region = node.Labels[util.NodeRegionKey]
	nodeServer.zone = node.Labels[util.NodeZoneKey]

	return nodeServer
}

func (d *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	annNoFormatVolume := d.config.DriverName + "/noformat"
	//publishInfoVolumeName := d.config.DriverName + "/volume-name"

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume ID must be provided")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Staging Target Path must be provided")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}

	devMountPoint, ok := req.GetPublishContext()["MountPoint"]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "node stage volume, could not find the dev disk in node")
	}
	glog.Infof("dev attach point:  %s", devMountPoint)
	// TODO  这里使用 /dev/disk/by-id/virtio-* 挂载，因为 openapi 返回的挂载点有时候与node实际挂载点（/dev/vd*）不符
	source := getDiskSource(req.VolumeId)
	//source := devMountPoint
	target := req.StagingTargetPath

	mnt := req.VolumeCapability.GetMount()
	options := mnt.MountFlags

	fsType := "ext4"
	if mnt.FsType != "" {
		fsType = mnt.FsType
	}

	_, ok = req.GetVolumeContext()[annNoFormatVolume]
	if !ok {
		d.Lock()
		defer d.Unlock()
		formatted, err := d.mounter.IsFormatted(source)
		if err != nil {
			return nil, err
		}

		if !formatted {
			glog.Info("formatting the volume for staging")
			if err := d.mounter.Format(source, fsType); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		} else {
			glog.Info("source device is already formatted")
		}

	} else {
		glog.Info("skipping formatting the source device")
	}
	glog.Info("mounting the volume for staging")
	mounted, err := d.mounter.IsMounted(target)
	if err != nil {
		return nil, err
	}
	if !mounted {
		if err := d.mounter.Mount(source, target, fsType, options...); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		glog.Info("source device is already mounted to the target path")
	}
	glog.Info("formatting and mounting stage volume is finished")
	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Volume ID must be provided")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Staging Target Path must be provided")
	}

	mounted, err := d.mounter.IsMounted(req.StagingTargetPath)
	if err != nil {
		return nil, err
	}

	if mounted {
		glog.Info("unmounting the staging target path")
		err := d.mounter.Unmount(req.StagingTargetPath)
		if err != nil {
			return nil, err
		}
	} else {
		glog.Info("staging target path is already unmounted")
	}

	glog.Info("unmounting stage volume is finished")
	return &csi.NodeUnstageVolumeResponse{}, nil
}

// NodePublishVolume mounts the volume mounted to the staging path to the target path
func (d *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Volume ID must be provided")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Staging Target Path must be provided")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Target Path must be provided")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Volume Capability must be provided")
	}
	source := req.StagingTargetPath
	target := req.TargetPath

	mnt := req.VolumeCapability.GetMount()
	options := mnt.MountFlags

	// TODO(arslan): do we need bind here? check it out
	// Perform a bind mount to the full path to allow duplicate mounts of the same PD.
	options = append(options, "bind")
	if req.Readonly {
		options = append(options, "ro")
	}

	fsType := "ext4"
	if mnt.FsType != "" {
		fsType = mnt.FsType
	}

	mounted, err := d.mounter.IsMounted(target)
	if err != nil {
		return nil, err
	}

	if !mounted {
		glog.Info("mounting the volume")
		if err := d.mounter.Mount(source, target, fsType, options...); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		glog.Info("volume is already mounted")
	}

	glog.Info("bind mounting the volume is finished")
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmounts the volume from the target path
func (d *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Volume ID must be provided")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Target Path must be provided")
	}

	mounted, err := d.mounter.IsMounted(req.TargetPath)
	if err != nil {
		return nil, err
	}

	if mounted {
		glog.Info("unmounting the target path")
		err := d.mounter.Unmount(req.TargetPath)
		if err != nil {
			return nil, err
		}
	} else {
		glog.Info("target path is already unmounted")
	}

	glog.Info("unmounting volume is finished")
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	var err error
	targetPath := req.GetVolumePath()
	if targetPath == "" {
		err = fmt.Errorf("NodeGetVolumeStats targetpath %v is empty", targetPath)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return util.GetMetrics(targetPath)
}

// TODO 目前不支持 xfs 文件系统扩容
func (d *NodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	if !d.config.EnableVolumeExpansion {
		return nil, status.Error(codes.Unimplemented, "NodeExpandVolume is not supported")
	}
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	capRange := req.GetCapacityRange()
	if capRange == nil {
		return nil, status.Error(codes.InvalidArgument, "Capacity range not provided")
	}

	devName := getDiskSource(volID)

	mnt := req.VolumeCapability.GetMount()
	switch mnt.FsType {
	case "xfs":
		d.mounter.Expand(mnt.FsType, req.VolumePath)
	case "ext4", "ext3", "ext2":
		d.mounter.Expand(mnt.FsType, devName)
	case "":
		ok, err := d.mounter.Expand("ext4", devName)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "expand failed with error: %v, fs type: %s, source: %s", err, "ext4", devName)
		}
		if !ok {
			return nil, status.Errorf(codes.Internal, "expand failed, fs type: %s, source: %s", "ext4", devName)
		}
	default:
		glog.Errorf("not supported fsType: %s", mnt.FsType)
		return nil, status.Errorf(codes.InvalidArgument, "not supported fsType: %s", mnt.FsType)
	}

	return &csi.NodeExpandVolumeResponse{
		//CapacityBytes: capRange.GetRequiredBytes(),
	}, nil
}

func (d *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	// currently there is a single EnableNodeServer capability according to the spec
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: d.getNodeServiceCapabilities(),
	}, nil
}

func (d *NodeServer) getNodeServiceCapabilities() []*csi.NodeServiceCapability {
	var capabilityRpcTypes []csi.NodeServiceCapability_RPC_Type

	capabilityRpcTypes = []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	}
	if d.config.EnableVolumeExpansion {
		capabilityRpcTypes = append(capabilityRpcTypes, csi.NodeServiceCapability_RPC_EXPAND_VOLUME)
	}
	if d.config.MetricEnabled {
		capabilityRpcTypes = append(capabilityRpcTypes, csi.NodeServiceCapability_RPC_GET_VOLUME_STATS)
	}
	var nodeServiceCapabilities []*csi.NodeServiceCapability

	for _, one := range capabilityRpcTypes {
		nodeServiceCapabilities = append(nodeServiceCapabilities, &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: one,
				},
			},
		})
	}

	return nodeServiceCapabilities
}

func (d *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	// fix bug, ca触发时，新增的节点 csi node ds pod 会依赖node label
	if len(d.region) == 0 || len(d.zone) == 0 {
		k8sCli := d.config.K8sClient
		node, err := k8sCli.CoreV1().Nodes().Get(context.Background(), d.nodeName, meta_v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		d.region = node.Labels[util.NodeRegionKey]
		d.zone = node.Labels[util.NodeZoneKey]
	}

	resp := &csi.NodeGetInfoResponse{
		NodeId: d.nodeID,
		//refer to  https://docs.ksyun.com/documents/5423 "单实例云硬盘数量"
		MaxVolumesPerNode: d.config.MaxVolumesPerNode,
		// make sure that the driver works on this particular region only
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				util.NodeRegionKey: d.region,
				util.NodeZoneKey:   d.zone,
			},
		},
	}
	return resp, nil
}

//
//// NodeExpandVolume is only implemented so the driver can be used for e2e testing.
//func (d *EnableNodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
//	if !hp.config.EnableVolumeExpansion {
//		return nil, status.Error(codes.Unimplemented, "NodeExpandVolume is not supported")
//	}
//
//	volID := req.GetVolumeId()
//	if len(volID) == 0 {
//		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
//	}
//
//	// Lock before acting on global state. A production-quality
//	// driver might use more fine-grained locking.
//	hp.mutex.Lock()
//	defer hp.mutex.Unlock()
//
//	vol, err := hp.state.GetVolumeByID(volID)
//	if err != nil {
//		return nil, err
//	}
//
//	volPath := req.GetVolumePath()
//	if len(volPath) == 0 {
//		return nil, status.Error(codes.InvalidArgument, "Volume path not provided")
//	}
//
//	info, err := os.Stat(volPath)
//	if err != nil {
//		return nil, status.Errorf(codes.InvalidArgument, "Could not get file information from %s: %v", volPath, err)
//	}
//
//	switch m := info.Mode(); {
//	case m.IsDir():
//		if vol.VolAccessType != state.MountAccess {
//			return nil, status.Errorf(codes.InvalidArgument, "Volume %s is not a directory", volID)
//		}
//	case m&os.ModeDevice != 0:
//		if vol.VolAccessType != state.BlockAccess {
//			return nil, status.Errorf(codes.InvalidArgument, "Volume %s is not a block device", volID)
//		}
//	default:
//		return nil, status.Errorf(codes.InvalidArgument, "Volume %s is invalid", volID)
//	}
//
//	return &csi.NodeExpandVolumeResponse{}, nil
//}
// getDiskSource returns the absolute path of the attached volume for the given
// DO volume name
func getDiskSource(volumeId string) string {
	return filepath.Join(diskIDPath, diskPrefix+volumeId[0:20])
}
