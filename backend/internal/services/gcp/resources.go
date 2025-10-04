package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"gorm.io/gorm"
)

type GCPResourcesService struct {
	db         *gorm.DB
	gcpService *GCPService
}

func NewGCPResourcesService(db *gorm.DB, gcpService *GCPService) *GCPResourcesService {
	return &GCPResourcesService{
		db:         db,
		gcpService: gcpService,
	}
}

func (s *GCPResourcesService) GetResources() (*config.GCPResources, error) {
	var cache models.GCPResources
	err := s.db.Order("updated_at DESC").First(&cache).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to fetch GCP resources from database: %w", err)
	}

	var resources config.GCPResources
	if err := json.Unmarshal(cache.Resources, &resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GCP resources: %w", err)
	}

	return &resources, nil
}

func (s *GCPResourcesService) SaveResources(resources *config.GCPResources) error {
	jsonData, err := json.Marshal(resources)
	if err != nil {
		return fmt.Errorf("failed to marshal GCP resources: %w", err)
	}

	// Check if a record exists
	var cache models.GCPResources
	err = s.db.Order("updated_at DESC").First(&cache).Error

	if err == gorm.ErrRecordNotFound {
		// Create new record
		cache = models.GCPResources{
			ID:          uuid.New(),
			LastUpdated: time.Now().UTC(),
			Resources:   jsonData,
		}
		if err := s.db.Create(&cache).Error; err != nil {
			return fmt.Errorf("failed to create GCP resources cache: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query GCP resources cache: %w", err)
	} else {
		// Update existing record
		cache.LastUpdated = time.Now().UTC()
		cache.Resources = jsonData
		if err := s.db.Save(&cache).Error; err != nil {
			return fmt.Errorf("failed to update GCP resources cache: %w", err)
		}
	}

	return nil
}

func (s *GCPResourcesService) DeleteAll() error {
	return s.db.Where("1 = 1").Delete(&models.GCPResources{}).Error
}

// loads GCP resources from database into the config
func (s *GCPResourcesService) LoadIntoConfig(cfg *config.Config) error {
	resources, err := s.GetResources()
	if err != nil {
		// If resources don't exist, fetch them from GCP API
		resources, err = s.RefreshFromGCP(context.Background())
		if err != nil {
			// Initialize with empty resources on error
			cfg.GCPResources = config.GCPResources{
				LastUpdated:        "",
				Zones:              []string{},
				MachineTypesByZone: make(map[string][]config.GCPMachineType),
			}
			return fmt.Errorf("failed to load GCP resources: %w", err)
		}
	}

	cfg.GCPResources = *resources
	return nil
}

// fetches data from GCP API
func (s *GCPResourcesService) RefreshFromGCP(ctx context.Context) (*config.GCPResources, error) {
	client, err := s.gcpService.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP client: %w", err)
	}

	zones, err := client.ListZonesInRegion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch zones: %w", err)
	}

	machineTypesByZone, err := client.ListAllMachineTypesByZone(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch machine types: %w", err)
	}

	gcpResources := config.GCPResources{
		LastUpdated:        time.Now().UTC().Format(time.RFC3339),
		Zones:              zones,
		MachineTypesByZone: make(map[string][]config.GCPMachineType),
	}

	for zone, machineTypes := range machineTypesByZone {
		configMachineTypes := make([]config.GCPMachineType, len(machineTypes))
		for i, mt := range machineTypes {
			configMachineTypes[i] = config.GCPMachineType{
				Name:        mt.Name,
				Description: mt.Description,
				GuestCpus:   mt.GuestCpus,
				MemoryMb:    mt.MemoryMb,
			}
		}
		gcpResources.MachineTypesByZone[zone] = configMachineTypes
	}

	// Save to database
	if err := s.SaveResources(&gcpResources); err != nil {
		return nil, fmt.Errorf("failed to save resources: %w", err)
	}

	return &gcpResources, nil
}
