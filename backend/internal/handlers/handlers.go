package handlers

import (
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	"gorm.io/gorm"
)

type Handlers struct {
	authHandlers *AuthHandlers
	teamHandlers *TeamHandlers
	userHandlers *UserHandlers
	isoHandlers  *ISOHandlers
	nodeHandlers *NodeHandlers
	gcpHandlers  *GCPHandlers
	jwtService   *middleware.JWTService
	db           *gorm.DB
}

func NewHandlers(db *gorm.DB, cfg *config.Config, providerManager *services.ProviderManager) *Handlers {
	jwtService := middleware.NewJWTService(cfg)

	return &Handlers{
		authHandlers: NewAuthHandlers(db, jwtService),
		teamHandlers: NewTeamHandlers(db),
		userHandlers: NewUserHandlers(db),
		isoHandlers:  NewISOHandlers(db, cfg),
		nodeHandlers: NewNodeHandlers(db, cfg, providerManager),
		gcpHandlers:  NewGCPHandlers(db, cfg, providerManager),
		jwtService:   jwtService,
		db:           db,
	}
}

func (h *Handlers) AuthHandlers() *AuthHandlers {
	return h.authHandlers
}

func (h *Handlers) TeamHandlers() *TeamHandlers {
	return h.teamHandlers
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

func (h *Handlers) JWTService() *middleware.JWTService {
	return h.jwtService
}

func (h *Handlers) DB() *gorm.DB {
	return h.db
}
