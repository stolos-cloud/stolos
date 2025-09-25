package database

import (
	"fmt"
	"log"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/config"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Initialize(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	if cfg.Host != "" {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port, cfg.SSLMode)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

		if err == nil {
			log.Println("Connected to PostgreSQL database")
		} else {
			log.Printf("PostgreSQL connection failed: %v", err)
		}
	}

	// Fallback to SQLite
	if db == nil {
		log.Println("Using SQLite as database (development mode)")
		db, err = gorm.Open(sqlite.Open("./data.db"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
		}
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *gorm.DB) error {
	// UUID extension no needed for SQLite
	if db.Dialector.Name() == "postgres" {
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
			return fmt.Errorf("failed to create uuid extension: %w", err)
		}
	}

	// auto-migrations
	return db.AutoMigrate(
		&models.Cluster{},
		&models.Node{},
		&models.GCPConfig{},
	)
}

func Seed(db *gorm.DB) error {
	// TODO: Add any initial data seeding 
	return nil
}