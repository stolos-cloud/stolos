package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/config"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/models"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
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

func (s *GCPService) IsConfigured() bool {
	return s.cfg.GCP.ProjectID != "" && s.cfg.GCP.ServiceAccountJSON != ""
}

func (s *GCPService) InitializeGCP(ctx context.Context) (*models.GCPConfig, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("GCP not configured. Please set ProjectID and ServiceAccountJSON in config")
	}

	existing, err := s.GetCurrentConfig()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing config: %w", err)
	}
	if existing != nil && existing.IsConfigured {
		return existing, nil
	}

	client, err := s.createGCPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP client: %w", err)
	}

	// Create storage service
	storageService, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	bucketName := fmt.Sprintf("stolos-tf-state-%s", s.generateRandomString(8))

	// Create storage bucket for Terraform state
	bucket := &storage.Bucket{
		Name:     bucketName,
		Location: s.cfg.GCP.Region,
		Versioning: &storage.BucketVersioning{
			Enabled: true,
		},
		// Enable uniform bucket-level access
		IamConfiguration: &storage.BucketIamConfiguration{
			UniformBucketLevelAccess: &storage.BucketIamConfigurationUniformBucketLevelAccess{
				Enabled: true,
			},
		},
	}

	_, err = storageService.Buckets.Insert(s.cfg.GCP.ProjectID, bucket).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage bucket: %w", err)
	}

	// Save config to db
	gcpConfig := &models.GCPConfig{
		ID:                    uuid.New(),
		ProjectID:             s.cfg.GCP.ProjectID,
		BucketName:            bucketName,
		ServiceAccountEmail:   s.getServiceAccountEmail(),
		ServiceAccountKeyJSON: s.cfg.GCP.ServiceAccountJSON, // talk about security..
		Region:                s.cfg.GCP.Region,
		IsConfigured:          true,
	}

	if err := s.db.Create(gcpConfig).Error; err != nil {
		return nil, fmt.Errorf("failed to save GCP config: %w", err)
	}

	gcpConfig.ServiceAccountKeyJSON = ""
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

func (s *GCPService) createGCPClient(ctx context.Context) (*http.Client, error) {
	creds, err := google.CredentialsFromJSON(ctx, []byte(s.cfg.GCP.ServiceAccountJSON), storage.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	ts := creds.TokenSource
	return oauth2.NewClient(ctx, ts), nil
}

func (s *GCPService) getServiceAccountEmail() string {
	var sa struct {
		ClientEmail string `json:"client_email"`
	}
	json.Unmarshal([]byte(s.cfg.GCP.ServiceAccountJSON), &sa)
	return sa.ClientEmail
}

func (s *GCPService) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}