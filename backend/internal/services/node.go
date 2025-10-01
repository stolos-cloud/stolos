package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/pkg/gcp"
	"gorm.io/gorm"
)

type NodeService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewNodeService(db *gorm.DB, cfg *config.Config) *NodeService {
	return &NodeService{db: db, cfg: cfg}
}

func (s *NodeService) QueryGCPInstances(ctx context.Context) error {
	// Sample: Create GCP client from environment and list all instances
	client, err := gcp.NewClientFromEnv()
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
		Status:       "pending",
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

func (s *NodeService) ListNodes(offset, limit int) ([]models.Node, error) {
	var nodes []models.Node
	if err := s.db.Offset(offset).Limit(limit).Order("created_at DESC").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

func (s *NodeService) ListPendingNodes() ([]models.Node, error) {
	var nodes []models.Node
	if err := s.db.Where("status = ?", "pending").Order("created_at DESC").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// Create sample of pending nodes in db to return in http handler
func (s *NodeService) CreateSamplePendingNodes() error {

	cluster := models.Cluster{
		ID:     uuid.New(),
		Name:   "sample-cluster",
	}

	if err := s.db.Create(&cluster).Error; err != nil {
		// return err
		fmt.Println("Cluster already exists, skipping creation")
	}

	sampleNodes := []models.Node{
		{
			ID:           uuid.New(),
			Name:         "node-1",
			Status:       "pending",
			Architecture: "amd64",
			Provider:     "onprem",
			ClusterID:    cluster.ID,
		},
		{
			ID:           uuid.New(),
			Name:         "node-2",
			Status:       "pending",
			Architecture: "arm64",
			Provider:     "onprem",
			ClusterID:    cluster.ID,
		},
	}

	for _, node := range sampleNodes {
		if err := s.db.Create(&node).Error; err != nil {
			fmt.Println("Node already exists, skipping creation")
			// return err
		}
	}

	return nil
}