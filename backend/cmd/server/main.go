package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strings"
	"time"

	"github.com/NVIDIA/gontainer/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron/v2"
	"github.com/joho/godotenv"
	_ "github.com/stolos-cloud/stolos/backend/docs"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/database"
	"github.com/stolos-cloud/stolos/backend/internal/handlers"
	"github.com/stolos-cloud/stolos/backend/internal/routes"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	discoveryservice "github.com/stolos-cloud/stolos/backend/internal/services/cluster"
	"github.com/stolos-cloud/stolos/backend/internal/services/node"
	talosservice "github.com/stolos-cloud/stolos/backend/internal/services/talos"
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

	// -- Load talos values

	// ---

	// Generate random if not provided
	if cfg.JWT.SecretKey == "" {
		log.Println("JWT_SECRET_KEY not set, generating random secret")
		cfg.JWT.SecretKey = generateRandomSecret(32)
	}

	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize context
	ctx := context.Background()

	// Register all services using modular wire.go files
	allServices := RegisterAllServices(db, cfg)

	// Add entrypoint
	allServices = append(allServices, gontainer.NewEntrypoint(
			func(talosService *talosservice.TalosService,
				providerManager *services.ProviderManager,
				nodeService *node.NodeService,
				clusterDiscovery *discoveryservice.DiscoveryService,
				h *handlers.Handlers,
				infrastructureService *services.InfrastructureService,
				resolver *gontainer.Resolver) {
				providerManager.SetInfrastructureService(infrastructureService)

				// Start EventSink after cluster init.
				talosService.StartEventSink()

				// Initialize providers
				if err := providerManager.InitializeProviders(ctx); err != nil {
					log.Fatal("Failed to initialize providers:", err)
				}

				if !providerManager.HasConfiguredProviders() {
					log.Println("No cloud providers configured")
				}

				// discover the cluster the backend is running on
				if err := clusterDiscovery.InitializeCluster(ctx); err != nil {
					log.Fatal("Failed to initialize cluster:", err)
				}

				// Migrate Talos configs from files to database (one-time migration)
				if err := talosService.MigrateTalosConfigFromFiles(); err != nil {
					log.Printf("Note: Talos config migration skipped: %v", err)
				}

				r := gin.Default()

				r.Use(cors.New(cors.Config{
					AllowOrigins:     []string{"*"},
					AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
					AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
					AllowCredentials: true,
				}))

				routes.SetupRoutes(r, h)

				r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

				port := os.Getenv("PORT")
				if port == "" {
					port = ":8080"
				}
				if !strings.HasPrefix(port, ":") {
					port = ":" + port
				}

				s, err := gocron.NewScheduler()
				if err != nil {
					log.Print("Failed to start scheduler:", err)
				}

				// add a job to the scheduler
				j, err := s.NewJob(
					gocron.DurationJob(
						30*time.Second,
					),
					gocron.NewTask(
						func(ctx context.Context) {
							//cli, _ := talosService.GetMachineryClient("192.168.2.71")
							//talosservice.GetTypedTalosResource()
							log.Println("Starting job")
						},
					),
				)
				if err != nil {
					log.Print("Failed to start job:", err)
				}
				log.Printf("Started healtch check job with id %s\n", j.ID())
				s.Start()

				log.Printf("Starting server on %s", port)
				if err := r.Run(port); err != nil {
					log.Fatal("Failed to start server:", err)
				}
			}))

	options := make([]gontainer.Option, len(allServices))
	for i, svc := range allServices {
		options[i] = svc.(gontainer.Option)
	}

	err = gontainer.Run(ctx, options...)
	if err != nil {
		log.Fatal("Failed to run application:", err)
	}
}

func generateRandomSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate random secret:", err)
	}
	return hex.EncodeToString(bytes)
}
