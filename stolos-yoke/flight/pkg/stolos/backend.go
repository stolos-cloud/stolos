package stolos

import (
	"fmt"
	"strconv"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagerv1meta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/cnpg"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/types"
	"github.com/stolos-cloud/stolos/stolos-yoke/flight/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubectl/pkg/scheme"
)

func CreateDeployment(input types.Stolos) *appsv1.Deployment {
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-backend",
			Namespace: input.Spec.StolosPlatform.Namespace,
			Labels: map[string]string{
				"app": "stolos-backend",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utils.PtrTo(int32(2)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "stolos-backend",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "stolos-backend",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "backend",
							Image: "ghcr.io/stolos-cloud/stolos-backend:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
								},
								{
									Name:          "grpc",
									ContainerPort: 8082,
								},
							},
							Env: []corev1.EnvVar{
								{Name: "PORT", Value: "8080"},
								{Name: "DB_HOST", Value: "postgresql-stolos-rw"},
								{Name: "DB_PORT", Value: "5432"},
								{Name: "DB_USER", Value: "stolos"},
								{Name: "DB_NAME", Value: "stolos"},
								{Name: "ADMIN_EMAIL", Value: input.Spec.StolosPlatform.DefaultAdminEmail},
								{Name: "ADMIN_PASSWORD", Value: input.Spec.StolosPlatform.DefaultAdminPassword},
								{Name: "DB_SSL_MODE", Value: "disable"},
								{
									Name: "DB_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: input.Spec.StolosPlatform.Database.DBPassowrdSecret,
											},
											Key: input.Spec.StolosPlatform.Database.DBPasswordKey,
										},
									},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "stolos-system-config",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "gitops-workspace",
									MountPath: "/root/gitops-workspace",
								}, // TODO add stolos-config-secret mount
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt32(8080),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt32(8080),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("250m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "gitops-workspace",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&deployment)
	deployment.SetGroupVersionKind(gvks[0])

	return &deployment
}

func CreateBackendService(input types.Stolos) *corev1.Service {
	backendService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-backend",
			Namespace: input.Spec.StolosPlatform.Namespace,
			Labels: map[string]string{
				"app": "stolos-backend",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "stolos-backend",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.FromInt32(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&backendService)
	backendService.SetGroupVersionKind(gvks[0])

	return &backendService

}

func CreateBackendGrpcService(input types.Stolos) *corev1.Service {
	backendService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-grpc-backend",
			Namespace: input.Spec.StolosPlatform.Namespace,
			Labels: map[string]string{
				"app": "stolos-backend",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "stolos-backend",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       8082,
					TargetPort: intstr.FromInt32(8082),
				},
			},
		},
	}

	gvks, _, _ := scheme.Scheme.ObjectKinds(&backendService)
	backendService.SetGroupVersionKind(gvks[0])

	return &backendService

}

func CreateBackendHttpProxy(input types.Stolos) *contourv1.HTTPProxy {
	return &contourv1.HTTPProxy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "projectcontour.io/v1",
			Kind:       "HTTPProxy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-grpc",
			Namespace: input.Spec.StolosPlatform.Namespace,
		},
		Spec: contourv1.HTTPProxySpec{
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
							Name: "stolos-grpc-backend",
							Port: 8082,
						},
					},
				},
			},
			VirtualHost: &contourv1.VirtualHost{
				Fqdn: fmt.Sprintf("grpc.%s.%s", input.Spec.StolosPlatform.BackendSubdomain, input.Spec.BaseDomain),
				TLS: &contourv1.TLS{
					MinimumProtocolVersion: "1.3",
					SecretName:             "stolos-grpc-ingress-tls",
				},
			},
		},
	}
}

func CreateBackendCertificate(input types.Stolos) *certmanagerv1.Certificate {
	return &certmanagerv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cert-manager.io/v1",
			Kind:       "Certificate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stolos-grpc-tls",
			Namespace: input.Spec.StolosPlatform.Namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: "stolos-grpc-ingress-tls",
			IssuerRef: certmanagerv1meta.ObjectReference{
				Name: input.Spec.CertManager.DefaultClusterIssuer,
				Kind: "ClusterIssuer",
			},
			CommonName: fmt.Sprintf("grpc.%s.%s", input.Spec.StolosPlatform.BackendSubdomain, input.Spec.BaseDomain),
			DNSNames:   []string{fmt.Sprintf("grpc.%s.%s", input.Spec.StolosPlatform.BackendSubdomain, input.Spec.BaseDomain)},
		},
	}
}

func CreateDatabase(input types.Stolos) *cnpg.Cluster {
	return &cnpg.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "postgresql.cnpg.io/v1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgresql-stolos",
			Namespace: input.Spec.StolosPlatform.Namespace,
		},
		Spec: cnpg.ClusterSpec{
			ImageName: input.Spec.StolosPlatform.Database.Image, //"ghcr.io/cloudnative-pg/postgresql:17.6",
			Instances: input.Spec.StolosPlatform.Database.InstanceCount,
			Bootstrap: &cnpg.BootstrapConfiguration{
				InitDB: &cnpg.BootstrapInitDB{
					Database: "stolos",
					Owner:    "stolos",
				},
			},
			StorageConfiguration: cnpg.StorageConfiguration{
				Size: strconv.Itoa(input.Spec.StolosPlatform.Database.SizeInGigabytes) + "Gi",
			},
			PostgresConfiguration: cnpg.PostgresConfiguration{
				Parameters: map[string]string{
					"wal_compression": "on",
				},
			},
		},
	}
}
