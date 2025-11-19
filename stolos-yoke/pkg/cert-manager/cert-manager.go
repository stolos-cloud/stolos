package cert_manager

import (
	cmacme "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/argocd"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/types"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/utils"
	"github.com/yokecd/yoke/pkg/flight"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func AllCertManager(input types.Stolos) []flight.Resource {
	all := []flight.Resource{
		CreateCertManagerNamespace(input),
		DeployCertManagerHelm(input),
	}
	all = append(all, DeployClusterIssuer(input)...)
	return all
}

func CreateCertManagerNamespace(input types.Stolos) *v1.Namespace {
	ns := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Spec.CertManager.Namespace,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&ns)
	ns.SetGroupVersionKind(gvks[0])

	return &ns
}

func DeployCertManagerHelm(input types.Stolos) *types.Application {
	app := types.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-manager",
			Namespace: input.Spec.ArgoCD.Namespace,
		},
		Spec: types.ApplicationSpec{
			Source: &types.ApplicationSource{
				RepoURL:        "https://charts.jetstack.io",
				TargetRevision: input.Spec.CertManager.Version,
				Chart:          "cert-manager",
				Helm: &types.ApplicationSourceHelm{
					Parameters: []types.HelmParameter{
						{
							Name:  "installCRDs",
							Value: "true",
						},
					},
				},
			},
			Destination: types.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: input.Spec.CertManager.Namespace,
			},
			Project:    "default",
			SyncPolicy: argocd.DefaultSyncPolicy,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&app)
	app.SetGroupVersionKind(gvks[0])

	return &app
}

func DeployClusterIssuer(input types.Stolos) []flight.Resource {
	var issuerConfigStg certmanagerv1.IssuerConfig
	var issuerConfigPrd certmanagerv1.IssuerConfig

	if input.Spec.CertManager.SelfSigned {
		issuerConfigStg = certmanagerv1.IssuerConfig{
			SelfSigned: &certmanagerv1.SelfSignedIssuer{},
		}
		issuerConfigPrd = certmanagerv1.IssuerConfig{
			SelfSigned: &certmanagerv1.SelfSignedIssuer{},
		}
	} else {
		issuerConfigStg = certmanagerv1.IssuerConfig{
			ACME: &cmacme.ACMEIssuer{
				Server: "https://acme-staging-v02.api.letsencrypt.org/directory",
				Email:  input.Spec.CertManager.Email,
				PrivateKey: cmmeta.SecretKeySelector{
					LocalObjectReference: cmmeta.LocalObjectReference{
						Name: input.Spec.CertManager.ClusterIssuerStaging,
					},
				},
				Solvers: []cmacme.ACMEChallengeSolver{
					{
						HTTP01: &cmacme.ACMEChallengeSolverHTTP01{
							Ingress: &cmacme.ACMEChallengeSolverHTTP01Ingress{
								Class: utils.PtrTo("contour"),
							},
						},
					},
				},
			},
		}

		issuerConfigPrd = certmanagerv1.IssuerConfig{
			ACME: &cmacme.ACMEIssuer{
				Server: "https://acme-v02.api.letsencrypt.org/directory",
				Email:  input.Spec.CertManager.Email,
				PrivateKey: cmmeta.SecretKeySelector{
					LocalObjectReference: cmmeta.LocalObjectReference{
						Name: input.Spec.CertManager.ClusterIssuerProd,
					},
				},
				Solvers: []cmacme.ACMEChallengeSolver{
					{
						HTTP01: &cmacme.ACMEChallengeSolverHTTP01{
							Ingress: &cmacme.ACMEChallengeSolverHTTP01Ingress{
								Class: utils.PtrTo("contour"),
							},
						},
					},
				},
			},
		}
	}

	return []flight.Resource{
		&certmanagerv1.ClusterIssuer{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterIssuer",
				APIVersion: "cert-manager.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Spec.CertManager.ClusterIssuerStaging,
				Namespace: input.Spec.CertManager.Namespace,
			},
			Spec: certmanagerv1.IssuerSpec{
				IssuerConfig: issuerConfigStg,
			},
		},
		&certmanagerv1.ClusterIssuer{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterIssuer",
				APIVersion: "cert-manager.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Spec.CertManager.ClusterIssuerProd,
				Namespace: input.Spec.CertManager.Namespace,
			},
			Spec: certmanagerv1.IssuerSpec{
				IssuerConfig: issuerConfigPrd,
			},
		},
	}
}
