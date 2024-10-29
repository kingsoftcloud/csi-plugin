package ebsClient

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

type StorageService interface {
	GetVolume(*ListVolumesReq) (*Volume, error)
	ListVolumes(*ListVolumesReq) (*ListVolumesResp, error)
	CreateVolume(*CreateVolumeReq) (*CreateVolumeResp, error)
	DeleteVolume(*DeleteVolumeReq) (*DeleteVolumeResp, error)
	ExpandVolume(*ExpandVolumeReq) (*ExpandVolumeResp, error)

	Attach(*AttachVolumeReq) (*AttachVolumeResp, error)
	Detach(*DetachVolumeReq) (*DetachVolumeResp, error)

	ValidateAttachInstance(*ValidateAttachInstanceReq) (*ValidateAttachInstanceResp, error)
	GetVolumeByName(getVolumesReq *ListVolumesReq) (*ListVolumesResp, error)
	DescribeInstanceVolumes(describeInstanceVolumesReq *DescribeInstanceVolumesReq) (*InstanceVolumes, error)

	CreateSnapshot(*CreateSnapshotReq) (*CreateSnapshotResp, error)
	GetSnapshot(*DescribeSnapshotsReq) (*Snapshot, error)
	ListSnapshots(resp *DescribeSnapshotsReq) (*DescribeSnapshotsResp, error)
	GetSnapshotsByName(*DescribeSnapshotsReq) (*DescribeSnapshotsResp, int, error)
	DeleteSnapshots(req *DeleteSnapshotsReq) (*DeleteSnapshotsResp, error)
	//DescribeSnapshots(describeSnapshotsReq *DescribeSnapshotsReq)(*)
}

type VolumeStatusType string

var VolumeTypes = []string{SSD2_0, SSD3_0, SATA3_0, EHDD, ESSD_PL1, ESSD_PL2, ESSD_PL3, ESSD_PL0}

const (
	Separator = "&"

	// volume type
	SSD2_0   string = "SSD2.0"
	SSD3_0   string = "SSD3.0"
	SATA3_0  string = "SATA3.0"
	EHDD     string = "EHDD"
	ESSD_PL0 string = "ESSD_PL0"
	ESSD_PL1 string = "ESSD_PL1"
	ESSD_PL2 string = "ESSD_PL2"
	ESSD_PL3 string = "ESSD_PL3"

	VolumeTypesRegexp = "^(SSD2.0|SSD3.0|SATA3.0|ESSD_PL0|ESSD_PL1|ESSD_PL2|ESSD_PL3|EHDD)$"

	// volume size
	// 单位 GB
	MIN_VOLUME_SIZE int64 = 10
	MAX_VOLUME_SIZE int64 = 32000

	// charge type
	MONTHLY_CHARGE_TYPE                   string = "Monthly"
	HOURLY_INSTANT_SETTLEMENT_CHARGE_TYPE string = "HourlyInstantSettlement"
	DAILY_CHARGE_TYPE                     string = "Daily"

	// volumecategory
	DATA_VOlUME_CATE   string = "data"
	SYSTEM_VOlUME_CATE string = "system"

	// volume status
	CREATING_STATUS  VolumeStatusType = "creating"
	AVAILABLE_STATUS VolumeStatusType = "available"
	ATTACHING_STATUS VolumeStatusType = "attaching"
	INUSE_STATUS     VolumeStatusType = "in-use"
	DETACHING_STATUS VolumeStatusType = "detaching"
	EXTENDING_STATUS VolumeStatusType = "extending"
	DELETING_STATUS  VolumeStatusType = "deleting"
	ERROR_STATUS     VolumeStatusType = "error"
)

type InstanceVolumes struct {
	RequestId   string        `json:"RequestId"`
	Attachments []*Attachment `json:"Attachments"`
}

type Volume struct {
	VolumeId           string           `json:"VolumeId"`
	VolumeName         string           `json:"VolumeName"`
	VolumeDesc         string           `json:"VolumeDesc"`
	Size               int64            `json:"Size"`
	VolumeStatus       VolumeStatusType `json:"VolumeStatus"`
	VolumeType         string           `json:"VolumeType"`
	VolumeCategory     string           `json:"VolumeCategory"`
	InstanceId         string           `json:"InstanceId"` //云硬盘状态为in-use时，该云硬盘关联的实例ID（主机ID）
	CreateTime         string           `json:"CreateTime"`
	AvailabilityZone   string           `json:"AvailabilityZone"`
	ProjectId          int              `json:"ProjectId"`
	DeleteWithInstance bool             `json:"DeleteWithInstance"`
	Attachments        []*Attachment    `json:"Attachment"` //硬盘的当前挂载信息
}

type Attachment struct {
	InstanceId string `json:"InstanceId"`
	VolumeId   string `json:"VolumeId"`
	MountPoint string `json:"MountPoint"`
}

type CreateVolumeReq struct {
	VolumeName       string
	VolumeType       string
	VolumeDesc       string
	SnapshotId       string
	Size             int64
	AvailabilityZone string
	ChargeType       string
	PurchaseTime     int
	ProjectId        string
	Tags             map[string]string
}

func (cv *CreateVolumeReq) ToQuery() string {
	querySlice := []string{"Action=CreateVolume"}

	if cv.VolumeName != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeName=%v", cv.VolumeName))
	}
	if cv.VolumeType != "" {
		for _, vt := range VolumeTypes {
			if cv.VolumeType != vt {
				continue
			}
			querySlice = append(querySlice, fmt.Sprintf("VolumeType=%v", cv.VolumeType))
			break
		}
	}

	if cv.VolumeDesc != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeDesc=%v", cv.VolumeDesc))
	}

	if cv.AvailabilityZone != "" {
		querySlice = append(querySlice, fmt.Sprintf("AvailabilityZone=%v", cv.AvailabilityZone))
	}

	if cv.ChargeType != "" {
		for _, chargeType := range []string{
			MONTHLY_CHARGE_TYPE, HOURLY_INSTANT_SETTLEMENT_CHARGE_TYPE, DAILY_CHARGE_TYPE,
		} {
			if cv.ChargeType != chargeType {
				continue
			}
			querySlice = append(querySlice, fmt.Sprintf("ChargeType=%v", cv.ChargeType))
			if cv.PurchaseTime != 0 {
				querySlice = append(querySlice, fmt.Sprintf("PurchaseTime=%v", cv.PurchaseTime))
			}
			break
		}
	}

	if cv.ProjectId != "" {
		querySlice = append(querySlice, fmt.Sprintf("ProjectId=%v", cv.ProjectId))
	}

	if cv.SnapshotId != "" {
		querySlice = append(querySlice, fmt.Sprintf("SnapshotId=%v", cv.SnapshotId))
	}

	if cv.Size <= MIN_VOLUME_SIZE {
		cv.Size = MIN_VOLUME_SIZE
	}
	if cv.Size >= MAX_VOLUME_SIZE {
		cv.Size = MAX_VOLUME_SIZE
	}
	querySlice = append(querySlice, fmt.Sprintf("Size=%v", cv.Size))
	//Tag.1.Key=123&Tag.1.Value=456&Tag.2.Key=Usage&Tag.2.Value=test123'
	if len(cv.Tags) > 0 {
		i := 1
		for k, v := range cv.Tags {
			if len(v) == 0 {
				klog.V(5).Infof("Invalid tag: key=%s, value=%s", k, v)
				continue
			}
			querySlice = append(querySlice, fmt.Sprintf("Tag.%d.Key=%s", i, k))
			querySlice = append(querySlice, fmt.Sprintf("Tag.%d.Value=%s", i, v))
			i++
		}
	}
	return strings.Join(querySlice, Separator)
}

type CreateVolumeResp struct {
	RequestId string `json:"RequestId"`
	VolumeId  string `json:"VolumeId"`
}

type DeleteVolumeReq struct {
	VolumeId string
}

func (dv *DeleteVolumeReq) ToQuery() string {
	querySlice := []string{"Action=DeleteVolume", fmt.Sprintf("VolumeId=%v", dv.VolumeId)}
	return strings.Join(querySlice, Separator)
}

type DeleteVolumeResp struct {
	RequestId string `json:"RequestId"`
	Return    bool   `json:"Return"`
}

type ExpandVolumeReq struct {
	RequestId    string `json:"RequestId"`
	VolumeId     string `json:"VolumeId"`
	Size         int64  `json:"Size"` //GB
	OnlineResize bool   `json:"OnlineResize"`
}

type ExpandVolumeResp struct {
	RequestId string `json:"RequestId"`
	Return    bool   `json:"Return"`
}

func (ev *ExpandVolumeReq) ToQuery() string {
	querySlice := []string{"Action=ResizeVolume"}
	querySlice = append(querySlice, fmt.Sprintf("VolumeId=%s", ev.VolumeId))
	querySlice = append(querySlice, fmt.Sprintf("Size=%d", ev.Size))
	querySlice = append(querySlice, fmt.Sprintf("OnlineResize=%t", ev.OnlineResize))

	return strings.Join(querySlice, Separator)
}

type ListVolumesReq struct {
	VolumeIds        []string
	VolumeCategory   string
	VolumeStatus     string
	VolumeType       string
	VolumeCreateDate string
	VolumeExactName  string
}

func (lv *ListVolumesReq) ToQuery() string {
	querySlice := []string{"Action=DescribeVolumes"}
	if lv.VolumeIds != nil && len(lv.VolumeIds) > 0 {
		for i, VolumeId := range lv.VolumeIds {
			querySlice = append(querySlice, fmt.Sprintf("VolumeId.%v=%v", i+1, VolumeId))
		}
	}
	for _, volumeCategory := range []string{DATA_VOlUME_CATE, SYSTEM_VOlUME_CATE} {
		if lv.VolumeCategory != volumeCategory {
			continue
		}
		querySlice = append(querySlice, fmt.Sprintf("VolumeCategory=%v", lv.VolumeCategory))
		break
	}
	if len(lv.VolumeStatus) > 0 {
		querySlice = append(querySlice, fmt.Sprintf("VolumeStatus=%v", lv.VolumeStatus))
	}

	for _, vt := range VolumeTypes {
		if lv.VolumeType != vt {
			continue
		}
		querySlice = append(querySlice, fmt.Sprintf("VolumeType=%v", lv.VolumeType))
		break
	}
	if len(lv.VolumeCreateDate) > 0 {
		querySlice = append(querySlice, fmt.Sprintf("VolumeCreateDate=%v", lv.VolumeCreateDate))
	}

	if len(lv.VolumeExactName) > 0 {
		querySlice = append(querySlice, fmt.Sprintf("VolumeExactName=%v", lv.VolumeExactName))
	}

	return strings.Join(querySlice, Separator)
}

type ListVolumesResp struct {
	RequestId  string    `json:"RequestId"`
	Volumes    []*Volume `json:"Volumes"`
	TotalCount int       `json:"TotalCount"`
}

type AttachVolumeReq struct {
	VolumeId           string
	InstanceId         string
	DeleteWithInstance bool
}

func (av *AttachVolumeReq) ToQuery() string {
	querySlice := []string{"Action=AttachVolume"}
	if av.VolumeId != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeId=%v", av.VolumeId))
	}
	if av.InstanceId != "" {
		querySlice = append(querySlice, fmt.Sprintf("InstanceId=%v", av.InstanceId))
	}
	querySlice = append(querySlice, fmt.Sprintf("DeleteWithInstance=%v", av.DeleteWithInstance))

	return strings.Join(querySlice, Separator)
}

type AttachVolumeResp struct {
	RequestId  string `json:"RequestId"`
	Return     bool   `json:"Return"`
	MountPoint string `json:"MountPoint"`
}

type DetachVolumeReq struct {
	VolumeId   string
	InstanceId string
}

func (dv *DetachVolumeReq) ToQuery() string {
	querySlice := []string{"Action=DetachVolume"}
	if dv.VolumeId != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeId=%v", dv.VolumeId))
	}
	if dv.InstanceId != "" {
		querySlice = append(querySlice, fmt.Sprintf("InstanceId=%v", dv.InstanceId))
	}

	return strings.Join(querySlice, Separator)
}

type DetachVolumeResp struct {
	RequestId string `json:"RequestId"`
	Return    bool   `json:"Return"`
}

type ValidateAttachInstanceReq struct {
	VolumeType string
	InstanceId string
}

func (va *ValidateAttachInstanceReq) ToQuery() string {
	querySlice := []string{"Action=ValidateAttachInstance"}
	if va.VolumeType != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeType=%v", va.VolumeType))
	}
	if va.InstanceId != "" {
		querySlice = append(querySlice, fmt.Sprintf("InstanceId=%v", va.InstanceId))
	}

	return strings.Join(querySlice, Separator)
}

type ValidateAttachInstanceResp struct {
	RequestId          string `json:"RequestId"`
	InstanceEnable     bool   `json:"InstanceEnable"`
	InstanceState      string `json:"InstanceState"`
	LargeVolumeSupport bool   `json:"LargeVolumeSupport"`
	AvailableVolumeNum int    `json:"AvailableVolumeNum"`
}

type DescribeInstanceVolumesReq struct {
	InstanceId string
}

func (va *DescribeInstanceVolumesReq) ToQuery() string {
	querySlice := []string{"Action=DescribeInstanceVolumes"}
	if va.InstanceId != "" {
		querySlice = append(querySlice, fmt.Sprintf("InstanceId=%v", va.InstanceId))
	}
	return strings.Join(querySlice, Separator)
}

type Snapshot struct {
	SnapshotID       string `json:"SnapshotId"`
	SnapshotName     string `json:"SnapshotName"`
	VolumeID         string `json:"VolumeId"`
	Size             int    `json:"Size"`
	CreateTime       string `json:"CreateTime"`
	SnapshotStatus   string `json:"SnapshotStatus"`
	VolumeCategory   string `json:"VolumeCategory"`
	VolumeName       string `json:"VolumeName"`
	VolumeType       string `json:"VolumeType"`
	Progress         string `json:"Progress"`
	AvailabilityZone string `json:"AvailabilityZone"`
	VolumeStatus     string `json:"VolumeStatus"`
	SnapshotType     string `json:"SnapshotType"`
}

type CreateSnapshotParams struct {
	VolumeID            string
	SnapshotType        string
	SnapshotName        string
	ScheduledDeleteTime string
	SnapShotDesc        string
	AutoSnapshot        bool
	RetentionDays       int
}

type DescribeSnapshotsReq struct {
	VolumeId         string
	VolumeCategory   string
	SnapshotId       string
	AvailabilityZone string
	SnapshotName     string
	InstanceId       string
}

type CreateSnapshotReq struct {
	VolumeId            string
	SnapshotName        string
	SnapshotDesc        string
	SnapshotType        string
	AutoSnapshot        string
	ScheduledDeleteTime string
}

type CreateSnapshotResp struct {
	RequestID  string `json:"RequestId"`
	SnapshotID string `json:"SnapshotId"`
}

type DescribeSnapshotsResp struct {
	RequestId string      `json:"RequestId"`
	Snapshots []*Snapshot `json:"Snapshots"`
}

func (cs *CreateSnapshotReq) ToQuery() string {
	querySlice := []string{"Action=CreateSnapshot"}
	if cs.VolumeId != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeId=%v", cs.VolumeId))
	}
	if cs.SnapshotType != "" {
		querySlice = append(querySlice, fmt.Sprintf("SnapshotType=%v", cs.SnapshotType))
	}
	if cs.SnapshotName != "" {
		querySlice = append(querySlice, fmt.Sprintf("SnapshotName=%v", cs.SnapshotName))
	}
	if cs.SnapshotDesc != "" {
		querySlice = append(querySlice, fmt.Sprintf("SnapshotDesc=%v", cs.SnapshotDesc))
	}
	if cs.ScheduledDeleteTime != "" {
		querySlice = append(querySlice, fmt.Sprintf("ScheduledDeleteTime=%v", cs.ScheduledDeleteTime))
	}
	return strings.Join(querySlice, Separator)
}

func (ds *DescribeSnapshotsReq) ToQuery() string {
	querySlice := []string{"Action=DescribeSnapshots"}
	if ds.VolumeId != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeId=%v", ds.VolumeId))
	}
	if ds.VolumeCategory != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeCategory=%v", ds.VolumeCategory))
	}
	if ds.SnapshotId != "" {
		querySlice = append(querySlice, fmt.Sprintf("SnapshotId=%v", ds.SnapshotId))
	}
	if ds.AvailabilityZone != "" {
		querySlice = append(querySlice, fmt.Sprintf("AvailabilityZone=%v", ds.AvailabilityZone))
	}
	if ds.SnapshotName != "" {
		querySlice = append(querySlice, fmt.Sprintf("SnapshotName=%v", ds.SnapshotName))
	}
	if ds.InstanceId != "" {
		querySlice = append(querySlice, fmt.Sprintf("InstanceId=%v", ds.InstanceId))
	}
	return strings.Join(querySlice, Separator)
}

type DeleteSnapshotsReq struct {
	SnapshotId string
}

func (ds *DeleteSnapshotsReq) ToQuery() string {
	querySlice := []string{"Action=DeleteSnapshots", fmt.Sprintf("SnapshotId=%v", ds.SnapshotId)}
	return strings.Join(querySlice, Separator)
}

type DeleteSnapshotsResp struct {
	RequestId string `json:"RequestId"`
	Return    bool   `json:"Return"`
}
