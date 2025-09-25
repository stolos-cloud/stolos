package handlers

import (
	"net/http"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/config"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handlers struct {
	isoService       *services.ISOService
	nodeService      *services.NodeService
	terraformService *services.TerraformService
	gcpService       *services.GCPService
}

func NewHandlers(db *gorm.DB, cfg *config.Config) *Handlers {
	return &Handlers{
		isoService:       services.NewISOService(db, cfg),
		nodeService:      services.NewNodeService(db, cfg),
		terraformService: services.NewTerraformService(db, cfg),
		gcpService:       services.NewGCPService(db, cfg),
	}
}

func SetupRoutes(r *gin.Engine, handlers *Handlers) {

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/isos/generate", handlers.generateISO)

		api.GET("/nodes", handlers.listNodes)
		api.GET("/pending-nodes", handlers.listPendingNodes)
		api.POST("/nodes", handlers.createNodes)
		api.GET("/nodes/:id", handlers.getNode)
		api.POST("/nodes/sync-gcp", handlers.syncGCPNodes)


		api.POST("/gcp/initialize", handlers.initializeGCP)
		api.GET("/gcp/status", handlers.getGCPStatus)
	}
}

func (h *Handlers) generateISO(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Generate ISO - TODO"})
}

func (h *Handlers) listNodes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List nodes - TODO"})
}

func (h *Handlers) listPendingNodes(c *gin.Context) {
	// Sample that creates some pending nodes and returns them
	if err := h.nodeService.CreateSamplePendingNodes(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	nodes, err := h.nodeService.ListPendingNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

func (h *Handlers) createNodes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create node - TODO"})
}

func (h *Handlers) getNode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get node - TODO"})
}

func (h *Handlers) syncGCPNodes(c *gin.Context) {
	// Sample: Query GCP instances
	err := h.nodeService.QueryGCPInstances(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sample : successfully queried GCP instances"})
}

func (h *Handlers) initializeGCP(c *gin.Context) {
	gcpConfig, err := h.gcpService.InitializeGCP(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gcpConfig)
}

func (h *Handlers) getGCPStatus(c *gin.Context) {
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