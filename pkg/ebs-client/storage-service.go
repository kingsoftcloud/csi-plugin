package ebsClient

import (
	"fmt"
	"strings"
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
}

type VolumeStatusType string

const (
	Separator = "&"

	// volume type
	SSD2_0  string = "SSD2.0"
	SSD3_0  string = "SSD3.0"
	SATA2_0 string = "SATA2.0"

	// volume size
	// 单位 GB
	MIN_VOLUME_SIZE int64 = 10
	MAX_VOLUME_SIZE int64 = 16000

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

type Volume struct {
	VolumeId           string           `json:"VolumeId"`
	VolumeName         string           `json:"VolumeName"`
	VolumeDesc         string           `json:"VolumeDesc"`
	Size               int64            `json:"Size"`
	VolumeStatus       VolumeStatusType `json:"VolumeStatus"`
	VolumeType         string           `json:"VolumeType"`
	VolumeCategory     string           `json:"VolumeCategory"`
	InstanceId         string           `json:"InstanceId"`
	CreateTime         string           `json:"CreateTime"`
	AvailabilityZone   string           `json:"AvailabilityZone"`
	ProjectId          int              `json:"ProjectId"`
	DeleteWithInstance bool             `json:"DeleteWithInstance"`
	Attachments        []*Attachment    `json:"Attachment"`
}

type Attachment struct {
	InstanceId string `json:"InstanceId"`
	MountPoint string `json:"MountPoint"`
}

type CreateVolumeReq struct {
	VolumeName       string
	VolumeType       string
	VolumeDesc       string
	Size             int64
	AvailabilityZone string
	ChargeType       string
	PurchaseTime     int
	ProjectId        string
}

func (cv *CreateVolumeReq) ToQuery() string {
	querySlice := []string{"Action=CreateVolume"}

	if cv.VolumeName != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeName=%v", cv.VolumeName))
	}
	if cv.VolumeType != "" {
		for _, vt := range []string{SSD2_0, SSD3_0, SATA2_0} {
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
			if cv.ChargeType == MONTHLY_CHARGE_TYPE || cv.ChargeType == DAILY_CHARGE_TYPE {
				querySlice = append(querySlice, fmt.Sprintf("PurchaseTime=%v", cv.PurchaseTime))
			}
			break
		}
	}

	if cv.ProjectId != "" {
		querySlice = append(querySlice, fmt.Sprintf("ProjectId=%v", cv.ProjectId))
	}

	if cv.Size <= MIN_VOLUME_SIZE {
		cv.Size = MIN_VOLUME_SIZE
	}
	if cv.Size >= MAX_VOLUME_SIZE {
		cv.Size = MAX_VOLUME_SIZE
	}
	querySlice = append(querySlice, fmt.Sprintf("Size=%v", cv.Size))

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
	return strings.Join(querySlice, Separator)
}

type ListVolumesReq struct {
	VolumeIds        []string
	VolumeCategory   string
	VolumeStatus     string
	VolumeType       string
	VolumeCreateDate string
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
	if lv.VolumeStatus != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeStatus=%v", lv.VolumeStatus))
	}

	for _, vt := range []string{
		SSD2_0, SSD3_0, SATA2_0,
	} {
		if lv.VolumeType != vt {
			continue
		}
		querySlice = append(querySlice, fmt.Sprintf("VolumeType=%v", lv.VolumeType))
		break
	}
	if lv.VolumeCreateDate != "" {
		querySlice = append(querySlice, fmt.Sprintf("VolumeCreateDate=%v", lv.VolumeCreateDate))
	}
	return strings.Join(querySlice, Separator)
}

type ListVolumesResp struct {
	RequestId string    `json:"RequestId"`
	Volumes   []*Volume `json:"Volumes"`
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
	RequestId string `json:"RequestId"`
	Return    bool   `json:"Return"`
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
	RequestId      string `json:"RequestId"`
	InstanceEnable bool   `json:"InstanceEnable"`
}
