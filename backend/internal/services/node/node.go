package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/google/uuid"
	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	machineconf "github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	"github.com/stolos-cloud/stolos/backend/internal/services/talos"
	"gorm.io/gorm"
)

type NodeService struct {
	db              *gorm.DB
	cfg             *config.Config
	providerManager *services.ProviderManager
	ts              *talos.TalosService
}

func NewNodeService(db *gorm.DB, cfg *config.Config, providerManager *services.ProviderManager, talosService *talos.TalosService) *NodeService {
	return &NodeService{
		db:              db,
		cfg:             cfg,
		providerManager: providerManager,
		ts:              talosService,
	}
}

// Sample: list all instances
func (s *NodeService) QueryGCPInstances(ctx context.Context) error {
	provider, ok := s.providerManager.GetProvider("gcp")
	if !ok {
		return fmt.Errorf("GCP provider not configured")
	}

	gcpService, ok := provider.(*gcpservices.GCPService)
	if !ok {
		return fmt.Errorf("provider is not GCP")
	}

	client, err := gcpService.GetClient()
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	// List all instances across all zones
	allInstances, err := client.ListAllInstances(ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Log the results
	for zone, instances := range allInstances {
		fmt.Printf("Zone %s: %d instances\n", zone, len(instances))
		for _, instance := range instances {
			fmt.Printf("  - %s (%s)\n", instance.Name, instance.Status)
		}
	}

	return nil
}

func (s *NodeService) CreateNode(name, architecture, provider string, clusterID uuid.UUID) (*models.Node, error) {
	node := &models.Node{
		ID:           uuid.New(),
		Name:         name,
		Status:       models.StatusPending,
		Architecture: architecture,
		Provider:     provider,
		ClusterID:    clusterID,
	}

	if err := s.db.Create(node).Error; err != nil {
		return nil, err
	}

	return node, nil
}

func (s *NodeService) GetNode(id uuid.UUID) (*models.Node, error) {
	var node models.Node
	if err := s.db.First(&node, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

// UpdateActiveNodeConfig updates role and labels for a single active node
func (s *NodeService) UpdateActiveNodeConfig(id uuid.UUID, role string, labels []string) (*models.Node, error) {
	var node models.Node
	if err := s.db.First(&node, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("node %s not found", id)
		}
		return nil, fmt.Errorf("failed to fetch node %s: %w", id, err)
	}

	// Only allow updating active nodes
	if node.Status != models.StatusActive {
		return nil, fmt.Errorf("node %s must be active to update config (current: %s)", id, node.Status)
	}

	node.Role = role
	if len(labels) > 0 {
		labelsJSON, err := json.Marshal(labels)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal labels: %w", err)
		}
		node.Labels = string(labelsJSON)
	}

	if err := s.db.Save(&node).Error; err != nil {
		return nil, fmt.Errorf("failed to update node %s: %w", id, err)
	}

	// TODO: Also update Talos node config
	// 1. Get current Talos config bundle
	// 2. Apply role/label patches to node config
	// 3. Re-apply config to node via Talos API

	return &node, nil
}

type NodeConfigUpdate struct {
	ID     uuid.UUID `json:"id"`
	Labels []string  `json:"labels"`
}

// UpdateActiveNodesConfig updates labels for multiple active nodes
func (s *NodeService) UpdateActiveNodesConfig(updates []NodeConfigUpdate) ([]models.Node, error) {
	var updatedNodes []models.Node

	for _, update := range updates {
		var node models.Node
		if err := s.db.First(&node, "id = ?", update.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("node %s not found", update.ID)
			}
			return nil, fmt.Errorf("failed to find node %s: %w", update.ID, err)
		}

		// Only allow updating active nodes
		if node.Status != models.StatusActive {
			return nil, fmt.Errorf("node %s must be active to update labels (current: %s)", update.ID, node.Status)
		}

		// Only update labels (do not change role or status)
		if len(update.Labels) > 0 {
			labelsJSON, err := json.Marshal(update.Labels)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal labels for node %s: %w", update.ID, err)
			}
			node.Labels = string(labelsJSON)
		}

		if err := s.db.Save(&node).Error; err != nil {
			return nil, fmt.Errorf("failed to update node %s: %w", update.ID, err)
		}

		updatedNodes = append(updatedNodes, node)
	}

	//todo also update Talos node config
	// 1. Get current Talos config bundle
	// 2. Apply label patches to node configs
	// 3. Re-apply configs to nodes via Talos API

	return updatedNodes, nil
}

// ProvisionNodes provisions multiple on-prem nodes by updating their role and labels,
// then applying Talos machine configuration. It continues processing all nodes even if
// some fail, returning a result list with per-node success/error details.
func (s *NodeService) ProvisionNodes(configs []models.OnPremNodeProvisionConfig) ([]models.NodeProvisionResult, error) {
	results := make([]models.NodeProvisionResult, 0, len(configs))

	// Get machine config bundle from TALOS_FOLDER, fatal if it fails
	configBundle, err := s.ts.GetMachineConfigBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get machine config bundle: %w", err)
	}

	for _, cfg := range configs {
		result := models.NodeProvisionResult{
			NodeID: cfg.NodeID,
			Role:   cfg.Role,
			Labels: cfg.Labels,
		}

		// ensure valid role
		if cfg.Role != "worker" && cfg.Role != "control-plane" {
			result.Error = fmt.Sprintf("invalid role '%s' (must be 'worker' or 'control-plane')", cfg.Role)
			results = append(results, result)
			continue
		}

		// Get node from db
		var node models.Node
		if err := s.db.Where("id = ? AND provider = ?", cfg.NodeID, "onprem").First(&node).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.Error = fmt.Sprintf("node %s not found or not an on-prem node", cfg.NodeID)
			} else {
				result.Error = fmt.Sprintf("failed to fetch node %s: %v", cfg.NodeID, err)
			}
			results = append(results, result)
			continue
		}

		// Sanity check node state
		if node.Status != models.StatusPending {
			result.Error = fmt.Sprintf("node %s must be pending to provision (current: %s)", cfg.NodeID, node.Status)
			results = append(results, result)
			continue
		}

		// Update role and labels
		node.Role = cfg.Role
		if len(cfg.Labels) > 0 {
			if labelsJSON, err := json.Marshal(cfg.Labels); err != nil {
				result.Error = fmt.Sprintf("failed to marshal labels: %v", err)
				results = append(results, result)
				continue
			} else {
				node.Labels = string(labelsJSON)
			}
		}
		if err := s.db.Save(&node).Error; err != nil {
			result.Error = fmt.Sprintf("failed to update node in DB: %v", err)
			results = append(results, result)
			continue
		}

		// get talos api client for node
		cli, err := talos.GetInsecureMachineryClient(context.Background(), node.IPAddress)
		if err != nil {
			result.Error = fmt.Sprintf("failed to get Talos client for %s: %v", node.IPAddress, err)
			results = append(results, result)
			continue
		}

		// get existing node count for this type
		var existingNodeCount int64
		if err := s.db.Model(&models.Node{}).
			Where("status = 'active' AND role = ?", node.Role).
			Count(&existingNodeCount).Error; err != nil {
			result.Error = fmt.Sprintf("failed to count existing nodes: %v", err)
			results = append(results, result)
			continue
		}
		nodeName := fmt.Sprintf("%s-%d", node.Role, int(existingNodeCount)+1)

		// create machineConfig for the node (part 1)
		var machineType machineconf.Type
		if node.Role == "control-plane" {
			machineType = machineconf.TypeControlPlane
		} else {
			machineType = machineconf.TypeWorker
		}
		rendered, err := configBundle.Serialize(encoder.CommentsDocs, machineType)
		if err != nil {
			result.Error = fmt.Sprintf("failed to serialize config: %v", err)
			results = append(results, result)
			continue
		}

		// use json patch to overwrite certain values (part 2)
		patch := jsonpatch.Patch{
			jsonpatch.Operation{
				"op":    raw("replace"),
				"path":  raw("/machine/network/hostname"),
				"value": raw(nodeName),
			},
			jsonpatch.Operation{
				"op":   raw("remove"),
				"path": raw("/machine/install/diskSelector"),
			},
			jsonpatch.Operation{
				"op":    raw("add"),
				"path":  raw("/machine/install/disk"),
				"value": raw(cfg.InstallDisk),
			},
		}

		patched, err := configpatcher.JSON6902(rendered, patch)
		if err != nil {
			result.Error = fmt.Sprintf("failed to apply JSON patch: %v", err)
			results = append(results, result)
			continue
		}

		// Send talos ApplyConfiguration request
		_, err = cli.ApplyConfiguration(context.Background(), &machineapi.ApplyConfigurationRequest{
			Data:   patched,
			Mode:   machineapi.ApplyConfigurationRequest_AUTO,
			DryRun: false,
		})
		if err != nil {
			result.Error = fmt.Sprintf("failed to apply configuration: %v", err)
			results = append(results, result)
			continue
		}

		// Set node to "provisioning" status
		node.Status = models.StatusProvisioning
		node.Name = nodeName
		if err := s.db.Save(&node).Error; err != nil {
			result.Error = fmt.Sprintf("failed to update DB after provisioning: %v", err)
			results = append(results, result)
			continue
		}

		result.Succeeded = true
		results = append(results, result)
	}

	return results, nil
}

func raw(v any) *json.RawMessage {
	b, _ := json.Marshal(v)
	rm := json.RawMessage(b)
	return &rm
}

// ListNodes lists nodes with optional status filter, offset, and limit.
// Pass empty string for status to list all nodes.
// Pass 0 for offset and limit to get all results without pagination.
func (s *NodeService) ListNodes(status string, offset, limit int) ([]models.Node, error) {
	var nodes []models.Node
	query := s.db.Order("created_at DESC")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	if err := query.Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// CreateSamplePendingNodes creates sample pending nodes if none exist in the database
func (s *NodeService) CreateSamplePendingNodes() error {
	// Check if there are any pending nodes
	var count int64
	if err := s.db.Model(&models.Node{}).Where("status = ?", models.StatusPending).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count pending nodes: %w", err)
	}

	// If pending nodes already exist, skip creation
	if count > 0 {
		return nil
	}

	// Get or create default cluster
	var cluster models.Cluster
	err := s.db.Where("name = ?", "sample-cluster").First(&cluster).Error
	if err == gorm.ErrRecordNotFound {
		cluster = models.Cluster{
			ID:   uuid.New(),
			Name: "sample-cluster",
		}
		if err := s.db.Create(&cluster).Error; err != nil {
			return fmt.Errorf("failed to create cluster: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query cluster: %w", err)
	}

	// Create sample nodes with auto-generated names
	sampleNodes := []models.Node{
		{
			ID:           uuid.New(),
			Name:         fmt.Sprintf("node-%s", uuid.New().String()[:8]),
			Status:       models.StatusPending,
			Architecture: "amd64",
			Provider:     "onprem",
			ClusterID:    cluster.ID,
		},
		{
			ID:           uuid.New(),
			Name:         fmt.Sprintf("node-%s", uuid.New().String()[:8]),
			Status:       models.StatusPending,
			Architecture: "arm64",
			Provider:     "onprem",
			ClusterID:    cluster.ID,
		},
	}

	for _, node := range sampleNodes {
		if err := s.db.Create(&node).Error; err != nil {
			return fmt.Errorf("failed to create node %s: %w", node.Name, err)
		}
	}

	return nil
}
