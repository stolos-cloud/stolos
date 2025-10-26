package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"
)

type JsonSchema = []byte

type Template struct {
	crd         *apiextensionsv1.CustomResourceDefinition
	Name        string `json:"name"`
	Description string `json:"description"`
}

func ListTemplates(config *rest.Config) ([]Template, error) {

	apiExtClient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	groupFilter := "stolos.cloud"

	// List all CRDs
	crdList, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		//log.Fatalf("Failed to list CRDs: %v", err)
		return nil, err
	}

	// Filter by group
	allTemplates := make([]Template, 0)

	for _, crd := range crdList.Items {
		if crd.Spec.Group == groupFilter || strings.HasSuffix(crd.Spec.Group, "."+groupFilter) {
			allTemplates = append(allTemplates, crdToTemplate(&crd))
		}
	}

	return allTemplates, nil
}

func GetTemplate(config *rest.Config, name string) (Template, error) {

	apiExtClient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return Template{}, err
	}

	crd, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return Template{}, err
	}
	return crdToTemplate(crd), nil
}

func crdToTemplate(crd *apiextensionsv1.CustomResourceDefinition) Template {
	newTemplate := Template{
		crd: crd,
	}

	name, ok := crd.Annotations["stolos.cloud/template-name"]
	if ok {
		newTemplate.Name = name
	} else {
		newTemplate.Name = crd.Name
	}

	description, ok := crd.Annotations["stolos.cloud/template-description"]
	if ok {
		newTemplate.Description = description
	}

	return newTemplate
}

func (t *Template) GetJsonSchema() (JsonSchema, error) {
	return toJSONSchema(t.crd)
}

func (t *Template) GetDefaultYaml() (string, error) {
	defaults, err := generateDefaultYAML(t.crd)
	if err != nil {
		return "", err
	}
	return string(defaults), nil
}

func toJSONSchema(crd *apiextensionsv1.CustomResourceDefinition) (JsonSchema, error) {

	schema := crd.Spec.Versions[0].Schema.OpenAPIV3Schema

	pruneKubernetesExtensions(schema)
	removeRequiredIfHasDefault(schema)

	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	// rehydrate into map[string]interface{} so we can inject $schema
	var crdSchema map[string]interface{}
	if err := json.Unmarshal(raw, &crdSchema); err != nil {
		return nil, err
	}

	jsonSchema := map[string]interface{}{
		"$schema": "https://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"apiVersion": map[string]interface{}{
				"type":  "string",
				"const": fmt.Sprintf("%s/%s", crd.Spec.Group, crd.Spec.Versions[0].Name),
			},
			"kind": map[string]interface{}{
				"type":  "string",
				"const": crd.Spec.Names.Kind,
			},
			"metadata": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"labels": map[string]interface{}{
						"type":                 "object",
						"additionalProperties": map[string]interface{}{"type": "string"},
					},
					"annotations": map[string]interface{}{
						"type":                 "object",
						"additionalProperties": map[string]interface{}{"type": "string"},
					},
				},
				"additionalProperties": false,
			},
			"spec": crdSchema["properties"].(map[string]interface{})["spec"],
		},
		"required":             []string{"apiVersion", "kind", "metadata"},
		"additionalProperties": false,
	}

	if crdRequired, ok := crdSchema["required"]; ok {
		jsonSchema["required"] = append(jsonSchema["required"].([]string), toStringSlice(crdRequired)...)
	}
	if anyOf, ok := crdSchema["anyOf"]; ok {
		jsonSchema["anyOf"] = anyOf
	}
	if oneOf, ok := crdSchema["oneOf"]; ok {
		jsonSchema["oneOf"] = oneOf
	}
	if allOf, ok := crdSchema["allOf"]; ok {
		jsonSchema["allOf"] = allOf
	}

	out, err := json.MarshalIndent(jsonSchema, "", "  ")
	if err != nil {
		return nil, err
	}

	return out, nil
}

func toStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, len(arr))
	for i, s := range arr {
		out[i], _ = s.(string)
	}
	return out
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

func removeRequiredIfHasDefault(schema *apiextensionsv1.JSONSchemaProps) {
	if schema == nil {
		return
	}

	// If the object has required fields, rebuild the list excluding ones with defaults.
	if len(schema.Required) > 0 {
		newRequired := make([]string, 0, len(schema.Required))
		for _, field := range schema.Required {
			prop, ok := schema.Properties[field]
			if !ok || prop.Default == nil {
				newRequired = append(newRequired, field)
				if prop.Type == "string" {
					prop.MinLength = pointer.Int64(1)
				}
				schema.Properties[field] = prop
			}
		}
		schema.Required = newRequired
	}

	// Recurse into children
	for k := range schema.Properties {
		child := schema.Properties[k]
		removeRequiredIfHasDefault(&child)
		schema.Properties[k] = child
	}
	if schema.Items != nil {
		if schema.Items.Schema != nil {
			removeRequiredIfHasDefault(schema.Items.Schema)
		}
		for i := range schema.Items.JSONSchemas {
			removeRequiredIfHasDefault(&schema.Items.JSONSchemas[i])
		}
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		removeRequiredIfHasDefault(schema.AdditionalProperties.Schema)
	}
	for i := range schema.AllOf {
		removeRequiredIfHasDefault(&schema.AllOf[i])
	}
	for i := range schema.OneOf {
		removeRequiredIfHasDefault(&schema.OneOf[i])
	}
	for i := range schema.AnyOf {
		removeRequiredIfHasDefault(&schema.AnyOf[i])
	}
}

func generateDefaultYAML(crd *apiextensionsv1.CustomResourceDefinition) ([]byte, error) {
	schema := crd.Spec.Versions[0].Schema.OpenAPIV3Schema
	if schema == nil {
		return nil, fmt.Errorf("CRD %s has no schema", crd.Name)
	}

	spec := generateObjectFromSchema(schema.Properties["spec"])

	root := map[string]interface{}{
		"apiVersion": fmt.Sprintf("%s/%s", crd.Spec.Group, crd.Spec.Versions[0].Name),
		"kind":       crd.Spec.Names.Kind,
		"metadata": map[string]interface{}{
			"name": "example",
		},
		"spec": spec,
	}

	return yaml.Marshal(root)
}

func generateObjectFromSchema(schema apiextensionsv1.JSONSchemaProps) interface{} {
	// If default is set, use it directly.
	if schema.Default != nil {
		var val interface{}
		_ = json.Unmarshal(schema.Default.Raw, &val)
		return val
	}

	switch schema.Type {
	case "object":
		out := map[string]interface{}{}
		for k, prop := range schema.Properties {
			out[k] = generateObjectFromSchema(prop)
		}
		return out
	case "array":
		// Create a single placeholder element
		if schema.Items != nil {
			if schema.Items.Schema != nil {
				return []interface{}{generateObjectFromSchema(*schema.Items.Schema)}
			} else if len(schema.Items.JSONSchemas) > 0 {
				return []interface{}{generateObjectFromSchema(schema.Items.JSONSchemas[0])}
			}
		}
		return []interface{}{}
	case "boolean":
		return false
	case "integer", "number":
		return 0
	case "string":
		return ""
	default:
		return nil
	}
}
