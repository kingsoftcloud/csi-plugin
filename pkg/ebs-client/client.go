package ebsClient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"regexp"

	api "csi-plugin/pkg/open-api"

	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	serviceName = "ebs"
)

type Client struct {
	*api.Client
}

func New(config *api.ClientConfig) *Client {
	return &Client{
		Client: api.New(config),
	}
}

func (cli *Client) CreateVolume(createVolumeReq *CreateVolumeReq) (*CreateVolumeResp, error) {
	if err := ValidateCreateVolumeReq(createVolumeReq); err != nil {
		return nil, err
	}

	createVolumeResp := &CreateVolumeResp{}
	query := createVolumeReq.ToQuery()
	resp, err := cli.DoRequest(serviceName, query)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &createVolumeResp)
	if err != nil {
		glog.Error("Error decoding json: ", err)
		return nil, err
	}

	return createVolumeResp, nil
}

func (cli *Client) DeleteVolume(deleteVolumeReq *DeleteVolumeReq) (*DeleteVolumeResp, error) {
	// query := "Action=DeleteVolume&Version=2016-03-04&VolumeId=" + volume_id
	deleteVolumeResp := &DeleteVolumeResp{}
	query := deleteVolumeReq.ToQuery()
	resp, err := cli.DoRequest(serviceName, query)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, deleteVolumeResp)
	if err != nil {
		glog.Error("Error decoding json: ", err)
		return nil, err
	}
	if !deleteVolumeResp.Return {
		return nil, errors.New("DeleteVolume return False")
	}
	return deleteVolumeResp, nil
}

func (cli *Client) ListVolumes(listVolumesReq *ListVolumesReq) (*ListVolumesResp, error) {
	for _, vid := range listVolumesReq.VolumeIds {
		if !validateReqParams(VolumeIdRegexp, vid) {
			return nil, status.Errorf(codes.InvalidArgument, "VolumeId (%v) is invalid", vid)
		}
	}
	listVolumesResp := &ListVolumesResp{}

	query := listVolumesReq.ToQuery()
	resp, err := cli.DoRequest(serviceName, query)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resp, listVolumesResp); err != nil {
		return nil, err
	}

	return listVolumesResp, nil
}

func (cli *Client) GetVolume(listVolumesReq *ListVolumesReq) (*Volume, error) {
	listVolumesResp, err := cli.ListVolumes(listVolumesReq)
	if err != nil {
		return nil, err
	}
	if len(listVolumesResp.Volumes) == 0 {
		return nil, errors.New("not found volume")
	}
	return listVolumesResp.Volumes[0], nil
}

//TODO
func (cli *Client) ExpandVolume(expandVolumeReq *ExpandVolumeReq) (*ExpandVolumeResp, error) {
	query := expandVolumeReq.ToQuery()
	resp, err := cli.DoRequest(serviceName, query)
	if err != nil {
		return nil, err
	}
	expandVolumeResp := &ExpandVolumeResp{}
	if err := json.Unmarshal(resp, expandVolumeResp); err != nil {
		return nil, err
	}

	return expandVolumeResp, nil
}

func (cli *Client) Attach(attachVolumeReq *AttachVolumeReq) (*AttachVolumeResp, error) {
	attachVolumeResp := &AttachVolumeResp{}

	query := attachVolumeReq.ToQuery()
	resp, err := cli.DoRequest(serviceName, query)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resp, attachVolumeResp); err != nil {
		return nil, err
	}
	if !attachVolumeResp.Return {
		return nil, errors.New("Attach return False")
	}
	return attachVolumeResp, nil
}

func (cli *Client) Detach(detachVolumeReq *DetachVolumeReq) (*DetachVolumeResp, error) {
	detachVolumeResp := &DetachVolumeResp{}

	query := detachVolumeReq.ToQuery()
	resp, err := cli.DoRequest(serviceName, query)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resp, detachVolumeResp); err != nil {
		return nil, err
	}
	if !detachVolumeResp.Return {
		return nil, errors.New("Detach return False")
	}
	return detachVolumeResp, nil
}

func (cli *Client) ValidateAttachInstance(validateAttachInstanceReq *ValidateAttachInstanceReq) (*ValidateAttachInstanceResp, error) {
	if !validateReqParams(VolumeTypeRegexp, validateAttachInstanceReq.VolumeType) {
		return nil, status.Errorf(codes.InvalidArgument, "Volume type (%v) is invalid", validateAttachInstanceReq.VolumeType)
	}

	validateAttachInstanceResp := &ValidateAttachInstanceResp{}
	query := validateAttachInstanceReq.ToQuery()
	resp, err := cli.DoRequest(serviceName, query)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resp, validateAttachInstanceResp); err != nil {
		return nil, err
	}
	return validateAttachInstanceResp, nil
}

func WaitVolumeStatus(storageService StorageService, volumeId string, targetStatus VolumeStatusType) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			listVolumesReq := &ListVolumesReq{
				VolumeIds: []string{volumeId},
			}
			listVolumesResp, err := storageService.ListVolumes(listVolumesReq)
			if err != nil {
				glog.Errorf("waitVolumeStatus:ListVolumes %v error: %v", volumeId, err)
				continue
			}
			if len(listVolumesResp.Volumes) == 0 {
				glog.Errorf("waitVolumeStatus:ListVolumes error: volume %v not found", volumeId)
				continue
			}
			vol := listVolumesResp.Volumes[0]
			glog.Infof("wating for volume status: %v, current status: %v", targetStatus, vol.VolumeStatus)
			if vol.VolumeStatus == targetStatus {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("timeout occured waiting for storage action of volume: %q", volumeId)
		}

	}
}

func ValidateCreateVolumeReq(req *CreateVolumeReq) error {
	if !validateReqParams(VolumeNameRegexp, req.VolumeName) {
		return status.Errorf(codes.InvalidArgument, "Volume name (%v) is invalid", req.VolumeName)
	}
	if !validateReqParams(VolumeTypeRegexp, req.VolumeType) {
		return status.Errorf(codes.InvalidArgument, "Volume type (%v) is invalid", req.VolumeType)
	}
	if !validateReqParams(VolumeDescRegexp, req.VolumeDesc) {
		return status.Errorf(codes.InvalidArgument, "Volume desc (%v) is invalid", req.VolumeDesc)
	}
	if !validateReqParams(AvailabilityZoneRegexp, req.AvailabilityZone) {
		return status.Errorf(codes.InvalidArgument, "AvailabilityZone (%v) is invalid", req.AvailabilityZone)
	}
	if !validateReqParams(ChargeTypeRegexp, req.ChargeType) {
		return status.Errorf(codes.InvalidArgument, "ChargeType (%v) is invalid", req.ChargeType)
	}
	if req.ChargeType == MONTHLY_CHARGE_TYPE || req.ChargeType == DAILY_CHARGE_TYPE {
		if !validateReqParams(PurchaseTimeRegexp, strconv.Itoa(req.PurchaseTime)) {
			return status.Errorf(codes.InvalidArgument, "purchase time (%v) is invalid", req.PurchaseTime)
		}
	}
	return nil
}

type RegexpType string

const (
	VolumeNameRegexp       RegexpType = "VolumeNameRegexp"
	VolumeTypeRegexp       RegexpType = "VolumeTypeRegexp"
	VolumeDescRegexp       RegexpType = "VolumeDescRegexp"
	AvailabilityZoneRegexp RegexpType = "AvailabilityZoneRegexp"
	ChargeTypeRegexp       RegexpType = "ChargeTypeRegexp"
	PurchaseTimeRegexp     RegexpType = "PurchaseTimeRegexp"
	VolumeIdRegexp         RegexpType = "VolumeIdRegexp"
)

var (
	ParamsRegexp = map[RegexpType]*regexp.Regexp{
		VolumeNameRegexp:       regexp.MustCompile(`(^$|^[a-zA-Z0-9\-_]{2,128}$)`),
		VolumeTypeRegexp:       regexp.MustCompile(fmt.Sprintf("^(%s|%s|%s)$", SSD2_0, SSD3_0, SATA2_0)),
		VolumeDescRegexp:       regexp.MustCompile(`(^$|^.{1,128}$)`),
		AvailabilityZoneRegexp: regexp.MustCompile(`^[a-zA-Z0-9\-_]+`),
		ChargeTypeRegexp: regexp.MustCompile(fmt.Sprintf("^(%s|%s|%s)$", MONTHLY_CHARGE_TYPE,
			HOURLY_INSTANT_SETTLEMENT_CHARGE_TYPE, DAILY_CHARGE_TYPE)),
		PurchaseTimeRegexp: regexp.MustCompile(`(^[1-9]$|^[1-2]\d$|^3[0-6]$)`),
		VolumeIdRegexp:     regexp.MustCompile(`^[a-zA-Z0-9\-_]{36}$`),
	}
)

func validateReqParams(regexpType RegexpType, regexpStr string) bool {
	r, ok := ParamsRegexp[regexpType]
	if !ok {
		return false
	}
	if r == nil {
		return false
	}
	return r.Match([]byte(regexpStr))
}
