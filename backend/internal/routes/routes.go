package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/handlers"
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	"github.com/stolos-cloud/stolos/backend/internal/models"
)

func SetupRoutes(r *gin.Engine, h *handlers.Handlers) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	r.HEAD("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	api := r.Group("/api/v1")
	{
		setupAuthRoutes(api, h)

		// temporary: don't require authentication for nodes routes
		setupNodeRoutes(api, h)
		// require authentication
		protected := api.Group("")
		protected.Use(middleware.JWTAuthMiddleware(h.JWTService(), h.DB()))
		{
			setupClusterRoutes(protected, h)
			setupISORoutes(protected, h)
			setupGCPRoutes(api, protected, h)
			setupNamespaceRoutes(protected, h)
			setupUserRoutes(protected, h)
			setupEventRoutes(protected, h)
			setupTemplateRoutes(protected, h)
		}
	}
}

func setupClusterRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	cluster := api.Group("/cluster")
	{
		cluster.GET("/info", h.GetClusterInfo)
	}
}

func setupISORoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	iso := api.Group("/iso")
	{
		iso.POST("/generate", h.ISOHandlers().GenerateISO)
	}
}

func setupNodeRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	nodes := api.Group("/nodes")
	{
		nodes.GET("", h.NodeHandlers().ListNodes)
		nodes.POST("", h.NodeHandlers().CreateNodes)
		nodes.GET("/:id", h.NodeHandlers().GetNode)
		nodes.DELETE("/:id", h.NodeHandlers().DeleteNode)
		nodes.PUT("/:id/config", h.NodeHandlers().UpdateActiveNodeConfig)
		nodes.PUT("/config", h.NodeHandlers().UpdateActiveNodesConfig)
		nodes.POST("/provision", h.NodeHandlers().ProvisionNodes)
		nodes.POST("/samples", h.NodeHandlers().CreateSampleNodes) // TODO: remove in production
		nodes.GET("/talosconfig", h.NodeHandlers().GetTalosconfig)
		nodes.GET("/:id/disks", h.NodeHandlers().GetNodeDisks)
		//nodes.GET("/kubeconfig", h.NodeHandlers().GetKubeconfig)
	}
}

func setupEventRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	events := api.Group("/events")
	{
		events.GET("/stream", middleware.RequireRole(models.RoleAdmin), h.EventHandlers().StreamEvents)
	}
}

func setupAuthRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	auth := api.Group("/auth")
	{
		auth.POST("/login", h.AuthHandlers().Login)

		// require authentication
		authenticated := auth.Group("")
		authenticated.Use(middleware.JWTAuthMiddleware(h.JWTService(), h.DB()))
		{
			authenticated.POST("/refresh", h.AuthHandlers().RefreshToken)
			authenticated.GET("/profile", h.AuthHandlers().GetProfile)
		}
	}
}

func setupNamespaceRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	namespaces := api.Group("/namespaces")
	{
		namespaces.GET("", h.NamespaceHandlers().GetNamespaces)
		namespaces.POST("", h.NamespaceHandlers().CreateNamespace) // Developers can create namespaces
		namespaces.GET("/:id", h.NamespaceHandlers().GetNamespace)
		namespaces.POST("/:id/users", h.NamespaceHandlers().AddUserToNamespace)                 // Namespace members can add users
		namespaces.DELETE("/:id/users/:user_id", h.NamespaceHandlers().RemoveUserFromNamespace) // Namespace members can remove users
		namespaces.DELETE("/:id", h.NamespaceHandlers().DeleteNamespace)                        // Developers can delete their own namespaces
	}
}

func setupUserRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	users := api.Group("/users")
	{
		// Admin-only
		users.Use(middleware.RequireRole(models.RoleAdmin))
		users.GET("", h.UserHandlers().ListUsers)
		users.GET("/:id", h.UserHandlers().GetUser)
		users.PUT("/:id/role", h.UserHandlers().UpdateUserRole)
		users.DELETE("/:id", h.UserHandlers().DeleteUser)
		users.POST("/create", h.UserHandlers().CreateUser)
	}
}

func setupGCPRoutes(public *gin.RouterGroup, protected *gin.RouterGroup, h *handlers.Handlers) {

	public.GET("/gcp/resources", h.GCPHandlers().GetGCPResources)

	// Provisioning endpoints
	public.GET("/gcp/nodes/provision/:request_id/stream", h.GCPHandlers().ProvisionGCPNodesStream)
	protected.GET("/gcp/nodes/provision/:request_id/plan", h.GCPHandlers().GetProvisionPlan)
	protected.GET("/gcp/nodes/provision/:request_id/apply", h.GCPHandlers().GetProvisionApply)

	// Protected routes
	gcp := protected.Group("/gcp")
	{
		// Checks if SA is configured
		gcp.GET("/status", h.GCPHandlers().GetGCPStatus)
		// Uploads with JSON body
		gcp.PUT("/configure", h.GCPHandlers().ConfigureGCP)
		gcp.POST("/configure/upload", h.GCPHandlers().ConfigureGCPUpload)
		// Uploads with form-data (file)
		gcp.POST("/bucket", h.GCPHandlers().CreateTerraformBucket)

		gcp.POST("/init-infra", h.GCPHandlers().InitInfra)
		gcp.POST("/delete-infra", h.GCPHandlers().DeleteInfra)

		gcp.POST("/terraform/force-unlock", h.GCPHandlers().ForceUnlockTerraformState)

		gcp.POST("/instances", h.GCPHandlers().QueryGCPInstances)

		gcp.POST("/resources/refresh", h.GCPHandlers().RefreshGCPResources)

		gcp.POST("/nodes/provision", h.GCPHandlers().ProvisionGCPNodes)
	}
}

func setupTemplateRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	templateRoutes := api.Group("/templates")
	{
		templateRoutes.GET("", h.TemplatesHandlers().GetTemplatesList)
		templateRoutes.GET("/:name", h.TemplatesHandlers().GetTemplate)
		templateRoutes.POST("/:id/validate/:instance_name", h.TemplatesHandlers().ValidateTemplate)
		templateRoutes.POST("/:id/apply/:instance_name", h.TemplatesHandlers().ApplyTemplate)
	}

	deploymentRoutes := api.Group("/deployments")
	{
		deploymentRoutes.GET("/list", h.TemplatesHandlers().ListDeployments)
		deploymentRoutes.GET("/get", h.TemplatesHandlers().GetDeployment)
		deploymentRoutes.POST("/delete", h.TemplatesHandlers().DeleteDeployment)
	}
}
