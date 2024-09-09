package driver

import (
	OpenApi "csi-plugin/pkg/open-api"
	"csi-plugin/util"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sclient "k8s.io/client-go/kubernetes"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	supportedAccessMode = &csi.VolumeCapability_AccessMode{
		Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	}
)

const (
	// PublishInfoVolumeName is used to pass the volume name from
	// `ControllerPublishVolume` to `NodeStageVolume or `NodePublishVolume`
	// PublishInfoVolumeName = DriverName + "/volume-name"

	// minimumVolumeSizeInBytes is used to validate that the user is not trying
	// to create a volume that is smaller than what we support
	minimumVolumeSizeInBytes int64 = 1 * GB

	// maximumVolumeSizeInBytes is used to validate that the user is not trying
	// to create a volume that is larger than what we support
	maximumVolumeSizeInBytes int64 = 32000 * GB

	// defaultVolumeSizeInBytes is used when the user did not provide a size or
	// the size they provided did not satisfy our requirements
	defaultVolumeSizeInBytes int64 = 16 * GB
)

var (
	AvailableVolumeTypes  = []string{SSD2_0, SSD3_0, SATA3_0, EHDD, ESSD, ESSD_PL1, ESSD_PL2, ESSD_PL3, ESSD_PL0}
	CustomDiskTypes       = map[string]int{ESSD: 0, SSD3_0: 1, SSD2_0: 2, SATA3_0: 3, EHDD: 4}
	CustomDiskPerfermance = map[string]string{DISK_PERFORMANCE_LEVEL0: "", DISK_PERFORMANCE_LEVEL1: "", DISK_PERFORMANCE_LEVEL2: "", DISK_PERFORMANCE_LEVEL3: ""}

	// the map of multizone and index
	storageClassZonePos = map[string]int{}
)

type AccountAllProjectListResp struct {
	ListProjectResult ListProjectResult `json:"ListProjectResult,omitempty"`
	RequestID         string            `json:"RequestId,omitempty"`
}
type ProjectList struct {
	ProjectID int    `json:"ProjectId,omitempty"`
	Status    int    `json:"Status,omitempty"`
	Krn       string `json:"Krn,omitempty"`
	AccountID string `json:"AccountId,omitempty"`
}
type ListProjectResult struct {
	Total       int           `json:"Total,omitempty"`
	ProjectList []ProjectList `json:"ProjectList,omitempty"`
}

type DescribeInstancesResp struct {
	Marker        int            `json:"Marker,omitempty"`
	InstanceCount int            `json:"InstanceCount,omitempty"`
	RequestID     string         `json:"RequestId,omitempty"`
	InstancesSet  []InstancesSet `json:"InstancesSet,omitempty"`
}
type InstanceState struct {
	Name string `json:"Name,omitempty"`
}
type InstancesSet struct {
	InstanceID       string        `json:"InstanceID,omitempty"`
	InstanceType     string        `json:"InstanceType,omitempty"`
	InstanceState    InstanceState `json:"InstanceState,omitempty"`
	AvailabilityZone string        `json:"AvailabilityZone,omitempty"`
}

type DescribeInstanceTypeConfigsResp struct {
	RequestID             string                  `json:"RequestId,omitempty"`
	InstanceTypeConfigSet []InstanceTypeConfigSet `json:"InstanceTypeConfigSet,omitempty"`
}
type AvailabilityZoneSet struct {
	AzCode string `json:"AzCode,omitempty"`
}
type DataDiskQuotaSet struct {
	DataDiskType        string                `json:"DataDiskType,omitempty"`
	DataDiskMinSize     float64               `json:"DataDiskMinSize,omitempty"`
	DataDiskMaxsize     float64               `json:"DataDiskMaxsize,omitempty"`
	DataDiskCount       int                   `json:"DataDiskCount,omitempty"`
	AvailabilityZoneSet []AvailabilityZoneSet `json:"AvailabilityZoneSet,omitempty"`
}
type InstanceTypeConfigSet struct {
	InstanceType     string             `json:"InstanceType,omitempty"`
	InstanceFamily   string             `json:"InstanceFamily,omitempty"`
	DataDiskQuotaSet []DataDiskQuotaSet `json:"DataDiskQuotaSet,omitempty"`
}

type VolumesInfoResp struct {
	RequestID string    `json:"RequestId,omitempty"`
	Volumes   []Volumes `json:"Volumes,omitempty"`
}
type Volumes struct {
	VolumeID   string `json:"VolumeId,omitempty"`
	VolumeName string `json:"VolumeName,omitempty"`
	VolumeType string `json:"VolumeType,omitempty"`
}

func validateCapabilities(caps []*csi.VolumeCapability) bool {
	vcaps := []*csi.VolumeCapability_AccessMode{supportedAccessMode}

	hasSupport := func(mode csi.VolumeCapability_AccessMode_Mode) bool {
		for _, m := range vcaps {
			if mode == m.Mode {
				return true
			}
		}
		return false
	}

	supported := false
	for _, cap := range caps {
		if hasSupport(cap.AccessMode.Mode) {
			supported = true
		} else {
			// we need to make sure all capabilities are supported. Revert back
			// in case we have a cap that is supported, but is invalidated now
			supported = false
		}
	}

	return supported
}

// extractStorage extracts the storage size in bytes from the given capacity
// range. If the capacity range is not satisfied it returns the default volume
// size. If the capacity range is below or above supported sizes, it returns an
// error.
func extractStorage(capRange *csi.CapacityRange) (int64, error) {
	if capRange == nil {
		return defaultVolumeSizeInBytes, nil
	}

	requiredBytes := capRange.GetRequiredBytes()
	requiredSet := 0 < requiredBytes
	limitBytes := capRange.GetLimitBytes()
	limitSet := 0 < limitBytes

	if !requiredSet && !limitSet {
		return defaultVolumeSizeInBytes, nil
	}

	if requiredSet && limitSet && limitBytes < requiredBytes {
		return 0, fmt.Errorf("limit (%v) can not be less than required (%v) size", formatBytes(limitBytes), formatBytes(requiredBytes))
	}

	if requiredSet && !limitSet && requiredBytes < minimumVolumeSizeInBytes {
		return 0, fmt.Errorf("required (%v) can not be less than minimum supported volume size (%v)", formatBytes(requiredBytes), formatBytes(minimumVolumeSizeInBytes))
	}

	if limitSet && limitBytes < minimumVolumeSizeInBytes {
		return 0, fmt.Errorf("limit (%v) can not be less than minimum supported volume size (%v)", formatBytes(limitBytes), formatBytes(minimumVolumeSizeInBytes))
	}

	if requiredSet && requiredBytes > maximumVolumeSizeInBytes {
		return 0, fmt.Errorf("required (%v) can not exceed maximum supported volume size (%v)", formatBytes(requiredBytes), formatBytes(maximumVolumeSizeInBytes))
	}

	if !requiredSet && limitSet && limitBytes > maximumVolumeSizeInBytes {
		return 0, fmt.Errorf("limit (%v) can not exceed maximum supported volume size (%v)", formatBytes(limitBytes), formatBytes(maximumVolumeSizeInBytes))
	}

	if requiredSet && limitSet && requiredBytes == limitBytes {
		return requiredBytes, nil
	}

	if requiredSet {
		return requiredBytes, nil
	}

	if limitSet {
		return limitBytes, nil
	}

	return defaultVolumeSizeInBytes, nil
}

func formatBytes(inputBytes int64) string {
	output := float64(inputBytes)
	unit := ""

	switch {
	case inputBytes >= TB:
		output = output / TB
		unit = "Ti"
	case inputBytes >= GB:
		output = output / GB
		unit = "Gi"
	case inputBytes >= MB:
		output = output / MB
		unit = "Mi"
	case inputBytes >= KB:
		output = output / KB
		unit = "Ki"
	case inputBytes == 0:
		return "0"
	}

	result := strconv.FormatFloat(output, 'f', 1, 64)
	result = strings.TrimSuffix(result, ".0")
	return result + unit
}

type SuperMapString map[string]string

func (sm SuperMapString) Get(key, backup string) string {
	value, ok := sm[key]
	if !ok {
		return backup
	}
	return value
}

func getLogLevel(method string) int32 {
	if method == "/csi.v1.Identity/Probe" ||
		method == "/csi.v1.Node/NodeGetCapabilities" ||
		method == "/csi.v1.Node/NodeGetVolumeStats" ||
		method == "/csi.v1.Controller/ControllerGetCapabilities" ||
		method == "/csi.v1.Controller/ControllerGetVolume" {
		return 5
	}
	return 2
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
	//var randNodes []k8s_v1.Node
	//TODO meta_v1.ListOptions 选择node
	labeSelector := meta_v1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/role": "node"},
	}
	mapLabel, err := meta_v1.LabelSelectorAsMap(&labeSelector)
	if err != nil {
		return "", "", err
	}
	nodes, err := kc.k8sclient.CoreV1().Nodes().List(context.Background(), meta_v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(mapLabel).String(),
	})
	if err != nil {
		return "", "", err
	}

	rand.Seed(time.Now().UnixNano())
	// sc 没有声明region和AZ, 这里只随机选择role为node的可用区
	// for _, v := range nodes.Items {
	// 	if role, ok := v.Labels[util.NodeRoleKey]; ok {
	// 		if role == "node" {
	// 			randNodes = append(randNodes, v)
	// 		}
	// 	}
	// }
	randNode := nodes.Items[rand.Intn(len(nodes.Items))]

	return randNode.Labels[util.NodeRegionKey], randNode.Labels[util.NodeZoneKey], nil
}

func (kc *K8sClientWrap) IsNodeStatusReadyByNodename(nodename string) (bool, error) {
	node, err := kc.k8sclient.CoreV1().Nodes().Get(context.Background(), nodename, meta_v1.GetOptions{})
	if err != nil {
		return false, err
	}
	for _, v := range node.Status.Conditions {
		if v.Type == "Ready" && v.Status == "True" {
			return true, nil
		}
	}
	return false, nil
}

func (kc *K8sClientWrap) IsNodeStatusReady(nodeID string) (bool, error) {
	labeSelector := meta_v1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/role": "node"},
	}
	mapLabel, err := meta_v1.LabelSelectorAsMap(&labeSelector)
	if err != nil {
		return false, err
	}
	nodes, err := kc.k8sclient.CoreV1().Nodes().List(context.Background(), meta_v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(mapLabel).String(),
	})
	if err != nil {
		return false, err
	}
	for _, node := range nodes.Items {
		if a, ok := node.Annotations["appengine.sdns.ksyun.com/instance-uuid"]; ok {
			if a != nodeID {
				continue
			}
		} else {
			return false, fmt.Errorf("node annotation missing: appengine.sdns.ksyun.com/instance-uuid")
		}
		for _, v := range node.Status.Conditions {
			if v.Type == "Ready" && v.Status == "True" {
				return true, nil
			}
		}
	}
	return false, nil
}

// getVolumeOptions
func getVolumeOptions(req *csi.CreateVolumeRequest) (*volumeArgs, error) {
	//var ok bool
	volArgArgs := &volumeArgs{
		DiskTags: map[string]string{},
	}
	volOptions := req.GetParameters()

	//TODO: 获取AZ
	//zone := parameters.Get("zone", "")
	//if volArgArgs.Zone, ok = volOptions["zone"]; !ok {
	//	if volArgArgs.Zone, ok = volOptions[strings.ToLower("zone")]; !ok {
	//		// 选择Zone
	//		volArgArgs.Zone = pickZone(req.GetAccessibilityRequirements())
	//		if volArgArgs.Zone == "" {
	//			klog.Errorf("CreateVolume: Can't get topology info , please check your setup or set zone in storage class. Use zone from service: %s", req.Name)
	//			volArgArgs.Zone, _ = utils.Get
	//		}
	//	}
	//}

	//TODO: Support Multi zones if set
	//zoneStr := volArgArgs.Zone
	//zones := strings.Split(zoneStr, ",")
	//zoneNum := len(zones)
	//if zoneNum > 1 {
	//	if _, ok := storageClassZonePos[zoneStr]; !ok {
	//		storageClassZonePos[zoneStr] = 0
	//	}
	//	zoneIndex := storageClassZonePos[zoneStr] % zoneNum
	//	volArgArgs.Zone = zones[zoneIndex]
	//	storageClassZonePos[zoneStr]++
	//}
	//volArgArgs.Region, ok = volOptions["region"]
	//if !ok {
	//	volArgArgs.Region = GlobalConfigVar.Region
	//}

	volArgArgs.NodeSelected, _ = volOptions[NodeSchedueTag]

	// disk Type
	diskType, err := validateDiskType(volOptions)
	if err != nil {
		return nil, fmt.Errorf("Illegal required parameter type: " + volArgArgs.Type)
	}
	volArgArgs.Type = diskType
	pls, err := validateDiskPerformaceLevel(volOptions)
	if err != nil {
		return nil, err
	}
	volArgArgs.PerformanceLevel = pls

	// diskTags
	diskTags, ok := volOptions["tags"]
	if ok {
		keyRegex := regexp.MustCompile("[^a-zA-Z0-9\u4e00-\u9fa5=._/@]")
		valueRegex := regexp.MustCompile("[^a-zA-Z0-9\u4e00-\u9fa5=._/@(){}]")

		for _, tag := range strings.Split(diskTags, ",") {
			k, v, found := strings.Cut(tag, ":")
			if !found {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid diskTags format name: %s tags: %s", req.GetName(), diskTags)
			}
			if len(k) > 128 || len(v) > 256 {
				return nil, status.Errorf(codes.InvalidArgument, "key or value length exceeds the limit，key: %s value: %s", k, v)
			}
			if k == "" || v == "" {
				return nil, status.Errorf(codes.InvalidArgument, "key or value is nil，key: %s value: %s", k, v)
			}
			if keyRegex.MatchString(k) || valueRegex.MatchString(v) {
				return nil, status.Errorf(codes.InvalidArgument, "key or value contains unsupported characters, key: %s value: %s", k, v)
			}
			volArgArgs.DiskTags[k] = v
		}
	}
	if len(volArgArgs.DiskTags) > 5 {
		return nil, errors.New("the number of labels cannot exceed 5")
	}
	//TODO: 将PV信息作为diskTags

	return volArgArgs, nil
}

func validateDiskType(opts map[string]string) (diskType string, err error) {

	if strings.Contains(opts["type"], ",") {
		orderedList := []string{}
		for _, cusType := range strings.Split(opts["type"], ",") {
			if _, ok := CustomDiskTypes[cusType]; ok {
				orderedList = append(orderedList, cusType)
			} else {
				return diskType, fmt.Errorf("Illegal required parameter type: " + cusType)
			}
		}
		diskType = strings.Join(orderedList, ",")
		return
	}
	for _, t := range AvailableVolumeTypes {
		if opts["type"] == t {
			diskType = t
		}
	}
	if diskType == "" {
		return diskType, fmt.Errorf("Illegal required parameter type: " + opts["type"])
	}
	return
}

func validateDiskPerformaceLevel(opts map[string]string) (performaceLevel string, err error) {
	pl, ok := opts[ESSD_PERFORMANCE_LEVEL]
	if !ok || pl == "" {
		return "", nil
	}
	klog.Infof("validateDiskPerformaceLevel: pl: %v", pl)
	if strings.Contains(pl, ",") {
		for _, cusPer := range strings.Split(pl, ",") {
			if _, ok := CustomDiskPerfermance[cusPer]; !ok {
				return "", fmt.Errorf("illegal performace level type: %s", cusPer)
			}
		}
	}
	return pl, nil
}

func getCsiVolumeInfo(diskType string, volumeId string, size int64, volumeContext map[string]string, zone string) *csi.Volume {
	accessibleTopology := []*csi.Topology{
		{
			Segments: map[string]string{
				util.NodeRegionKey:                      strings.TrimSuffix(zone[:len(zone)-1], "-"),
				util.NodeZoneKey:                        zone,
				fmt.Sprintf(nodeStorageLabel, diskType): "available",
			},
		},
	}
	if diskType != "" {
		//Add PV Label
		diskTypePL := diskType
		if diskType == ESSD {
			if pl, ok := volumeContext[ESSD_PERFORMANCE_LEVEL]; ok && pl != "" {
				diskTypePL = fmt.Sprintf("%s,%s", ESSD, pl)
			} else {
				diskTypePL = fmt.Sprintf("%s.%s", ESSD, "PL1")
			}
		}
		volumeContext[labelAppendPrefix+labelVolumeType] = diskTypePL

		// Add PV NodeAffinity
		labelKey := fmt.Sprintf(nodeStorageLabel, diskType)
		expressions := []v1.NodeSelectorRequirement{{
			Key:      labelKey,
			Operator: v1.NodeSelectorOpIn,
			Values:   []string{"available"},
		}}
		terms := []v1.NodeSelectorTerm{{
			MatchExpressions: expressions,
		}}
		diskTypeTopo := &v1.NodeSelector{
			NodeSelectorTerms: terms,
		}
		diskTypeTopoBytes, _ := json.Marshal(diskTypeTopo)
		volumeContext[annAppendPrefix+annVolumeTopoKey] = string(diskTypeTopoBytes)
	}

	klog.V(5).Infof("volumeCreate: volumeContext: %+v", volumeContext)
	tmpVol := &csi.Volume{
		CapacityBytes:      size,
		VolumeId:           volumeId,
		VolumeContext:      volumeContext,
		AccessibleTopology: accessibleTopology,
	}

	return tmpVol
}

func deleteEmpty(s []string) []string {
	var a []string
	for _, str := range s {
		if str != "" {
			a = append(a, str)
		}
	}
	return a
}

func intersect(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	for _, v := range slice1 {
		m[v]++
	}
	for _, v := range slice2 {
		times, _ := m[v]
		if times == 1 {
			nn = append(nn, v)
		}
	}
	return nn
}

func UpdateNode(nodes core_v1.NodeInterface, instanceID string) {
	ctx, cancel := context.WithTimeout(context.Background(), UpdateNodeTimeout)
	defer cancel()
	nodeName := os.Getenv(KubeNodeName)
	nodeInfo, err := nodes.Get(ctx, nodeName, meta_v1.GetOptions{})
	if err != nil {
		klog.Errorf("UpdateNode:: get node info error : %s", err.Error())
	}
	instanceType := nodeInfo.Labels[instanceTypeLabel]
	instanceRegion := nodeInfo.Labels[NodeRegionKey]
	instanceZone := nodeInfo.Labels[NodeZoneKey]

	if instanceType == "" || instanceRegion == "" || instanceZone == "" {
		//There will be a certain delay in obtaining the instance ID using annotations,
		//resulting in failure to obtain the instance model.
		//instanceID := nodeInfo.Annotations[InstanceUuid]
		instanceInfo, err := GetInstanceInfo(instanceID)
		instanceType = instanceInfo.InstanceType
		instanceZone = instanceInfo.AvailabilityZone
		instanceRegion = strings.TrimSuffix(instanceZone[:len(instanceZone)-1], "-")
		if err != nil {
			return
		}
		//instanceType = nodeInfo.Labels[KceLabelZoneKey]
		//zone = nodeInfo.Labels[KecLabelZoneKey]
	}

	instanceRegionLables := map[string]string{
		NodeRegionKey: instanceRegion,
		NodeZoneKey:   instanceZone,
	}

	instanceStorageLabels := map[string]string{}
	if instanceType != "" {
		diskTypes, err := GetAvailableDiskTypes(instanceType)
		if err != nil {
			klog.Errorf("UpdateNode:: failed to get available disk types: %v", err)
		} else {
			for _, diskType := range diskTypes {
				labelKey := fmt.Sprintf(nodeStorageLabel, diskType)
				instanceStorageLabels[labelKey] = "available"
			}
		}
	} else {
		klog.Warningf("UpdateNode:: instaceType or zone is empty, skipping disk label update, instanceType: %s, zone: %s")
	}

	needUpdate := false
	for l, v := range instanceStorageLabels {
		if nodeInfo.Labels[l] != v {
			needUpdate = true
			break
		}
	}

	needUpdateRegion := false
	for l, v := range instanceRegionLables {
		if nodeInfo.Labels[l] != v {
			needUpdateRegion = true
			break
		}
	}

	if !needUpdate && !needUpdateRegion {
		klog.Infof("UpdateNode:: no need to update node")
		return
	}
	patch, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": instanceStorageLabels,
		},
	})
	if err != nil {
		klog.Errorf("UpdateNode:: failed to marshal patch json")
	}

	patchRegion, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": instanceRegionLables,
		},
	})
	if err != nil {
		klog.Errorf("UpdateNode:: failed to marshal patchRegion json")
	}

	backoff := wait.Backoff{
		Duration: time.Second,
		Factor:   2.,
		Steps:    9.,
	}

	if needUpdateRegion {
		for {
			_, err := nodes.Patch(ctx, nodeName, types.StrategicMergePatchType, patchRegion, meta_v1.PatchOptions{})
			if err == nil {
				break
			}
			klog.Errorf("UpdateNode:: failed to update node status: %v", err)
			if errors.Is(err, ctx.Err()) {
				return
			}
			time.Sleep(backoff.Step())
		}
	}

	if needUpdate {
		for {
			_, err := nodes.Patch(ctx, nodeName, types.StrategicMergePatchType, patch, meta_v1.PatchOptions{})
			if err == nil {
				break
			}
			klog.Errorf("UpdateNode:: failed to update node status: %v", err)
			if errors.Is(err, ctx.Err()) {
				return
			}
			time.Sleep(backoff.Step())
		}
	}

	klog.V(5).Infof("UpdateNode:: finished")
}

func GetProjectId() (ProjectId []int, err error) {
	cli := OpenApi.New(GlobalConfigVar.OpenApiConfig)
	AccountAllProjectListResp := &AccountAllProjectListResp{}

	resp, err := cli.DoRequest("iam", "Action=GetAccountAllProjectList", "")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &AccountAllProjectListResp)
	if err != nil {
		klog.Error("Error decoding json: ", err)
		return nil, err
	}
	ListProjectResult := AccountAllProjectListResp.ListProjectResult
	ProjectList := ListProjectResult.ProjectList

	for _, Project := range ProjectList {
		ProjectId = append(ProjectId, Project.ProjectID)
	}

	return ProjectId, nil
}

func GetInstanceInfo(InstanceId string) (*InstancesSet, error) {
	ProjectIds, err := GetProjectId()
	if err != nil {
		return nil, err
	}

	cli := OpenApi.New(GlobalConfigVar.OpenApiConfig)
	DescribeInstancesResp := &DescribeInstancesResp{}
	payload := fmt.Sprintf("InstanceId.1=%s", InstanceId)
	for n, ProjectId := range ProjectIds {
		payload = payload + fmt.Sprintf("&ProjectId.%v=%v", n+1, ProjectId)
	}

	resp, err := cli.DoRequest("kec", "Action=DescribeInstances", payload)
	if err != nil {
		klog.Errorf("GetInstanceType::payload is %v, err is %v", payload, err)
		return nil, err
	}

	err = json.Unmarshal(resp, &DescribeInstancesResp)
	if err != nil {
		klog.Error("Error decoding json: ", err)
		return nil, err
	}

	if DescribeInstancesResp.InstanceCount == 0 {
		return nil, fmt.Errorf("this instance was not found,please confirm whether it exists")
	}
	instancesInfo := DescribeInstancesResp.InstancesSet[0]
	return &instancesInfo, nil
}

func GetAvailableDiskTypes(instanceType string) (DataDiskTypes []string, err error) {
	cli := OpenApi.New(GlobalConfigVar.OpenApiConfig)
	DescribeInstanceTypeConfigsResp := &DescribeInstanceTypeConfigsResp{}
	payloads := fmt.Sprintf("Filter.1.Name.1=instance-type&Filter.1.Value.1=%s", instanceType)

	resp, err := cli.DoRequest("kec", "Action=DescribeInstanceTypeConfigs", payloads)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &DescribeInstanceTypeConfigsResp)
	if err != nil {
		klog.Error("Error decoding json: ", err)
		return nil, err
	}
	InstanceTypeConfigSet := DescribeInstanceTypeConfigsResp.InstanceTypeConfigSet
	DataDiskQuotaSet := InstanceTypeConfigSet[0].DataDiskQuotaSet

	for _, DataDiskQuota := range DataDiskQuotaSet {
		DataDiskTypes = append(DataDiskTypes, DataDiskQuota.DataDiskType)
	}

	return DataDiskTypes, nil
}

func getVolumeCount(instanceID string) (int64, error) {
	var availableVolumeCount int
	instanceInfo, err := GetInstanceInfo(instanceID)
	instanceType := instanceInfo.InstanceType
	if err != nil {
		return DefaultMaxVolumesPerNode, err
	}

	cli := OpenApi.New(GlobalConfigVar.OpenApiConfig)
	DescribeInstanceTypeConfigsResp := &DescribeInstanceTypeConfigsResp{}
	payloads := fmt.Sprintf("Filter.1.Name.1=instance-type&Filter.1.Value.1=%s", instanceType)

	resp, err := cli.DoRequest("kec", "Action=DescribeInstanceTypeConfigs", payloads)
	if err != nil {
		return DefaultMaxVolumesPerNode, err
	}

	err = json.Unmarshal(resp, &DescribeInstanceTypeConfigsResp)
	if err != nil {
		klog.Error("Error decoding json: ", err)
		return DefaultMaxVolumesPerNode, err
	}
	InstanceTypeConfigSet := DescribeInstanceTypeConfigsResp.InstanceTypeConfigSet
	DataDiskQuotaSet := InstanceTypeConfigSet[0].DataDiskQuotaSet

	//Do not add statements within the following loop.
Loop:
	for _, DataDiskQuota := range DataDiskQuotaSet {
		for _, volumeType := range AvailableVolumeTypes {
			if DataDiskQuota.DataDiskType == volumeType {
				availableVolumeCount = DataDiskQuota.DataDiskCount
				break Loop
			}
		}
	}
	return int64(availableVolumeCount), nil
}
