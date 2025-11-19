package stolos

import (
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagerv1meta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	"github.com/stolos-cloud/stolos/stolos-yoke/pkg/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/utils/pointer"
)

func CreateFrontendDeployment(input types.Stolos) *appsv1.Deployment {
	frontendDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-frontend",
			Namespace: input.Spec.StolosPlatform.Namespace,
			Labels: map[string]string{
				"app": "stolos-frontend",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "stolos-frontend",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "stolos-frontend",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "frontend",
							Image: "ghcr.io/stolos-cloud/stolos-frontend:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "VITE_API_BASE_URL",
									Value: "http://stolos-backend:8080",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("128Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("200m"),
								},
							},
						},
					},
				},
			},
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&frontendDeployment)
	frontendDeployment.SetGroupVersionKind(gvks[0])

	return &frontendDeployment
}

func CreateFrontendService(input types.Stolos) *corev1.Service {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-frontend",
			Namespace: input.Spec.StolosPlatform.Namespace,
			Labels: map[string]string{
				"app": "stolos-frontend",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "stolos-frontend",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt32(80),
				},
			},
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&svc)
	svc.SetGroupVersionKind(gvks[0])

	return &svc
}

func CreateHTTPProxy(input types.Stolos) *contourv1.HTTPProxy {
	return &contourv1.HTTPProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos",
			Namespace: input.Spec.StolosPlatform.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPProxy",
			APIVersion: "projectcontour.io/v1",
		},
		Spec: contourv1.HTTPProxySpec{
			VirtualHost: &contourv1.VirtualHost{
				Fqdn: input.Spec.StolosPlatform.FrontendSubdomain + "." + input.Spec.BaseDomain,
				TLS: &contourv1.TLS{
					MinimumProtocolVersion: "1.3",
					SecretName:             "stolos-ingress-tls",
				},
			},
			Routes: []contourv1.Route{
				{
					Conditions: []contourv1.MatchCondition{
						{
							Prefix: "/api/v1",
						},
					},
					Services: []contourv1.Service{
						{
							Name: "stolos-backend",
							Port: 8080,
						},
					},
				},
				{
					Conditions: []contourv1.MatchCondition{
						{
							Prefix: "/",
						},
					},
					Services: []contourv1.Service{
						{
							Name: "stolos-frontend",
							Port: 80,
						},
					},
				},
			},
		},
	}
}

func CreateFrontendCertificate(input types.Stolos) *certmanagerv1.Certificate {
	return &certmanagerv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cert-manager.io/v1",
			Kind:       "Certificate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-tls",
			Namespace: input.Spec.StolosPlatform.Namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: "stolos-ingress-tls",
			IssuerRef: certmanagerv1meta.ObjectReference{
				Name: input.Spec.CertManager.DefaultClusterIssuer,
				Kind: "ClusterIssuer",
			},
			CommonName: input.Spec.StolosPlatform.FrontendSubdomain + "." + input.Spec.BaseDomain,
			DNSNames:   []string{input.Spec.StolosPlatform.FrontendSubdomain + "." + input.Spec.BaseDomain},
		},
	}
}
