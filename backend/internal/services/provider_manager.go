package services

import (
	"context"
	"log"

	"github.com/stolos-cloud/stolos/backend/internal/config"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	gitopsservices "github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	"gorm.io/gorm"
)

type ProviderManager struct {
	db        *gorm.DB
	cfg       *config.Config
	providers map[string]Provider
}

func NewProviderManager(db *gorm.DB, cfg *config.Config) *ProviderManager {
	return &ProviderManager{
		db:        db,
		cfg:       cfg,
		providers: make(map[string]Provider),
	}
}

// discovers and initializes all available cloud providers
func (pm *ProviderManager) InitializeProviders(ctx context.Context) error {

	if err := pm.initializeGitOps(ctx); err != nil {
		log.Printf("Warning: GitOps initialization failed: %v", err)
	}

	if err := pm.initializeGCP(ctx); err != nil {
		return err
	}

	// Future example
	// if err := pm.initializeAWS(ctx); err != nil {
	//     return err
	// }

	return nil
}

func (pm *ProviderManager) initializeGCP(ctx context.Context) error {
	gcpService := gcpservices.NewGCPService(pm.db, pm.cfg)

	gcpConfig, err := gcpService.InitializeGCP(ctx)
	if err != nil {
		return err
	}

	if gcpConfig != nil {
		log.Printf("GCP initialized successfully with project: %s", gcpConfig.ProjectID)
		pm.providers["gcp"] = gcpService

		// Load GCP resources into config (zones, machine types, etc)
		gcpResourcesService := gcpservices.NewGCPResourcesService(pm.db, gcpService)
		if err := gcpResourcesService.LoadIntoConfig(pm.cfg); err != nil {
			log.Printf("Warning: Failed to load GCP resources: %v", err)
		}
	} else {
		log.Println("GCP not configured. Skipping initialization")
	}

	return nil
}

func (pm *ProviderManager) GetProvider(name string) (Provider, bool) {
	provider, ok := pm.providers[name]
	return provider, ok
}

func (pm *ProviderManager) GetConfiguredProviders() map[string]Provider {
	return pm.providers
}

func (pm *ProviderManager) HasConfiguredProviders() bool {
	return len(pm.providers) > 0
}

func (pm *ProviderManager) initializeGitOps(ctx context.Context) error {
	gitopsService := gitopsservices.NewGitOpsService(pm.db, pm.cfg)

	gitopsConfig, err := gitopsService.InitializeGitOps(ctx)
	if err != nil {
		return err
	}

	if gitopsConfig != nil {
		log.Printf("GitOps initialized successfully: %s/%s (branch: %s, workdir: %s)",
			gitopsConfig.RepoOwner, gitopsConfig.RepoName, gitopsConfig.Branch, gitopsConfig.WorkingDir)
	} else {
		log.Println("GitOps not configured. Will use environment variables if available")
	}

	return nil
}
