/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/zwei/appclient/pkg/util"
	"github.com/zwei/appclient/pkg/util/random"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"strconv"
	"strings"
)

const (
	ENDPOINT = "http://appengine.sdns.ksyun.com:80"
	CLUSTER  = "cluster"
	NODE     = "node"
	CA       = "ca"
	VROUTE   = "vroute"
	POST     = http.MethodPost
	PUT      = http.MethodPut
	UPDATE   = http.MethodPut
	GET      = http.MethodGet
	DELETE   = http.MethodDelete
)

type IAppDataClient interface {
	SetEndpoint(value string) IAppDataClient
	SetTenantId(value string) IAppDataClient
	SetInstanceId(value string) IAppDataClient
	SetMethod(value string) IAppDataClient
	SetHeader(value map[string]string) IAppDataClient
	SetBody(i interface{}) IAppDataClient
	SetByteBody(value []byte) IAppDataClient
	SetUrl(value string) IAppDataClient
	SetUrlQuery(value string, i interface{}) IAppDataClient
	SetSigner(ServerName, region, AccessKeyId, AccessKeySecret string) IAppDataClient
	Go() ([]byte, error)
}

type AppDataClient struct {
	endpoint   string
	tenantId   string
	instanceId string
	method     string
	url        string
	headers    map[string]string
	body       *bytes.Buffer
	client     *http.Client
	s          *v4.Signer
	region     string
	servername string
}

func NewAppDataClient() *AppDataClient {
	return &AppDataClient{
		endpoint: ENDPOINT,
		client:   &http.Client{},
	}
}

func (app *AppDataClient) SetEndpoint(value string) IAppDataClient {
	app.endpoint = value
	return app
}

func (app *AppDataClient) SetInstanceId(value string) IAppDataClient {
	app.instanceId = value
	return app
}

func (app *AppDataClient) SetTenantId(value string) IAppDataClient {
	app.tenantId = value
	return app
}

func (app *AppDataClient) SetMethod(value string) IAppDataClient {
	app.method = value
	return app
}

func (app *AppDataClient) SetHeader(value map[string]string) IAppDataClient {
	app.headers = value
	return app
}

func (app *AppDataClient) SetUrlQuery(value string, i interface{}) IAppDataClient {
	u := util.ConvertToQueryValues(i)
	if len(value) == 0 {
		app.url = fmt.Sprintf("%s?%s", app.endpoint, u.Encode())
	} else {
		app.url = fmt.Sprintf("%s/%s?%s", app.endpoint, value, u.Encode())
	}
	return app
}

func (app *AppDataClient) SetBody(i interface{}) IAppDataClient {
	bodyStr := bytes.NewBuffer(util.ConvertToMap(i))
	app.body = bodyStr
	return app
}

func (app *AppDataClient) SetByteBody(value []byte) IAppDataClient {
	bodyStr := bytes.NewBuffer(value)
	app.body = bodyStr
	return app
}

func (app *AppDataClient) SetUrl(value string) IAppDataClient {
	app.url = fmt.Sprintf("%s/%s", app.endpoint, value)
	return app
}

func (app *AppDataClient) SetSigner(ServerName, region, AccessKeyId, AccessKeySecret string) IAppDataClient {
	app.region = region
	app.servername = ServerName
	app.s = v4.NewSigner(credentials.NewStaticCredentials(AccessKeyId, AccessKeySecret, ""))
	return app
}

var retry = util.AttemptStrategy{
	Min:   5,
	Total: 5 * time.Second,
	Delay: 200 * time.Millisecond,
}

func (app *AppDataClient) Go() (body []byte, err error) {
	for r := retry.Start(); r.Next(); {
		body, err = app.send()
		if !shouldRetry(err) {
			break
		}
	}
	return body, err
}

func (app *AppDataClient) send() ([]byte, error) {
	glog.V(9).Infof("req url: %s %s body: %v", app.method, app.url, app.body)
	// var body map[string]interface{}
	requ, err := http.NewRequest(app.method, app.url, nil)
	switch app.method {
	case POST:
		requ, err = http.NewRequest(app.method, app.url, app.body)
		if err != nil {
			return nil, err
		}
	case UPDATE:
		requ, err = http.NewRequest(app.method, app.url, app.body)
		if err != nil {
			return nil, err
		}
	default:
		requ, err = http.NewRequest(app.method, app.url, nil)
		if err != nil {
			return nil, err
		}
	}

	if app.s != nil {
		if app.body != nil {
			bodyLen := app.body.Len()
			if bodyLen > 0 {
				requ.Header.Add("Content-Length", strconv.Itoa(bodyLen))
			}
		}
		body := strings.NewReader(app.body.String())
		requ, err = http.NewRequest(app.method, app.url, body)
		if err != nil {
			glog.Error(err)
			return nil, err
		}
		if _, err := app.s.Sign(requ, getSeek(body), app.servername, app.region, time.Now()); err != nil {
			glog.Error(err)
			return nil, err
		}
	}

	requ.Header.Add("Content-Type", "application/json")
	requ.Header.Add("Accept", "*/*")
	requ.Header.Add("User-Agent", "app-agent")
	requ.Header.Add("X-Request-ID", fmt.Sprintf("app-agent-%s", generator()))

	// define headers
	for k, v := range app.headers {
		requ.Header.Set(k, v)
	}

	glog.V(9).Infof("req url: %s %s body: %v header %v", app.method, app.url, app.body, requ.Header)
	resp, err := app.client.Do(requ)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		glog.Errorf("resq body %s", string(data))
		respErr := new(util.Error)
		json.Unmarshal(data, respErr)
		return nil, respErr
	}

	defer resp.Body.Close()

	// json.Unmarshal([]byte(data), &body)
	// result := fmt.Sprintln(body[app.resource])
	// return strings.Replace(result, "\n", "", -1), nil
	return data, nil
}

type TimeoutError interface {
	error
	Timeout() bool // Is the error a timeout?
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(TimeoutError)
	if ok {
		return true
	}

	switch err {
	case io.ErrUnexpectedEOF, io.EOF:
		return true
	}
	switch e := err.(type) {
	case *net.DNSError:
		return true
	case *net.OpError:
		switch e.Op {
		case "read", "write":
			return true
		}
	case *url.Error:
		// url.Error can be returned either by net/url if a URL cannot be
		// parsed, or by net/http if the response is closed before the headers
		// are received or parsed correctly. In that later case, e.Op is set to
		// the HTTP method name with the first letter uppercased. We don't want
		// to retry on POST operations, since those are not idempotent, all the
		// other ones should be safe to retry.
		switch e.Op {
		case "Get", "Put", "Delete", "Head":
			return shouldRetry(e.Err)
		default:
			return false
		}
	}
	return false
}

func generator() string {
	return random.String(32)
}

func getSeek(body io.Reader) io.ReadSeeker {
	var seeker io.ReadSeeker
	if sr, ok := body.(io.ReadSeeker); ok {
		seeker = sr
	} else {
		seeker = aws.ReadSeekCloser(body)
	}
	return seeker
}
