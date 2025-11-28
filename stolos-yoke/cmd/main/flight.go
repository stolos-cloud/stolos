package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/argocd"
	cert_manager "github.com/stolos-cloud/stolos/stolos-yoke/pkg/cert-manager"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/cnpg"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/contour"
	localpathprovisioner "github.com/stolos-cloud/stolos/stolos-yoke/pkg/local-path-provisioner"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/metallb"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/stolos"
	types "github.com/stolos-cloud/stolos/stolos-yoke/pkg/types"
	stolos_yoke "github.com/stolos-cloud/stolos/yoke-base/pkg/stolos-yoke"
	airway "github.com/yokecd/yoke/pkg/apis/airway/v1alpha1"
	"github.com/yokecd/yoke/pkg/flight"
	k8s "github.com/yokecd/yoke/pkg/flight/wasi/k8s"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func main() {

	airway := stolos_yoke.AirwayInputs{
		NamePlural:             "stolosplatforms",
		NameSingular:           "stolosplatform",
		Kind:                   "StolosPlatform",
		Version:                "v1alpha",
		DisplayName:            "Stolos Platform",
		Timeout:                ptr.To(1 * time.Minute),
		AirwayMode:             ptr.To(airway.AirwayModeSubscription),
		ClusterAccess:          true,
		ResourceAccessMatchers: []string{"*"},
		FixDriftInterval:       ptr.To(5 * time.Minute),
		CrossNamespace:         true,
		Scope:                  ptr.To(apiextv1.ClusterScoped),
	}

	stolos_yoke.Run[types.Stolos](airway, run)

}

func run() ([]byte, error) {

	var input types.Stolos
	err := yaml.NewYAMLToJSONDecoder(os.Stdin).Decode(&input)
	if err != nil {
		return nil, err
	}

	argoApp := &types.Application{}
	scheme.Scheme.AddKnownTypes(argoApp.GetObjectKind().GroupVersionKind().GroupVersion(), argoApp)
	resources := []flight.Resource{}
	if input.Spec.ArgoCD.Deploy {
		resources = append(resources, argocd.AllArgoCD(input)...)
	}

	if input.Spec.LocalPathProvisioner.Deploy {
		resources = append(resources, localpathprovisioner.AllLocalPathProvisioner(input)...)
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

	// resources = append(resources, selfArgoApp(input))

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

	return json.Marshal(resultResources)
}

// func selfArgoApp(input types.Stolos) *types.Application {
// 	app := types.Application{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind:       "Application",
// 			APIVersion: "argoproj.io/v1alpha1",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "stolos-flight",
// 			Namespace: input.Spec.ArgoCD.Namespace,
// 		},
// 		Spec: types.ApplicationSpec{
// 			Source: &types.ApplicationSource{
// 				RepoURL:        fmt.Sprintf("https://github.com/%s/%s", input.Spec.ArgoCD.RepositoryOwner, input.Spec.ArgoCD.RepositoryName),
// 				TargetRevision: input.Spec.ArgoCD.RepositoryRevision,
// 				Path:           filepath.Dir(input.Spec.StolosPlatform.PathToYaml),
// 				Directory: &types.ApplicationSourceDirectory{
// 					Include: filepath.Base(input.Spec.StolosPlatform.PathToYaml),
// 				},
// 			},
// 			Destination: types.ApplicationDestination{
// 				Server: "https://kubernetes.default.svc",
// 			},
// 			Project:    "default",
// 			SyncPolicy: argocd.DefaultSyncPolicy,
// 		},
// 	}

// 	gvks, _, _ := scheme.Scheme.ObjectKinds(&app)
// 	app.SetGroupVersionKind(gvks[0])

// 	return &app
// }
