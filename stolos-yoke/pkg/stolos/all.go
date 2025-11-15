package stolos

import (
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/types"
	"github.com/yokecd/yoke/pkg/flight"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
)

func AllStolos(input types.Stolos) []flight.Resource {
	return []flight.Resource{
		CreateStolosNamespace(input),
		CreateDeployment(input),
		CreateBackendService(input),
		CreateBackendGrpcService(input),
		CreateBackendHttpProxy(input),
		CreateBackendCertificate(input),
		CreateDatabase(input),
		CreateFrontendDeployment(input),
		CreateFrontendService(input),
		CreateHTTPProxy(input),
		CreateFrontendCertificate(input),
	}
}

func CreateStolosNamespace(input types.Stolos) *corev1.Namespace {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Spec.StolosPlatform.Namespace,
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&ns)
	ns.SetGroupVersionKind(gvks[0])

	return &ns
}
