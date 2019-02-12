package node

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/zwei/appclient/config"
	"github.com/zwei/appclient/config/types"
	appHttp "github.com/zwei/appclient/pkg/http"
	"github.com/zwei/appclient/pkg/util/node"
	"path"
)

type NodeClient struct {
	conf     *config.Config
	client   *appHttp.AppDataClient
	tenantID string
}

func NewNodeClient(conf *config.Config) (*NodeClient, error) {
	var tenantID string
	dataClient := appHttp.NewAppDataClient()
	dataClient.SetEndpoint(conf.Endpoint)
	if len(conf.TenantID) != 0 {
		tenantID = conf.TenantID
	} else {
		tenantID = config.DefaultTenant
	}
	return &NodeClient{
		client:   dataClient,
		conf:     conf,
		tenantID: tenantID,
	}, nil
}

func (n *NodeClient) GetLocalNode() (*types.Node, error) {
	uuid, err := node.GetSystemUUID()
	if err != nil {
		return nil, err
	}

	url := path.Join(n.tenantID, "nodes", uuid)
	glog.V(9).Infof("get cluster node info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.GET)
	data, err := n.client.Go()
	if err != nil {
		return nil, err
	}
	nodeInfo := new(types.Node)
	err = json.Unmarshal([]byte(data), &nodeInfo)
	if err != nil {
		return nil, err
	}
	return nodeInfo, nil
}

func (n *NodeClient) GetNode(uuid string) (*types.Node, error) {
	url := path.Join(n.tenantID, "nodes", uuid)
	glog.V(9).Infof("get cluster node info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.GET)
	data, err := n.client.Go()
	if err != nil {
		return nil, err
	}
	nodeInfo := new(types.Node)
	err = json.Unmarshal([]byte(data), &nodeInfo)
	if err != nil {
		return nil, err
	}
	return nodeInfo, nil
}

func (n *NodeClient) UpdateNode(uuid, clusterUUID, status, taskStatus string, errInfo types.ErrorInfo) (*types.Node, error) {
	nodeInfo := new(types.Node)
	nodeInfo.Status = status
	nodeInfo.Task_Status = taskStatus
	nodeInfo.Error_stage = errInfo.ErrorStage
	nodeInfo.Error_msg = errInfo.ErrorMsg
	url := path.Join(n.tenantID, "clusters", clusterUUID, "nodes", uuid)
	glog.V(9).Infof("update cluster node info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.UPDATE)
	n.client.SetBody(nodeInfo)
	data, err := n.client.Go()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), &nodeInfo)
	if err != nil {
		return nil, err
	}
	return nodeInfo, nil
}

func (n *NodeClient) DeleteNode(uuid, clusterUUID string) error {
	url := path.Join(n.tenantID, "clusters", clusterUUID, "nodes", uuid)
	glog.V(9).Infof("delete cluster node info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.DELETE)
	_, err := n.client.Go()
	if err != nil {
		return err
	}
	return nil
}
