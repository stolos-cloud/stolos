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
  r.HEAD("/health", func(c *gin.Context) {
    c.Status(http.StatusOK)
  })

  api := r.Group("/api/v1")
  {
    setupISORoutes(api, handlers)
    setupNodeRoutes(api, handlers)
    setupGCPRoutes(api, handlers)
  }
}


func setupISORoutes(api *gin.RouterGroup, handlers *Handlers) {
	isos := api.Group("/isos")
	{
		isos.POST("/generate", handlers.generateISO)
	}
}

func setupNodeRoutes(api *gin.RouterGroup, handlers *Handlers) {
	nodes := api.Group("/nodes")
	{
		nodes.GET("", handlers.listNodes)
		nodes.POST("", handlers.createNodes)
		nodes.GET("/:id", handlers.getNode)
		nodes.POST("/sync-gcp", handlers.syncGCPNodes)
	}
}

func setupGCPRoutes(api *gin.RouterGroup, handlers *Handlers) {
	gcp := api.Group("/gcp")
	{
		gcp.POST("/initialize", handlers.initializeGCP)
		gcp.GET("/status", handlers.getGCPStatus)
		gcp.PUT("/service-account", handlers.updateGCPServiceAccount)
		gcp.POST("/bucket", handlers.createTerraformBucket)
		gcp.POST("/init-infra", handlers.initInfra)
		gcp.POST("/delete-infra", handlers.deleteInfra)
	}
}

