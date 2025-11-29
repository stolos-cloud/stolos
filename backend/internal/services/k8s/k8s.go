package k8s

import (
	"context"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Used to add prefixes to app namespaces (developers)
const K8sNamespacePrefix = "app-"

type K8sResourceFilter struct {
	Namespace  string
	Kind       string
	Group      string
	ApiVersion string
}

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

func (K8sClient K8sClient) GetAllResourcesWithFilter(filter K8sResourceFilter) ([]unstructured.Unstructured, error) {
	fmt.Printf("Getting all resources with filter %+v\n", filter)
	disc, _ := discovery.NewDiscoveryClientForConfig(K8sClient.Config)
	var resources []unstructured.Unstructured

	if filter.Kind != "" {
		if filter.Group == "" {
			return nil, fmt.Errorf("Group is required when Kind is specified")
		}

		gvrs, err := findAllGroupVersions(disc, filter.Group, filter.Kind)
		if err != nil {
			return nil, err
		}

		return K8sClient.getAllFromGvrsList(gvrs, filter.Namespace)
	}

	if filter.Group != "" {
		if filter.ApiVersion != "" {
			resourcesGv, err := disc.ServerResourcesForGroupVersion(fmt.Sprintf("%s/%s", filter.Group, filter.ApiVersion))
			if err != nil {
				return nil, err
			}
			allResources, err := K8sClient.findFromResourcesGv(resourcesGv, filter.Namespace)
			if err != nil {
				return nil, err
			}
			resources = append(resources, allResources...)
		} else {

			gvrs, err := findAllGroupVersions(disc, filter.Group, "")
			if err != nil {
				return nil, err
			}

			return K8sClient.getAllFromGvrsList(gvrs, filter.Namespace)
		}
	}

	return resources, nil
}

func (K8sClient K8sClient) findFromResourcesGv(resourcesGv *metav1.APIResourceList, namespace string) ([]unstructured.Unstructured, error) {
	resources := []unstructured.Unstructured{}
	var err error
	for _, r := range resourcesGv.APIResources {
		// Skip subresources like "deployments/status"
		if strings.Contains(r.Name, "/") {
			continue
		}

		gvr := schema.GroupVersionResource{
			Group:    r.Group,
			Version:  r.Version,
			Resource: r.Name, // plural
		}

		var list *unstructured.UnstructuredList
		if namespace != "" {
			list, err = K8sClient.DynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		} else {
			list, err = K8sClient.DynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
		}
		if err != nil {
			// Some kinds may not support LIST (e.g., scale subresources)
			continue
		}

		resources = append(resources, list.Items...)
	}

	return resources, nil
}

func findAllGroupVersions(dc discovery.DiscoveryInterface, group string, kind string) ([]schema.GroupVersionResource, error) {
	groups, err := dc.ServerGroups()
	if err != nil {
		return nil, err
	}

	var gvrs []schema.GroupVersionResource

	for _, g := range groups.Groups {
		if g.Name != group {
			continue
		}

		for _, v := range g.Versions {
			gv := schema.GroupVersion{Group: g.Name, Version: v.Version}

			// Now discover resources inside this group/version
			rl, err := dc.ServerResourcesForGroupVersion(gv.String())
			if err != nil {
				continue
			}

			for _, res := range rl.APIResources {
				if strings.Contains(res.Name, "/") || !res.Namespaced {
					continue
				}
				if res.Kind == kind || kind == "" {
					gvrs = append(gvrs, gv.WithResource(res.Name))
				}
			}
		}
	}

	return gvrs, nil
}

func (K8sClient K8sClient) getAllFromGvrsList(gvrs []schema.GroupVersionResource, namespace string) ([]unstructured.Unstructured, error) {
	resources := []unstructured.Unstructured{}
	for _, gvr := range gvrs {
		fmt.Printf("Getting resources %+v\n", gvr)
		res := K8sClient.DynamicClient.Resource(gvr)

		if namespace != "" {
			result, err := res.Namespace(K8sNamespacePrefix+namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			resources = append(resources, result.Items...)
		} else {
			result, err := res.List(context.Background(), metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			resources = append(resources, result.Items...)
		}
	}

	return resources, nil
}

// CreateNamespace creates a Kubernetes namespace
func (k8sClient K8sClient) CreateNamespace(ctx context.Context, namespaceName string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "stolos",
			},
		},
	}

	_, err := k8sClient.Clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespaceName, err)
	}

	fmt.Printf("Successfully created namespace %s\n", namespaceName)
	return nil
}

// DeleteNamespace deletes a Kubernetes namespace
func (k8sClient K8sClient) DeleteNamespace(ctx context.Context, namespaceName string) error {
	err := k8sClient.Clientset.CoreV1().Namespaces().Delete(ctx, namespaceName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", namespaceName, err)
	}

	fmt.Printf("Successfully deleted namespace %s\n", namespaceName)
	return nil
}
