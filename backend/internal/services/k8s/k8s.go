package k8s

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Used to add prefixes to app namespaces (developers)
const K8sNamespacePrefix = "app-"

type K8sClient struct {
	Config             *rest.Config
	ApiExtensionClient *apiextensionsclient.Clientset
	DynamicClient      *dynamic.DynamicClient
	Clientset          *kubernetes.Clientset
}

func NewK8sClient() (*K8sClient, error) {
	var err error
	k8sClient := K8sClient{}

	// Try in-cluster config first
	k8sClient.Config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig for development
		if os.Getenv("KUBECONFIG") != "" {
			filePath := os.Getenv("KUBECONFIG")
			k8sClient.Config, err = clientcmd.BuildConfigFromFlags("", filePath)
		} else if os.Getenv("KUBECONFIG_CONTENT") != "" {
			fileContent := os.Getenv("KUBECONFIG_CONTENT")
			k8sClient.Config, err = clientcmd.RESTConfigFromKubeConfig([]byte(fileContent))
		}
	}

	if err == nil {
		k8sClient.ApiExtensionClient, err = apiextensionsclient.NewForConfig(k8sClient.Config)
	}

	if err == nil {
		k8sClient.DynamicClient, err = dynamic.NewForConfig(k8sClient.Config)
	}

	if err == nil {
		k8sClient.Clientset, err = kubernetes.NewForConfig(k8sClient.Config)
	}

	return &k8sClient, err
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

// CreateNamespace creates a Kubernetes namespace with the app- prefix
func (k8sClient K8sClient) CreateNamespace(ctx context.Context, namespaceName string) error {
	k8sNamespaceName := K8sNamespacePrefix + namespaceName

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sNamespaceName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "stolos",
			},
		},
	}

	_, err := k8sClient.Clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", k8sNamespaceName, err)
	}

	fmt.Printf("Successfully created namespace %s\n", k8sNamespaceName)
	return nil
}

// DeleteNamespace deletes a Kubernetes namespace
func (k8sClient K8sClient) DeleteNamespace(ctx context.Context, namespaceName string) error {
	k8sNamespaceName := K8sNamespacePrefix + namespaceName

	err := k8sClient.Clientset.CoreV1().Namespaces().Delete(ctx, k8sNamespaceName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", k8sNamespaceName, err)
	}

	fmt.Printf("Successfully deleted namespace %s\n", k8sNamespaceName)
	return nil
}
