package main

import (
	"encoding/json"
	"flag"
	"os"

	glog "github.com/Sirupsen/logrus"
	"github.com/container-storage-interface/spec/lib/go/csi"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type disk struct {
	k8sclient        *k8sclient.Clientset
	driver           *csicommon.CSIDriver
	idServer         *identityServer
	nodeServer       *nodeServer
	controllerServer *controllerServer
}

type ClusterInfo struct {
	UUID   string `json:"cluster_uuid"`
	Region string `json:"region"`
}

func (ci *ClusterInfo) Init() {
	file, err := os.Open(clusterInfoPath)
	if err != nil {
		glog.Error("Failed to read clusterinfo: ", err)
		return
	}
	defer file.Close()
	if err = json.NewDecoder(file).Decode(ci); err != nil {
		glog.Error("Failed to get region and accountId from clusterinfo: ", err)
		return
	}
}

var (
	clusterinfo     ClusterInfo
	clusterInfoPath = "/opt/app-agent/arrangement/clusterinfo"
	openApiEndpoint = "api.ksyun.com"
	openApiPrefix   = "https"

	driverName = "csi-diskplugin"
	version    = "0.1"
	endpoint   = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	nodeid     = flag.String("nodeid", "", "Node ID")
	master     = flag.String("master", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

func init_environmentvariable() {
	if os.Getenv("OPENAPI_ENDPOINT") != "" {
		openApiEndpoint = os.Getenv("OPENAPI_ENDPOINT")
	}

	if os.Getenv("OPENAPI_PREFIX") != "" {
		openApiPrefix = os.Getenv("OPENAPI_PREFIX")
	}
}

func init_k8sclient() *k8sclient.Clientset {
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

func init() {
	clusterinfo.Init()
	init_environmentvariable()
}

func main() {
	flag.Parse()
	glog.Infof("CSI plugin, version: %s", version)

	d := &disk{}
	d.k8sclient = init_k8sclient()
	d.driver = csicommon.NewCSIDriver(driverName, version, *nodeid)
	d.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
	})
	d.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})

	d.idServer = &identityServer{}
	d.nodeServer = &nodeServer{}
	d.controllerServer = &controllerServer{DefaultControllerServer: csicommon.NewDefaultControllerServer(d.driver)}

	glog.Info("Staring GRPC server...")
	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(*endpoint, d.idServer, d.controllerServer, d.nodeServer)
	s.Wait()
}
