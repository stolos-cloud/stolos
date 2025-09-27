package services

import (
	"context"
	"fmt"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/config"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/models"
	"github.com/etsmtl-pfe-cloudnative/backend/pkg/gcp"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GCPService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewGCPService(db *gorm.DB, cfg *config.Config) *GCPService {
	return &GCPService{
		db:  db,
		cfg: cfg,
	}
}

func (s *GCPService) IsConfiguredFromDatabase() bool {
	config, err := s.GetCurrentConfig()
	return err == nil && config != nil && config.IsConfigured
}

func (s *GCPService) IsConfiguredFromEnv() bool {
	return s.cfg.GCP.ProjectID != "" && s.cfg.GCP.ServiceAccountJSON != ""
}

func (s *GCPService) InitializeGCP(ctx context.Context) (*models.GCPConfig, error) {
	if s.IsConfiguredFromDatabase() {
		return s.GetCurrentConfig()
	}

	if !s.IsConfiguredFromEnv() {
		return nil, fmt.Errorf("GCP not configured. Please set ProjectID and ServiceAccountJSON in config or database")
	}

	bucketName, err := s.CreateTerraformBucket(ctx, s.cfg.GCP.ProjectID, s.cfg.GCP.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform bucket: %w", err)
	}

	gcpConfig, err := s.UpdateServiceAccount(ctx, s.cfg.GCP.ProjectID, s.cfg.GCP.Region, s.cfg.GCP.ServiceAccountJSON, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to save GCP config: %w", err)
	}

	return gcpConfig, nil
}

func (s *GCPService) GetCurrentConfig() (*models.GCPConfig, error) {
	var config models.GCPConfig
	err := s.db.Where("is_configured = ?", true).First(&config).Error
	if err != nil {
		return nil, err
	}

	// clear key before returning
	config.ServiceAccountKeyJSON = ""
	return &config, nil
}

// returns config including service account credentials // TODO handle better
func (s *GCPService) GetCurrentConfigWithCredentials() (*models.GCPConfig, error) {
	var config models.GCPConfig
	err := s.db.Where("is_configured = ?", true).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *GCPService) GetTerraformBackendConfig() (map[string]string, error) {
	config, err := s.GetCurrentConfig()
	if err != nil {
		return nil, fmt.Errorf("GCP not configured: %w", err)
	}

	return map[string]string{
		"backend": "gcs",
		"bucket":  config.BucketName,
		"prefix":  "terraform/state",
	}, nil
}



func (s *GCPService) UpdateServiceAccount(ctx context.Context, projectID, region, serviceAccountJSON string, bucketName ...string) (*models.GCPConfig, error) {
	gcpConfig := &gcp.Config{
		ProjectID:          projectID,
		Region:             region,
		ServiceAccountJSON: serviceAccountJSON,
	}

	_, err := gcp.NewClientFromConfig(gcpConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid service account configuration: %w", err)
	}

	var dbConfig models.GCPConfig
	err = s.db.Where("is_configured = ?", true).First(&dbConfig).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to fetch existing config: %w", err)
	}

	if err == gorm.ErrRecordNotFound {
		dbConfig = models.GCPConfig{
			ID:           uuid.New(),
			IsConfigured: true,
		}
	}

	dbConfig.ProjectID = projectID
	dbConfig.Region = region
	dbConfig.ServiceAccountEmail = gcpConfig.ServiceAccountEmail
	dbConfig.ServiceAccountKeyJSON = serviceAccountJSON

	// Set bucket name if provided
	if len(bucketName) > 0 && bucketName[0] != "" {
		dbConfig.BucketName = bucketName[0]
	}

	if err == gorm.ErrRecordNotFound {
		err = s.db.Create(&dbConfig).Error
	} else {
		err = s.db.Save(&dbConfig).Error
	}

	if err != nil {
		return nil, fmt.Errorf("failed to save service account config: %w", err)
	}

	dbConfig.ServiceAccountKeyJSON = ""
	return &dbConfig, nil
}

func (s *GCPService) CreateTerraformBucket(ctx context.Context, projectID, region string) (string, error) {
	// Try to get from database first, if not available use env config
	var gcpClient *gcp.Client
	var err error

	if s.IsConfiguredFromDatabase() {
		config, err := s.GetCurrentConfigWithCredentials()
		if err != nil {
			return "", fmt.Errorf("failed to get database config: %w", err)
		}
		gcpClient, err = gcp.NewClientFromConfig(&gcp.Config{
			ProjectID:           config.ProjectID,
			Region:              config.Region,
			ServiceAccountJSON:  config.ServiceAccountKeyJSON,
			ServiceAccountEmail: config.ServiceAccountEmail,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create GCP client from database config: %w", err)
		}
	} else if s.IsConfiguredFromEnv() {
		gcpClient, err = gcp.NewClientFromConfig(&gcp.Config{
			ProjectID:          projectID,
			Region:             region,
			ServiceAccountJSON: s.cfg.GCP.ServiceAccountJSON,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create GCP client from env config: %w", err)
		}
	} else {
		return "", fmt.Errorf("GCP not configured in database or environment")
	}

	bucketName, err := gcpClient.CreateTerraformBucket(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create terraform bucket: %w", err)
	}

	// Only update database config if it exists
	if s.IsConfiguredFromDatabase() {
		config, err := s.GetCurrentConfigWithCredentials()
		if err == nil {
			config.BucketName = bucketName
			if err := s.db.Save(config).Error; err != nil {
				return "", fmt.Errorf("failed to update bucket name in config: %w", err)
			}
		}
	}

	return bucketName, nil
}




