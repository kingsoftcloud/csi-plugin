package appclient

import (
	"github.com/zwei/appclient/config"
	"github.com/zwei/appclient/pkg/client/cluster"
	"github.com/zwei/appclient/pkg/client/etcd"
	"github.com/zwei/appclient/pkg/client/node"
	"github.com/zwei/appclient/pkg/client/plugins"
	"github.com/zwei/appclient/pkg/client/vroute"
	"github.com/zwei/appclient/pkg/client/version"

	utilNode "github.com/zwei/appclient/pkg/util/node"
)

func NewDefaultConfig() *config.Config {
	return &config.Config{
		Endpoint: config.ENDPOINT,
		TenantID: config.DefaultTenant,
	}
}

func NewConfig(endpoint string) *config.Config {
	return &config.Config{
		Endpoint: endpoint,
		TenantID: config.DefaultTenant,
	}
}

func NewConfigWithClusterUUID(endpoint, clusterUUID string) *config.Config {
	return &config.Config{
		Endpoint:    endpoint,
		TenantID:    config.DefaultTenant,
		ClusterUUID: clusterUUID,
	}
}

func NewConfigWithInstanceUUID(endpoint, instanceUUID string) *config.Config {
	return &config.Config{
		Endpoint:     endpoint,
		TenantID:     config.DefaultTenant,
		InstanceUUID: instanceUUID,
	}
}

func Version(conf *config.Config) (*version.VersionClient, error) {
	return version.NewVersionClient(conf)
}

func Cluster(conf *config.Config) (*cluster.ClusterClient, error) {
	return cluster.NewClusterClient(conf)
}

func Etcd(conf *config.Config) (*etcd.EtcdClient, error) {
	return etcd.NewEtcdClient(conf)
}

func Node(conf *config.Config) (*node.NodeClient, error) {
	return node.NewNodeClient(conf)
}

func Plugins(conf *config.Config) (*plugins.PluginsClient, error) {
	return plugins.NewPluginsClient(conf)
}

func Vroute(conf *config.Config) (*vroute.VrouteClient, error) {
	return vroute.NewVrouteClient(conf)
}

func Vpc(conf *config.Config) (*vroute.VpcClient, error) {
	return vroute.NewVpcClient(conf)
}

func GetInstaceUUID() (string, error) {
	uuid, err := utilNode.GetSystemUUID()
	if err != nil {
		return "", err
	}

	return uuid, nil
}
