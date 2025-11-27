package localpathprovisioner

import (
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/argocd"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/types"
	"github.com/yokecd/yoke/pkg/flight"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func AllLocalPathProvisioner(input types.Stolos) []flight.Resource {
	return []flight.Resource{
		CreateLocalPathProvisionerNamespace(input),
		DeployLocalPathProvisioner(input),
	}
}

func CreateLocalPathProvisionerNamespace(input types.Stolos) *v1.Namespace {
	ns := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Spec.LocalPathProvisioner.Namespace,
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "privileged", // helper pods use hostPath volumes
			},
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&ns)
	ns.SetGroupVersionKind(gvks[0])

	return &ns
}

func DeployLocalPathProvisioner(input types.Stolos) *types.Application {
	app := types.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-path-provisioner",
			Namespace: input.Spec.ArgoCD.Namespace,
		},
		Spec: types.ApplicationSpec{
			Source: &types.ApplicationSource{
				RepoURL:        "https://github.com/rancher/local-path-provisioner",
				TargetRevision: input.Spec.LocalPathProvisioner.Version,
				Path:           "deploy/chart/local-path-provisioner",
				Helm: &types.ApplicationSourceHelm{
					Parameters: []types.HelmParameter{
						{
							Name:  "storageClass.defaultClass",
							Value: "true",
						},
					},
				},
			},
			Destination: types.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: input.Spec.LocalPathProvisioner.Namespace,
			},
			Project:    "default",
			SyncPolicy: argocd.DefaultSyncPolicy,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&app)
	app.SetGroupVersionKind(gvks[0])

	return &app
}
