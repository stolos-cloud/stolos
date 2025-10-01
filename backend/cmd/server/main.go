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
)

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

	// Initialize GCP if environment variables are set
	gcpService := services.NewGCPService(db, cfg)
	ctx := context.Background()
	gcpConfig, err := gcpService.InitializeGCP(ctx)
	if err != nil {
		log.Fatal("Failed to initialize GCP:", err)
	}
	if gcpConfig != nil {
		log.Printf("GCP initialized successfully with project: %s", gcpConfig.ProjectID)
	} else {
		log.Println("GCP not configured. Skipping initialization")
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	h, err := handlers.NewHandlers(db, cfg)
	if err != nil {
		log.Fatal("Failed to initialize handlers:", err)
	}

	routes.SetupRoutes(r, h)

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