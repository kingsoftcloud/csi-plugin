package version

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/zwei/appclient/config"
	"github.com/zwei/appclient/config/types"
	appHttp "github.com/zwei/appclient/pkg/http"
)

type VersionClient struct {
	conf     *config.Config
	client   *appHttp.AppDataClient
	tenantID string
}

func NewVersionClient(conf *config.Config) (*VersionClient, error) {
	var tenantID string
	dataClient := appHttp.NewAppDataClient()
	dataClient.SetEndpoint(conf.Endpoint)
	if len(conf.TenantID) != 0 {
		tenantID = conf.TenantID
	} else {
		tenantID = config.DefaultTenant
	}
	return &VersionClient{
		client:   dataClient,
		conf:     conf,
		tenantID: tenantID,
	}, nil
}

func (n *VersionClient) GetVersion() (string, error) {
	url := "version"
	glog.V(9).Infof("get node vroute info from appengine: %s url: %s", n.conf.Endpoint, url)
	n.client.SetUrl(url)
	n.client.SetMethod(appHttp.GET)
	data, err := n.client.Go()
	if err != nil {
		return "", err
	}
	
	vInfo := new(types.Info)
	err = json.Unmarshal([]byte(data), vInfo)
	if err != nil {
		return "", err
	}
	return vInfo.Version, nil
}
