package cnpg

import (
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/argocd"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/types"
	"github.com/yokecd/yoke/pkg/flight"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func AllCnpg(input types.Stolos) []flight.Resource {
	return []flight.Resource{
		CreateCnpgNamespace(input),
		DeployCnpgApplication(input),
	}
}

func CreateCnpgNamespace(input types.Stolos) *v1.Namespace {
	ns := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Spec.CNPG.Namespace,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&ns)
	ns.SetGroupVersionKind(gvks[0])

	return &ns
}

func DeployCnpgApplication(input types.Stolos) *types.Application {
	app := types.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cnpg",
			Namespace: input.Spec.ArgoCD.Namespace,
		},
		Spec: types.ApplicationSpec{
			Sources: []types.ApplicationSource{
				{
					RepoURL:        "https://cloudnative-pg.github.io/charts",
					TargetRevision: input.Spec.CNPG.Version,
					Chart:          "cloudnative-pg",
					Helm: &types.ApplicationSourceHelm{
						Namespace: input.Spec.CNPG.Namespace,
						Parameters: []types.HelmParameter{
							{
								Name:  "namespaceOverride",
								Value: input.Spec.CNPG.Namespace,
							},
						},
					},
				},
				{
					RepoURL:        "https://github.com/cloudnative-pg/plugin-barman-cloud",
					TargetRevision: input.Spec.CNPG.BarmanVersion,
					Directory: &types.ApplicationSourceDirectory{
						Include: "manifest.yaml",
					},
					Path: ".",
				},
			},
			Destination: types.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: input.Spec.CNPG.Namespace,
			},
			Project:    "default",
			SyncPolicy: argocd.DefaultSyncPolicy,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&app)
	app.SetGroupVersionKind(gvks[0])

	return &app
}
