package util

import (
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
	AK        string
	SK        string
	K8sclient *k8sclient.Clientset
}

var aksk = AKSK{}

func InitAksk(k8sclient *k8sclient.Clientset) {
	aksk.K8sclient = k8sclient
}

func GetAKSK() (string, string, error) {
	cm, err := aksk.K8sclient.CoreV1().ConfigMaps(Namespace).Get(ConfigMapName, v1.GetOptions{})
	if err != nil {
		glog.Errorf("get configmap %v: %v", ConfigMapName, err)
		return aksk.AK, aksk.SK, err
	}
	aksk.AK = cm.Data["ak"]
	aksk.SK = cm.Data["sk"]

	glog.Info("get ak: ", aksk)
	return aksk.AK, aksk.SK, nil
}
