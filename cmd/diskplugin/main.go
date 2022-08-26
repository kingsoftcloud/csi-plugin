package main

import (
	ebs "csi-plugin/driver/disk"
	nfs "csi-plugin/driver/nfs"
	ebsClient "csi-plugin/pkg/ebs-client"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "csi-plugin/pkg/open-api"

	"csi-plugin/util"

	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const (
	EBSdriverName = "com.ksc.csi.diskplugin"
	NFSDriverName = "com.kce.csi.nfs"
	version       = "1.6.0"
)

var (
	endpoint         = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	master           = flag.String("master", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	kubeconfig       = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
	controllerServer = flag.Bool("controller-server", true, "value: controller-server=true|false")
	nodeServer       = flag.Bool("node-server", false, "value: node-server=true|false")

	volumeExpansion = flag.Bool("node-expand-required", true, "Enables NodeServiceCapability_RPC_EXPAND_VOLUME capacity.")
	maxVolumeSize   = flag.Int64("max-volume-size", 16000, "maximum size of volumes in GB (inclusive)")

	accessKeyId     = flag.String("access-key-id", "", "")
	accessKeySecret = flag.String("access-key-secret", "", "")

	openApiEndpoint = flag.String("open-api-endpoint", "internal.api.ksyun.com", "")
	openApiSchema   = flag.String("open-api-schema", "http", "")
	region          = flag.String("region", "", "")
	timeout         = flag.Duration("timeout", 30*time.Second, "Timeout specifies a time limit for requests made by this Client.")
	//clusterInfoPath = flag.String("cluster-info-path", "/opt/app-agent/arrangement/clusterinfo", "")
	metric            = flag.Bool("metric", false, "Enable monitoring volume statistics")
	driverName        = flag.String("driver", EBSdriverName, "CSI Driver")
	maxVolumesPerNode = flag.Int64("max-volumes-pernode", 8, "Only EBS: maximum number of volumes that can be attached to node")
	//nfs
	mountPermissions = flag.Uint64("mount-permissions", 0777, "mounted folder permissions")
	workingMountDir  = flag.String("working-mount-dir", "/tmp", "working directory for provisioner to mount nfs shares temporarily")
)

func new_k8sclient() *k8sclient.Clientset {
	var config *rest.Config
	var err error
	if *master != "" || *kubeconfig != "" {
		klog.V(5).Infof("Either master or kubeconfig specified. building kube config from that..")
		config, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	} else {
		klog.V(5).Infof("Building kube configs for running in cluster...")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := k8sclient.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create client: %v", err)
	}

	return clientset
}

type ClusterInfo struct {
	AccountID int64  `json:"user_id"`
	UUID      string `json:"cluster_uuid"`
	Region    string `json:"region"`
}

func getEBSDriver() *ebs.Driver {
	OpenApiConfig := &api.ClientConfig{
		AccessKeyId:     *accessKeyId,
		AccessKeySecret: *accessKeySecret,
		OpenApiEndpoint: *openApiEndpoint,
		OpenApiPrefix:   *openApiSchema,
		Region:          *region,
		Timeout:         *timeout,
	}

	cfg := &ebs.Config{
		EndPoint:               *endpoint,
		EnableNodeServer:       *nodeServer,
		EnableControllerServer: *controllerServer,
		EnableVolumeExpansion:  *volumeExpansion,
		MaxVolumeSize:          *maxVolumeSize,
		DriverName:             *driverName,
		K8sClient:              new_k8sclient(),
		EbsClient:              ebsClient.New(OpenApiConfig),
		MetricEnabled:          *metric,
		Version:                version,
		MaxVolumesPerNode:      *maxVolumesPerNode,
	}
	klog.V(5).Infof("disk driver config: %+v", cfg)

	return ebs.NewDriver(cfg)
}

func getNFSDriver() *nfs.Driver {
	nodeID,err:= util.GetSystemUUID()
	if err !=nil{
		klog.Warning("nodeid is empty")
	}
	driverOptions := nfs.DriverOptions{
		NodeID:           nodeID,
		DriverName:       *driverName,
		Endpoint:         *endpoint,
		MountPermissions: *mountPermissions,
		WorkingMountDir:  *workingMountDir,
	}
	return  nfs.NewDriver(&driverOptions)

}
func main() {
	flag.Parse()
	klog.V(5).Infof("CSI Driver Name: %s, version: %s, endPoints: %s", *driverName, version, *endpoint)

	util.InitAksk(new_k8sclient())
	stop := make(chan struct{})
	switch *driverName {
	case EBSdriverName:
		d := getEBSDriver()
		go func() {
			if err := d.Run(); err != nil {
				klog.Fatal(err)
				d.Stop()
				stop <- struct{}{}
			}
		}()
	case NFSDriverName:
		// TODO
		r := getNFSDriver()
		r.Run()
		r.Stop()
		stop <- struct{}{}
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		s := <-c
		klog.V(5).Infof("got system signal: %v, exiting", s)
		stop <- struct{}{}
	}()

	<-stop
	//d.Stop()

}
