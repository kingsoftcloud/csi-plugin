package util

import (
	"context"
	"encoding/json"
	"fmt"
	prvd "github.com/kingsoftcloud/aksk-provider"
	"github.com/kingsoftcloud/aksk-provider/env"
	"github.com/kingsoftcloud/aksk-provider/file"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"os"
)

const (
	Resource      = "configmaps"
	Namespace     = "kube-system"
	ConfigMapName = "aksk-config"
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
	AkskType     string               `json:"aksk_type"`
	AkskFilePath string               `json:"aksk_file_path"`
	Encrypt      bool                 `json:"encrypt"`
	Akskprovider prvd.AKSKProvider    `json:"-"`
	K8sClient    *k8sclient.Clientset `json:"-"`
	Region       string               `json:"region"`
}

var akskconfig = AkskConfig{}
var DefaultCipherKey string

func InitAksk(k8sclient *k8sclient.Clientset) {
	akskconfig.K8sClient = k8sclient
}

func SetAksk() (*AkskConfig, error) {
	//cm, err := akskconfig.K8sClient.CoreV1().ConfigMaps(Namespace).Get(context.Background(), ConfigMapName, meta_v1.GetOptions{})
	//if err != nil {
	//	klog.Errorf("get configmap %v:%v", ConfigMapName, err)
	//	return &akskconfig, err
	//}
	//akskconfig.AkskType = cm.Data["type"]

	var aksk AkskConfig

	content := os.Getenv("AKSK_CONF")
	if content == "" {
		return nil, fmt.Errorf("aksk config is null")
	}

	if err := json.Unmarshal([]byte(content), &aksk); err != nil {
		return nil, fmt.Errorf("json unmarshal %s error: %v", content, err)
	}

	switch aksk.AkskType {
	case "configmap", "secret", "", "file":
		aksk.AkskFilePath = aksk.AkskFilePath
		aksk.Akskprovider = file.NewFileAKSKProvider(aksk.AkskFilePath, DefaultCipherKey)
	case "env":
		//akskconfig.Encrypt, err = strconv.ParseBool(cm.Data["Encrypt"])\
		aksk.Akskprovider = env.NewEnvAKSKProvider(aksk.Encrypt, DefaultCipherKey)
	default:
		return nil, fmt.Errorf("please set aksk type")
	}
	return &aksk, nil
}

func (c AkskConfig) GetRegion() (string, error) {
	c.Region = os.Getenv("Region")

	nodeName := os.Getenv("KUBE_NODE_NAME")
	if nodeName == "" {
		klog.Errorf("nodeName is empty")
	}
	k8sCli := c.K8sClient
	node, err := k8sCli.CoreV1().Nodes().Get(context.Background(), nodeName, meta_v1.GetOptions{})
	if err != nil {
		klog.Errorf("AKSK get node Region error.")
	}
	c.Region = node.Labels[NodeRegionKey]

	return c.Region, nil
}
