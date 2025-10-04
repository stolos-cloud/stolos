package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	gcpconfig "github.com/stolos-cloud/stolos-bootstrap/pkg/gcp"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	"github.com/stolos-cloud/stolos/backend/pkg/gcp"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	ctx := context.Background()

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: failed to load .env file: %v", err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := connectToDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Ensure tables exist
	if err := db.AutoMigrate(&models.GCPConfig{}, &models.GCPResources{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Fetching GCP config from database...")
	client, err := createClientFromDB(db)
	if err != nil {
		log.Fatalf("Failed to create GCP client from database: %v", err)
	}

	log.Println("Fetching zones in region...")
	zones, err := client.ListZonesInRegion(ctx)
	if err != nil {
		log.Fatalf("Failed to list zones: %v", err)
	}
	log.Printf("Found %d zones\n", len(zones))

	log.Println("Fetching machine types for all zones...")
	machineTypesByZone, err := client.ListAllMachineTypesByZone(ctx)
	if err != nil {
		log.Fatalf("Failed to list machine types: %v", err)
	}

	gcpResources := config.GCPResources{
		LastUpdated:        time.Now().UTC().Format(time.RFC3339),
		Zones:              zones,
		MachineTypesByZone: make(map[string][]config.GCPMachineType),
	}

	for zone, machineTypes := range machineTypesByZone {
		configMachineTypes := make([]config.GCPMachineType, len(machineTypes))
		for i, mt := range machineTypes {
			configMachineTypes[i] = config.GCPMachineType{
				Name:        mt.Name,
				Description: mt.Description,
				GuestCpus:   mt.GuestCpus,
				MemoryMb:    mt.MemoryMb,
			}
		}
		gcpResources.MachineTypesByZone[zone] = configMachineTypes
		log.Printf("Zone %s: %d machine types\n", zone, len(configMachineTypes))
	}

	gcpService := gcpservices.NewGCPService(db, cfg)
	resourcesService := gcpservices.NewGCPResourcesService(db, gcpService)
	if err := resourcesService.SaveResources(&gcpResources); err != nil {
		log.Fatalf("Failed to save resources to database: %v", err)
	}

	log.Println("Successfully saved GCP resources to database")
	fmt.Printf("Last updated: %s\n", gcpResources.LastUpdated)
	fmt.Printf("Total zones: %d\n", len(gcpResources.Zones))
	fmt.Printf("Total machine types across all zones: %d\n", countTotalMachineTypes(gcpResources.MachineTypesByZone))
}

func countTotalMachineTypes(m map[string][]config.GCPMachineType) int {
	total := 0
	for _, types := range m {
		total += len(types)
	}
	return total
}

func connectToDatabase(cfg *config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// Use PostgreSQL if configured
	if cfg.Database.Host != "" {
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Database,
			cfg.Database.SSLMode,
		)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
		log.Println("Connected to PostgreSQL database")
		return db, nil
	}

	// Use SQLite only if PostgreSQL is not configured
	log.Println("Using SQLite database (./data.db)")
	db, err = gorm.Open(sqlite.Open("./data.db"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	return db, nil
}

func createClientFromDB(db *gorm.DB) (*gcp.Client, error) {
	var gcpConfig models.GCPConfig
	err := db.Where("is_configured = ?", true).First(&gcpConfig).Error
	if err != nil {
		return nil, fmt.Errorf("GCP not configured in database: %w", err)
	}

	gcpCfg, err := gcpconfig.NewConfig(
		gcpConfig.ProjectID,
		gcpConfig.Region,
		gcpConfig.ServiceAccountKeyJSON,
		gcpConfig.ServiceAccountEmail,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP config: %w", err)
	}

	return gcp.NewClientFromConfig(gcpCfg)
}
