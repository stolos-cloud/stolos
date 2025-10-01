package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	"gorm.io/gorm"
)

type ISOHandlers struct {
	db         *gorm.DB
	isoService *services.ISOService
}

func NewISOHandlers(db *gorm.DB, cfg *config.Config) *ISOHandlers {
	return &ISOHandlers{
		db:         db,
		isoService: services.NewISOService(db, cfg),
	}
}

func (h *ISOHandlers) GenerateISO(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Generate ISO - TODO"})
}