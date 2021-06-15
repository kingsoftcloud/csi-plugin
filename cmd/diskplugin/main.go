package main

import (
	"csi-plugin/driver"
	ebsClient "csi-plugin/pkg/ebs-client"
	"encoding/json"
	"flag"
	"os"
	"os/signal"

	api "csi-plugin/pkg/open-api"

	"csi-plugin/util"

	"github.com/golang/glog"

	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	driverName = "com.ksc.csi.diskplugin"
	version    = "0.1"
)

var (
	endpoint         = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	master           = flag.String("master", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	kubeconfig       = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
	controllerServer = flag.Bool("controller-server", false, "value: controller-server=true|false")
	nodeServer       = flag.Bool("node-server", false, "value: node-server=true|false")

	volumeExpansion = flag.Bool("node-expand-required", true, "Enables NodeServiceCapability_RPC_EXPAND_VOLUME capacity.")
	maxVolumeSize   = flag.Int64("max-volume-size", 500, "maximum size of volumes in GB (inclusive)")

	accessKeyId     = flag.String("access-key-id", "", "")
	accessKeySecret = flag.String("access-key-secret", "", "")

	openApiEndpoint = flag.String("open-api-endpoint", "internal.api.ksyun.com", "")
	openApiSchema   = flag.String("open-api-schema", "http", "")
	clusterInfoPath = flag.String("cluster-info-path", "/opt/app-agent/arrangement/clusterinfo", "")
)

func new_k8sclient() *k8sclient.Clientset {
	var config *rest.Config
	var err error
	if *master != "" || *kubeconfig != "" {
		glog.Infof("Either master or kubeconfig specified. building kube config from that..")
		config, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	} else {
		glog.Infof("Building kube configs for running in cluster...")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := k8sclient.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	return clientset
}

type ClusterInfo struct {
	AccountID int64  `json:"user_id"`
	UUID      string `json:"cluster_uuid"`
	Region    string `json:"region"`
}

func loadClusterInfo(clusterInfoPath string) (*ClusterInfo, error) {
	clusterInfo := &ClusterInfo{}
	file, err := os.Open(clusterInfoPath)
	if err != nil {
		glog.Error("Failed to read clusterinfo: ", err)
		return nil, err
	}
	defer file.Close()
	if err = json.NewDecoder(file).Decode(clusterInfo); err != nil {
		glog.Error("Failed to get region and accountId from clusterinfo: ", err)
		return nil, err
	}
	return clusterInfo, nil

}

func getDriver() *driver.Driver {
	ci, err := loadClusterInfo(*clusterInfoPath)
	if err != nil {
		panic(err)
	}
	glog.Infof("cluster info: %v", ci)

	OpenApiConfig := &api.ClientConfig{
		AccessKeyId:     *accessKeyId,
		AccessKeySecret: *accessKeySecret,
		OpenApiEndpoint: *openApiEndpoint,
		OpenApiPrefix:   *openApiSchema,
		Region:          ci.Region,
	}

	cfg := &driver.Config{
		EndPoint:               *endpoint,
		EnableNodeServer:       *nodeServer,
		EnableControllerServer: *controllerServer,
		EnableVolumeExpansion:  *volumeExpansion,
		MaxVolumeSize:          *maxVolumeSize,
		DriverName:             driverName,
		K8sClient:              new_k8sclient(),
		EbsClient:              ebsClient.New(OpenApiConfig),

		Version: version,
	}

	return driver.NewDriver(cfg)
}

func main() {
	flag.Parse()
	glog.Infof("CSI plugin, version: %s", version)

	util.InitAksk(new_k8sclient())

	d := getDriver()
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
		d.Stop()
	}()

	if err := d.Run(); err != nil {
		glog.Fatal(err)
	}
}
