package util

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	Resource      = "configmaps"
	Namespace     = "kube-system"
	ConfigMapName = "user-temp-aksk"
)

type AKSK struct {
	AK            string
	SK            string
	SecurityToken string
	K8sclient     *k8sclient.Clientset
	Region        string
}

var aksk = AKSK{}

func InitAksk(k8sclient *k8sclient.Clientset) {
	aksk.K8sclient = k8sclient
}

func GetAKSK() (AKSK, error) {
	cm, err := aksk.K8sclient.CoreV1().ConfigMaps(Namespace).Get(context.Background(), ConfigMapName, v1.GetOptions{})
	if err != nil {
		klog.Errorf("get configmap %v: %v", ConfigMapName, err)
		return aksk, err
	}
	aksk.AK = cm.Data["ak"]
	aksk.SK = cm.Data["sk"]
	aksk.Region = cm.Data["region"]
	securityToken, ok := cm.Data["securityToken"]
	if !ok {
		return aksk, fmt.Errorf("securityToken not found in configmap %s", ConfigMapName)
	}
	aksk.SecurityToken = securityToken

	//klog.V(5).Infof("get AK: %s, SK: %s, region: %s", aksk.AK, aksk.SK, aksk.Region)
	return aksk, nil
}
