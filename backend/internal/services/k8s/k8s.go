package k8s

import (
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewK8sClient() (*rest.Config, error) {
	var config *rest.Config
	var err error
	if os.Getenv("KUBECONFIG") != "" {
		filePath := os.Getenv("KUBECONFIG")
		config, err = clientcmd.BuildConfigFromFlags("", filePath)
	} else if os.Getenv("KUBECONFIG_CONTENT") != "" {
		fileContent := os.Getenv("KUBECONFIG_CONTENT")
		config, err = clientcmd.RESTConfigFromKubeConfig([]byte(fileContent))
	} else {
		config, err = rest.InClusterConfig()
	}

	return config, err
}
