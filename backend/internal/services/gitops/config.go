package gitops

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	githubpkg "github.com/stolos-cloud/stolos/backend/pkg/github"
	"gorm.io/gorm"
)

type GitOpsService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewGitOpsService(db *gorm.DB, cfg *config.Config) *GitOpsService {
	return &GitOpsService{
		db:  db,
		cfg: cfg,
	}
}

func (s *GitOpsService) IsConfiguredFromDatabase() bool {
	config, err := s.GetCurrentConfig()
	return err == nil && config != nil && config.IsConfigured
}

func (s *GitOpsService) IsConfiguredFromEnv() bool {
	return s.cfg.GitOps.RepoOwner != "" && s.cfg.GitOps.RepoName != ""
}

// initializes GitOps configuration on server startup
func (s *GitOpsService) InitializeGitOps(ctx context.Context) (*models.GitOpsConfig, error) {
	// Return existing config if already set up
	if s.IsConfiguredFromDatabase() {
		return s.GetCurrentConfig()
	}

	// If no DB config and no env config, skip initialization
	if !s.IsConfiguredFromEnv() {
		return nil, nil
	}

	// Initialize from env config
	return s.ConfigureGitOps(ctx, s.cfg.GitOps.RepoOwner, s.cfg.GitOps.RepoName, s.cfg.GitOps.Branch, s.cfg.GitOps.WorkingDir, s.cfg.GitOps.Username, s.cfg.GitOps.Email)
}

func (s *GitOpsService) ConfigureGitOps(ctx context.Context, repoOwner, repoName, branch, workingDir, username, email string) (*models.GitOpsConfig, error) {
	var dbConfig models.GitOpsConfig
	err := s.db.Where("is_configured = ?", true).First(&dbConfig).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to fetch existing config: %w", err)
	}

	if err == gorm.ErrRecordNotFound {
		dbConfig = models.GitOpsConfig{
			ID:           uuid.New(),
			IsConfigured: true,
		}
	}

	dbConfig.RepoOwner = repoOwner
	dbConfig.RepoName = repoName

	// Set branch with default
	if branch != "" {
		dbConfig.Branch = branch
	} else {
		dbConfig.Branch = "main"
	}

	// Set working dir with default
	if workingDir != "" {
		dbConfig.WorkingDir = workingDir
	} else {
		dbConfig.WorkingDir = "terraform"
	}

	// Set username with default
	if username != "" {
		dbConfig.Username = username
	} else {
		dbConfig.Username = "Stolos Bot"
	}

	// Set email with default
	if email != "" {
		dbConfig.Email = email
	} else {
		dbConfig.Email = "bot@stolos.cloud"
	}

	if err == gorm.ErrRecordNotFound {
		err = s.db.Create(&dbConfig).Error
	} else {
		err = s.db.Save(&dbConfig).Error
	}

	if err != nil {
		return nil, fmt.Errorf("failed to save GitOps config: %w", err)
	}

	return &dbConfig, nil
}

func (s *GitOpsService) GetCurrentConfig() (*models.GitOpsConfig, error) {
	var config models.GitOpsConfig
	err := s.db.Where("is_configured = ?", true).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetConfigOrDefault returns the config from DB, or falls back to env/defaults
func (s *GitOpsService) GetConfigOrDefault() (*models.GitOpsConfig, error) {
	// Try DB first
	if s.IsConfiguredFromDatabase() {
		return s.GetCurrentConfig()
	}

	// Fall back to env/defaults
	if s.IsConfiguredFromEnv() {
		branch := s.cfg.GitOps.Branch
		if branch == "" {
			branch = "main"
		}

		workingDir := s.cfg.GitOps.WorkingDir
		if workingDir == "" {
			workingDir = "terraform"
		}

		username := s.cfg.GitOps.Username
		if username == "" {
			username = "Stolos Bot"
		}

		email := s.cfg.GitOps.Email
		if email == "" {
			email = "bot@stolos.cloud"
		}

		return &models.GitOpsConfig{
			RepoOwner:  s.cfg.GitOps.RepoOwner,
			RepoName:   s.cfg.GitOps.RepoName,
			Branch:     branch,
			WorkingDir: workingDir,
			Username:   username,
			Email:      email,
		}, nil
	}

	return nil, fmt.Errorf("GitOps not configured in database or environment")
}

// GetGitHubClient creates a GitHub client using app config + GitOps config
func (s *GitOpsService) GetGitHubClient() (*githubpkg.Client, error) {
	gitopsConfig, err := s.GetConfigOrDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitOps config: %w", err)
	}

	return githubpkg.NewClientFromConfig(
		s.cfg.GitHub.AppID,
		s.cfg.GitHub.InstallationID,
		s.cfg.GitHub.PrivateKey,
		gitopsConfig.RepoOwner,
		gitopsConfig.RepoName,
		gitopsConfig.Branch,
	)
}
