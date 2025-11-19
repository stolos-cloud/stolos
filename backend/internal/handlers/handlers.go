package handlers

import (
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"gorm.io/gorm"
)

type Handlers struct {
	authHandlers      *AuthHandlers
	namespaceHandlers *NamespaceHandlers
	userHandlers      *UserHandlers
	isoHandlers       *ISOHandlers
	nodeHandlers      *NodeHandlers
	gcpHandlers       *GCPHandlers
	eventHandlers     *EventHandlers
	templatesHandlers *TemplatesHandler
	scaffoldsHandlers *ScaffoldsHandler
	jwtService        *middleware.JWTService
	db                *gorm.DB
	wsManager         *wsservices.Manager
}

func NewHandlers(
	authHandlers *AuthHandlers,
	namespaceHandlers *NamespaceHandlers,
	userHandlers *UserHandlers,
	isoHandlers *ISOHandlers,
	nodeHandlers *NodeHandlers,
	gcpHandlers *GCPHandlers,
	eventHandlers *EventHandlers,
	templatesHandlers *TemplatesHandler,
	scaffoldsHandlers *ScaffoldsHandler,
	jwtService *middleware.JWTService,
	db *gorm.DB,
	wsManager *wsservices.Manager,
) *Handlers {
	return &Handlers{
		authHandlers:      authHandlers,
		namespaceHandlers: namespaceHandlers,
		userHandlers:      userHandlers,
		isoHandlers:       isoHandlers,
		nodeHandlers:      nodeHandlers,
		gcpHandlers:       gcpHandlers,
		eventHandlers:     eventHandlers,
		templatesHandlers: templatesHandlers,
		scaffoldsHandlers: scaffoldsHandlers,
		jwtService:        jwtService,
		db:                db,
		wsManager:         wsManager,
	}
}

func (h *Handlers) AuthHandlers() *AuthHandlers {
	return h.authHandlers
}

func (h *Handlers) NamespaceHandlers() *NamespaceHandlers {
	return h.namespaceHandlers
}

func (h *Handlers) UserHandlers() *UserHandlers {
	return h.userHandlers
}

func (h *Handlers) ISOHandlers() *ISOHandlers {
	return h.isoHandlers
}

func (h *Handlers) NodeHandlers() *NodeHandlers {
	return h.nodeHandlers
}

func (h *Handlers) GCPHandlers() *GCPHandlers {
	return h.gcpHandlers
}

func (h *Handlers) EventHandlers() *EventHandlers {
	return h.eventHandlers
}

func (h *Handlers) JWTService() *middleware.JWTService {
	return h.jwtService
}

func (h *Handlers) DB() *gorm.DB {
	return h.db
}

func (h *Handlers) TemplatesHandlers() *TemplatesHandler { return h.templatesHandlers }

func (h *Handlers) ScaffoldsHandlers() *ScaffoldsHandler { return h.scaffoldsHandlers }
