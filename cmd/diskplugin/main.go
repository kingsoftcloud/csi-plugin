package main

import (
	"csi-plugin/driver"
	ebsClient "csi-plugin/pkg/ebs-client"
	"encoding/json"
	"flag"
	"os"
	"os/signal"

	api "csi-plugin/pkg/open-api"

	kecClient "csi-plugin/pkg/kec-client"

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
	endpoint   = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	nodeid     = flag.String("nodeid", "", "Node ID")
	master     = flag.String("master", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")

	openApiEndpoint = flag.String("open-api-endpoint", "api.ksyun.com", "")
	openApiSchema   = flag.String("open-api-schema", "https", "")
	clusterInfoPath = flag.String("cluster-info-path", "/opt/app-agent/arrangement/clusterinfo", "")

	accessKeyId     = flag.String("access-key-id", "", "")
	accessKeySecret = flag.String("access-key-secret", "", "")
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

type clusterInfo struct {
	accountId string `json:"user_id"`
	uuid      string `json:"cluster_uuid"`
	region    string `json:"region"`
}

func loadClusterInfo(clusterInfoPath string) *clusterInfo {
	clusterInfo := &clusterInfo{}
	file, err := os.Open(clusterInfoPath)
	if err != nil {
		glog.Error("Failed to read clusterinfo: ", err)
		return nil
	}
	defer file.Close()
	if err = json.NewDecoder(file).Decode(clusterInfo); err != nil {
		glog.Error("Failed to get region and accountId from clusterinfo: ", err)
		return nil
	}
	return clusterInfo
}

func getDriver() *driver.Driver {
	ci := loadClusterInfo(*clusterInfoPath)

	OpenApiConfig := &api.ClientConfig{
		AccessKeyId:     *accessKeyId,
		AccessKeySecret: *accessKeySecret,
		OpenApiEndpoint: *openApiEndpoint,
		OpenApiPrefix:   *openApiSchema,
		Region:          ci.region,
	}

	driverConfig := &driver.DriverConfig{
		EndPoint:   *endpoint,
		DriverName: driverName,
		NodeID:     *nodeid,
		Version:    version,
		Region:     ci.region,
		EbsClient:  ebsClient.New(OpenApiConfig),
		KecClient:  kecClient.New(OpenApiConfig),
		K8sclient:  new_k8sclient(),
	}

	return driver.NewDriver(driverConfig)
}

func main() {
	flag.Parse()
	glog.Infof("CSI plugin, version: %s", version)

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
