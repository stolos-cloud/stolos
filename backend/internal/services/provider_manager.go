package services

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	gitopsservices "github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	talosservices "github.com/stolos-cloud/stolos/backend/internal/services/talos"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	tfpkg "github.com/stolos-cloud/stolos/backend/pkg/terraform"
	"gorm.io/gorm"
)

type ProviderManager struct {
	db                    *gorm.DB
	cfg                   *config.Config
	providers             map[string]Provider
	gcpService            *gcpservices.GCPService
	gcpResourcesService   *gcpservices.GCPResourcesService
	talosService          *talosservices.TalosService
	gitopsService         *gitopsservices.GitOpsService
	infrastructureService *InfrastructureService
	wsManager             *wsservices.Manager
}

func NewProviderManager(
	db *gorm.DB,
	cfg *config.Config,
	gcpService *gcpservices.GCPService,
	gcpResourcesService *gcpservices.GCPResourcesService,
	talosService *talosservices.TalosService,
	gitopsService *gitopsservices.GitOpsService,
	wsManager *wsservices.Manager,
) *ProviderManager {
	return &ProviderManager{
		db:                  db,
		cfg:                 cfg,
		providers:           make(map[string]Provider),
		gcpService:          gcpService,
		gcpResourcesService: gcpResourcesService,
		talosService:        talosService,
		gitopsService:       gitopsService,
		wsManager:           wsManager,
	}
}

func (pm *ProviderManager) SetInfrastructureService(infrastructureService *InfrastructureService) {
	pm.infrastructureService = infrastructureService
}

// discovers and initializes all available cloud providers
func (pm *ProviderManager) InitializeProviders(ctx context.Context) error {

	if err := tfpkg.CheckTerraformInstalled(); err != nil {
		log.Printf("Terraform not installed - cloud provider features will be unavailable: %v", err)

		if err := pm.initializeGitOps(ctx); err != nil {
			log.Printf("Warning: GitOps initialization failed: %v", err)
		}
		return nil
	}

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
	gcpConfig, err := pm.gcpService.InitializeGCP(ctx)
	if err != nil {
		return err
	}

	if gcpConfig != nil {
		log.Printf("GCP initialized successfully with project: %s", gcpConfig.ProjectID)
		pm.providers["gcp"] = pm.gcpService

		// Load GCP resources into config (zones, machine types, etc)
		if err := pm.gcpResourcesService.LoadIntoConfig(pm.cfg); err != nil {
			log.Printf("Warning: Failed to load GCP resources: %v", err)
		}

		// Only initialize infrastructure if not already ready
		if gcpConfig.InfrastructureStatus != "ready" {
			log.Printf("Infrastructure status: %s - starting initialization", gcpConfig.InfrastructureStatus)
			go pm.initializeGCPInfrastructure(ctx, gcpConfig.ID)
		} else {
			log.Println("Infrastructure already ready - skipping initialization")
		}
	} else {
		log.Println("GCP not configured. Skipping initialization")
	}

	return nil
}

// initializeGCPInfrastructure initializes GCP infrastructure (VPC, subnet) in the background
func (pm *ProviderManager) initializeGCPInfrastructure(ctx context.Context, configID uuid.UUID) {
	// Update status to initializing
	pm.db.Model(&models.GCPConfig{}).
		Where("id = ?", configID).
		Update("infrastructure_status", "initializing")

	if pm.wsManager != nil {
		pm.wsManager.BroadcastInfrastructureStatus("initializing", "gcp")
	}

	log.Println("Starting GCP infrastructure initialization...")

	// Get GCP config with credentials
	var gcpConfig models.GCPConfig
	if err := pm.db.Where("id = ?", configID).First(&gcpConfig).Error; err != nil {
		log.Printf("Failed to get GCP config: %v", err)
		pm.db.Model(&models.GCPConfig{}).
			Where("id = ?", configID).
			Update("infrastructure_status", "failed")
		if pm.wsManager != nil {
			pm.wsManager.BroadcastInfrastructureStatus("failed", "gcp")
		}
		return
	}

	// Ensure Talos images are uploaded and registered
	log.Println("Checking Talos GCP images...")
	if err := pm.talosService.EnsureTalosGCPImages(ctx, &gcpConfig); err != nil {
		log.Printf("Failed to initialize Talos images: %v", err)
		pm.db.Model(&models.GCPConfig{}).
			Where("id = ?", configID).
			Update("infrastructure_status", "failed")
		if pm.wsManager != nil {
			pm.wsManager.BroadcastInfrastructureStatus("failed", "gcp")
		}
		return
	}

	log.Println("Initializing GCP infrastructure (VPC, subnet)...")
	if err := pm.infrastructureService.InitializeInfrastructure(ctx, "gcp"); err != nil {
		log.Printf("Failed to initialize GCP infrastructure: %v", err)
		pm.db.Model(&models.GCPConfig{}).
			Where("id = ?", configID).
			Update("infrastructure_status", "failed")
		if pm.wsManager != nil {
			pm.wsManager.BroadcastInfrastructureStatus("failed", "gcp")
		}
		return
	}

	// Update status to ready
	pm.db.Model(&models.GCPConfig{}).
		Where("id = ?", configID).
		Update("infrastructure_status", "ready")

	if pm.wsManager != nil {
		pm.wsManager.BroadcastInfrastructureStatus("ready", "gcp")
	}

	log.Println("GCP infrastructure initialized successfully")
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
	gitopsConfig, err := pm.gitopsService.InitializeGitOps(ctx)
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
