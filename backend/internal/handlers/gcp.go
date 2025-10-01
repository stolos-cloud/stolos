package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	"gorm.io/gorm"
)

type GCPHandlers struct {
	db               *gorm.DB
	gcpService       *services.GCPService
	nodeService      *services.NodeService
	terraformService *services.TerraformService
}

func NewGCPHandlers(db *gorm.DB, cfg *config.Config) *GCPHandlers {
	return &GCPHandlers{
		db:               db,
		gcpService:       services.NewGCPService(db, cfg),
		nodeService:      services.NewNodeService(db, cfg),
		terraformService: services.NewTerraformService(db, cfg),
	}
}

func (h *GCPHandlers) InitializeGCP(c *gin.Context) {
	gcpConfig, err := h.gcpService.InitializeGCP(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gcpConfig)
}

func (h *GCPHandlers) GetGCPStatus(c *gin.Context) {
	config, err := h.gcpService.GetCurrentConfig()
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusOK, gin.H{
				"configured": false,
				"message":    "GCP not initialized",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configured": true,
		"config":     config,
	})
}

func (h *GCPHandlers) ConfigureGCP(c *gin.Context) {
	var req struct {
		ProjectID          string `json:"project_id" binding:"required"`
		Region             string `json:"region" binding:"required"`
		ServiceAccountJSON string `json:"service_account_json" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gcpConfig, err := h.gcpService.ConfigureGCP(c.Request.Context(), req.ProjectID, req.Region, req.ServiceAccountJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gcpConfig)
}

func (h *GCPHandlers) CreateTerraformBucket(c *gin.Context) {
	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
		Region    string `json:"region" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bucketName, err := h.gcpService.CreateTerraformBucket(c.Request.Context(), req.ProjectID, req.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bucket_name": bucketName,
		"message":     "Terraform bucket created successfully",
	})
}

// Allows to query directly GCP instances
func (h *GCPHandlers) QueryGCPInstances(c *gin.Context) {
	err := h.nodeService.QueryGCPInstances(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sample : successfully queried GCP instances"})
}

func (h *GCPHandlers) InitInfra(c *gin.Context) {
	err := h.terraformService.InitializeInfrastructure(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Terraform infrastructure initialized successfully"})
}

func (h *GCPHandlers) DeleteInfra(c *gin.Context) {
	err := h.terraformService.DestroyInfrastructure(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Terraform infrastructure destroyed successfully"})
}