package main

import (
	"github.com/NVIDIA/gontainer/v2"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/handlers"
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	discoveryservice "github.com/stolos-cloud/stolos/backend/internal/services/cluster"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	"github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	"github.com/stolos-cloud/stolos/backend/internal/services/job"
	"github.com/stolos-cloud/stolos/backend/internal/services/node"
	talosservice "github.com/stolos-cloud/stolos/backend/internal/services/talos"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"gorm.io/gorm"
)

// RegisterInfrastructure registers core infrastructure components
func RegisterInfrastructure(db *gorm.DB, cfg *config.Config) []any {
	return []any{
		gontainer.NewFactory(func() *gorm.DB {
			return db
		}),
		gontainer.NewFactory(func() *config.Config {
			return cfg
		}),
	}
}

// RegisterMiddleware registers middleware services
func RegisterMiddleware() []any {
	return []any{
		gontainer.NewFactory(func(cfg *config.Config) *middleware.JWTService {
			return middleware.NewJWTService(cfg)
		}),
		gontainer.NewFactory(func() *wsservices.Manager {
			wsManager := wsservices.NewManager()
			go wsManager.Run()
			return wsManager
		}),
	}
}

// RegisterCoreServices registers core business services
func RegisterCoreServices() []any {
	return []any{
		gontainer.NewFactory(func(db *gorm.DB, cfg *config.Config, wsManager *wsservices.Manager) *talosservice.TalosService {
			return talosservice.NewTalosService(db, cfg, wsManager)
		}),
		gontainer.NewFactory(func(db *gorm.DB, cfg *config.Config, ts *talosservice.TalosService) *discoveryservice.DiscoveryService {
			return discoveryservice.NewDiscoveryService(db, cfg, ts)
		}),
		gontainer.NewFactory(func(db *gorm.DB, cfg *config.Config, pm *services.ProviderManager, ts *talosservice.TalosService) *node.NodeService {
			return node.NewNodeService(db, cfg, pm, ts)
		}),
		gontainer.NewFactory(func(db *gorm.DB, cfg *config.Config) *gitops.GitOpsService {
			return gitops.NewGitOpsService(db, cfg)
		}),
	}
}

// RegisterGCPServices registers GCP-specific services
func RegisterGCPServices() []any {
	return []any{
		gontainer.NewFactory(func(db *gorm.DB, cfg *config.Config) *gcpservices.GCPService {
			return gcpservices.NewGCPService(db, cfg)
		}),
		gontainer.NewFactory(func(db *gorm.DB, gcpService *gcpservices.GCPService) *gcpservices.GCPResourcesService {
			return gcpservices.NewGCPResourcesService(db, gcpService)
		}),
		gontainer.NewFactory(func(db *gorm.DB, cfg *config.Config, ts *talosservice.TalosService, gcpService *gcpservices.GCPService, gitopsService *gitops.GitOpsService) *gcpservices.ProvisioningService {
			return gcpservices.NewProvisioningService(db, cfg, ts, gcpService, gitopsService)
		}),
	}
}

// RegisterInfrastructureServices registers infrastructure orchestration services
func RegisterInfrastructureServices() []any {
	return []any{
		gontainer.NewFactory(func(db *gorm.DB, cfg *config.Config, pm *services.ProviderManager, gitopsService *gitops.GitOpsService, ts *talosservice.TalosService) *services.InfrastructureService {
			return services.NewInfrastructureService(db, cfg, pm, gitopsService, ts)
		}),
		gontainer.NewFactory(func(
			db *gorm.DB,
			cfg *config.Config,
			gcpService *gcpservices.GCPService,
			gcpResourcesService *gcpservices.GCPResourcesService,
			talosService *talosservice.TalosService,
			gitopsService *gitops.GitOpsService,
			wsManager *wsservices.Manager,
		) *services.ProviderManager {
			return services.NewProviderManager(db, cfg, gcpService, gcpResourcesService, talosService, gitopsService, wsManager)
		}),
		gontainer.NewFactory(func(resolver *gontainer.Resolver) *job.JobService {
			s, _ := job.NewJobService(resolver)
			return s
		}),
	}
}

// RegisterAllServices combines all service registrations into a single slice
func RegisterAllServices(db *gorm.DB, cfg *config.Config) []any {
	var allServices []any

	// Infrastructure (DB, Config)
	allServices = append(allServices, RegisterInfrastructure(db, cfg)...)

	// Middleware (JWT, WebSocket)
	allServices = append(allServices, RegisterMiddleware()...)

	// Core business services (Talos, Node, GitOps, etc.)
	allServices = append(allServices, RegisterCoreServices()...)

	// GCP-specific services
	allServices = append(allServices, RegisterGCPServices()...)

	// Infrastructure orchestration (must come after core services)
	allServices = append(allServices, RegisterInfrastructureServices()...)

	// HTTP handlers
	allServices = append(allServices, handlers.RegisterHandlers()...)

	return allServices
}
