package utils

import (
	"bytes"
	"io"

	yokeK8s "github.com/yokecd/yoke/pkg/flight/wasi/k8s"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func ReadMultiDocument(documents []byte) []unstructured.Unstructured {
	var result []unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(documents), 4096)
	for {
		var res unstructured.Unstructured
		if err := decoder.Decode(&res); err != nil {
			if err == io.EOF { // end of file, break loop
				break
			}
			panic(err)
		}
		result = append(result, res)
	}
	return result
}

func ConvertUnstructured[T any](u unstructured.Unstructured) T {
	var result T
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(
		u.Object,
		&result,
	)
	return result
}

func PtrTo[T any](v T) *T { return &v }

func CheckCrdPresence(crdName string) bool {
	crd, err := yokeK8s.Lookup[apiextv1.CustomResourceDefinition](yokeK8s.ResourceIdentifier{
		ApiVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Name:       crdName,
	})

	panic(err)
	return err == nil && crd != nil
}
