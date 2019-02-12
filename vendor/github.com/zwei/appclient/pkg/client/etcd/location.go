package etcd

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/zwei/appclient/config/types"
	appHttp "github.com/zwei/appclient/pkg/http"
	"path"
	"fmt"
)

func (e *EtcdClient) CreateEtcdLocation(clusterUUID, snapUUID, location string, size int64, checksum, status string) (*types.EtcdLocation, error) {
	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdlocation")
	glog.V(9).Infof("create etcd cluster all snapshot info from appengine: %s url: %s", e.conf.Endpoint, url)

	if len(status) == 0 {
		status = types.ETCDSNAP_INIT
	}

	etcdLocation := &types.EtcdLocation{
		Cluster_uuid: clusterUUID,
		Snap_uuid:    snapUUID,
		Location:     location,
		Size:         size,
		Checksum:     checksum,
		Status:       status,
	}

	e.client.SetBody(etcdLocation)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.POST)
	data, err := e.client.Go()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(data), etcdLocation); err != nil {
		return nil, err
	}
	return etcdLocation, nil
}

func (e *EtcdClient) UpdateEtcdLocation(clusterUUID, snapUUID, location string, size int64, checksum, status string) (*types.EtcdLocation, error) {
	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdlocation", snapUUID)
	glog.V(9).Infof("update etcd cluster all snapshot info from appengine: %s url: %s", e.conf.Endpoint, url)
	etcdLocation := &types.EtcdLocation{
		Location: location,
		Size:     size,
		Checksum: checksum,
		Status:   status,
	}
	e.client.SetBody(etcdLocation)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.PUT)
	data, err := e.client.Go()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(data), etcdLocation); err != nil {
		return nil, err
	}
	return etcdLocation, nil
}

func (e *EtcdClient) DeleteEtcdLocation(clusterUUID, snapUUID string) error {
	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdlocation", snapUUID)
	glog.V(9).Infof("delete etcd cluster all snapshot info from appengine: %s url: %s", e.conf.Endpoint, url)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.DELETE)
	if _, err := e.client.Go(); err != nil {
		return err
	}
	return nil
}

func (e *EtcdClient) GetEtcdLocations(clusterUUID string) (*types.EtcdLocations, error) {
	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdlocations")
	glog.V(9).Infof("get etcd cluster all snapshot info from appengine: %s url: %s", e.conf.Endpoint, url)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.GET)
	data, err := e.client.Go()
	if err != nil {
		return nil, err
	}
	etcdLocations := new(types.EtcdLocations)
	if err := json.Unmarshal([]byte(data), etcdLocations); err != nil {
		return nil, err
	}
	return etcdLocations, nil
}

func (e *EtcdClient) GetEtcdLocationsWithLimit(clusterUUID, limit string) (*types.EtcdLocations, error) {
	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdlocations") + fmt.Sprintf("?limit=%s", limit)
	glog.V(9).Infof("get etcd cluster all snapshot info from appengine: %s url: %s", e.conf.Endpoint, url)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.GET)
	data, err := e.client.Go()
	if err != nil {
		return nil, err
	}
	etcdLocations := new(types.EtcdLocations)
	if err := json.Unmarshal([]byte(data), etcdLocations); err != nil {
		return nil, err
	}
	return etcdLocations, nil
}


func (e *EtcdClient) GetEtcdLocation(clusterUUID, snapUUID string) (*types.EtcdLocation, error) {
	url := path.Join(e.tenantID, "clusters", clusterUUID, "etcdlocation", snapUUID)
	glog.V(9).Infof("get etcd cluster location snapstho uuid %s info from appengine: %s url: %s", snapUUID, e.conf.Endpoint, url)
	e.client.SetUrl(url)
	e.client.SetMethod(appHttp.GET)
	data, err := e.client.Go()
	if err != nil {
		return nil, err
	}
	etcdLocation := new(types.EtcdLocation)
	if err := json.Unmarshal([]byte(data), etcdLocation); err != nil {
		return nil, err
	}
	return etcdLocation, nil
}
