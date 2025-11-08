//go:build airway

package stolos_yoke

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	airways "github.com/yokecd/yoke/pkg/apis/airway/v1alpha1"
	"github.com/yokecd/yoke/pkg/openapi"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type AirwayInputs struct {
	NamePlural             string
	NameSingular           string
	DisplayName            string
	Kind                   string
	Version                string
	AirwayMode             *airways.AirwayMode // optional
	ClusterAccess          bool                // optional
	ResourceAccessMatchers []string            // optional
	Timeout                *time.Duration      // optional
	FixDriftInterval       *time.Duration      // optional
}

func BuildAirwayFor[crdType any](inputs AirwayInputs, flightUrl string) ([]byte, error) {

	if inputs.NamePlural == "" || inputs.NameSingular == "" || inputs.Kind == "" || inputs.Version == "" {
		return nil, fmt.Errorf("missing inputs for airway")
	}

	if inputs.AirwayMode == nil {
		inputs.AirwayMode = ptr.To[airways.AirwayMode](airways.AirwayModeDynamic)
	}

	if inputs.ResourceAccessMatchers == nil {
		inputs.ResourceAccessMatchers = []string{}
	}

	if inputs.Timeout == nil {
		inputs.Timeout = ptr.To[time.Duration](time.Minute)
	}

	if inputs.FixDriftInterval == nil {
		inputs.FixDriftInterval = ptr.To[time.Duration](5 * time.Minute)
	}

	return json.Marshal(airways.Airway{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.stolos.cloud", inputs.NamePlural),
			Annotations: map[string]string{
				"stolos.cloud/template-display-name": inputs.DisplayName,
			},
		},
		Spec: airways.AirwaySpec{
			Mode: airways.AirwayModeSubscription,
			WasmURLs: airways.WasmURLs{
				Flight: flightUrl,
			},
			CrossNamespace:         false,
			FixDriftInterval:       metav1.Duration{Duration: *inputs.FixDriftInterval},
			ClusterAccess:          inputs.ClusterAccess,
			ResourceAccessMatchers: inputs.ResourceAccessMatchers,
			Prune: airways.PruneOptions{
				CRDs:       true,
				Namespaces: false,
			},
			Timeout: metav1.Duration{Duration: *inputs.Timeout},

			Template: apiextv1.CustomResourceDefinitionSpec{
				Group: "stolos.cloud",
				Names: apiextv1.CustomResourceDefinitionNames{
					Plural:   inputs.NamePlural,
					Singular: inputs.NameSingular,
					Kind:     inputs.Kind,
				},
				Scope: apiextv1.NamespaceScoped,
				Versions: []apiextv1.CustomResourceDefinitionVersion{
					{
						Name:    inputs.Version,
						Served:  true,
						Storage: true,
						Schema: &apiextv1.CustomResourceValidation{
							OpenAPIV3Schema: openapi.SchemaFrom(reflect.TypeFor[crdType]()),
						},
					},
				},
			},
		},
	})
}
