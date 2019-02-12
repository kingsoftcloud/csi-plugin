package plugins

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/zwei/appclient/config/types"
	appHttp "github.com/zwei/appclient/pkg/http"
	"net/http"
	"path"
)

func (p *PluginsClient) CreatePluginMetaData(clusterUUID string, metadataInfo *types.PluginMetaData) (*types.PluginMetaData, error) {
	url := path.Join(p.tenantID, "clusters", clusterUUID, "pluginmetadata")
	glog.V(9).Infof("create cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.POST)
	p.client.SetBody(metadataInfo)
	data, err := p.client.Go()

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), metadataInfo)
	if err != nil {
		return nil, err
	}
	return metadataInfo, nil
}

func (p *PluginsClient) GetPluginMetaData(clusterUUID, pluginUUID string) (*types.PluginMetaData, error) {
	url := path.Join(p.tenantID, "clusters", clusterUUID, "pluginmetadata", pluginUUID)
	glog.V(9).Infof("get cluster plugins metadata info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.GET)
	data, err := p.client.Go()
	if err != nil {
		return nil, err
	}

	metadataInfo := new(types.PluginMetaData)
	err = json.Unmarshal([]byte(data), metadataInfo)
	if err != nil {
		return nil, err
	}

	return metadataInfo, nil
}

func (p *PluginsClient) UpdatePluginMetaData(clusterUUID, pluginUUID string, status, taskStatus string) (*types.PluginMetaData, error) {
	metadataInfo := &types.PluginMetaData{
		Status:      status,
		Task_Status: taskStatus,
	}
	url := path.Join(p.tenantID, "clusters", clusterUUID, "pluginmetadata", pluginUUID)
	glog.V(9).Infof("put cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(http.MethodPut)
	p.client.SetBody(metadataInfo)
	data, err := p.client.Go()

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), metadataInfo)
	if err != nil {
		return nil, err
	}
	return metadataInfo, nil
}

func (p *PluginsClient) DeletePluginMetaData(clusterUUID, pluginUUID string) error {
	url := path.Join(p.tenantID, "clusters", clusterUUID, "pluginmetadata", pluginUUID)
	glog.V(9).Infof("delete cluster plugins metadata info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.DELETE)
	_, err := p.client.Go()
	if err != nil {
		return err
	}
	return nil
}
