package ebsClient

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/golang/glog"
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

func (cli *Client) CreateVolume(volume_name, volume_type, availability_zone, charge_type, project_id string, size, purchase_time int) (string, error) {
	query := "Action=CreateVolume&Version=2016-03-04&VolumeType=" + volume_type + "&AvailabilityZone=" + availability_zone + "&ChargeType=" + charge_type + "&Size=" + strconv.Itoa(size)
	if volume_name != "" {
		query = query + "&VolumeName=" + volume_name
	}
	if project_id != "" {
		query = query + "&ProjectId=" + project_id
	}
	if purchase_time != 0 {
		query = query + "&PurchaseTime=" + strconv.Itoa(purchase_time)
	}

	resp, err := cli.doRequest("ebs", query)
	if err != nil {
		return "", err
	}
	type CreateVolumeResp struct {
		VolumeId string `json:"VolumeId"`
	}
	var volume CreateVolumeResp
	err = json.Unmarshal(resp, &volume)
	if err != nil {
		glog.Error("Error decoding json: ", err)
		return "", err
	}

	return volume.VolumeId, nil
}

func (cli *Client) DeleteVolume(volume_id string) error {
	query := "Action=DeleteVolume&Version=2016-03-04&VolumeId=" + volume_id
	resp, err := cli.doRequest("ebs", query)
	if err != nil {
		return err
	}

	type DeleteVolumeResp struct {
		Return bool `json:"Return"`
	}
	var res DeleteVolumeResp
	err = json.Unmarshal(resp, &res)
	if err != nil {
		glog.Error("Error decoding json: ", err)
		return err
	}
	if !res.Return {
		return errors.New("DeleteVolume return False")
	}
	return nil
}

type KecInfo struct {
	InstanceId       string `json:"InstanceId"`
	AvailabilityZone string `json:"AvailabilityZone"`
}

type KecList struct {
	Instances []KecInfo `json:"InstancesSet"`
}

func (cli *Client) DescribeInstances(instance_id string) (*KecInfo, error) {
	query := "Action=DescribeInstances&Version=2016-03-04&InstanceId.1=" + instance_id
	resp, err := cli.doRequest("kec", query)
	if err != nil {
		return nil, err
	}

	var instances KecList
	err = json.Unmarshal(resp, &instances)
	if err != nil {
		glog.Error("Error decoding json", err)
		return nil, err
	}
	return &instances.Instances[0], nil
}
