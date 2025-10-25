package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yokecd/yoke/pkg/apis/airway/v1alpha1"
	"github.com/yokecd/yoke/pkg/openapi"

	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/types"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {

	return json.NewEncoder(os.Stdout).Encode(v1alpha1.Airway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "stolosplatforms.stolos.cloud",
		},
		Spec: v1alpha1.AirwaySpec{
			Mode: v1alpha1.AirwayModeStandard,
			WasmURLs: v1alpha1.WasmURLs{
				Flight: "oci://ghcr.io/stolos-cloud/stolos/flight:v1-alpha.21",
			},
			CrossNamespace:         true,
			FixDriftInterval:       metav1.Duration{Duration: 5 * time.Minute},
			ClusterAccess:          true,
			ResourceAccessMatchers: []string{"*"},
			Prune: v1alpha1.PruneOptions{
				CRDs:       true,
				Namespaces: true,
			},
			Timeout: metav1.Duration{Duration: 1 * time.Minute},

			Template: apiextv1.CustomResourceDefinitionSpec{
				Group: "stolos.cloud",
				Names: apiextv1.CustomResourceDefinitionNames{
					Plural:   "stolosplatforms",
					Singular: "stolosplatform",
					Kind:     "StolosPlatform",
				},
				Scope: apiextv1.ClusterScoped,
				Versions: []apiextv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1alpha",
						Served:  true,
						Storage: true,
						Schema: &apiextv1.CustomResourceValidation{
							OpenAPIV3Schema: openapi.SchemaFrom(reflect.TypeFor[types.Stolos]()),
						},
					},
				},
			},
		},
	})
}
