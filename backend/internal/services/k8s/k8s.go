package k8s

import (
	"context"
	"fmt"
	"os"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const K8sTeamsPrefix = "team-"

type K8sClient struct {
	Config             *rest.Config
	ApiExtensionClient *apiextensionsclient.Clientset
	DynamicClient      *dynamic.DynamicClient
}

func NewK8sClient() (*K8sClient, error) {
	var err error
	k8sClient := K8sClient{}

	if os.Getenv("KUBECONFIG") != "" {
		filePath := os.Getenv("KUBECONFIG")
		k8sClient.Config, err = clientcmd.BuildConfigFromFlags("", filePath)
	} else if os.Getenv("KUBECONFIG_CONTENT") != "" {
		fileContent := os.Getenv("KUBECONFIG_CONTENT")
		k8sClient.Config, err = clientcmd.RESTConfigFromKubeConfig([]byte(fileContent))
	} else {
		k8sClient.Config, err = rest.InClusterConfig()
	}

	if err == nil {
		k8sClient.ApiExtensionClient, err = apiextensionsclient.NewForConfig(k8sClient.Config)
	}

	if err == nil {
		k8sClient.DynamicClient, err = dynamic.NewForConfig(k8sClient.Config)
	}

	return &k8sClient, nil
}

func (k8sClient K8sClient) ApplyCR(crd map[string]interface{}, gvr schema.GroupVersionResource, onlyDryRun bool) error {

	fmt.Printf("Applying CRD %s.%s/%s\n", gvr.Resource, gvr.Group, gvr.Version)
	name := crd["metadata"].(map[string]interface{})["name"].(string)
	unstructuredCrd := &unstructured.Unstructured{
		Object: crd,
	}

	applyOptions := metav1.ApplyOptions{
		FieldManager: "stolos-k8s",
	}
	if onlyDryRun {
		applyOptions.DryRun = []string{metav1.DryRunAll}
	}

	_, err := k8sClient.DynamicClient.Resource(gvr).Namespace(unstructuredCrd.GetNamespace()).
		Apply(context.Background(), name, unstructuredCrd, applyOptions)

	return err
}
