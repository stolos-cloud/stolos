package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/database"
	"github.com/stolos-cloud/stolos/backend/internal/handlers"
	"github.com/stolos-cloud/stolos/backend/internal/routes"
	"github.com/stolos-cloud/stolos/backend/internal/services"

	_ "github.com/stolos-cloud/stolos/backend/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Stolos API
// @version 1.0
// @description API for Stolos Cloud Platform
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/stolos-cloud/stolos
// @contact.email support@stolos.cloud

// @license.name TBD
// @license.url http://TBD

// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize providers
	ctx := context.Background()
	providerManager := services.NewProviderManager(db, cfg)
	if err := providerManager.InitializeProviders(ctx); err != nil {
		log.Fatal("Failed to initialize providers:", err)
	}

	if !providerManager.HasConfiguredProviders() {
		log.Println("No cloud providers configured")
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	h := handlers.NewHandlers(db, cfg, providerManager)

	routes.SetupRoutes(r, h)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	log.Printf("Starting server on %s", port)
	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
