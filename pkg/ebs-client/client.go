package ebsClient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	serviceName = "ebs"
)

type Client struct {
	accessKeyId     string //Access Key Id
	accessKeySecret string //Access Key Secret
	region          string
	httpClient      *http.Client

	openApiEndpoint string
	openApiPrefix   string
}

type ClientConfig struct {
	AccessKeyId     string //Access Key Id
	AccessKeySecret string //Access Key Secret
	Region          string
	OpenApiEndpoint string
	OpenApiPrefix   string
}

func New(config *ClientConfig) *Client {
	return &Client{
		accessKeyId:     config.AccessKeyId,
		accessKeySecret: config.AccessKeySecret,
		region:          config.Region,
		httpClient:      &http.Client{},
	}
}

func (cli *Client) CreateVolume(createVolumeReq *CreateVolumeReq) (*CreateVolumeResp, error) {
	if err := ValidateCreateVolumeReq(createVolumeReq); err != nil {
		return nil, err
	}

	createVolumeResp := &CreateVolumeResp{}
	query := createVolumeReq.ToQuery()
	resp, err := cli.doRequest(serviceName, query)
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
	resp, err := cli.doRequest(serviceName, query)
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
	resp, err := cli.doRequest(serviceName, query)
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

func (cli *Client) Attach(attachVolumeReq *AttachVolumeReq) (*AttachVolumeResp, error) {
	attachVolumeResp := &AttachVolumeResp{}

	query := attachVolumeReq.ToQuery()
	resp, err := cli.doRequest(serviceName, query)
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
	resp, err := cli.doRequest(serviceName, query)
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

// type KecInfo struct {
// 	InstanceId       string `json:"InstanceId"`
// 	AvailabilityZone string `json:"AvailabilityZone"`
// }

// type KecList struct {
// 	Instances []KecInfo `json:"InstancesSet"`
// }

// func (cli *Client) DescribeInstances(instance_id string) (*KecInfo, error) {
// 	query := "Action=DescribeInstances&Version=2016-03-04&InstanceId.1=" + instance_id
// 	resp, err := cli.doRequest("kec", query)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var instances KecList
// 	err = json.Unmarshal(resp, &instances)
// 	if err != nil {
// 		glog.Error("Error decoding json", err)
// 		return nil, err
// 	}
// 	return &instances.Instances[0], nil
// }

func (cli *Client) buildRequest(serviceName, body string) (*http.Request, io.ReadSeeker) {
	reader := strings.NewReader(body)
	return cli.buildRequestWithBodyReader(serviceName, reader)
}

func (cli *Client) buildRequestWithBodyReader(serviceName string, body io.Reader) (*http.Request, io.ReadSeeker) {
	var bodyLen int

	type lenner interface {
		Len() int
	}
	if lr, ok := body.(lenner); ok {
		bodyLen = lr.Len()
	}

	endpoint := "https://" + serviceName + "." + cli.openApiEndpoint
	req, _ := http.NewRequest("GET", endpoint, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if bodyLen > 0 {
		req.Header.Set("Content-Length", strconv.Itoa(bodyLen))
	}

	var seeker io.ReadSeeker
	if sr, ok := body.(io.ReadSeeker); ok {
		seeker = sr
	} else {
		seeker = aws.ReadSeekCloser(body)
	}

	return req, seeker
}

func (cli *Client) doRequest(service string, query string) ([]byte, error) {
	s := v4.Signer{Credentials: credentials.NewStaticCredentials(cli.accessKeyId, cli.accessKeySecret, "")}

	req, body := cli.buildRequest(service, "")
	req.URL.RawQuery = query
	_, err := s.Sign(req, body, service, cli.region, time.Now())
	if err != nil {
		glog.Error("Request Sign failed: ", err)
		return nil, err
	}

	glog.Info("Do HTTP Request: ", query)
	resp, err := cli.httpClient.Do(req)
	if err != nil {
		glog.Error("HTTP Request failed: ", err)
		return nil, err
	}
	statusCode := resp.StatusCode

	defer resp.Body.Close()
	res_body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		glog.Error("Get Response failed: ", err)
		return nil, err
	}

	glog.Info("OpenAPI return: ", string(res_body))

	type Error struct {
		Code    string
		Message string
	}
	type ErrorResponse struct {
		RequestID string
		Error     Error
	}

	if statusCode >= 400 && statusCode <= 599 {
		var error_resp ErrorResponse
		if err = json.Unmarshal(res_body, &error_resp); err != nil {
			glog.Error("JSON unmarshal failed:", err)
		}
		return res_body, errors.New(error_resp.Error.Message)
	}

	return res_body, nil
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
		return status.Errorf(codes.InvalidArgument, "Region (%v) is invalid", req.AvailabilityZone)
	}
	if !validateReqParams(ChargeTypeRegexp, req.ChargeType) {
		return status.Errorf(codes.InvalidArgument, "ChargeType (%v) is invalid", req.ChargeType)
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
		VolumeIdRegexp: regexp.MustCompile(`^[a-zA-Z0-9\-_]{36}$`),
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
