package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"strconv"

	"github.com/stolos-cloud/stolos/stolos-yoke-test/pkg/v1"
	stolos_yoke "github.com/stolos-cloud/stolos/yoke-base/pkg/stolos-yoke"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/yokecd/yoke/pkg/flight"
)

func main() {
	airway := stolos_yoke.AirwayInputs{
		NamePlural:   "backends",
		NameSingular: "backend",
		Kind:         "Backend",
		Version:      "v1apha1",
		DisplayName:  "Backend API service",
	}

	stolos_yoke.Run[v1.Backend](airway, run)
}

func run() ([]byte, error) {
	// When this flight is invoked, the atc will pass the JSON representation of the Backend instance to this program via standard input.
	// We can use the yaml to json decoder so that we can pass yaml definitions manually when testing for convenience.
	var backend v1.Backend
	if err := yaml.NewYAMLToJSONDecoder(os.Stdin).Decode(&backend); err != nil && err != io.EOF {
		return nil, err
	}

	// Configure some sane defaults
	backend.Spec.ServicePort = cmp.Or(backend.Spec.ServicePort, 3000)

	// Make sure that our labels include our custom selector.
	if backend.Spec.Labels == nil {
		backend.Spec.Labels = map[string]string{}
	}
	maps.Copy(backend.Spec.Labels, selector(backend))

	if backend.Spec.Replicas == 0 {
		return nil, fmt.Errorf("replicas cannot be 0")
	}

	// Create our resources (Deployment and Service) and encode them back out via Stdout.
	return json.Marshal([]flight.Resource{
		createDeployment(backend),
		createService(backend),
	})
}

// The following functions create standard kubernetes resources from our backend resource definition.
// It utilizes the base types found in `k8s.io/api` and is essentially the same as writing the types free-hand via yaml
// except that we have strong typing, type-checking, and documentation at our finger tips. All this at the reasonable
// cost of a little more verbosity.

func createDeployment(backend v1.Backend) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      backend.Name,
			Namespace: backend.Namespace,
			Labels:    backend.Spec.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &backend.Spec.Replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector(backend)},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: backend.Spec.Labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            backend.Name,
							Image:           backend.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "PORT",
									Value: strconv.Itoa(backend.Spec.ServicePort),
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          backend.Name,
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: int32(backend.Spec.ServicePort),
								},
							},
						},
					},
				},
			},
		},
	}
}

func createService(backend v1.Backend) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.Identifier(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      backend.Name,
			Namespace: backend.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector(backend),
			Type: func() corev1.ServiceType {
				if backend.Spec.NodePort > 0 {
					return corev1.ServiceTypeNodePort
				}
				return corev1.ServiceTypeClusterIP
			}(),
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					NodePort:   int32(backend.Spec.NodePort),
					Port:       80,
					TargetPort: intstr.FromString(backend.Name),
				},
			},
		},
	}
}

// Our selector for our backend application. Independent from the regular labels passed in the backend spec.
func selector(backend v1.Backend) map[string]string {
	return map[string]string{"app": backend.Name}
}
