package talos

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	factoryClient "github.com/siderolabs/image-factory/pkg/client"
	"github.com/siderolabs/image-factory/pkg/schematic"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	machineryClientConfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/container"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	netres "github.com/siderolabs/talos/pkg/machinery/resources/network"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"gorm.io/gorm"
)

var ignoredNodesCache []string

type TalosService struct {
	db            *gorm.DB
	cfg           *config.Config
	factoryClient *factoryClient.Client
	wsManager     *wsservices.Manager
}

// MachineConfigRequest represents parameters for generating machine configs
type MachineConfigRequest struct {
	ClusterName       string `json:"cluster_name"`
	KubernetesVersion string `json:"kubernetes_version"`
	TalosVersion      string `json:"talos_version"`
	ControlPlaneIP    string `json:"control_plane_ip"`
}

func NewTalosService(db *gorm.DB, cfg *config.Config, wsManager *wsservices.Manager) *TalosService {
	factory := talos.CreateFactoryClient()
	return &TalosService{
		db:            db,
		cfg:           cfg,
		factoryClient: factory,
		wsManager:     wsManager,
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

// GetMachineConfigBundle gets bundle.Bundle from database or TALOS_FOLDER
// DB first  then file fallback
func (s *TalosService) GetMachineConfigBundle() (*bundle.Bundle, error) {
	// get configs from database first
	cpConfig, workerConfig, err := s.GetMachineConfigsFromDB()
	if err == nil && len(cpConfig) > 0 && len(workerConfig) > 0 {
		// Construct bundle from DB configs
		tmpDir, err := os.MkdirTemp("", "talos-configs-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// Write configs to temp files
		cpPath := filepath.Join(tmpDir, "controlplane.yaml")
		if err := os.WriteFile(cpPath, cpConfig, 0600); err != nil {
			return nil, fmt.Errorf("failed to write controlplane config: %w", err)
		}

		workerPath := filepath.Join(tmpDir, "worker.yaml")
		if err := os.WriteFile(workerPath, workerConfig, 0600); err != nil {
			return nil, fmt.Errorf("failed to write worker config: %w", err)
		}

		// Create bundle from temp files
		return bundle.NewBundle(bundle.WithExistingConfigs(tmpDir))
	}

	// Fallback to file configs
	if s.cfg.TalosFolder != "" {
		return bundle.NewBundle(bundle.WithExistingConfigs(s.cfg.TalosFolder))
	}

	return nil, fmt.Errorf("no machine configs in database and TALOS_FOLDER not configured")
}

// GetMachineryClient gets Talos machinery client from database or TALOS_FOLDER
// DB first, then file fallback
func (s *TalosService) GetMachineryClient(nodeIP string) (*machineryClient.Client, error) {
	endpoint := net.JoinHostPort(nodeIP, "50000")

	talosConfigBytes, err := s.GetTalosConfigFromDB()
	if err != nil {
		// Fallback to file
		if s.cfg.TalosFolder != "" {
			talosConfigBytes, err = os.ReadFile(filepath.Join(s.cfg.TalosFolder, "talosconfig"))
			if err != nil {
				return nil, fmt.Errorf("failed to load talosconfig from DB or file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no talosconfig in database and TALOS_FOLDER not configured: %w", err)
		}
	}

	talosconfig, err := machineryClientConfig.FromBytes(talosConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse talosconfig: %w", err)
	}

	return machineryClient.New(
		context.Background(),
		machineryClient.WithConfig(talosconfig), // talosconfig provides certs
		machineryClient.WithEndpoints(endpoint),
	)
}

// GetReachableMachineryClient iterates over nodes to find the first reachable Talos machinery client.
func (s *TalosService) GetReachableMachineryClient(ctx context.Context) (*machineryClient.Client, *models.Node, error) {
	var nodes []models.Node
	if err := s.db.Where("ip_address <> ''").Find(&nodes).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to load nodes: %w", err)
	}

	for _, node := range nodes {
		client, err := s.GetMachineryClient(node.IPAddress)
		if err != nil {
			continue
		}

		if _, err := GetMachineStatus(client); err != nil {
			continue
		}

		return client, &node, nil
	}

	return nil, nil, fmt.Errorf("no reachable Talos nodes found")
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
	machinesJsonBytes, err := os.ReadFile(filepath.Join(s.cfg.TalosFolder, "machines.json")) // TODO : Store the config bundles in DB ?
	if err != nil {
		return nil, err
	}

	var machines talos.Machines
	if err := json.Unmarshal(machinesJsonBytes, &machines); err != nil {
		return nil, err
	}

	var nodes []*models.Node

	for ip := range machines.ControlPlanes {
		node, err := s.CreateExistingNodeFromIP(context.Background(), ip, "control-plane")
		if err != nil {
			// fallback: still return minimal node with IP
			node = &models.Node{
				ID:        uuid.New(),
				Role:      "control-plane",
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
	node.IPAddress = nodeIP

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

	if iface := GetMachineBestExternalNetworkInterface(ctx, cli); iface != nil {
		node.MACAddress = iface.Mac
	}

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

// StoreTalosConfig stores the Talos configuration bundle in the database
// Includes talosconfig, controlplane config, worker config, and bundle
func (s *TalosService) StoreTalosConfig(clusterID uuid.UUID, talosVersion, kubeVersion string, talosConfig, cpConfig, workerConfig, fullBundle []byte) error {
	var cluster models.Cluster
	if err := s.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		return fmt.Errorf("cluster not found: %w", err)
	}

	// Update cluster with Talos configuration
	cluster.TalosVersion = talosVersion
	cluster.KubeVersion = kubeVersion
	cluster.TalosConfig = talosConfig
	cluster.ControlPlaneConfig = cpConfig
	cluster.WorkerConfig = workerConfig
	cluster.ConfigBundle = fullBundle

	if err := s.db.Save(&cluster).Error; err != nil {
		return fmt.Errorf("failed to save Talos config to database: %w", err)
	}

	return nil
}

// GetTalosConfigFromDB retrieves the talosconfig from the database
func (s *TalosService) GetTalosConfigFromDB() ([]byte, error) {
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return nil, fmt.Errorf("no cluster found in database: %w", err)
	}

	if len(cluster.TalosConfig) == 0 {
		return nil, fmt.Errorf("no talosconfig stored in database")
	}

	return cluster.TalosConfig, nil
}

// GetMachineConfigsFromDB retrieves controlplane and worker configs from database
func (s *TalosService) GetMachineConfigsFromDB() (controlplane []byte, worker []byte, err error) {
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return nil, nil, fmt.Errorf("no cluster found in database: %w", err)
	}

	if len(cluster.ControlPlaneConfig) == 0 || len(cluster.WorkerConfig) == 0 {
		return nil, nil, fmt.Errorf("machine configs not stored in database")
	}

	return cluster.ControlPlaneConfig, cluster.WorkerConfig, nil
}

// MigrateTalosConfigFromFiles migrates Talos configs from TALOS_FOLDER to database
func (s *TalosService) MigrateTalosConfigFromFiles() error {
	if s.cfg.TalosFolder == "" {
		return fmt.Errorf("TALOS_FOLDER not configured, cannot migrate")
	}

	// Check if already migrated
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return fmt.Errorf("no cluster found in database: %w", err)
	}

	if len(cluster.TalosConfig) > 0 {
		// Already migrated
		return nil
	}

	// Read talosconfig
	talosConfigPath := filepath.Join(s.cfg.TalosFolder, "talosconfig")
	talosConfig, err := os.ReadFile(talosConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read talosconfig: %w", err)
	}

	// Read controlplane.yaml
	cpConfigPath := filepath.Join(s.cfg.TalosFolder, "controlplane.yaml")
	cpConfig, err := os.ReadFile(cpConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read controlplane.yaml: %w", err)
	}

	// Read worker.yaml
	workerConfigPath := filepath.Join(s.cfg.TalosFolder, "worker.yaml")
	workerConfig, err := os.ReadFile(workerConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read worker.yaml: %w", err)
	}

	// default .. todo
	talosVersion := "v1.11.1"
	kubeVersion := "v1.32.1"

	// Store in database
	if err := s.StoreTalosConfig(cluster.ID, talosVersion, kubeVersion, talosConfig, cpConfig, workerConfig, nil); err != nil {
		return fmt.Errorf("failed to store configs in database: %w", err)
	}

	fmt.Printf("Successfully migrated Talos configs from %s to database\n", s.cfg.TalosFolder)
	return nil
}

// CreateMachineConfigPatch creates a machine config patch for a node.
// It applies hostname, disk, network settings, and node labels (including static subnets for hybrid cloud).
func CreateMachineConfigPatch(hostname, installDisk string, labels []string) (configpatcher.Patch, error) {
	kubeletConfig := &v1alpha1.KubeletConfig{
		KubeletNodeIP: &v1alpha1.KubeletNodeIPConfig{
			// Use static private network subnets for node IP selection
			KubeletNodeIPValidSubnets: []string{
				"10.0.0.0/8",     // RFC 1918 Class A private networks
				"172.16.0.0/12",  // RFC 1918 Class B private networks
				"192.168.0.0/16", // RFC 1918 Class C private networks
				"fdd6::/16",      // KubeSpan IPv6 overlay network
			},
		},
	}

	// Add node labels if provided
	if len(labels) > 0 {
		kubeletConfig.KubeletExtraArgs = map[string]string{
			"node-labels": strings.Join(labels, ","),
		}
	}

	cfg := &v1alpha1.Config{
		ConfigVersion: "v1alpha1",
		MachineConfig: &v1alpha1.MachineConfig{
			MachineNetwork: &v1alpha1.NetworkConfig{
				NetworkHostname: hostname,
			},
			MachineInstall: &v1alpha1.InstallConfig{
				InstallDisk: installDisk,
				// Explicitly set diskSelector to nil to remove hardware-specific busPath
				InstallDiskSelector: nil,
			},
			MachineKubelet: kubeletConfig,
		},
	}

	// Wrap in container and create strategic merge patch
	ctr := container.NewV1Alpha1(cfg)
	return configpatcher.NewStrategicMergePatch(ctr), nil
}
