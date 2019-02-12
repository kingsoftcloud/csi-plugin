package aksk

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"github.com/zwei/appclient/config/types"
)


func GetAKSK(kubeconfig string, aksk *types.AKSK)  error {
	// set up the client config
	var clientConfig *rest.Config
	var err error
	if len(kubeconfig) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

		clientConfig, err = loader.ClientConfig()
	} else {
		clientConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return fmt.Errorf("unable to construct lister client config: %v", err)
	}

	// set up the informers
	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return fmt.Errorf("unable to construct lister client: %v", err)
	}

	cm, err := kubeClient.CoreV1().ConfigMaps("kube-system").Get("user-temp-aksk", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get config maps: %s", err)
	}

	aksk.AK = cm.Data["ak"]
	aksk.SK = cm.Data["sk"]
	return nil
}

