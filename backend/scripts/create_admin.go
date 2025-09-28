package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/config"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/database"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/models"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	var (
		email    = flag.String("email", "", "Admin email address")
		password = flag.String("password", "", "Admin password")
		force    = flag.Bool("force", false, "Force create admin even if one exists")
	)
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	if *email == "" {
		log.Fatal("Email is required. Use -email flag or set ADMIN_EMAIL environment variable")
	}

	if *password == "" {
		*password = os.Getenv("ADMIN_PASSWORD")
		if *password == "" {
			log.Fatal("Password is required. Use -password flag or set ADMIN_PASSWORD environment variable")
		}
	}

	if envEmail := os.Getenv("ADMIN_EMAIL"); envEmail != "" {
		*email = envEmail
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := createAdminUser(db, *email, *password, *force); err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Printf("Admin user created successfully: %s", *email)
}

func createAdminUser(db *gorm.DB, email, password string, force bool) error {
	email = strings.ToLower(email)

	var existingUser models.User
	err := db.Where("email = ?", email).First(&existingUser).Error
	if err == nil {
		if !force {
			return fmt.Errorf("user with email %s already exists. Use -force to recreate", email)
		}

		if err := db.Delete(&existingUser).Error; err != nil {
			return fmt.Errorf("failed to delete existing user: %w", err)
		}
		log.Printf("Deleted existing user: %s", email)
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", err)
	}

	admin := models.User{
		Email: email,
		Role:  models.RoleAdmin,
	}

	if err := admin.SetPassword(password); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	var adminTeam models.Team
	err = db.Where("name = ?", "administrators").First(&adminTeam).Error
	switch err {
		case gorm.ErrRecordNotFound:
			adminTeam = models.Team{Name: "administrators"}
			if err := db.Create(&adminTeam).Error; err != nil {
				log.Printf("Failed to create administrators team: %v", err)
			} else {
				// Add admin to team
				if err := db.Model(&admin).Association("Teams").Append(&adminTeam); err != nil {
					log.Printf("Failed to add admin to administrators team: %v", err)
				} else {
					log.Printf("Added admin to administrators team")
				}
			}
	case nil:
		// Team exists
		if err := db.Model(&admin).Association("Teams").Append(&adminTeam); err != nil {
			log.Printf("Failed to add admin to existing administrators team: %v", err)
		} else {
			log.Printf("Added admin to existing administrators team")
		}
	}

	return nil
}