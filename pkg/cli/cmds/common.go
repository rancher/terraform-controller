package cmds

import (
	"github.com/rancher/terraform-operator/types/apis/terraform-operator.cattle.io/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getKubeClientSet(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func newModuleClient(kubeconfig string) (*v1.Clients, error) {
	var config *rest.Config
	var err error

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientConfig, err := v1.NewForConfig(*config)
	if err != nil {
		return nil, err
	}
	return v1.NewClientsFromInterface(clientConfig), nil
}
