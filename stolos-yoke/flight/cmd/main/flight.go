package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/invopop/jsonschema"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/argocd"
	cert_manager "github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/cert-manager"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/cnpg"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/contour"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/metallb"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/stolos"
	types "github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/types"
	"github.com/yokecd/yoke/pkg/flight"
	k8s "github.com/yokecd/yoke/pkg/flight/wasi/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func main() {
	if runtime.GOARCH != "wasm" {
		schema := jsonschema.Reflect(&types.Stolos{})
		schemaBytes, _ := schema.MarshalJSON()
		_ = os.WriteFile("schema.json", schemaBytes, 0644)
		os.Exit(0)
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {

	var input types.Stolos
	err := yaml.NewYAMLToJSONDecoder(os.Stdin).Decode(&input)
	if err != nil {
		return err
	}

	argoApp := &types.Application{}
	scheme.Scheme.AddKnownTypes(argoApp.GetObjectKind().GroupVersionKind().GroupVersion(), argoApp)
	resources := []flight.Resource{}
	if input.Spec.ArgoCD.Deploy {
		resources = append(resources, argocd.AllArgoCD(input)...)
	}

	if input.Spec.MetalLB.Deploy {
		resources = append(resources, metallb.AllMetalLB(input)...)
	}

	if input.Spec.CertManager.Deploy {
		resources = append(resources, cert_manager.AllCertManager(input)...)
	}

	if input.Spec.CNPG.Deploy {
		resources = append(resources, cnpg.AllCnpg(input)...)
	}

	if input.Spec.Contour.Deploy {
		resources = append(resources, contour.AllContour(input)...)
	}

	if input.Spec.StolosPlatform.Deploy {
		resources = append(resources, stolos.AllStolos(input)...)
	}

	resources = append(resources, selfArgoApp(input))

	resultResources := []flight.Resource{}
	coreScheme := k8sruntime.NewScheme()
	scheme.AddToScheme(coreScheme)
	allTypes := coreScheme.AllKnownTypes()
	existingCrds := map[schema.GroupVersionKind]bool{}

	for _, res := range resources {
		_, isCoreRes := allTypes[res.GroupVersionKind()]
		_, crdExists := existingCrds[res.GroupVersionKind()]
		if isCoreRes || crdExists {
			resultResources = append(resultResources, res)
		} else {

			_, err := k8s.GetRestMapping(res.GroupVersionKind().GroupVersion().String(), res.GroupVersionKind().Kind)
			if err == nil {
				resultResources = append(resultResources, res)
				existingCrds[res.GroupVersionKind()] = true
			} else {
				fmt.Fprint(os.Stderr, err.Error())
			}

		}
	}

	return json.NewEncoder(os.Stdout).Encode(resultResources)
}

func selfArgoApp(input types.Stolos) *types.Application {
	app := types.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-flight",
			Namespace: input.Spec.ArgoCD.Namespace,
		},
		Spec: types.ApplicationSpec{
			Source: &types.ApplicationSource{
				RepoURL:        "https://github.com/stolos-cloud/stolos",
				TargetRevision: "feature/yoke",
				Path:           "stolos-yoke",
				Directory: &types.ApplicationSourceDirectory{
					Include: "stolos-platform.yaml",
				},
			},
			Destination: types.ApplicationDestination{
				Server: "https://kubernetes.default.svc",
			},
			Project:    "default",
			SyncPolicy: argocd.DefaultSyncPolicy,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&app)
	app.SetGroupVersionKind(gvks[0])

	return &app
}
