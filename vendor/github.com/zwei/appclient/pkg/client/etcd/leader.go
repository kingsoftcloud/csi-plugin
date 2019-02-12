package etcd

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/zwei/appclient/config"
	"github.com/zwei/appclient/config/types"
	appHttp "github.com/zwei/appclient/pkg/http"
	"path"
)

type EtcdClient struct {
	conf     *config.Config
	client   *appHttp.AppDataClient
	tenantID string
}

func NewEtcdClient(conf *config.Config) (*EtcdClient, error) {
	var tenantID string
	dataClient := appHttp.NewAppDataClient()
	dataClient.SetEndpoint(conf.Endpoint)
	if len(conf.TenantID) != 0 {
		tenantID = conf.TenantID
	} else {
		tenantID = config.DefaultTenant
	}
	EtcdLocationsInfo := &EtcdClient{
		client:   dataClient,
		conf:     conf,
		tenantID: tenantID,
	}
	return EtcdLocationsInfo, nil
}

func (e *EtcdClient) GetEtcdLeader(uuid string) (*types.EtcdLeader, error) {
	url := path.Join(e.tenantID, "clusters", uuid, "etcdleader")
	glog.V(9).Infof("get etcd cluster leader info from appengine: %s url: %s", e.conf.Endpoint, url)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.GET)
	data, err := e.client.Go()
	if err != nil {
		return nil, err
	}
	etcdLeaderInfo := new(types.EtcdLeader)
	err = json.Unmarshal([]byte(data), etcdLeaderInfo)
	if err != nil {
		return nil, err
	}
	return etcdLeaderInfo, nil
}

func (e *EtcdClient) CreateEtcdLeader(clusterUUID, instanceUUID string, metaData string) (*types.EtcdLeader, error) {
	etcdLeaderInfo := new(types.EtcdLeader)
	etcdLeaderInfo.Cluster_uuid = clusterUUID
	etcdLeaderInfo.MetaData = metaData
	etcdLeaderInfo.EtcdLeader = instanceUUID

	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdleader")
	glog.V(9).Infof("get etcd cluster leader info from appengine: %s url: %s", e.conf.Endpoint, url)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.POST)
	e.client.SetBody(etcdLeaderInfo)
	data, err := e.client.Go()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), etcdLeaderInfo)
	if err != nil {
		return nil, err
	}
	return etcdLeaderInfo, nil
}

func (e *EtcdClient) UpdateEtcdLeader(clusterUUID, instanceUUID string, metaData string) (*types.EtcdLeader, error) {
	etcdLeaderInfo := new(types.EtcdLeader)
	etcdLeaderInfo.Cluster_uuid = clusterUUID
	etcdLeaderInfo.MetaData = metaData
	etcdLeaderInfo.EtcdLeader = instanceUUID
	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdleader")
	glog.V(9).Infof("get etcd cluster leader info from appengine: %s url: %s", e.conf.Endpoint, url)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.UPDATE)
	e.client.SetBody(etcdLeaderInfo)
	data, err := e.client.Go()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), etcdLeaderInfo)
	if err != nil {
		return nil, err
	}
	return etcdLeaderInfo, nil
}

func (e *EtcdClient) DeleteEtcdLeader(clusterUUID string) error {
	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdleader")
	glog.V(9).Infof("get etcd cluster leader info from appengine: %s url: %s", e.conf.Endpoint, url)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.DELETE)
	_, err := e.client.Go()
	if err != nil {
		return err
	}
	return nil
}
