package platform

import (
	"context"
	"fmt"

	"github.com/stolos-cloud/stolos-bootstrap/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PlatformConfig struct {
	ClusterName            string `json:"cluster_name"`
	BaseDomain             string `json:"base_domain"`
	TalosEventSinkHostname string `json:"talos_event_sink_hostname"`
	TalosEventSinkPort     string `json:"talos_event_sink_port"`
}

// NewPlatformConfig creates a new platform configuration
func NewPlatformConfig(clusterName, baseDomain string) *PlatformConfig {
	return &PlatformConfig{
		ClusterName:            clusterName,
		BaseDomain:             baseDomain,
		TalosEventSinkHostname: fmt.Sprintf("grpc.backend.%s", baseDomain),
		TalosEventSinkPort:     "8082",
	}
}

// ToSecret serializes platform config to Kubernetes secret data
func (c *PlatformConfig) ToSecret(namespace, secretName string) *corev1.Secret {
	data := map[string][]byte{
		"CLUSTER_NAME":              []byte(c.ClusterName),
		"BASE_DOMAIN":               []byte(c.BaseDomain),
		"TALOS_EVENT_SINK_HOSTNAME": []byte(c.TalosEventSinkHostname),
		"TALOS_EVENT_SINK_PORT":     []byte(c.TalosEventSinkPort),
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "stolos-platform",
				"app.kubernetes.io/component": "stolos-backend",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
}

// FromSecret deserializes Kubernetes secret to platform config
func FromSecret(secret *corev1.Secret) (*PlatformConfig, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret data is nil")
	}

	return &PlatformConfig{
		ClusterName:            string(secret.Data["CLUSTER_NAME"]),
		BaseDomain:             string(secret.Data["BASE_DOMAIN"]),
		TalosEventSinkHostname: string(secret.Data["TALOS_EVENT_SINK_HOSTNAME"]),
		TalosEventSinkPort:     string(secret.Data["TALOS_EVENT_SINK_PORT"]),
	}, nil
}

func (c *PlatformConfig) CreateOrUpdateSecret(ctx context.Context, client kubernetes.Interface, namespace, secretName string) error {
	secret := c.ToSecret(namespace, secretName)
	return k8s.CreateOrUpdateSecret(ctx, client, secret, true)
}
