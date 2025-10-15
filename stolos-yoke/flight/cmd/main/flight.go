package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/invopop/jsonschema"
	argoappv1 "github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/argocd"
	cert_manager "github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/cert-manager"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/cnpg"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/contour"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/metallb"
	types "github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/types"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/utils"
	"github.com/yokecd/yoke/pkg/flight"
	k8s "github.com/yokecd/yoke/pkg/flight/wasi/k8s"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
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
		resources = append(resources, argoappv1.AllArgoCD(input)...)
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

	resultResources := []flight.Resource{}
	coreScheme := k8sruntime.NewScheme()
	scheme.AddToScheme(coreScheme)
	allTypes := coreScheme.AllKnownTypes()
	existingCrds := map[string]bool{}

	pluralNamer := utils.NewAllLowercasePluralNamer(make(map[string]string))
	for _, res := range resources {
		pluralKind := pluralNamer.Name(res.GroupVersionKind().Kind)
		crdName := pluralKind + "." + res.GroupVersionKind().Group
		_, isCoreRes := allTypes[res.GroupVersionKind()]
		_, crdExists := existingCrds[crdName]
		if isCoreRes || crdExists {
			resultResources = append(resultResources, res)
		} else {
			_, err := k8s.Lookup[apiextv1.CustomResourceDefinition](k8s.ResourceIdentifier{
				ApiVersion: "apiextensions.k8s.io/v1",
				Kind:       "CustomResourceDefinition",
				Name:       crdName,
			})

			if err == nil {
				existingCrds[crdName] = true
				resultResources = append(resultResources, res)
			}
		}
	}

	return json.NewEncoder(os.Stdout).Encode(resultResources)

}
