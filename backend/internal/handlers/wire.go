package handlers

import (
	"github.com/NVIDIA/gontainer/v2"
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	"github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	"github.com/stolos-cloud/stolos/backend/internal/services/node"
	talosservice "github.com/stolos-cloud/stolos/backend/internal/services/talos"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"gorm.io/gorm"
)

// RegisterHandlers registers all HTTP handlers with the DI container
func RegisterHandlers() []any {
	return []any{
		// Individual handlers
		gontainer.NewFactory(func(db *gorm.DB, jwt *middleware.JWTService) *AuthHandlers {
			return NewAuthHandlers(db, jwt)
		}),
		gontainer.NewFactory(func(db *gorm.DB) *TeamHandlers {
			return NewTeamHandlers(db)
		}),
		gontainer.NewFactory(func(db *gorm.DB) *UserHandlers {
			return NewUserHandlers(db)
		}),
		gontainer.NewFactory(func(db *gorm.DB, ts *talosservice.TalosService) *ISOHandlers {
			return NewISOHandlers(db, ts)
		}),
		gontainer.NewFactory(func(db *gorm.DB, ns *node.NodeService, ts *talosservice.TalosService, wsManager *wsservices.Manager) *NodeHandlers {
			return NewNodeHandlers(db, ns, ts, wsManager)
		}),
		gontainer.NewFactory(func(wsManager *wsservices.Manager) *EventHandlers {
			return NewEventHandlers(wsManager)
		}),
		gontainer.NewFactory(func(
			db *gorm.DB,
			gcpService *gcpservices.GCPService,
			gitopsService *gitops.GitOpsService,
			nodeService *node.NodeService,
			infrastructureService *services.InfrastructureService,
			gcpResourcesService *gcpservices.GCPResourcesService,
			provisioningService *gcpservices.ProvisioningService,
			wsManager *wsservices.Manager,
		) *GCPHandlers {
			return NewGCPHandlers(
				db,
				gcpService,
				gitopsService,
				nodeService,
				infrastructureService,
				gcpResourcesService,
				provisioningService,
				wsManager,
			)
		}),

		// Handler aggregator
		gontainer.NewFactory(func(
			authHandlers *AuthHandlers,
			teamHandlers *TeamHandlers,
			userHandlers *UserHandlers,
			isoHandlers *ISOHandlers,
			nodeHandlers *NodeHandlers,
			gcpHandlers *GCPHandlers,
			eventHandlers *EventHandlers,
			jwtService *middleware.JWTService,
			db *gorm.DB,
			wsManager *wsservices.Manager,
		) *Handlers {
			return NewHandlers(
				authHandlers,
				teamHandlers,
				userHandlers,
				isoHandlers,
				nodeHandlers,
				gcpHandlers,
				eventHandlers,
				jwtService,
				db,
				wsManager,
			)
		}),
	}
}
