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
	"k8s.io/klog/v2"
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

var aksk = AkskConfig{}
var DefaultCipherKey string

func InitAksk(k8sclient *k8sclient.Clientset) {
	aksk.K8sClient = k8sclient
}

func SetAksk() (*AkskConfig, error) {
	content := os.Getenv("AKSK_CONF")
	if content == "" {
		return nil, fmt.Errorf("aksk config is null")
	}

	if err := json.Unmarshal([]byte(content), &aksk); err != nil {
		return nil, fmt.Errorf("json unmarshal %s error: %v", content, err)
	}

	switch aksk.AkskType {
	case "configmap", "secret", "", "file":
		aksk.Akskprovider = file.NewFileAKSKProvider(aksk.AkskFilePath, DefaultCipherKey)
	case "env":
		aksk.Akskprovider = env.NewEnvAKSKProvider(aksk.Encrypt, DefaultCipherKey)
	default:
		return nil, fmt.Errorf("please set aksk type")
	}
	return &aksk, nil
}

func (c *AkskConfig) GetRegion() (string, error) {
	if c.Region == "" {
		nodeName := os.Getenv("KUBE_NODE_NAME")
		if nodeName == "" {
			klog.Errorf("nodeName is empty")
		}
		node, err := c.K8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, meta_v1.GetOptions{})
		if err != nil {
			klog.Errorf("AKSK get node Region error.")
		}
		c.Region = node.Labels[NodeRegionKey]
	}
	return c.Region, nil
}
