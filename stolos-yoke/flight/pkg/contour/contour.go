package contour

import (
	_ "embed"

	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/argocd"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/types"
	"github.com/yokecd/yoke/pkg/flight"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func AllContour(input types.Stolos) []flight.Resource {
	return []flight.Resource{
		CreateContourNamespace(input),
		DeployContourYaml(input),
	}
}

func CreateContourNamespace(input types.Stolos) *v1.Namespace {
	ns := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Spec.Contour.Namespace,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&ns)
	ns.SetGroupVersionKind(gvks[0])

	return &ns
}

func DeployContourYaml(input types.Stolos) *types.Application {
	app := types.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: input.Spec.ArgoCD.Namespace,
		},
		Spec: types.ApplicationSpec{
			Source: &types.ApplicationSource{
				RepoURL:        "https://github.com/projectcontour/contour",
				TargetRevision: input.Spec.Contour.Version, //release-1.33
				Path:           "examples/render",
				Directory: &types.ApplicationSourceDirectory{
					Include: "contour.yaml",
				},
			},
			Destination: types.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: input.Spec.Contour.Namespace,
			},
			Project:    "default",
			SyncPolicy: argocd.DefaultSyncPolicy,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&app)
	app.SetGroupVersionKind(gvks[0])

	return &app
}
