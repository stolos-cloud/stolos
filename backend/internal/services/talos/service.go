package talos

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	factoryClient "github.com/siderolabs/image-factory/pkg/client"
	"github.com/siderolabs/image-factory/pkg/schematic"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	machineryClientConfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	netres "github.com/siderolabs/talos/pkg/machinery/resources/network"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"gorm.io/gorm"
)

var ignoredNodesCache []string

type TalosService struct {
	db            *gorm.DB
	cfg           *config.Config
	factoryClient *factoryClient.Client
}

// MachineConfigRequest represents parameters for generating machine configs
type MachineConfigRequest struct {
	ClusterName       string `json:"cluster_name"`
	KubernetesVersion string `json:"kubernetes_version"`
	TalosVersion      string `json:"talos_version"`
	ControlPlaneIP    string `json:"control_plane_ip"`
}

func NewTalosService(db *gorm.DB, cfg *config.Config) *TalosService {
	factory := talos.CreateFactoryClient()
	return &TalosService{
		db:            db,
		cfg:           cfg,
		factoryClient: factory,
	}
}

// GenerateISO creates a custom Talos ISO using the image factory
func (s *TalosService) GenerateISO(req *models.ISORequest) (*models.ISOResponse, error) {
	ctx := context.Background()

	// Build kernel args with event sink configuration
	kernelArgs := make([]string, 0)

	// Add event sink configuration if hostname is configured
	if s.cfg.Talos.EventSinkHostname != "" {
		sinkConf := fmt.Sprintf("talos.events.sink=%s:%s",
			s.cfg.Talos.EventSinkHostname,
			s.cfg.Talos.EventSinkPort)
		kernelArgs = append(kernelArgs, sinkConf)
	}

	// Add any additional kernel args from request
	kernelArgs = append(kernelArgs, req.ExtraKernelArgs...)

	// Build schematic
	sch := schematic.Schematic{
		Customization: schematic.Customization{
			ExtraKernelArgs: kernelArgs,
		},
	}

	// Add overlay for SBCs if provided
	if req.OverlayImage != "" && req.OverlayName != "" {
		sch.Overlay = schematic.Overlay{
			Image: req.OverlayImage,
			Name:  req.OverlayName,
		}
	}

	// Create schematic (returns schematic ID as string)
	schematicID, err := s.factoryClient.SchematicCreate(ctx, sch)
	if err != nil {
		return nil, fmt.Errorf("failed to create schematic: %w", err)
	}

	// Build download URL
	talosVersion := req.TalosVersion
	if talosVersion == "" {
		talosVersion = "v1.11.1" // default
	}

	architecture := req.Architecture
	if architecture == "" {
		architecture = "amd64" // default
	}

	downloadURL := fmt.Sprintf("https://factory.talos.dev/image/%s/%s/metal-%s.iso",
		schematicID, talosVersion, architecture)

	filename := fmt.Sprintf("stolos-talos-%s-%s.iso", talosVersion, architecture)

	return &models.ISOResponse{
		DownloadURL:  downloadURL,
		Filename:     filename,
		SchematicID:  schematicID,
		TalosVersion: talosVersion,
		Architecture: architecture,
	}, nil
}

// GetMachineConfigBundle gets bundle.Bundle constructed from files in TALOS_FOLDER
func (s *TalosService) GetMachineConfigBundle() (*bundle.Bundle, error) {
	configBundleOpts := []bundle.Option{
		bundle.WithExistingConfigs(s.cfg.TalosFolder),
	}

	return bundle.NewBundle(configBundleOpts...)
}

// GetMachineryClient gets Talos machinery client from talosconfig in TALOS_FOLDER
func (s *TalosService) GetMachineryClient(nodeIP string) (*machineryClient.Client, error) {
	endpoint := net.JoinHostPort(nodeIP, "50000")
	talosConfigBytes, _ := os.ReadFile(filepath.Join(s.cfg.TalosFolder, "talosconfig"))
	talosconfig, err := machineryClientConfig.FromBytes(talosConfigBytes)

	if err != nil {
		return nil, err
	}

	return machineryClient.New(
		context.Background(),
		machineryClient.WithConfig(talosconfig), // talosconfig provides certs
		machineryClient.WithEndpoints(endpoint),
	)

}

func GetInsecureMachineryClient(ctx context.Context, nodeIP string) (*machineryClient.Client, error) {
	endpoint := net.JoinHostPort(nodeIP, "50000")
	tlsCfg := &tls.Config{}
	tlsCfg.InsecureSkipVerify = true

	return machineryClient.New(context.Background(),
		machineryClient.WithEndpoints(endpoint),
		machineryClient.WithTLSConfig(tlsCfg),
	)
}

// GetNodeDisks retrieves disk information from a Talos node. NOTE: The node is set via the machinery client's endpoint!
func (s *TalosService) GetNodeDisks(ctx context.Context, client *machineryClient.Client) ([]string, error) {
	disksRes, err := client.Disks(ctx)
	if err != nil {
		return nil, err
	}
	disks := disksRes.GetMessages()[0].Disks

	var diskPaths []string
	for _, disk := range disks {
		diskPaths = append(diskPaths, disk.DeviceName)
	}

	return diskPaths, nil
}

// GetBootstrapCachedNodes reads cached machine definitions and builds Node models.
func (s *TalosService) GetBootstrapCachedNodes(clusterID uuid.UUID) ([]*models.Node, error) {
	machinesJsonBytes, err := os.ReadFile(filepath.Join(s.cfg.TalosFolder, "machines.json"))
	if err != nil {
		return nil, err
	}

	var machines talos.Machines
	if err := json.Unmarshal(machinesJsonBytes, &machines); err != nil {
		return nil, err
	}

	var nodes []*models.Node

	for ip := range machines.ControlPlanes {
		node, err := s.CreateExistingNodeFromIP(context.Background(), ip, "controlplane")
		if err != nil {
			// fallback: still return minimal node with IP
			node = &models.Node{
				ID:        uuid.New(),
				Role:      "controlplane",
				IPAddress: ip,
				Status:    "active",
			}
		}
		node.ClusterID = clusterID
		node.Provider = "onprem"
		nodes = append(nodes, node)
	}

	for ip := range machines.Workers {
		node, err := s.CreateExistingNodeFromIP(context.Background(), ip, "worker")
		if err != nil {
			node = &models.Node{
				ID:        uuid.New(),
				Role:      "worker",
				IPAddress: ip,
				Status:    "active",
			}
		}
		node.ClusterID = clusterID
		node.Provider = "onprem"
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// CreateExistingNodeFromIP contacts an existing Talos node and fills in: Name, Architecture, MACAddress.
func (s *TalosService) CreateExistingNodeFromIP(ctx context.Context, nodeIP string, role string) (*models.Node, error) {

	var node models.Node

	node.Role = role

	cli, err := s.GetMachineryClient(nodeIP) //Get authenticated client.
	if err != nil {
		return nil, fmt.Errorf("machinery client: %w", err)
	}
	defer cli.Close()

	status, err := GetMachineStatus(cli)
	if err == nil && status.Stage == runtime.MachineStageRunning {
		node.Status = models.StatusActive
	} else {
		node.Status = models.StatusPending
	}

	// Get hostname
	hostname, err := GetTypedTalosResource[*netres.HostnameStatus](ctx, cli, netres.NamespaceName, netres.HostnameStatusType, "hostname")
	if err != nil {
		return nil, err
	}
	node.Name = hostname.TypedSpec().Hostname

	// Get talos version
	// version, err := GetTypedTalosResource[*runtime.Version](ctx, cli, runtime.NamespaceName, runtime.VersionType, "runtime")
	// node.Architecture = version.TypedSpec().Version

	node.MACAddress = GetMachineBestExternalMacCandidate(ctx, cli)

	return &node, nil
}

// GetGCPImageName returns the Talos image name for the specified architecture
// Images are uploaded to the user's project and stored in GCPConfig
func (s *TalosService) GetGCPImageName(architecture string) (string, error) {
	// Get GCP config from database
	var gcpConfig models.GCPConfig
	if err := s.db.Where("is_configured = ?", true).First(&gcpConfig).Error; err != nil {
		return "", fmt.Errorf("failed to get GCP config: %w", err)
	}

	// Get image name based on architecture
	var image string
	switch architecture {
	case "amd64", "":
		image = gcpConfig.TalosImageAMD64
	case "arm64":
		image = gcpConfig.TalosImageARM64
	default:
		return "", fmt.Errorf("unsupported architecture: %s", architecture)
	}

	if image == "" {
		return "", fmt.Errorf("talos image not found for architecture %s - image upload may still be in progress", architecture)
	}

	return image, nil
}
