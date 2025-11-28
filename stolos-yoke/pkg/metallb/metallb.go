package metallb

import (
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/argocd"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/types"
	"github.com/yokecd/yoke/pkg/flight"
	metallb "go.universe.tf/metallb/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func AllMetalLB(input types.Stolos) []flight.Resource {
	return []flight.Resource{
		CreateMetalLBNamespace(input),
		DeployMetalLBHelm(input),
		DeployIPAddressPool(input),
		DeployL2Advertisement(input),
	}
}

func CreateMetalLBNamespace(input types.Stolos) *v1.Namespace {
	ns := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Spec.MetalLB.Namespace,
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "privileged",
				"pod-security.kubernetes.io/audit":   "privileged",
				"pod-security.kubernetes.io/warn":    "privileged",
			},
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&ns)
	ns.SetGroupVersionKind(gvks[0])

	return &ns
}

func DeployMetalLBHelm(input types.Stolos) *types.Application {
	app := types.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metallb",
			Namespace: input.Spec.ArgoCD.Namespace,
		},
		Spec: types.ApplicationSpec{
			Source: &types.ApplicationSource{
				RepoURL:        "https://metallb.github.io/metallb",
				TargetRevision: input.Spec.MetalLB.Version, //0.15.2
				Chart:          "metallb",
			},
			Destination: types.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: input.Spec.MetalLB.Namespace,
			},
			Project:    "default",
			SyncPolicy: argocd.DefaultSyncPolicy,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&app)
	app.SetGroupVersionKind(gvks[0])

	return &app
}

func DeployIPAddressPool(input types.Stolos) *metallb.IPAddressPool {
	autoAssign := true
	return &metallb.IPAddressPool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IPAddressPool",
			APIVersion: "metallb.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "public-ip",
			Namespace: input.Spec.MetalLB.Namespace,
		},
		Spec: metallb.IPAddressPoolSpec{
			Addresses:  []string{input.Spec.MetalLB.ArpIp + "/32"},
			AutoAssign: &autoAssign,
		},
	}
}

func DeployL2Advertisement(input types.Stolos) *metallb.L2Advertisement {
	return &metallb.L2Advertisement{
		TypeMeta: metav1.TypeMeta{
			Kind:       "L2Advertisement",
			APIVersion: "metallb.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "public-ip",
			Namespace: input.Spec.MetalLB.Namespace,
		},
		Spec: metallb.L2AdvertisementSpec{
			IPAddressPools: []string{"public-ip"},
		},
	}
}
