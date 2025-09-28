package routes

import (
	"net/http"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/handlers"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/middleware"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/models"
	"github.com/gin-gonic/gin"
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

		// require authentication
		protected := api.Group("")
		protected.Use(middleware.JWTAuthMiddleware(h.JWTService(), h.DB()))
		{
			setupISORoutes(protected, h)
			setupNodeRoutes(protected, h)
			setupGCPRoutes(protected, h)
			setupTeamRoutes(protected, h)
			setupUserRoutes(protected, h)
		}
	}
}

func setupISORoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	isos := api.Group("/isos")
	{
		isos.POST("/generate", h.ISOHandlers().GenerateISO)
	}
}

func setupNodeRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	nodes := api.Group("/nodes")
	{
		nodes.GET("", h.NodeHandlers().ListNodes)
		nodes.POST("", h.NodeHandlers().CreateNodes)
		nodes.GET("/:id", h.NodeHandlers().GetNode)
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

		// Admin-only routes
		admin := auth.Group("/admin")
		admin.Use(middleware.JWTAuthMiddleware(h.JWTService(), h.DB()))
		admin.Use(middleware.RequireRole(models.RoleAdmin))
		{
			admin.POST("/users", h.AuthHandlers().CreateUser)
			admin.POST("/register", h.AuthHandlers().Register)
		}
	}
}

func setupTeamRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	teams := api.Group("/teams")
	{
		teams.GET("", h.TeamHandlers().GetTeams)
		teams.POST("", middleware.RequireRole(models.RoleAdmin), h.TeamHandlers().CreateTeam)
		teams.GET("/:id", h.TeamHandlers().GetTeam)
		teams.POST("/:id/users", middleware.RequireRole(models.RoleAdmin), h.TeamHandlers().AddUserToTeam)
		teams.DELETE("/:id/users/:user_id", middleware.RequireRole(models.RoleAdmin), h.TeamHandlers().RemoveUserFromTeam)
		teams.DELETE("/:id", middleware.RequireRole(models.RoleAdmin), h.TeamHandlers().DeleteTeam)
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
	}
}

func setupGCPRoutes(api *gin.RouterGroup, h *handlers.Handlers) {
	gcp := api.Group("/gcp")
	{
		gcp.POST("/initialize", h.GCPHandlers().InitializeGCP)
		gcp.GET("/status", h.GCPHandlers().GetGCPStatus)
		gcp.PUT("/service-account", h.GCPHandlers().UpdateGCPServiceAccount)
		gcp.POST("/bucket", h.GCPHandlers().CreateTerraformBucket)
		gcp.POST("/init-infra", h.GCPHandlers().InitInfra)
		gcp.POST("/delete-infra", h.GCPHandlers().DeleteInfra)
		gcp.POST("/instances", h.GCPHandlers().QueryGCPInstances)
	}
}