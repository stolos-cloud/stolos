package platform_talos

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/k8s"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type TalosSecretData struct {
	MachinesCache  talos.Machines
	BootstrapFiles map[string][]byte
}

// file list to be included in secret
var secretFilePaths = []string{
	"kubeconfig",
	"talosconfig",
	"init.yaml",
	"controlplane.yaml",
	"worker.yaml",
}

// NewBootstrapSecret reads files and prepares TalosSecretData structure
func NewBootstrapSecret(machines talos.Machines) (*TalosSecretData, error) {
	files := make(map[string][]byte)

	for _, path := range secretFilePaths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		files[path] = data
	}

	return &TalosSecretData{
		MachinesCache:  machines,
		BootstrapFiles: files,
	}, nil
}

// ToSecret serializes TalosSecretData to Kubernetes Secret
func (s *TalosSecretData) ToSecret(namespace, name string) (*corev1.Secret, error) {
	machinesJSON, err := json.Marshal(s.MachinesCache)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal machines: %w", err)
	}

	data := map[string][]byte{
		"machines.json": machinesJSON,
	}

	for name, content := range s.BootstrapFiles {
		data[name] = content
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "stolos-platform",
				"app.kubernetes.io/component": "stolos-bootstrap",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	return secret, nil
}

// FromSecret reconstructs TalosSecretData from a Kubernetes Secret
func FromSecret(secret *corev1.Secret) (*TalosSecretData, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret data is nil")
	}

	machinesData, ok := secret.Data["machines.json"]
	if !ok {
		return nil, fmt.Errorf("machines.json not found in secret")
	}

	var machines talos.Machines
	if err := json.Unmarshal(machinesData, &machines); err != nil {
		return nil, fmt.Errorf("failed to unmarshal machines: %w", err)
	}

	files := make(map[string][]byte)
	for _, path := range secretFilePaths {
		if val, ok := secret.Data[path]; ok {
			files[path] = val
		}
	}

	return &TalosSecretData{
		MachinesCache:  machines,
		BootstrapFiles: files,
	}, nil
}

// CreateOrUpdateSecret creates or updates Kubernetes secret
func (s *TalosSecretData) CreateOrUpdateSecret(ctx context.Context, client kubernetes.Interface, namespace, secretName string) error {
	secret, err := s.ToSecret(namespace, secretName)
	if err != nil {
		return err
	}
	return k8s.CreateOrUpdateSecret(ctx, client, secret, true)
}

// GetKubernetesClient creates a Kubernetes client from kubeconfig in Secret
func (s *TalosSecretData) GetKubernetesClient() (kubernetes.Interface, error) {
	kubeconfig, ok := s.BootstrapFiles["./kubeconfig.yaml"]
	if !ok {
		return nil, fmt.Errorf("kubeconfig.yaml not found in secret data")
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build rest config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// GetTalosClient creates a Talos client from talosconfig in Secret
func (s *TalosSecretData) GetTalosClient() (*machineryClient.Client, error) {
	cfgData, ok := s.BootstrapFiles["./talosconfig.yaml"]
	if !ok {
		return nil, fmt.Errorf("talosconfig.yaml not found in secret data")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.FromBytes(cfgData)

	if err != nil {
		return nil, fmt.Errorf("failed reading talosconfig bytes: %w", err)
	}

	machinery, err := machineryClient.New(
		ctx,
		machineryClient.WithConfig(cfg),
		machineryClient.WithGRPCDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create talos machinery client: %w", err)
	}

	return machinery, nil
}

//// TODO : GetTalosConfigBundle builds a Talos ConfigBundle from init/controlplane/worker YAMLs in the secret
//func (s *TalosSecretData) GetTalosConfigBundle() (*bundle.Bundle, error) {
//	configBundleOpts := []bundle.Option{
//		bundle.WithExistingConfigs("./"),
//	}
//
//	return bundle.NewBundle(configBundleOpts...)
//}
