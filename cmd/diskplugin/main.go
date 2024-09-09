package main

import (
	ebs "csi-plugin/driver/disk"
	"csi-plugin/driver/ks3"
	nfs "csi-plugin/driver/nfs"
	ebsClient "csi-plugin/pkg/ebs-client"
	"flag"
	"os"
	"strings"
	"sync"
	"time"

	api "csi-plugin/pkg/open-api"

	"csi-plugin/util"

	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

//func init() {
//	flag.Set("logtostderr", "true")
//}

const (
	EBSdriverName             = "com.ksc.csi.diskplugin"
	NFSDriverName             = "com.ksc.csi.nfsplugin"
	KS3DriverName             = "com.ksc.csi.ks3plugin"
	DiskNFSKS3MultiDriverName = "com.ksc.csi.diskplugin,com.ksc.csi.nfsplugin,com.ksc.csi.ks3plugin"
	TypePluginVar             = "com.ksc.csi.driverplugin-replace"
	version                   = "2.0.0"
)

var (
	endpoint         = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	master           = flag.String("master", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	kubeconfig       = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
	controllerServer = flag.Bool("controller-server", true, "value: controller-server=true|false")
	nodeServer       = flag.Bool("node-server", false, "value: node-server=true|false")

	volumeExpansion = flag.Bool("node-expand-required", true, "Enables NodeServiceCapability_RPC_EXPAND_VOLUME capacity.")
	maxVolumeSize   = flag.Int64("max-volume-size", 32000, "maximum size of volumes in GB (inclusive)")

	accessKeyId     = flag.String("access-key-id", "", "")
	accessKeySecret = flag.String("access-key-secret", "", "")

	openApiEndpoint = flag.String("open-api-endpoint", "internal.api.ksyun.com", "")
	openApiSchema   = flag.String("open-api-schema", "http", "")
	region          = flag.String("region", "", "")
	timeout         = flag.Duration("timeout", 30*time.Second, "Timeout specifies a time limit for requests made by this Client.")
	//clusterInfoPath = flag.String("cluster-info-path", "/opt/app-agent/arrangement/clusterinfo", "")
	metric            = flag.Bool("metric", false, "Enable monitoring volume statistics")
	driverName        = flag.String("driver", DiskNFSKS3MultiDriverName, "CSI Driver, support multi driver and  separated by ','")
	maxVolumesPerNode = flag.Int64("max-volumes-pernode", 8, "Only EBS: maximum number of volumes that can be attached to node")
	//nfs
	mountPermissions             = flag.Uint64("mount-permissions", 0, "mounted folder permissions")
	workingMountDir              = flag.String("working-mount-dir", "/tmp", "working directory for provisioner to mount nfs shares temporarily")
	defaultOnDeletePolicy        = flag.String("default-ondelete-policy", "", "default policy for deleting subdirectory when deleting a volume")
	volStatsCacheExpireInMinutes = flag.Int("vol-stats-cache-expire-in-minutes", 10, "The cache expire time in minutes for volume stats cache")
)

func newK8SClient() *k8sclient.Clientset {
	var config *rest.Config
	var err error
	if *master != "" || *kubeconfig != "" {
		klog.V(2).Infof("Either master or kubeconfig specified. building kube config from that..")
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

func getEBSDriver(epName string) *ebs.Driver {
	ebs.GlobalConfigVar.OpenApiConfig = &api.ClientConfig{
		AccessKeyId:     *accessKeyId,
		AccessKeySecret: *accessKeySecret,
		OpenApiEndpoint: *openApiEndpoint,
		OpenApiPrefix:   *openApiSchema,
		Region:          *region,
		Timeout:         *timeout,
	}
	ebs.GlobalConfigVar.K8sClient = newK8SClient()
	ebs.GlobalConfigVar.EbsClient = ebsClient.New(ebs.GlobalConfigVar.OpenApiConfig)

	cfg := &ebs.Config{
		EndPoint:               epName,
		EnableNodeServer:       *nodeServer,
		EnableControllerServer: *controllerServer,
		EnableVolumeExpansion:  *volumeExpansion,
		MaxVolumeSize:          *maxVolumeSize,
		DriverName:             EBSdriverName,
		K8sClient:              ebs.GlobalConfigVar.K8sClient,
		EbsClient:              ebs.GlobalConfigVar.EbsClient,
		MetricEnabled:          *metric,
		Version:                version,
		MaxVolumesPerNode:      *maxVolumesPerNode,
	}
	klog.V(5).Infof("disk driver config: %+v", cfg)

	klog.V(5).Infof("GlobalConfigVar driver config: %+v", ebs.GlobalConfigVar.K8sClient)

	return ebs.NewDriver(cfg)
}

func getNFSDriver(epName string) *nfs.Driver {
	nodeID, err := util.GetSystemUUID()
	if err != nil {
		klog.Warningf("nodeid is empty, err: %v", err)
	}
	driverOptions := nfs.DriverOptions{
		NodeID:                       nodeID,
		DriverName:                   NFSDriverName,
		Endpoint:                     epName,
		MountPermissions:             *mountPermissions,
		WorkingMountDir:              *workingMountDir,
		DefaultOnDeletePolicy:        *defaultOnDeletePolicy,
		VolStatsCacheExpireInMinutes: *volStatsCacheExpireInMinutes,
		K8sClient:                    newK8SClient(),
	}
	return nfs.NewDriver(&driverOptions)

}

func replaceEndpoint(driverType, endpointName string) string {
	return strings.Replace(endpointName, TypePluginVar, driverType, -1)
}

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Parse()
	klog.Infof("CSI Driver Name: %s, version: %s, endPoints: %s", *driverName, version, *endpoint)
	util.InitAksk(newK8SClient())
	multiDriverNames := *driverName
	driverNames := strings.Split(multiDriverNames, ",")
	nodeId, err := util.GetSystemUUID()
	if err != nil {
		klog.Warningf("nodeid is empty, err: %v", err)
	}
	var epName = *endpoint
	var wg sync.WaitGroup
	for _, driverName := range driverNames {
		wg.Add(1)
		if strings.Contains(*endpoint, TypePluginVar) {
			epName = replaceEndpoint(driverName, *endpoint)
		} else {
			klog.Fatal("csi endpoint: %s", *endpoint)
		}
		switch driverName {
		case EBSdriverName:
			go func(ep string) {
				defer wg.Done()
				d := getEBSDriver(ep)
				if err := d.Run(); err != nil {
					klog.Fatal(err)
					d.Stop()
				}
			}(epName)
		case NFSDriverName:
			go func(ep string) {
				defer wg.Done()
				r := getNFSDriver(ep)
				r.Run(false)
			}(epName)
		case KS3DriverName:
			klog.V(2).Infof("KS3 Driver start: %s", driverName)
			go func(ep string) {
				defer wg.Done()
				k := ks3.NewDriver(KS3DriverName, version, nodeId)
				k.Run(ep)
			}(epName)

		default:
			klog.Fatalf("CSI start failed, not support driver: %s", driverName)
		}
	}
	// wg.Add(1)
	// go func() {
	// 	c := make(chan os.Signal, 1)
	// 	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	// 	s := <-c
	// 	klog.Infof("got system signal: %v, exiting", s)
	// 	wg.Done()
	// }()
	wg.Wait()
	os.Exit(0)
}
