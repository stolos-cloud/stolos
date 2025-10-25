package templates

import (
	"context"
	"encoding/json"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type JsonSchema = []byte

func listTemplates() (map[*apiextensionsv1.CustomResourceDefinitionNames]JsonSchema, error) {
	groupFilter := "stolos.cloud"

	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		//log.Fatalf("Failed to load in-cluster config: %v", err)
		return nil, err
	}

	// Create apiextensions client
	apiExtClient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		//log.Fatalf("Failed to create apiextensions client: %v", err)
		return nil, err
	}

	// List all CRDs
	crdList, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		//log.Fatalf("Failed to list CRDs: %v", err)
		return nil, err
	}

	// Filter by group
	var matchingCrds map[*apiextensionsv1.CustomResourceDefinitionNames]JsonSchema
	for _, crd := range crdList.Items {
		if crd.Spec.Group == groupFilter || strings.HasSuffix(crd.Spec.Group, "."+groupFilter) {
			schema, err := toJSONSchema(crd.Spec.Versions[0].Schema.OpenAPIV3Schema)
			if err != nil {
				return nil, err
			}
			matchingCrds[&crd.Spec.Names] = schema
		}
	}

	return matchingCrds, nil
}

func toJSONSchema(schema *apiextensionsv1.JSONSchemaProps) (JsonSchema, error) {
	if schema == nil {
		return nil, nil
	}

	pruneKubernetesExtensions(schema)

	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	// rehydrate into map[string]interface{} so we can inject $schema
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	m["$schema"] = "https://json-schema.org/draft-07/schema#"

	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return out, nil
}

func pruneKubernetesExtensions(schema *apiextensionsv1.JSONSchemaProps) {
	if schema == nil {
		return
	}

	schema.XPreserveUnknownFields = nil
	schema.XEmbeddedResource = false
	schema.XIntOrString = false
	schema.XListMapKeys = nil
	schema.XListType = nil
	schema.XMapType = nil
	schema.XValidations = nil

	for k := range schema.Properties {
		child := schema.Properties[k]
		pruneKubernetesExtensions(&child)
		schema.Properties[k] = child
	}
	if schema.Items != nil {
		if schema.Items.Schema != nil {
			pruneKubernetesExtensions(schema.Items.Schema)
		}
		for i := range schema.Items.JSONSchemas {
			pruneKubernetesExtensions(&schema.Items.JSONSchemas[i])
		}
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		pruneKubernetesExtensions(schema.AdditionalProperties.Schema)
	}
	for i := range schema.AllOf {
		pruneKubernetesExtensions(&schema.AllOf[i])
	}
	for i := range schema.OneOf {
		pruneKubernetesExtensions(&schema.OneOf[i])
	}
	for i := range schema.AnyOf {
		pruneKubernetesExtensions(&schema.AnyOf[i])
	}
}
