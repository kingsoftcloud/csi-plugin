package vroute

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/zwei/appclient/config"
	"github.com/zwei/appclient/config/types"
	appHttp "github.com/zwei/appclient/pkg/http"
	"path"
)

type VrouteClient struct {
	conf     *config.Config
	client   *appHttp.AppDataClient
	tenantID string
}

func NewVrouteClient(conf *config.Config) (*VrouteClient, error) {
	var tenantID string
	dataClient := appHttp.NewAppDataClient()
	dataClient.SetEndpoint(conf.Endpoint)
	if len(conf.TenantID) != 0 {
		tenantID = conf.TenantID
	} else {
		tenantID = config.DefaultTenant
	}
	return &VrouteClient{
		client:   dataClient,
		conf:     conf,
		tenantID: tenantID,
	}, nil
}

func (n *VrouteClient) CreateVroute(vroute *types.Vroute) (*types.Vroute, error) {
	url := path.Join(n.tenantID, "clusters", vroute.Cluster_uuid, "vroutes")
	glog.V(9).Infof("create nove vroute info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.POST)
	data, err := n.client.Go()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), vroute)
	if err != nil {
		return nil, err
	}
	return vroute, nil
}

func (n *VrouteClient) GetVroute(clusterUUID, instanceUUID string) (*types.Vroute, error) {
	url := path.Join(n.tenantID, "clusters", clusterUUID, "vroutes", instanceUUID)
	glog.V(9).Infof("get node vroute info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.GET)
	data, err := n.client.Go()
	if err != nil {
		return nil, err
	}
	vrouteInfo := new(types.Vroute)
	err = json.Unmarshal([]byte(data), vrouteInfo)
	if err != nil {
		return nil, err
	}
	return vrouteInfo, nil
}

func (n *VrouteClient) UpdateVroute(vroute *types.Vroute) (*types.Vroute, error) {
	url := path.Join(n.tenantID, "clusters", vroute.Cluster_uuid, "vroutes", vroute.Instance_uuid)
	glog.V(9).Infof("update node vroute info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.UPDATE)
	n.client.SetBody(vroute)
	data, err := n.client.Go()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), vroute)
	if err != nil {
		return nil, err
	}
	return vroute, nil
}

func (n *VrouteClient) DeleteVroute(instanceUUID, clusterUUID string) error {
	url := path.Join(n.tenantID, "clusters", clusterUUID, "vroutes", instanceUUID)
	glog.V(9).Infof("delete cluster node info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.DELETE)
	_, err := n.client.Go()
	if err != nil {
		return err
	}
	return nil
}
