package util

import (
	"context"
	"fmt"
	prvd "github.com/kingsoftcloud/aksk-provider"
	"github.com/kingsoftcloud/aksk-provider/env"
	"github.com/kingsoftcloud/aksk-provider/file"
	"os"
	"strconv"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	Resource      = "configmaps"
	Namespace     = "kube-system"
	ConfigMapName = "user-temp-aksk"
	SecretName    = "kce-security-token"
)

type AKSK struct {
	AK            string
	SK            string
	SecurityToken string
	K8sclient     *k8sclient.Clientset
	Region        string
}

type AkskConfig struct {
	AkskType     string
	AkskFilePath string
	Encrypt      bool
	Akskprovider prvd.AKSKProvider
	K8sClient    *k8sclient.Clientset
	region       string
}

var akskconfig = AkskConfig{}
var DefaultCipherKey string

func InitAksk(k8sclient *k8sclient.Clientset) {
	akskconfig.K8sClient = k8sclient
}

func SetAksk() (AkskConfig, error) {
	cm, err := akskconfig.K8sClient.CoreV1().ConfigMaps(Namespace).Get(context.Background(), ConfigMapName, meta_v1.GetOptions{})
	if err != nil {
		klog.Errorf("get configmap %v:%v", ConfigMapName, err)
		return akskconfig, err
	}
	akskconfig.AkskType = cm.Data["type"]

	switch akskconfig.AkskType {
	case "configmap", "secret", "", "file":
		akskconfig.AkskFilePath = cm.Data["filepath"]
		akskconfig.Akskprovider = file.NewFileAKSKProvider(akskconfig.AkskFilePath, DefaultCipherKey)
	case "env":
		akskconfig.Encrypt, err = strconv.ParseBool(cm.Data["Encrypt"])
		if err != nil {
			klog.Errorf("String conversion to bool type failed")
		}
		akskconfig.Akskprovider = env.NewEnvAKSKProvider(akskconfig.Encrypt, DefaultCipherKey)
	default:
		return akskconfig, fmt.Errorf("please set aksk type")
	}
	return akskconfig, err
}

func (c AkskConfig) GetRegion() (string, error) {
	c.region = os.Getenv("Region")

	nodeName := os.Getenv("KUBE_NODE_NAME")
	if nodeName == "" {
		klog.Errorf("nodeName is empty")
	}
	k8sCli := c.K8sClient
	node, err := k8sCli.CoreV1().Nodes().Get(context.Background(), nodeName, meta_v1.GetOptions{})
	if err != nil {
		klog.Errorf("AKSK get node Region error.")
	}
	c.region = node.Labels[NodeRegionKey]

	return c.region, nil
}
