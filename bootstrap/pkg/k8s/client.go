package k8s

import (
	"context"
	"maps"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClientFromKubeconfig(kubeconfig []byte) (kubernetes.Interface, error) {

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CreateOrUpdateSecret creates a new secret or updates an existing one.
// If merge is true, the new data is merged into existing secret data.
// If merge is false, the secret data is completely replaced.
func CreateOrUpdateSecret(ctx context.Context, client kubernetes.Interface, secret *corev1.Secret, merge bool) error {
	existingSecret, err := client.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// Secret doesn't exist, create it
		_, err = client.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	// Secret exists, update it
	if merge {
		// Merge new data into existing secret
		if existingSecret.Data == nil {
			existingSecret.Data = make(map[string][]byte)
		}
		maps.Copy(existingSecret.Data, secret.Data)
		if existingSecret.StringData == nil && secret.StringData != nil {
			existingSecret.StringData = make(map[string]string)
		}
		maps.Copy(existingSecret.StringData, secret.StringData)
	} else {
		// Replace completely
		existingSecret.Data = secret.Data
		existingSecret.StringData = secret.StringData
	}

	existingSecret.Labels = secret.Labels
	_, err = client.CoreV1().Secrets(secret.Namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	return err
}