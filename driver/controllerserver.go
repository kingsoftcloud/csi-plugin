package driver

import (
	ebsClient "csi-plugin/pkg/ebs-client"
	"fmt"

	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	// check parameters
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume name must be provided")
	}
	if req.VolumeCapabilities == nil || len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume capabilities must be provided")
	}

	if !validateCapabilities(req.VolumeCapabilities) {
		return nil, status.Error(codes.InvalidArgument, "invalid volume capabilities requested. Only SINGLE_NODE_WRITER is supported ('accessModes.ReadWriteOnce' on Kubernetes)")
	}

	size, err := extractStorage(req.CapacityRange)
	if err != nil {
		return nil, status.Errorf(codes.OutOfRange, "invalid capacity range: %v", err)
	}

	if req.AccessibilityRequirements != nil {
		for _, t := range req.AccessibilityRequirements.Requisite {
			region, ok := t.Segments["region"]
			if !ok {
				continue // nothing to do
			}

			if region != d.region {
				return nil, status.Errorf(codes.ResourceExhausted, "volume can be only created in region: %q, got: %q", d.region, region)
			}
		}
	}

	volumeName := req.Name

	// get volume first, if it's created do no thing
	listVolumesResp, err := d.ebsClient.ListVolumes(&ebsClient.ListVolumesReq{})
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
				Id:            vol.VolumeId,
				CapacityBytes: vol.Size * GB,
			},
		}, nil
	}

	parameters := SuperMapString(req.Parameters)
	chargeType := parameters.Get("charge_type", defaultChargeType)
	volumeType := parameters.Get("volume_type", defaultVolumeType)

	createVolumeReq := &ebsClient.CreateVolumeReq{
		AvailabilityZone: d.region,
		VolumeName:       volumeName,
		VolumeDesc:       createdByDO,
		Size:             size / GB,
		ChargeType:       chargeType,
		VolumeType:       volumeType,
	}
	if chargeType == ebsClient.DAILY_CHARGE_TYPE || chargeType == ebsClient.MONTHLY_CHARGE_TYPE {
		purchaseTime, err := strconv.Atoi(parameters.Get("purchase_time", defaultPurchaseTime))
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		createVolumeReq.PurchaseTime = purchaseTime
	}

	createVolumeResp, err := d.ebsClient.CreateVolume(createVolumeReq)
	if err != nil {
		return nil, err
	}

	// todo
	// checking volume limit
	// 是否检查每个账号可以创建的最多 volume 数量

	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			Id:            createVolumeResp.VolumeId,
			CapacityBytes: size,
			AccessibleTopology: []*csi.Topology{
				{
					Segments: map[string]string{
						"region": d.region,
					},
				},
			},
		},
	}

	return resp, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "DeleteVolume Volume ID must be provided")
	}
	deleteVolumeReq := &ebsClient.DeleteVolumeReq{
		VolumeId: req.VolumeId,
	}
	_, err := d.ebsClient.DeleteVolume(deleteVolumeReq)
	if err != nil {
		return nil, err
	}
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerUnpublishVolume deattaches the given volume from the node
func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
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
	_, err := d.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	// check if node exist before trying to detach the volume from the node
	if _, err := d.kecClient.DescribeInstances(req.NodeId); err != nil {
		return nil, status.Errorf(codes.NotFound, "node %q not found", req.NodeId)
	}
	detachVolumeReq := &ebsClient.DetachVolumeReq{
		req.VolumeId,
		req.NodeId,
	}
	if _, err := d.ebsClient.Detach(detachVolumeReq); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	glog.Info("volume is detached")

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

// ControllerPublishVolume attaches the given volume to the node
func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	publishInfoVolumeName := d.name + "/volume-name"

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
	vol, err := d.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	// check if kec node exist before trying to attach the volume to the node
	if _, err := d.kecClient.DescribeInstances(req.NodeId); err != nil {
		return nil, status.Errorf(codes.NotFound, "node %q not found", req.NodeId)
	}

	attachedID := ""
	for _, attachment := range vol.Attachments {
		attachedID = attachment.InstanceId
		if attachment.InstanceId == req.NodeId {
			glog.Info("volume is already attached")
			return &csi.ControllerPublishVolumeResponse{
				PublishInfo: map[string]string{
					publishInfoVolumeName: vol.VolumeName,
				},
			}, nil
		}
	}

	// waiting until volume is available
	if vol.VolumeStatus != ebsClient.AVAILABLE_STATUS {
		if err := ebsClient.WaitVolumeStatus(d.ebsClient, vol.VolumeId, ebsClient.AVAILABLE_STATUS); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// node is attached to a different node, return an error
	if attachedID != "" {
		return nil, status.Errorf(codes.FailedPrecondition,
			"volume is attached to the wrong node(%q), dettach the volume to fix it", attachedID)
	}

	// attach the volume to the correct node
	attachVolumeReq := &ebsClient.AttachVolumeReq{
		VolumeId:   req.VolumeId,
		InstanceId: req.NodeId,
	}
	if _, err := d.ebsClient.Attach(attachVolumeReq); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// waiting until volume is attached
	if err := ebsClient.WaitVolumeStatus(d.ebsClient, req.VolumeId, ebsClient.INUSE_STATUS); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	glog.Info("volume attaced")
	return &csi.ControllerPublishVolumeResponse{
		PublishInfo: map[string]string{
			publishInfoVolumeName: vol.VolumeName,
		},
	}, nil
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities Volume ID must be provided")
	}

	if req.VolumeCapabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities Volume Capabilities must be provided")
	}
	listVolumesReq := &ebsClient.ListVolumesReq{
		VolumeIds: []string{req.VolumeId},
	}
	_, err := d.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	if req.AccessibleTopology != nil {
		for _, t := range req.AccessibleTopology {
			region, ok := t.Segments["region"]
			if !ok {
				continue // nothing to do
			}

			if region != d.region {
				// return early if a different region is expected
				glog.Info("supported capabilities false")
				return &csi.ValidateVolumeCapabilitiesResponse{
					Supported: false,
				}, nil
			}
		}
	}

	// if it's not supported (i.e: wrong region), we shouldn't override it
	resp := &csi.ValidateVolumeCapabilitiesResponse{
		Supported: validateCapabilities(req.VolumeCapabilities),
	}
	return resp, nil
}

func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	listVolumesResp, err := d.ebsClient.ListVolumes(&ebsClient.ListVolumesReq{})
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
				Id:            vol.VolumeId,
				CapacityBytes: int64(vol.Size * GB),
			},
		})
	}

	resp := &csi.ListVolumesResponse{
		Entries: entries,
	}

	return resp, nil
}

func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	newCap := func(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
		return &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
	}

	var caps []*csi.ControllerServiceCapability
	for _, cap := range []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
	} {
		caps = append(caps, newCap(cap))
	}

	resp := &csi.ControllerGetCapabilitiesResponse{
		Capabilities: caps,
	}

	return resp, nil
}

func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
