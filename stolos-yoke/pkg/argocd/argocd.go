package argocd

import (
	_ "embed"
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	types "github.com/stolos-cloud/stolos/stolos-yoke/pkg/types"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/utils"
	"github.com/yokecd/yoke/pkg/flight"
	"github.com/yokecd/yoke/pkg/helm"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	//rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

//go:embed argocd-values.yaml
var ArgoValuesYaml []byte

//go:embed argocd-image-updater.yaml
var ImageUpdaterYaml []byte

//go:embed argo-cd-8.3.0.tgz
var ArgoCDChart []byte

var DefaultSyncPolicy = &types.SyncPolicy{
	Automated: &types.SyncPolicyAutomated{
		Prune:    true,
		SelfHeal: true,
	},
	SyncOptions: types.SyncOptions{
		"ServerSideApply=true",
	},
}

func AllArgoCD(input types.Stolos) []flight.Resource {
	all := []flight.Resource{
		CreateArgoNamespace(input),
		DeployArgoHelm(input),
		DeployArgocdProxy(input),
		DeployArgocdCert(input),
		DeploySystemApps(input),
	}
	all = append(all, DeployArgoCDImageUpdaterResources(input)...)

	//_, err := k8s.Lookup[types.Application](k8s.ResourceIdentifier{
	//	ApiVersion: "argoproj.io/v1alpha1",
	//	Kind:       "Application",
	//	Name:       "argocd",
	//	Namespace:  input.Spec.ArgoCD.Namespace,
	//})
	//
	//if err != nil {
	//	resources, err := DeployInitChart(input)
	//	if err == nil {
	//		for _, res := range resources {
	//			all = append(all, res)
	//		}
	//	} else {
	//		fmt.Fprintf(os.Stderr, "error deploying argocd chart: %v\n", err)
	//	}
	//}

	return all
}

func CreateArgoNamespace(input types.Stolos) *v1.Namespace {
	ns := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Spec.ArgoCD.Namespace,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&ns)
	ns.SetGroupVersionKind(gvks[0])

	return &ns
}

func DeployInitChart(input types.Stolos) ([]*unstructured.Unstructured, error) {
	chart, err := helm.LoadChartFromZippedArchive(ArgoCDChart)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from zipped archive: %w", err)
	}

	resources, err := chart.Render(
		"argocd",
		input.Spec.ArgoCD.Namespace,
		map[string]any{
			"namespaceOverride": input.Spec.ArgoCD.Namespace,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	return resources, nil

}

func DeployArgoHelm(input types.Stolos) *types.Application {

	app := types.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd",
			Namespace: input.Spec.ArgoCD.Namespace,
		},
		Spec: types.ApplicationSpec{
			Source: &types.ApplicationSource{
				RepoURL:        "https://argoproj.github.io/argo-helm",
				TargetRevision: input.Spec.ArgoCD.Version, //0.15.2,
				Helm: &types.ApplicationSourceHelm{
					Values: string(ArgoValuesYaml),
					Parameters: []types.HelmParameter{
						{
							Name:  "global.domain",
							Value: input.Spec.ArgoCD.Subdomain + "." + input.Spec.BaseDomain,
						},
						{
							Name:  "namespaceOverride",
							Value: input.Spec.ArgoCD.Namespace,
						},
					},
				},
				Chart: "argo-cd",
			},
			Destination: types.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: input.Spec.ArgoCD.Namespace,
			},
			Project:    "default",
			SyncPolicy: DefaultSyncPolicy,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&app)
	app.SetGroupVersionKind(gvks[0])

	return &app
}

func DeployArgocdProxy(input types.Stolos) *contourv1.HTTPProxy {
	return &contourv1.HTTPProxy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPProxy",
			APIVersion: "projectcontour.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd",
			Namespace: input.Spec.ArgoCD.Namespace,
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer": input.Spec.CertManager.DefaultClusterIssuer,
			},
		},
		Spec: contourv1.HTTPProxySpec{
			VirtualHost: &contourv1.VirtualHost{
				Fqdn: input.Spec.ArgoCD.Subdomain + "." + input.Spec.BaseDomain,
				TLS: &contourv1.TLS{
					SecretName: "argocd-tls",
				},
			},
			Routes: []contourv1.Route{
				{
					Conditions: []contourv1.MatchCondition{
						{
							Header: &contourv1.HeaderMatchCondition{
								Name:     "Content-Type",
								Contains: "application/grpc",
							},
						},
					},
					Services: []contourv1.Service{
						{
							Name:     "argocd-server",
							Port:     80,
							Protocol: utils.PtrTo("h2c"),
						},
					},
				},
				{
					Services: []contourv1.Service{
						{
							Name: "argocd-server",
							Port: 80,
						},
					},
				},
			},
		},
	}
}

func DeployArgocdCert(input types.Stolos) *certmanagerv1.Certificate {
	return &certmanagerv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: "cert-manager.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-tls",
			Namespace: input.Spec.ArgoCD.Namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: "argocd-tls",
			IssuerRef: certmanagermetav1.ObjectReference{
				Name: input.Spec.CertManager.DefaultClusterIssuer,
				Kind: "ClusterIssuer",
			},
			CommonName: input.Spec.ArgoCD.Subdomain + "." + input.Spec.BaseDomain,
			DNSNames: []string{
				input.Spec.ArgoCD.Subdomain + "." + input.Spec.BaseDomain,
			},
		},
	}
}

func DeployArgoCDImageUpdaterResources(input types.Stolos) []flight.Resource {

	var results []flight.Resource
	resources := utils.ReadMultiDocument(ImageUpdaterYaml)
	for _, res := range resources {
		res.SetNamespace(input.Spec.ArgoCD.Namespace)
		if res.GetKind() == "Deployment" {
			dep := utils.ConvertUnstructured[appsv1.Deployment](res)
			dep.Spec.Template.Spec.Containers[0].Image = "quay.io/argoprojlabs/argocd-image-updater:" + input.Spec.ArgoCD.ImageUpdaterVersion
			results = append(results, &dep)
		} else {
			results = append(results, &res)
		}
		if res.GetNamespace() != "" {
			res.SetNamespace(input.Spec.ArgoCD.Namespace)
		}
	}

	return results
}

func DeploySystemApps(input types.Stolos) flight.Resource {
	app := types.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-argoapps",
			Namespace: input.Spec.ArgoCD.Namespace,
			Annotations: map[string]string{
				"argocd.argoproj.io/sync-wave": "-10",
			},
		},
		Spec: types.ApplicationSpec{
			Source: &types.ApplicationSource{
				RepoURL: fmt.Sprintf("https://github.com/%s/%s", input.Spec.ArgoCD.RepositoryOwner, input.Spec.ArgoCD.RepositoryName),
				Ref:     input.Spec.ArgoCD.RepositoryRevision,
				Path:    "system/argoapps",
			},
			Destination: types.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: input.Spec.ArgoCD.Namespace,
			},
			Project:    "default",
			SyncPolicy: DefaultSyncPolicy,
		},
	}

	return &app
}
