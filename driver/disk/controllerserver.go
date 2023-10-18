package driver

import (
	ebsClient "csi-plugin/pkg/ebs-client"
	"csi-plugin/util"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"strings"
	"sync"
	"time"

	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	//k8s_v1 "k8s.io/api/core/v1"

	"k8s.io/klog"
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
	defaultPurchaseTime = "0"
)

type KscEBSControllerServer struct {
	config Config

	mutex     sync.Mutex
	k8sClient K8sClientWrapper
	ebsClient ebsClient.StorageService
}

// volume parameters
type volumeArgs struct {
	Type             string            `json:"type"`
	Region           string            `json:"regionId"`
	Zone             string            `json:"zoneId"`
	PerformanceLevel string            `json:"performanceLevel"`
	DiskTags         map[string]string `json:"diskTags"`
	NodeSelected     string            `json:"nodeSelected"`
	FsType           string            `json:"fsType"`
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
		//csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_GET_VOLUME,
		csi.ControllerServiceCapability_RPC_VOLUME_CONDITION,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES,
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
	klog.V(5).Infof("CreateVolume: Starting CreateVolume: %+v", req)

	if err := cs.validateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		klog.V(2).Infof("CreateVolume: invalid create volume req: %v", req)
		return nil, err
	}

	//TODO:修改防止重复创建盘的逻辑
	//if value, ok:=createdVolumeMap[req.Name];ok{
	//	klog.V(2).Infof("CreateVolume: volume already be created pvName: %s, VolumeId: %s, volumeContext: %v", req.Name, value.VolumeId, value.VolumeContext)
	//	return &csi.CreateVolumeResponse{Volume: value},nil
	//}

	// check parameters
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume: Volume name must be provided")
	}
	if req.VolumeCapabilities == nil || len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume: volume capabilities must be provided")
	}

	if !validateCapabilities(req.VolumeCapabilities) {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume: invalid volume capabilities requestecs . Only SINGLE_NODE_WRITER is supported ('accessModes.ReadWriteOnce' on Kubernetes)")
	}

	size, err := extractStorage(req.CapacityRange)
	if err != nil {
		return nil, status.Errorf(codes.OutOfRange, "CreateVolume: invalid capacity range: %v", err)
	}

	volArg, err := getVolumeOptions(req)
	if err != nil {
		klog.Errorf("CreateVolume: error parameters from input: %v, with error: %v", req.Name, err)
		return nil, status.Errorf(codes.InvalidArgument, "CreateVolume: Invalid parameters from input: %v, with error: %v", req.Name, err)
	}

	volumeName := req.GetName()

	// 给 ebs 创建硬盘的时间
	time.Sleep(5 * time.Second)

	// get volume first, if it's created do nothing
	listVolumesResp, err := cs.ebsClient.GetVolumeByName(&ebsClient.ListVolumesReq{
		VolumeExactName: volumeName,
		VolumeCategory:  "data",
	})
	if err != nil {
		return nil, err
	}

	// volume already exist, do nothing
	if listVolumesResp.TotalCount > 0 && len(listVolumesResp.Volumes) > 0 {
		klog.V(2).Infoln("Query the hard disk list volume already exists")
		if listVolumesResp.TotalCount > 1 {
			// external-provisioner 调用了三次 rpc createvolume
			// 目前解决办法是，需要用户手动去控制台删除一块ebs盘
			// TODO 是否需要csi 删除一块
			if listVolumesResp.TotalCount == 2 {
				if cs.delvolume(listVolumesResp.Volumes[0], size) {
					return &csi.CreateVolumeResponse{
						Volume: &csi.Volume{
							VolumeId:      listVolumesResp.Volumes[1].VolumeId,
							CapacityBytes: size,
						},
					}, nil
				} else if cs.delvolume(listVolumesResp.Volumes[1], size) {
					return &csi.CreateVolumeResponse{
						Volume: &csi.Volume{
							VolumeId:      listVolumesResp.Volumes[0].VolumeId,
							CapacityBytes: size,
						},
					}, nil
				} else {
					return nil, fmt.Errorf("fatal issue: duplicate volume %q exists", volumeName)
				}
			}
		}
		vol := listVolumesResp.Volumes[0]
		if vol.VolumeStatus == "creating" || vol.VolumeStatus == "available" {
			if vol.Size*GB != size {
				return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("invalid option requested size: %d", size))
			}
			klog.V(2).Info("volume already created")
			return &csi.CreateVolumeResponse{
				Volume: &csi.Volume{
					VolumeId:      vol.VolumeId,
					CapacityBytes: size,
				},
			}, nil
		}
	}
	// todo
	// checking volume limit
	// 是否检查每个账号可以创建的最多 volume 数量
	parameters := SuperMapString(req.Parameters)

	//var region string
	zone := parameters.Get("zone", "")
	isZoneSpecified := true
	if len(zone) == 0 {
		isZoneSpecified = false
		if len(req.AccessibilityRequirements.Preferred) != 0 {
			// choose the most preffer zone
			segments := req.AccessibilityRequirements.Preferred[0]
			zone = segments.Segments[util.NodeZoneKey]
		} else {
			_, zone, err = cs.k8sClient.GetNodeRegionZone()
			if err != nil {
				return nil, err
			}
			klog.V(5).Info(fmt.Sprintf("rand region and zone: %s, %s", "", zone))
		}
	}
	volArg.Zone = zone

	createVolumeReq, diskType, err := preCreateVolume(req.GetName(), size, volArg, parameters)

	purchaseTime, err := strconv.Atoi(parameters.Get("purchasetime", defaultPurchaseTime))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if purchaseTime != 0 {
		createVolumeReq.PurchaseTime = purchaseTime
	}

	//Read the information in storageclasses and use it to create ebs volume
	volumeContext := req.GetParameters()
	if volumeContext == nil {
		volumeContext = make(map[string]string)
	}
	if diskType != "" {
		volumeContext["type"] = diskType
	}
	klog.V(5).Infof("CreateVolume: volume: %s", req.GetName())

	var createVolumeResp *ebsClient.CreateVolumeResp

	for i := 0; i < len(req.AccessibilityRequirements.Preferred); i++ {
		if !isZoneSpecified {
			segments := req.AccessibilityRequirements.Preferred[i]
			createVolumeReq.AvailabilityZone = segments.Segments[util.NodeZoneKey]
			zone = createVolumeReq.AvailabilityZone
		}
		createVolumeResp, err = cs.ebsClient.CreateVolume(createVolumeReq)
		// if createVolume success
		if err == nil {
			break
		}
		// if createVolume err and zoneSelection or last preffered
		if (err != nil && isZoneSpecified) || (err != nil && i == len(req.AccessibilityRequirements.Preferred)-1) {
			return nil, err
		}
	}

	tmpVol := getCsiVolumeInfo(diskType, createVolumeResp.VolumeId, size, volumeContext, volArg.Zone)

	return &csi.CreateVolumeResponse{Volume: tmpVol}, nil
}

func preCreateVolume(diskName string, size int64, volArg *volumeArgs, parameters SuperMapString) (*ebsClient.CreateVolumeReq, string, error) {
	createVolumeRequest := &ebsClient.CreateVolumeReq{}
	createVolumeRequest.VolumeName = diskName
	createVolumeRequest.Size = size / GB
	createVolumeRequest.AvailabilityZone = volArg.Zone
	createVolumeRequest.VolumeDesc = createdByDO
	createVolumeRequest.Tags = volArg.DiskTags
	createVolumeRequest.ChargeType = parameters.Get("chargetype", defaultChargeType)
	createVolumeRequest.ProjectId = parameters.Get("projectid", "")

	diskTypes, diskPLs, err := getDiskType(volArg)
	klog.V(5).Infof("createDisk: diskName: %s, valid disktype: %v, valid diskpls: %v", diskName, diskTypes, diskPLs)
	if err != nil {
		return nil, "", err
	}

	// TODO:支持多diskTypes
	for _, dType := range diskTypes {
		createVolumeRequest.VolumeType = dType
		return createVolumeRequest, dType, nil
	}

	return nil, "", status.Errorf(codes.Internal, "createDisk: err: %v, the zone:[%s] is not support specific disk type, please change the request disktype: %s or disk pl: %s", err, volArg.Zone, diskTypes, diskPLs)
}

func getDiskType(volArg *volumeArgs) ([]string, []string, error) {
	nodeSupportDiskType := []string{}
	if volArg.NodeSelected != "" {
		client := GlobalConfigVar.K8sClient
		nodeInfo, err := client.CoreV1().Nodes().Get(context.Background(), volArg.NodeSelected, metav1.GetOptions{})
		//If the nodeinfo is obtained normally,
		//obtain the Label and verify the matching value between the node and the hard disk.
		//If the acquisition fails, skip the verification and follow the immediate binding process.
		if err != nil {
			klog.Errorf("getDiskType: failed to get node labels: %v", err)
			goto cusDiskType
		}
		re := regexp.MustCompile(`com.ksc.csi.node/disktype.(.*)`)
		for key := range nodeInfo.Labels {
			if result := re.FindStringSubmatch(key); len(result) != 0 {
				nodeSupportDiskType = append(nodeSupportDiskType, result[1])
			}
		}
		klog.V(5).Infof("CreateVolume:: node support disk types: %v, nodeSelected: %v", nodeSupportDiskType, volArg.NodeSelected)
	}
cusDiskType:
	provisionPerformanceLevel := []string{}
	if volArg.PerformanceLevel != "" {
		provisionPerformanceLevel = strings.Split(volArg.PerformanceLevel, ",")
	}
	provisionDiskTypes := []string{}
	allTypes := deleteEmpty(strings.Split(volArg.Type, ","))
	for arr, disktype := range allTypes {
		if disktype == "ESSD" && provisionPerformanceLevel != nil {
			allTypes[arr] = allTypes[arr] + provisionPerformanceLevel[0]
		}
	}
	if len(nodeSupportDiskType) != 0 {
		provisionDiskTypes = intersect(nodeSupportDiskType, allTypes)
		if len(provisionDiskTypes) == 0 {
			klog.Errorf("CreateVolume:: node(%s) support type: [%v] is incompatible with provision disk type: [%s]", volArg.NodeSelected, nodeSupportDiskType, allTypes)
			//return nil, nil, status.Errorf(codes.InvalidArgument, "CreateVolume:: node support type: [%v] is incompatible with provision disk type: [%s]", nodeSupportDiskType, allTypes)
			if allTypes != nil {
				provisionDiskTypes = append(provisionDiskTypes, allTypes[0])
			}
		}
	} else {
		provisionDiskTypes = allTypes
	}
	return provisionDiskTypes, provisionPerformanceLevel, nil
}

// delvolume For CreateVolume rpc timeout error
func (cs *KscEBSControllerServer) delvolume(delVol *ebsClient.Volume, size int64) bool {
	createTime, _ := time.Parse("2006-01-02 15:04:05", delVol.CreateTime)
	if delVol.VolumeDesc == createdByDO && delVol.Size == size/GB && time.Since(createTime).Seconds() < 60 {
		_, err := cs.ebsClient.DeleteVolume(&ebsClient.DeleteVolumeReq{
			VolumeId: delVol.VolumeId,
		})
		if err != nil {
			klog.Errorf("Error deleting volume on duplicate createvolume, volume id: %s", delVol.VolumeId)
		}
		return true
	}
	return false
}

func parseTags(p string) (map[string]string, error) {
	res := make(map[string]string)
	parts := strings.Split(p, ";")
	//fmt.Println(parts)
	if len(parts) > 5 {
		return nil, errors.New("the number of labels cannot exceed 5")
	}
	for _, label := range parts {
		temp := strings.Split(label, "~")
		if len(temp) != 2 {
			klog.Warningf("Invalid label: %v; %s", temp, label)
			//return nil, fmt.Errorf("invalid tag: %s", label)
			continue
		}
		res[temp[0]] = temp[1]
	}
	return res, nil
}

func (cs *KscEBSControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if err := cs.validateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		klog.V(2).Infof("invalid delete volume req: %v", req)
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
	ebs, err := cs.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		if err.Error() == "not found volume" {
			klog.Errorf("volume id: %s error: %v. volume is deleted", req.VolumeId, err)
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if ebs.VolumeStatus != "in-use" {
		klog.V(2).Infof("volume id: %s, volume status %s. volume is detached ", req.VolumeId, ebs.VolumeStatus)
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}

	if len(ebs.Attachments) > 0 {
		if ebs.Attachments[0].InstanceId != req.NodeId {
			klog.V(2).Infof("volume id: %s, volume status %s, target node id: %s. volume is used by other node: %s.", req.VolumeId, ebs.VolumeStatus, req.NodeId, ebs.Attachments[0].InstanceId)
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		}
	} else {
		klog.Fatalf("volume id: %s, volume status %s, target node id: %s. volume is not used. please contact ebs", req.VolumeId, ebs.VolumeStatus, req.NodeId)
	}

	detachVolumeReq := &ebsClient.DetachVolumeReq{
		VolumeId:   req.VolumeId,
		InstanceId: req.NodeId,
	}
	if _, err := cs.ebsClient.Detach(detachVolumeReq); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	klog.V(2).Info("volume is detached")

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

// ControllerPublishVolume attaches the given volume to the node
func (cs *KscEBSControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	//publishInfoVolumeName := cs.config.DriverName + "/volume-name"

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
		if attachment.InstanceId == req.NodeId && vol.VolumeStatus == "in-use" {
			klog.V(2).Info("volume is already attached")
			return &csi.ControllerPublishVolumeResponse{
				PublishContext: map[string]string{
					"MountPoint": attachment.MountPoint,
					//publishInfoVolumeName: vol.VolumeName,
				},
			}, nil
		}
	}
	// node is attached to a different node, return an error
	if len(attachedID) > 0 && vol.VolumeStatus == "in-use" {
		detachVolumeReq := &ebsClient.DetachVolumeReq{
			VolumeId:   req.VolumeId,
			InstanceId: vol.InstanceId,
		}
		if _, err := cs.ebsClient.Detach(detachVolumeReq); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
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
		if err := ebsClient.WaitVolumeStatus(cs.ebsClient, vol.VolumeId, ebsClient.AVAILABLE_STATUS, req.NodeId); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// attach the volume to the correct node
	attachVolumeReq := &ebsClient.AttachVolumeReq{
		VolumeId:   req.VolumeId,
		InstanceId: req.NodeId,
	}
	resp, err := cs.ebsClient.Attach(attachVolumeReq)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// waiting until volume is attached
	if err := ebsClient.WaitVolumeStatus(cs.ebsClient, req.VolumeId, ebsClient.INUSE_STATUS, req.NodeId); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	klog.V(5).Info("volume attached")
	// 给openapi和cinder异步任务执行时间
	time.Sleep(5 * time.Second)
	return &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{
			//publishInfoVolumeName: vol.VolumeName,
			"MountPoint": resp.MountPoint,
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
	volumes := make([]*ebsClient.Volume, len(listVolumesResp.Volumes))
	copy(volumes, listVolumesResp.Volumes)
	// for _, volume := range listVolumesResp.Volumes {
	// 	volumes = append(volumes, volume)
	// }

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
			klog.V(2).Infof("Expand volume-%s failed response: %v , error: %v", volID, expandVolResp, err)
			return nil, err
		} else {
			klog.V(5).Infof("volume-%s expanded success.", volID)
		}
	}

	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         capRange.GetRequiredBytes(),
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
	//return nil, status.Error(codes.Unimplemented, "")
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Volume ID must be provided")
	}

	listVolumesReq := &ebsClient.ListVolumesReq{
		VolumeIds: []string{req.VolumeId},
	}
	vol, err := cs.ebsClient.GetVolume(listVolumesReq)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	if vol.VolumeStatus == "error" {
		return &csi.ControllerGetVolumeResponse{Status: &csi.ControllerGetVolumeResponse_VolumeStatus{VolumeCondition: &csi.VolumeCondition{Abnormal: false, Message: "ebs volume status error"}}}, nil
	}
	if vol.VolumeStatus != "in-use" {
		klog.V(5).Info("volume is already attached")
		return &csi.ControllerGetVolumeResponse{}, nil
	}

	res := &csi.ControllerGetVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      vol.VolumeId,
			CapacityBytes: int64(vol.Size * GB),
		},
		Status: &csi.ControllerGetVolumeResponse_VolumeStatus{
			PublishedNodeIds: []string{vol.InstanceId},
			VolumeCondition:  &csi.VolumeCondition{},
		},
	}

	nodeReady, err := cs.k8sClient.IsNodeStatusReady(vol.InstanceId)
	if err != nil {
		return nil, err
	}
	res.Status.VolumeCondition.Abnormal = !nodeReady
	if !nodeReady {
		res.Status.VolumeCondition.Message = "node notready: " + vol.InstanceId
	}

	return res, nil
}

type K8sClientWrapper interface {
	GetNodeRegionZone() (string, string, error)
	IsNodeStatusReady(nodename string) (bool, error)
}
