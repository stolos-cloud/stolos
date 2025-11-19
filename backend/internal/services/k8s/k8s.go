package k8s

import (
	"context"
	"fmt"
	"os"
	"strings"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
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
	disc, _ := discovery.NewDiscoveryClientForConfig(K8sClient.Config)
	var resources []unstructured.Unstructured
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
			apiGroupList, _ := disc.ServerGroups()

			for _, g := range apiGroupList.Groups {
				if g.Name == filter.Group {
					for _, v := range g.Versions {
						gv := v.GroupVersion
						resourcesGv, err := disc.ServerResourcesForGroupVersion(gv)
						if err != nil {
							return nil, err
						}
						loopResources, err := K8sClient.findFromResourcesGv(resourcesGv, filter.Namespace)
						if err != nil {
							return nil, err
						}
						resources = append(resources, loopResources...)
					}
				}
			}
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
