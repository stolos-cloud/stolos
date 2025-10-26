package cluster

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/internal/services/talos"
	"gorm.io/gorm"
)

// DiscoveryService handles cluster discovery and initialization
type DiscoveryService struct {
	db  *gorm.DB
	cfg *config.Config
	ts  *talos.TalosService
}

// NewDiscoveryService creates a new cluster discovery service
func NewDiscoveryService(db *gorm.DB, cfg *config.Config, ts *talos.TalosService) *DiscoveryService {
	return &DiscoveryService{
		db:  db,
		cfg: cfg,
		ts:  ts,
	}
}

// InitializeCluster ensures a cluster exists in the database
func (s *DiscoveryService) InitializeCluster(ctx context.Context) error {
	log.Println("Starting cluster discovery...")

	var existingCluster models.Cluster
	err := s.db.First(&existingCluster).Error
	if err == nil {
		// Cluster already exists
		log.Printf("Found existing cluster: %s (ID: %s)", existingCluster.Name, existingCluster.ID)
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to query clusters: %w", err)
	}

	log.Println("No cluster found in database, creating cluster record...")

	// Get cluster name from config or use default
	clusterName := s.cfg.ClusterName
	if clusterName == "" {
		clusterName = "stolos-cluster"
	}

	cluster := models.Cluster{
		ID:   uuid.New(),
		Name: clusterName,
	}

	if err := s.db.Create(&cluster).Error; err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	log.Printf("Created cluster: %s (ID: %s)", cluster.Name, cluster.ID)

	// Discover existing nodes
	if err := s.discoverNodes(ctx, cluster.ID); err != nil {
		log.Printf("Warning: failed to discover nodes: %v", err)
		// Don't fail initialization if node discovery fails
	}

	return nil
}

// discoverNodes discovers existing nodes in the Talos cluster
func (s *DiscoveryService) discoverNodes(ctx context.Context, clusterID uuid.UUID) error {
	log.Println("Discovering existing cluster nodes from TALOS_FOLDER ...")

	nodes, err := s.ts.GetBootstrapCachedNodes(clusterID)

	if err != nil {
		return fmt.Errorf("failed to get bootstrap nodes: %w", err)
	}

	if err := s.db.Save(&nodes).Error; err != nil {
		log.Printf("Failed to save nodes: %v", err)
	}

	return nil
}
