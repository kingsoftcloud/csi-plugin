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
package cluster

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/zwei/appclient/pkg/client/util"
	"github.com/zwei/appclient/config"
	"github.com/zwei/appclient/config/types"
	appHttp "github.com/zwei/appclient/pkg/http"
	"os"
	"path"
)

type ClusterClient struct {
	conf     *config.Config
	client   *appHttp.AppDataClient
	tenantID string
}

func NewClusterClient(conf *config.Config) (*ClusterClient, error) {
	var tenantID string
	dataClient := appHttp.NewAppDataClient()
	dataClient.SetEndpoint(conf.Endpoint)
	if len(conf.TenantID) != 0 {
		tenantID = conf.TenantID
	} else {
		tenantID = config.DefaultTenant
	}

	return &ClusterClient{
		client:   dataClient,
		conf:     conf,
		tenantID: tenantID,
	}, nil
}

func (c *ClusterClient) GetCluster(uuid string) (*types.Cluster, error) {
	url := path.Join(c.tenantID, "clusters", uuid)
	glog.V(9).Infof("get cluster info from appengine: %s url: %s", c.conf.Endpoint, url)
	c.client.SetUrl(url)
	c.client.SetMethod(appHttp.GET)
	data, err := c.client.Go()
	if err != nil {
		return nil, err
	}
	cluster := new(types.Cluster)
	err = json.Unmarshal([]byte(data), cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c *ClusterClient) GetClusterCa(uuid string) (*types.Certificate, error) {
	url := path.Join(c.tenantID, "clusters", uuid, "ca")
	glog.V(9).Infof("get cluster info from appengine: %s url: %s", c.conf.Endpoint, url)
	c.client.SetUrl(url)
	c.client.SetMethod(appHttp.GET)
	data, err := c.client.Go()
	if err != nil {
		return nil, err
	}
	clusterCa := new(types.Certificate)
	err = json.Unmarshal([]byte(data), clusterCa)
	if err != nil {
		return nil, err
	}
	return clusterCa, nil
}

func (c *ClusterClient) GetClusterConfigFile(uuid string) ([]byte, error) {
	url := path.Join(c.tenantID, "clusters", uuid, "ca", "file")
	glog.V(9).Infof("get cluster info from appengine: %s url: %s", c.conf.Endpoint, url)
	c.client.SetUrl(url)
	c.client.SetMethod(appHttp.GET)
	data, err := c.client.Go()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *ClusterClient) PutClusterConfigFile(uuid, eip, eiptype, configFile string) (*types.CrtLocation, error) {
	headers := make(map[string]string)
	headers["X-K8s-Eip"] = eip
	headers["X-K8s-Slb"] = eiptype
	headers["Content-Type"] = "application/octet-stream"

	if _, err := os.Stat(configFile); err != nil {
		return nil, err
	}
	body, err := util.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	url := path.Join(c.tenantID, "clusters", uuid, "ca", "file")
	glog.V(9).Infof("create cluster info from appengine: %s url: %s body: %v", c.conf.Endpoint, url, string(body))
	c.client.SetUrl(url)
	c.client.SetHeader(headers)
	c.client.SetMethod(appHttp.UPDATE)
	c.client.SetByteBody(body)
	data, err := c.client.Go()
	if err != nil {
		return nil, err
	}
	crtLocation := new(types.CrtLocation)
	err = json.Unmarshal([]byte(data), crtLocation)
	if err != nil {
		return nil, err
	}
	return crtLocation, nil
}
