package main

import (
	"log"
	"os"
	"strings"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/config"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/database"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/handlers"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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