package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"

	yokeK8s "github.com/yokecd/yoke/pkg/flight/wasi/k8s"
	corev1 "k8s.io/api/core/v1"
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

	if err != nil {
		panic(err)
	}
	return crd != nil
}

// GetExistingSecret to fetch an existing secret
func GetExistingSecret(name, namespace string) (*corev1.Secret, error) {
	secret, err := yokeK8s.Lookup[corev1.Secret](yokeK8s.ResourceIdentifier{
		ApiVersion: "v1",
		Kind:       "Secret",
		Name:       name,
		Namespace:  namespace,
	})

	if err != nil {
		return nil, err
	}
	return secret, nil
}

// GenerateRandomString generates a cryptographically secure random string
func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic("Failed to generate random secret: " + err.Error())
	}
	return hex.EncodeToString(bytes)
}
