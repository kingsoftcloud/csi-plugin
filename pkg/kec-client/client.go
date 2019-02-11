package kecClient

import (
	api "csi-plugin/pkg/open-api"
	"encoding/json"

	"github.com/golang/glog"
)

const (
	serviceName = "kec"
)

type Client struct {
	*api.Client
}

func New(config *api.ClientConfig) *Client {
	return &Client{
		Client: api.New(config),
	}
}

func (cli *Client) DescribeInstances(instance_id string) (*KecInfo, error) {
	query := "Action=DescribeInstances&Version=2016-03-04&InstanceId.1=" + instance_id
	resp, err := cli.DoRequest(serviceName, query)
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
