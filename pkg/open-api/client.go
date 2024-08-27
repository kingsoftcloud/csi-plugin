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
	"k8s.io/klog"
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
	Timeout         time.Duration
}

func New(config *ClientConfig) *Client {
	return &Client{
		accessKeyId:     config.AccessKeyId,
		accessKeySecret: config.AccessKeySecret,
		region:          config.Region,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},

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

	method := "GET"
	req, _ := http.NewRequest(method, endpoint, body)

	if serviceName == "ebs" {
		req.Header.Set("Content-Type", "application/json")
	} else if serviceName == "kec" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Accept", "application/json")
	// test
	//req.Header.Set("X-KSC-ACCOUNT-ID", "73404680")
	//t := time.Now().Unix()
	//req.Header.Set("X-KSC-REQUEST-ID", "xiangqian-test-"+strconv.Itoa(int(t)))
	//req.Header.Set("X-KSC-REGION", "cn-shanghai-3")
	//req.Header.Set("X-KSC-SOURCE", "user")

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

func (cli *Client) DoRequest(service string, query string, payloads ...string) ([]byte, error) {
	akskConfig, err := util.SetAksk()
	if err != nil {
		return nil, err
	}

	aksk, err := akskConfig.Akskprovider.GetAKSK()
	if err != nil {
		return nil, err
	}

	ak, sk := aksk.AK, aksk.SK

	if len(cli.region) == 0 {
		cli.region, _ = akskConfig.GetRegion()
	}
	s := v4.Signer{Credentials: credentials.NewStaticCredentials(ak, sk, "")}

	if service == "ebs" || service == "kec" {
		query = fmt.Sprintf("%v&Version=%v", query, Version)
	} else if service == "iam" {
		query = fmt.Sprintf("%v&Version=%v", query, "2015-11-01")
	}

	var payload string
	if len(payloads) == 0 {
		payload = ""
	} else {
		payload = payloads[0]
	}

	req, body := cli.buildRequest(service, payload)

	req.Header.Set("X-Ksc-Security-Token", aksk.SecurityToken)

	req.URL.RawQuery = query
	_, err = s.Sign(req, body, service, cli.region, time.Now())
	if err != nil {
		klog.Error("Request Sign failed: ", err)
		return nil, err
	}
	klog.V(5).Info("Do HTTP Request: ", query)
	resp, err := cli.httpClient.Do(req)
	if err != nil {
		klog.Error("HTTP Request failed: ", err)
		return nil, err
	}
	statusCode := resp.StatusCode

	defer resp.Body.Close()
	res_body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		klog.Error("Get Response failed: ", err)
		return nil, err
	}
	//TODO:
	if len(res_body) > 1024 {
		klog.V(2).Info("OpenAPI return: ", string(res_body[:1024]))
	} else {
		klog.V(2).Info("OpenAPI return: ", string(res_body))
	}

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
			klog.Error("JSON unmarshal failed:", err)
		}
		return res_body, errors.New(error_resp.Error.Message)
	}

	return res_body, nil
}
