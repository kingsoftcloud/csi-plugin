package plugins

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/zwei/appclient/config"
	"github.com/zwei/appclient/config/types"
	appHttp "github.com/zwei/appclient/pkg/http"
	"path"
	"strings"
)

var ExcludeListKubes = []string{
	types.DefaultKubeApiServer,
	types.DefaultKubeControllerManager,
	types.DefaultKubeScheduler,
	types.DefaultKubelet,
	types.DefaultKubeProxy,
	types.DefaultKubeDNS,
	types.DefaultCanalFlanneld,
	types.DefualtDocker,
	types.DefaultKubeEtcd,
	types.DefaultPlugins,
	types.DefaultDemon,
	types.DefaultHosts,
}


type PluginsClient struct {
	conf     *config.Config
	client   *appHttp.AppDataClient
	tenantID string
}

func NewPluginsClient(conf *config.Config) (*PluginsClient, error) {
	var tenantID string
	dataClient := appHttp.NewAppDataClient()
	dataClient.SetEndpoint(conf.Endpoint)
	if len(conf.TenantID) != 0 {
		tenantID = conf.TenantID
	} else {
		tenantID = config.DefaultTenant
	}
	return &PluginsClient{
		client:   dataClient,
		conf:     conf,
		tenantID: tenantID,
	}, nil
}

func (p *PluginsClient) GetPlugins() ([]types.Plugin, error) {
	url := path.Join(p.tenantID, "plugins")
	glog.V(9).Infof("get cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.GET)
	data, err := p.client.Go()
	if err != nil {
		return nil, err
	}
	plugins := new(types.Plugins)
	err = json.Unmarshal([]byte(data), plugins)
	if err != nil {
		return nil, err
	}
	// init plugins id
	if len(plugins.Plugins) == 0 {
		err = fmt.Errorf("not found k8s cluster plugins")
		return nil, err
	}

	return plugins.Plugins, nil
}

func (p *PluginsClient) GetPluginsWithVersion(version string) ([]types.Plugin, error) {
	url := path.Join(p.tenantID, "plugins")
	url = fmt.Sprintf("%s?version=%s", url, version)
	glog.V(9).Infof("get cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.GET)
	data, err := p.client.Go()
	if err != nil {
		return nil, err
	}
	plugins := new(types.Plugins)
	err = json.Unmarshal([]byte(data), plugins)
	if err != nil {
		return nil, err
	}
	// init plugins id
	if len(plugins.Plugins) == 0 {
		err = fmt.Errorf("not found k8s cluster plugins")
		return nil, err
	}

	return plugins.Plugins, nil
}

func (p *PluginsClient) GetPlugin(pluginUUID string) (*types.Plugin, error) {
	url := path.Join(p.tenantID, "plugins", pluginUUID)
	glog.V(9).Infof("get cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.GET)
	data, err := p.client.Go()
	if err != nil {
		return nil, err
	}

	plugin := new(types.Plugin)
	err = json.Unmarshal([]byte(data), plugin)
	if err != nil {
		return nil, err
	}

	return plugin, nil
}

func (p *PluginsClient) GetPluginByName(name string) (*types.Plugin, error) {
	url := path.Join(p.tenantID, "plugins", name)
	glog.V(9).Infof("get cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.GET)
	data, err := p.client.Go()
	if err != nil {
		return nil, err
	}
	plugin := new(types.Plugin)
	err = json.Unmarshal([]byte(data), plugin)
	if err != nil {
		return nil, err
	}

	return plugin, nil
}

func (p *PluginsClient) GetPluginByNameAndVersion(name, version string) (*types.Plugin, error) {
	url := path.Join(p.tenantID, "plugins", name)
	url = fmt.Sprintf("%s?version=%s", url, version)
	glog.V(9).Infof("get cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.GET)
	data, err := p.client.Go()
	if err != nil {
		return nil, err
	}
	plugin := new(types.Plugin)
	err = json.Unmarshal([]byte(data), plugin)
	if err != nil {
		return nil, err
	}

	return plugin, nil
}

func (p *PluginsClient) UpdatePlugin(pluginUUID string, status string) (*types.Plugin, error) {
	url := path.Join(p.tenantID, "plugins", pluginUUID)
	glog.V(9).Infof("put cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	plugin := &types.Plugin{
		Status: status,
	}
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.PUT)
	p.client.SetBody(plugin)
	data, err := p.client.Go()

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), plugin)
	if err != nil {
		return nil, err
	}
	return plugin, nil
}

func (p *PluginsClient) DeletePlugin(pluginUUID string) error {
	url := path.Join(p.tenantID, "plugins", pluginUUID)
	glog.V(9).Infof("delete cluster plugins info from appengine: %s url: %s", p.conf.Endpoint, url)
	p.client.SetUrl(url)
	p.client.SetMethod(appHttp.DELETE)
	_, err := p.client.Go()
	if err != nil {
		return err
	}
	return nil
}

func (p *PluginsClient) ExcludePlugin(name string) bool {
	for _, n := range ExcludeListKubes {
		if strings.ToUpper(name) == strings.ToUpper(n) {
			return true
		}
	}
	return false
}
