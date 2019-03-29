package api

import (
	"csi-plugin/util"
	"encoding/json"
	"errors"
	"fmt"
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

const (
	Version = "2016-03-04"
)

type Client struct {
	accessKeyId     string
	accessKeySecret string

	region     string
	httpClient *http.Client

	openApiEndpoint string
	openApiPrefix   string
}

type ClientConfig struct {
	AccessKeyId     string
	AccessKeySecret string

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

		openApiEndpoint: config.OpenApiEndpoint,
		openApiPrefix:   config.OpenApiPrefix,
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
	endpoint := fmt.Sprintf("%v://%v.%v.%v/", cli.openApiPrefix, serviceName, cli.region, cli.openApiEndpoint)

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

func (cli *Client) DoRequest(service string, query string) ([]byte, error) {
	ak, sk := cli.accessKeyId, cli.accessKeySecret
	if ak == "" || sk == "" {
		ak, sk, _ = util.GetAKSK()
	}

	s := v4.Signer{Credentials: credentials.NewStaticCredentials(ak, sk, "")}
	query = fmt.Sprintf("%v&Version=%v", query, Version)

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
