package util

import (
	"fmt"

	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
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
}

var aksk = AKSK{}

func InitAksk(k8sclient *k8sclient.Clientset) {
	aksk.K8sclient = k8sclient
}

func GetAKSK() (AKSK, error) {
	cm, err := aksk.K8sclient.CoreV1().ConfigMaps(Namespace).Get(ConfigMapName, v1.GetOptions{})
	if err != nil {
		glog.Errorf("get configmap %v: %v", ConfigMapName, err)
		return aksk, err
	}
	aksk.AK = cm.Data["ak"]
	aksk.SK = cm.Data["sk"]
	securityToken, ok := cm.Data["securityToken"]
	if !ok {
		return aksk, fmt.Errorf("securityToken not found in configmap %s", ConfigMapName)
	}
	aksk.SecurityToken = securityToken

	glog.Info("get ak: ", aksk)
	return aksk, nil
}
