package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	talosservices "github.com/stolos-cloud/stolos/backend/internal/services/talos"
	"gorm.io/gorm"
)

type ISOHandlers struct {
	db           *gorm.DB
	talosService *talosservices.TalosService
}

func NewISOHandlers(db *gorm.DB, talosService *talosservices.TalosService) *ISOHandlers {
	return &ISOHandlers{
		db:           db,
		talosService: talosService,
	}
}

// GenerateISO godoc
// @Summary Generate a custom Talos ISO
// @Description Generate a custom Talos ISO image for on-prem node provisioning using the image factory
// @Tags iso
// @Accept json
// @Produce json
// @Param request body models.ISORequest true "ISO generation parameters"
// @Success 200 {object} models.ISOResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /iso/generate [post]
// @Security BearerAuth
func (h *ISOHandlers) GenerateISO(c *gin.Context) {
	var req models.ISORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Generate ISO using the Talos service
	response, err := h.talosService.GenerateISO(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate ISO", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
