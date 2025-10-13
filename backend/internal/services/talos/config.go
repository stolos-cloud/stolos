package talos

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	factoryClient "github.com/siderolabs/image-factory/pkg/client"
	"github.com/siderolabs/image-factory/pkg/schematic"
	"github.com/siderolabs/siderolink/pkg/events"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	machineryClientConfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"gorm.io/gorm"
)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	talosConfigBytes, _ := os.ReadFile(filepath.Join(s.cfg.TalosFolder, "talosconfig"))

	talosconfig, err := machineryClientConfig.FromBytes(talosConfigBytes)

	if err != nil {
		return nil, err
	}

	machinery, err := machineryClient.New(
		ctx,
		machineryClient.WithConfig(talosconfig),
		machineryClient.WithEndpoints(nodeIP),
		// TODO : We may need credentials since we are no longer in maintenance mode for provisioned nodes.
		//machineryClient.WithGRPCDialOptions(
		//	grpc.WithTransportCredentials(insecure.NewCredentials()),
		//),
	)

	return machinery, nil
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
		node, err := s.CreateNodeFromIP(context.Background(), ip, "controlplane")
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
		node, err := s.CreateNodeFromIP(context.Background(), ip, "worker")
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

// CreateNodeFromIP contacts a Talos node and fills in: Name, Architecture, MACAddress.
func (s *TalosService) CreateNodeFromIP(ctx context.Context, nodeIP string, role string) (*models.Node, error) {

	var node models.Node

	node.Role = role
	node.Status = "active"

	cli, err := s.GetMachineryClient(nodeIP)
	if err != nil {
		return nil, fmt.Errorf("machinery client: %w", err)
	}
	defer cli.Close()

	name, err := readFileTrim(ctx, cli, "/proc/sys/kernel/hostname")
	if err == nil && name != "" {
		node.Name = name
	}

	verResp, err := cli.Version(ctx)
	if err == nil && verResp != nil {
		msgs := verResp.GetMessages()
		if len(msgs) > 0 {
			if v := msgs[0].GetVersion(); v != nil {
				if arch := v.GetArch(); arch != "" {
					node.Architecture = arch
				}
			}
		}
	} else if err != nil {
		_ = err
	}

	iface, err := findFirstIfaceMacAddr(ctx, cli)
	if err == nil && iface != "" {
		mac, macErr := readFileTrim(ctx, cli, "/sys/class/net/"+iface+"/address")
		if macErr == nil && mac != "" && mac != "00:00:00:00:00:00" {
			node.MACAddress = strings.ToLower(mac)
		}
	}

	return &node, nil
}

// readFileTrim reads a file via Talos client.Read and returns trimmed string.
func readFileTrim(ctx context.Context, cli *machineryClient.Client, path string) (string, error) {
	rc, err := cli.Read(ctx, path)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// findFirstIfaceMacAddr returns the first reasonable non-loopback interface name as seen in /proc/net/dev.
func findFirstIfaceMacAddr(ctx context.Context, cli *machineryClient.Client) (string, error) {
	rc, err := cli.Read(ctx, "/proc/net/dev")
	if err != nil {
		return "", err
	}
	defer rc.Close()

	sc := bufio.NewScanner(rc)
	skipPrefixes := []string{"lo", "bond", "br", "veth", "docker", "cni", "flannel", "kube", "wg", "tun", "tap"}

	isVirtual := func(name string) bool {
		for _, p := range skipPrefixes {
			if strings.HasPrefix(name, p) {
				return true
			}
		}
		return false
	}

	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "" || isVirtual(iface) {
			continue
		}

		// Confirm it has a sane MAC address file and non-zero MAC.
		mac, err := readFileTrim(ctx, cli, "/sys/class/net/"+iface+"/address")
		if err != nil || mac == "" || mac == "00:00:00:00:00:00" {
			continue
		}

		// Optional: prefer interfaces that are up; if operstate is readable and says "up", return immediately.
		if oper, operErr := readFileTrim(ctx, cli, "/sys/class/net/"+iface+"/operstate"); operErr == nil && oper == "up" {
			return iface, nil
		}

		// Fallback: first non-virtual with MAC.
		return iface, nil
	}

	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no suitable interface found")
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

// starts the Talos event sink gRPC server to receive events from booting nodes
func (s *TalosService) StartEventSink() {
	// Skip if event sink hostname is not configured
	if s.cfg.Talos.EventSinkHostname == "" {
		log.Println("Talos event sink hostname not configured, skipping event sink startup")
		return
	}

	log.Printf("Starting Talos event sink on %s:%s", s.cfg.Talos.EventSinkHostname, s.cfg.Talos.EventSinkPort)

	// Prepare talosInfo struct for EventSink
	talosInfo := &talos.TalosInfo{
		HTTPHostname: s.cfg.Talos.EventSinkHostname,
		HTTPPort:     s.cfg.Talos.EventSinkPort,
	}

	// Start EventSink
	go func() {
		err := talos.EventSink(talosInfo, func(ctx context.Context, event events.Event) error {
			// Extract IP from event.Node
			ip := strings.Split(event.Node, ":")[0]

			// Check if node already exists
			var existing models.Node
			err := s.db.Where("ip_address = ? AND provider = ?", ip, "onprem").First(&existing).Error

			if err == gorm.ErrRecordNotFound {
				// Auto-register new node
				node := models.Node{
					Name:         fmt.Sprintf("node-%s", strings.ReplaceAll(ip, ".", "-")),
					IPAddress:    ip,
					Provider:     "onprem",
					Status:       models.StatusPending,
					Architecture: "amd64", // todo detect architecture
				}

				if err := s.db.Create(&node).Error; err != nil {
					log.Printf("Failed to auto-register node %s: %v", ip, err)
					return err
				}

				log.Printf("Auto-registered new on-prem node: %s (IP: %s)", node.Name, ip)
			} else if err != nil {
				log.Printf("Error checking node existence for IP %s: %v", ip, err)
				return err
			} else {
				log.Printf("Node with IP %s already registered", ip)
			}

			return nil
		})

		if err != nil {
			log.Printf("Talos event sink error: %v", err)
		}
	}()

	log.Printf("Talos event sink started successfully")
}
