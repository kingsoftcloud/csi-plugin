package driver

import (
	ebsClient "csi-plugin/pkg/ebs-client"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"strconv"

	"csi-plugin/util"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8s_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB

	// createdByDO is used to tag volumes that are created by this CSI plugin
	createdByDO = "Created by KSC  CSI driver"

	defaultChargeType   = ebsClient.DAILY_CHARGE_TYPE
	defaultVolumeType   = ebsClient.SSD3_0
	defaultPurchaseTime = "1"
)

type KscEBSControllerServer struct {
	config Config

	mutex     sync.Mutex
	k8sClient K8sClientWrapper
	ebsClient ebsClient.StorageService
}

func GetControllerServer(cfg *Config) *KscEBSControllerServer {
	return &KscEBSControllerServer{
		config:    *cfg,
		ebsClient: cfg.EbsClient,
		k8sClient: GetK8sClientWrapper(cfg.K8sClient),
	}
}

func (cs *KscEBSControllerServer) getControllerServiceCapabilities() []*csi.ControllerServiceCapability {
	var cl []csi.ControllerServiceCapability_RPC_Type

	cl = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
	}
	if cs.config.EnableVolumeExpansion {
		cl = append(cl, csi.ControllerServiceCapability_RPC_EXPAND_VOLUME)
	}

	var csc []*csi.ControllerServiceCapability

	for _, cap := range cl {
		csc = append(csc, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		})
	}

	return csc
}

func (cs *KscEBSControllerServer) validateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	if c == csi.ControllerServiceCapability_RPC_UNKNOWN {
		return nil
	}

	for _, cap := range cs.getControllerServiceCapabilities() {
		if c == cap.GetRpc().GetType() {
			return nil
		}
	}
	return status.Errorf(codes.InvalidArgument, "unsupported capability %s", c)
}

func (cs *KscEBSControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if err := cs.validateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid create volume req: %v", req)
		return nil, err
	}
	// check parameters
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume name must be provided")
	}
	if req.VolumeCapabilities == nil || len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume capabilities must be provided")
	}

	if !validateCapabilities(req.VolumeCapabilities) {
		return nil, status.Error(codes.InvalidArgument, "invalid volume capabilities requestecs . Only SINGLE_NODE_WRITER is supported ('accessModes.ReadWriteOnce' on Kubernetes)")
	}

	size, err := extractStorage(req.CapacityRange)
	if err != nil {
		return nil, status.Errorf(codes.OutOfRange, "invalid capacity range: %v", err)
	}

	volumeName := req.Name
	// get volume first, if it's created do no thing
	listVolumesResp, err := cs.ebsClient.ListVolumes(&ebsClient.ListVolumesReq{})
	if err != nil {
		return nil, err
	}
	filterVolumes := make([]*ebsClient.Volume, 0)
	for _, volume := range listVolumesResp.Volumes {
		if volumeName == volume.VolumeName {
			filterVolumes = append(filterVolumes, volume)
		}
	}

	// volume already exist, do nothing
	if len(filterVolumes) != 0 {
		if len(filterVolumes) > 1 {
			return nil, fmt.Errorf("fatal issue: duplicate volume %q exists", volumeName)
		}
		vol := filterVolumes[0]
		if vol.Size*GB != size {
			return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("invalid option requested size: %d", size))
		}
		glog.Info("volume already created")
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				VolumeId:      vol.VolumeId,
				CapacityBytes: vol.Size * GB,
			},
		}, nil
	}
	// todo
	// checking volume limit
	// 是否检查每个账号可以创建的最多 volume 数量
	parameters := SuperMapString(req.Parameters)
	chargeType := parameters.Get("chargetype", defaultChargeType)
	volumeType := parameters.Get("type", defaultVolumeType)
	projectId := parameters.Get("projectid", "")
	region := parameters.Get("region", "")
	zone := parameters.Get("zone", "")
	if region == "" || zone == "" {
		region, zone, err = cs.k8sClient.GetNodeRegionZone()
		if err != nil {
			return nil, err
		}
		glog.Info(fmt.Sprintf("rand region and zone: %s, %s", region, zone))
	}

	createVolumeReq := &ebsClient.CreateVolumeReq{
		AvailabilityZone: zone,
		VolumeName:       volumeName,
		VolumeDesc:       createdByDO,
		Size:             size / GB,
		ChargeType:       chargeType,
		VolumeType:       volumeType,
		ProjectId:        projectId,
	}

	if chargeType == ebsClient.DAILY_CHARGE_TYPE || chargeType == ebsClient.MONTHLY_CHARGE_TYPE {
		purchaseTime, err := strconv.Atoi(parameters.Get("purchasetime", defaultPurchaseTime))
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		createVolumeReq.PurchaseTime = purchaseTime
	}

	createVolumeResp, err := cs.ebsClient.CreateVolume(createVolumeReq)
	if err != nil {
		return nil, err
	}

	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      createVolumeResp.VolumeId,
			CapacityBytes: size,
			AccessibleTopology: []*csi.Topology{
				{
					Segments: map[string]string{
						util.NodeRegionKey: region,
						util.NodeZoneKey:   zone,
					},
				},
			},
		},
	}

	return resp, nil
}

func (cs *KscEBSControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if err := cs.validateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid delete volume req: %v", req)
		return nil, err
	}

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "DeleteVolume Volume ID must be provided")
	}

	deleteVolumeReq := &ebsClient.DeleteVolumeReq{
		VolumeId: req.VolumeId,
	}
	_, err := cs.ebsClient.DeleteVolume(deleteVolumeReq)
	if err != nil {
		return nil, err
	}
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerUnpublishVolume deattaches the given volume from the node
func (cs *KscEBSControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Volume ID must be provided")
	}
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Node ID must be provided")
	}

	// check if volume exist before trying to detach it
	listVolumesReq := &ebsClient.ListVolumesReq{
		VolumeIds: []string{req.VolumeId},
	}
	_, err := cs.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	detachVolumeReq := &ebsClient.DetachVolumeReq{
		req.VolumeId,
		req.NodeId,
	}
	if _, err := cs.ebsClient.Detach(detachVolumeReq); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	glog.Info("volume is detached")

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

// ControllerPublishVolume attaches the given volume to the node
func (cs *KscEBSControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	publishInfoVolumeName := cs.config.DriverName + "/volume-name"

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Volume ID must be provided")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Node ID must be provided")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Volume capability must be provided")
	}

	if req.Readonly {
		// TODO(arslan): we should return codes.InvalidArgument, but the CSI
		// test fails, because according to the CSI Spec, this flag cannot be
		// changed on the same volume. However we don't use this flag at all,
		// as there are no `readonly` attachable volumes.
		return nil, status.Error(codes.AlreadyExists, "read only Volumes are not supported")
	}

	// check if volume exist before trying to attach it
	listVolumesReq := &ebsClient.ListVolumesReq{
		VolumeIds: []string{req.VolumeId},
	}
	vol, err := cs.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	attachedID := ""
	for _, attachment := range vol.Attachments {
		attachedID = attachment.InstanceId
		if attachment.InstanceId == req.NodeId {
			glog.Info("volume is already attached")
			return &csi.ControllerPublishVolumeResponse{
				PublishContext: map[string]string{
					publishInfoVolumeName: vol.VolumeName,
				},
			}, nil
		}
	}
	// node is attached to a different node, return an error
	if attachedID != "" {
		return nil, status.Errorf(codes.FailedPrecondition,
			"volume is attached to the wrong node(%q), dettach the volume to fix it", attachedID)
	}

	// validate attach instance
	validateAttachInstanceResp, err := cs.ebsClient.ValidateAttachInstance(&ebsClient.ValidateAttachInstanceReq{
		VolumeType: vol.VolumeType,
		InstanceId: req.NodeId,
	})
	if err != nil {
		return nil, err
	}
	if !validateAttachInstanceResp.InstanceEnable {
		return nil, status.Errorf(codes.ResourceExhausted, "attach volume limit has been reached on node %v", req.NodeId)
	}

	// waiting until volume is available
	if vol.VolumeStatus != ebsClient.AVAILABLE_STATUS {
		if err := ebsClient.WaitVolumeStatus(cs.ebsClient, vol.VolumeId, ebsClient.AVAILABLE_STATUS); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// attach the volume to the correct node
	attachVolumeReq := &ebsClient.AttachVolumeReq{
		VolumeId:   req.VolumeId,
		InstanceId: req.NodeId,
	}
	if _, err := cs.ebsClient.Attach(attachVolumeReq); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// waiting until volume is attached
	if err := ebsClient.WaitVolumeStatus(cs.ebsClient, req.VolumeId, ebsClient.INUSE_STATUS); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	glog.Info("volume attached")
	return &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{
			publishInfoVolumeName: vol.VolumeName,
		},
	}, nil
}

func (cs *KscEBSControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities Volume ID must be provided")
	}

	if req.VolumeCapabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities Volume Capabilities must be provided")
	}
	if !validateCapabilities(req.VolumeCapabilities) {
		return nil, status.Error(codes.InvalidArgument, "invalid volume capabilities requestecs . Only SINGLE_NODE_WRITER is supported ('accessModes.ReadWriteOnce' on Kubernetes)")
	}

	listVolumesReq := &ebsClient.ListVolumesReq{
		VolumeIds: []string{req.VolumeId},
	}
	_, err := cs.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	// if it's not supported (i.e: wrong region), we shouldn't override it
	resp := &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext:      req.GetVolumeContext(),
			VolumeCapabilities: req.GetVolumeCapabilities(),
			Parameters:         req.GetParameters()},
	}
	return resp, nil
}

func (cs *KscEBSControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	listVolumesResp, err := cs.ebsClient.ListVolumes(&ebsClient.ListVolumesReq{})
	if err != nil {
		return nil, err
	}
	volumes := make([]*ebsClient.Volume, 0)
	for _, volume := range listVolumesResp.Volumes {
		volumes = append(volumes, volume)
	}

	var entries []*csi.ListVolumesResponse_Entry
	for _, vol := range volumes {
		entries = append(entries, &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				VolumeId:      vol.VolumeId,
				CapacityBytes: int64(vol.Size * GB),
			},
		})
	}

	resp := &csi.ListVolumesResponse{
		Entries: entries,
	}

	return resp, nil
}

func (cs *KscEBSControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	if !cs.config.EnableVolumeExpansion {
		return nil, status.Error(codes.Unimplemented, "ControllerExpandVolume is not supported")
	}
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	capRange := req.GetCapacityRange()
	if capRange == nil {
		return nil, status.Error(codes.InvalidArgument, "Capacity range not provided")
	}

	capacity := int64(capRange.GetRequiredBytes()) / 1024 / 1024 / 1024
	if capacity > cs.config.MaxVolumeSize {
		return nil, status.Errorf(codes.OutOfRange, "Requested capacity %d exceeds maximum allowed %d", capacity, cs.config.MaxVolumeSize)
	}

	// Lock before acting on global state. A production-quality
	// driver might use more fine-grained locking.
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	listVolumesReq := &ebsClient.ListVolumesReq{
		VolumeIds: []string{volID},
	}

	exVol, err := cs.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		return nil, err
	}

	if exVol.Size < capacity {
		var expandVolResp *ebsClient.ExpandVolumeResp
		var expandVolReq = &ebsClient.ExpandVolumeReq{Size: capacity, OnlineResize: true, VolumeId: volID}
		if expandVolResp, err = cs.ebsClient.ExpandVolume(expandVolReq); err != nil {
			glog.Infof("Expand volume-%s failed response: %s , error: %s", volID, expandVolResp, err)
			return nil, err
		} else {
			glog.Info("volume-%s expanded success.", volID)
		}
	}

	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         exVol.Size * 1024 * 1024 * 1024,
		NodeExpansionRequired: true,
	}, nil
}

func (cs *KscEBSControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *KscEBSControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	resp := &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.getControllerServiceCapabilities(),
	}

	return resp, nil
}

func (cs *KscEBSControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *KscEBSControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *KscEBSControllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *KscEBSControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

type K8sClientWrapper interface {
	GetNodeRegionZone() (string, string, error)
}

type K8sClientWrap struct {
	k8sclient *k8sclient.Clientset
}

func GetK8sClientWrapper(k8sclient *k8sclient.Clientset) K8sClientWrapper {
	return &K8sClientWrap{
		k8sclient: k8sclient,
	}
}

func (kc *K8sClientWrap) GetNodeRegionZone() (string, string, error) {
	var randNode k8s_v1.Node

	nodes, err := kc.k8sclient.CoreV1().Nodes().List(context.Background(), meta_v1.ListOptions{})
	if err != nil {
		return "", "", err
	}

	rand.Seed(time.Now().UnixNano())
	randNode = nodes.Items[rand.Intn(len(nodes.Items))]
	return randNode.Labels[util.NodeRegionKey], randNode.Labels[util.NodeZoneKey], nil
}
