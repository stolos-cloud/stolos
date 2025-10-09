package talos

import (
	"context"
	"fmt"
	"log"
	"strings"

	factoryClient "github.com/siderolabs/image-factory/pkg/client"
	"github.com/siderolabs/image-factory/pkg/schematic"
	"github.com/siderolabs/siderolink/pkg/events"
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

// MachineConfigRequest represents parameters for generating machine configs
type MachineConfigRequest struct {
	ClusterName       string `json:"cluster_name"`
	KubernetesVersion string `json:"kubernetes_version"`
	TalosVersion      string `json:"talos_version"`
	ControlPlaneIP    string `json:"control_plane_ip"`
}

// GenerateMachineConfigBundle creates a new Talos config bundle
func (s *TalosService) GenerateMachineConfigBundle(req *MachineConfigRequest) (*bundle.Bundle, error) {
	talosInfo := &talos.TalosInfo{
		ClusterName:       req.ClusterName,
		KubernetesVersion: req.KubernetesVersion,
		TalosVersion:      req.TalosVersion,
	}

	configBundle, err := talos.CreateMachineConfigBundle(req.ControlPlaneIP, talosInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create config bundle: %w", err)
	}

	return configBundle, nil
}

// GetNodeDisks retrieves disk information from a Talos node
func (s *TalosService) GetNodeDisks(ctx context.Context, nodeIP string) ([]string, error) {
	disks, err := talos.GetDisks(ctx, nodeIP)
	if err != nil {
		return nil, fmt.Errorf("failed to get disks from node %s: %w", nodeIP, err)
	}

	var diskPaths []string
	for _, disk := range disks {
		diskPaths = append(diskPaths, disk.DeviceName)
	}

	return diskPaths, nil
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
