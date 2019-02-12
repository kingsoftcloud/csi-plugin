package config

type Config struct {
	// Appengine service endpoint address
	Endpoint string `json:"endpoint"`
	// Neutron service endpoint address
	NetworkEndpoint string `json:"network_endpoint"`
	// k8s cluster uuid info from appengine
	ClusterUUID string `json:"cluster_uuid"`
	// nova vm instance uuid
	InstanceUUID string `json:"instance_uuid"`
	// this node vm ip
	NodeIP string `json:"node_ip"`
	// nova tenant id
	TenantID string `json:"tenant_id"`
	// nova user id
	UserID string `json:"user_id"`
	// nova token id
	Token string `json:"token"`
	// nova region
	Region string `json:"region"`
	// kube config file
	Kubeconfig string `json:"kubeconfig"`
}
