package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	gitopsservices "github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	talosservices "github.com/stolos-cloud/stolos/backend/internal/services/talos"
	githubpkg "github.com/stolos-cloud/stolos/backend/pkg/github"
	tfpkg "github.com/stolos-cloud/stolos/backend/pkg/terraform"
	"gorm.io/gorm"
)

type InfrastructureService struct {
	db              *gorm.DB
	cfg             *config.Config
	providerManager *ProviderManager
	gitopsService   *gitopsservices.GitOpsService
}

type NodeConfig struct {
	Name         string
	Zone         string
	MachineType  string
	Architecture string
}

func NewInfrastructureService(db *gorm.DB, cfg *config.Config, providerManager *ProviderManager, gitopsService *gitopsservices.GitOpsService) *InfrastructureService {
	return &InfrastructureService{
		db:              db,
		cfg:             cfg,
		providerManager: providerManager,
		gitopsService:   gitopsService,
	}
}

// sanitizeGCPResourceName converts a cluster name to a valid GCP resource name
// GCP requirements: lowercase, letters/numbers/hyphens only, start with letter, no trailing hyphen
func sanitizeGCPResourceName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	name = reg.ReplaceAllString(name, "-")

	// Remove leading non-letter characters
	name = regexp.MustCompile(`^[^a-z]+`).ReplaceAllString(name, "")

	// Remove trailing hyphens
	name = strings.TrimRight(name, "-")

	// If empty after sanitization, use default
	if name == "" {
		name = "cluster"
	}

	return name
}

// sets up the base infrastructure (VPC, subnets, etc.) needed for VM provisioning
func (s *InfrastructureService) InitializeInfrastructure(ctx context.Context, providerName string) error {
	// Get cluster name from database
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Create a temporary directory for terraform files
	workDir, err := os.MkdirTemp("", "terraform-infra-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Get provider and config
	provider, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		return fmt.Errorf("provider %s not configured", providerName)
	}

	gcpService, ok := provider.(*gcpservices.GCPService)
	if !ok {
		return fmt.Errorf("unsupported provider type")
	}

	gcpConfig, err := gcpService.GetCurrentConfigWithCredentials()
	if err != nil {
		return fmt.Errorf("failed to get GCP config: %w", err)
	}

	// Get backend configuration
	backendConfig, err := provider.GetTerraformBackendConfig()
	if err != nil {
		return fmt.Errorf("failed to get backend config: %w", err)
	}

	// Create orchestrator
	envVars := map[string]string{
		"GOOGLE_CREDENTIALS": gcpConfig.ServiceAccountKeyJSON,
		"GOOGLE_PROJECT":     gcpConfig.ProjectID,
	}

	orchestrator, err := tfpkg.NewOrchestrator(tfpkg.OrchestratorConfig{
		WorkDir:         workDir,
		TemplateBaseDir: "terraform-templates",
		EnvVars:         envVars,
	})
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Render infrastructure template
	templateData := InfrastructureTemplateData{
		ClusterName: sanitizeGCPResourceName(cluster.Name),
		BucketName:  backendConfig["bucket"],
		ProjectID:   gcpConfig.ProjectID,
		Region:      gcpConfig.Region,
	}

	if err := orchestrator.RenderTemplateToFile("gcp/infrastructure.tf.tmpl", "main.tf", templateData); err != nil {
		return fmt.Errorf("failed to render infrastructure template: %w", err)
	}

	// Initialize terraform
	if err := orchestrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize terraform: %w", err)
	}

	// Plan and apply
	hasChanges, err := orchestrator.Plan(ctx)
	if err != nil {
		return fmt.Errorf("terraform plan failed: %w", err)
	}

	if hasChanges {
		if err := orchestrator.Apply(ctx); err != nil {
			return fmt.Errorf("terraform apply failed: %w", err)
		}
		fmt.Println("Infrastructure provisioned successfully")

		// Commit terraform files to repository
		gitopsConfig, err := s.gitopsService.GetConfigOrDefault()
		if err != nil {
			return fmt.Errorf("failed to get GitOps config: %w", err)
		}

		ghClient, err := githubpkg.NewClientFromConfig(
			s.cfg.GitHub.AppID,
			s.cfg.GitHub.InstallationID,
			s.cfg.GitHub.PrivateKey,
			gitopsConfig.RepoOwner,
			gitopsConfig.RepoName,
			gitopsConfig.Branch,
		)
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}

		if err := orchestrator.CommitToGitOps(ctx, ghClient.Client, tfpkg.GitOpsConfig{
			Owner:    gitopsConfig.RepoOwner,
			Repo:     gitopsConfig.RepoName,
			Branch:   gitopsConfig.Branch,
			BasePath: filepath.Join(gitopsConfig.WorkingDir, providerName),
			Username: gitopsConfig.Username,
			Email:    gitopsConfig.Email,
		}, "Update infrastructure terraform configuration"); err != nil {
			return fmt.Errorf("failed to commit to repository: %w", err)
		}

		// Publish node module to gitops repo (only on first-time setup)
		if err := s.PublishNodeModuleToRepo(providerName, ""); err != nil {
			return fmt.Errorf("failed to publish node module: %w", err)
		}
	} else {
		fmt.Println("No infrastructure changes required - skipping GitOps updates")
	}

	return nil
}

type InfrastructureTemplateData struct {
	ClusterName string
	BucketName  string
	ProjectID   string
	Region      string
}

func (s *InfrastructureService) DestroyInfrastructure(ctx context.Context, providerName string) error {
	// Get cluster name from database
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// temporary directory for terraform operations
	workDir, err := os.MkdirTemp("", "terraform-destroy-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Get provider and config
	provider, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		return fmt.Errorf("provider %s not configured", providerName)
	}

	gcpService, ok := provider.(*gcpservices.GCPService)
	if !ok {
		return fmt.Errorf("unsupported provider type")
	}

	gcpConfig, err := gcpService.GetCurrentConfigWithCredentials()
	if err != nil {
		return fmt.Errorf("failed to get GCP config: %w", err)
	}

	// Get backend configuration
	backendConfig, err := provider.GetTerraformBackendConfig()
	if err != nil {
		return fmt.Errorf("failed to get backend config: %w", err)
	}

	// Create orchestrator
	envVars := map[string]string{
		"GOOGLE_CREDENTIALS": gcpConfig.ServiceAccountKeyJSON,
		"GOOGLE_PROJECT":     gcpConfig.ProjectID,
	}

	orchestrator, err := tfpkg.NewOrchestrator(tfpkg.OrchestratorConfig{
		WorkDir:         workDir,
		TemplateBaseDir: "terraform-templates",
		EnvVars:         envVars,
	})
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Render infrastructure template
	templateData := InfrastructureTemplateData{
		ClusterName: sanitizeGCPResourceName(cluster.Name),
		BucketName:  backendConfig["bucket"],
		ProjectID:   gcpConfig.ProjectID,
		Region:      gcpConfig.Region,
	}

	if err := orchestrator.RenderTemplateToFile("gcp/infrastructure.tf.tmpl", "main.tf", templateData); err != nil {
		return fmt.Errorf("failed to render infrastructure template: %w", err)
	}

	// Initialize terraform
	if err := orchestrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize terraform: %w", err)
	}

	// Destroy infrastructure
	if err := orchestrator.Destroy(ctx); err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	fmt.Println("Infrastructure destroyed successfully")
	return nil
}

func (s *InfrastructureService) GetInfrastructureStatus(ctx context.Context, providerName string) (map[string]any, error) {
	// Get cluster name from database
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Get provider config
	provider, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		return map[string]any{
			"status": "unconfigured",
		}, nil
	}

	gcpService, ok := provider.(*gcpservices.GCPService)
	if !ok {
		return nil, fmt.Errorf("unsupported provider type")
	}

	gcpConfig, err := gcpService.GetCurrentConfig()
	if err != nil {
		return map[string]any{
			"status": "unconfigured",
		}, nil
	}

	sanitizedName := sanitizeGCPResourceName(cluster.Name)

	return map[string]any{
		"status": gcpConfig.InfrastructureStatus,
		"vpc":    sanitizedName + "-vpc",
		"subnet": sanitizedName + "-subnet",
		"region": gcpConfig.Region,
	}, nil
}

type NodeModuleTemplateData struct {
	ClusterName       string
	TalosImageProject string
	TalosImageName    string
}

// PublishNodeModuleToRepo publishes the Terraform node module to the GitOps repository
func (s *InfrastructureService) PublishNodeModuleToRepo(providerName, talosVersion string) error {
	ctx := context.Background()

	// Get cluster name from database
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Get GCP config for project ID
	provider, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		return fmt.Errorf("provider %s not configured", providerName)
	}
	gcpService, ok := provider.(*gcpservices.GCPService)
	if !ok {
		return fmt.Errorf("unsupported provider type")
	}
	gcpConfig, err := gcpService.GetCurrentConfig()
	if err != nil {
		return fmt.Errorf("failed to get GCP config: %w", err)
	}

	// default to AMD
	talosService := talosservices.NewTalosService(s.db, s.cfg)
	talosImageName, err := talosService.GetGCPImageName("amd64")
	if err != nil {
		return fmt.Errorf("failed to get Talos image name: %w", err)
	}

	// Create temporary directory for module files
	workDir, err := os.MkdirTemp("", "terraform-node-module-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Create orchestrator (no env vars needed for template rendering only)
	orchestrator, err := tfpkg.NewOrchestrator(tfpkg.OrchestratorConfig{
		WorkDir:         workDir,
		TemplateBaseDir: "terraform-templates",
		EnvVars:         nil,
	})
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Template data for module templates
	templateData := NodeModuleTemplateData{
		ClusterName:       sanitizeGCPResourceName(cluster.Name),
		TalosImageProject: gcpConfig.ProjectID,
		TalosImageName:    talosImageName,
	}

	// Module template files to process
	moduleFiles := map[string]string{
		"main.tf.tmpl":      "main.tf",
		"variables.tf.tmpl": "variables.tf",
		"outputs.tf.tmpl":   "outputs.tf",
		"provider.tf.tmpl":  "provider.tf",
	}

	// Render all module templates to work directory
	for templateName, outputName := range moduleFiles {
		templatePath := filepath.Join(providerName, "modules", "node", templateName)
		if err := orchestrator.RenderTemplateToFile(templatePath, outputName, templateData); err != nil {
			return fmt.Errorf("failed to render template %s: %w", templateName, err)
		}
	}

	// Get GitOps config
	gitopsConfig, err := s.gitopsService.GetConfigOrDefault()
	if err != nil {
		return fmt.Errorf("failed to get GitOps config: %w", err)
	}

	// Initialize GitHub client
	ghClient, err := githubpkg.NewClientFromConfig(
		s.cfg.GitHub.AppID,
		s.cfg.GitHub.InstallationID,
		s.cfg.GitHub.PrivateKey,
		gitopsConfig.RepoOwner,
		gitopsConfig.RepoName,
		gitopsConfig.Branch,
	)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Commit to GitOps repository
	moduleBasePath := filepath.Join(gitopsConfig.WorkingDir, providerName, "modules", "node")
	if err := orchestrator.CommitToGitOps(ctx, ghClient.Client, tfpkg.GitOpsConfig{
		Owner:    gitopsConfig.RepoOwner,
		Repo:     gitopsConfig.RepoName,
		Branch:   gitopsConfig.Branch,
		BasePath: moduleBasePath,
		Username: gitopsConfig.Username,
		Email:    gitopsConfig.Email,
	}, fmt.Sprintf("Publish Terraform node module for %s", providerName)); err != nil {
		return fmt.Errorf("failed to commit to repository: %w", err)
	}

	fmt.Printf("Published node module to %s/%s (branch: %s)\n", gitopsConfig.RepoOwner, gitopsConfig.RepoName, gitopsConfig.Branch)
	fmt.Printf("  Module directory: %s\n", moduleBasePath)

	return nil
}

// removes a stuck Terraform state lock
func (s *InfrastructureService) ForceUnlockState(ctx context.Context, providerName, lockID string) error {
	// Get cluster name from database
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Create a temporary directory for terraform operations
	workDir, err := os.MkdirTemp("", "terraform-unlock-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Get provider and config
	provider, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		return fmt.Errorf("provider %s not configured", providerName)
	}

	gcpService, ok := provider.(*gcpservices.GCPService)
	if !ok {
		return fmt.Errorf("unsupported provider type")
	}

	gcpConfig, err := gcpService.GetCurrentConfigWithCredentials()
	if err != nil {
		return fmt.Errorf("failed to get GCP config: %w", err)
	}

	// Get backend configuration
	backendConfig, err := provider.GetTerraformBackendConfig()
	if err != nil {
		return fmt.Errorf("failed to get backend config: %w", err)
	}

	// Create orchestrator
	envVars := map[string]string{
		"GOOGLE_CREDENTIALS": gcpConfig.ServiceAccountKeyJSON,
		"GOOGLE_PROJECT":     gcpConfig.ProjectID,
	}

	orchestrator, err := tfpkg.NewOrchestrator(tfpkg.OrchestratorConfig{
		WorkDir:         workDir,
		TemplateBaseDir: "terraform-templates",
		EnvVars:         envVars,
	})
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Render infrastructure template (needed to connect to backend)
	templateData := InfrastructureTemplateData{
		ClusterName: sanitizeGCPResourceName(cluster.Name),
		BucketName:  backendConfig["bucket"],
		ProjectID:   gcpConfig.ProjectID,
		Region:      gcpConfig.Region,
	}

	if err := orchestrator.RenderTemplateToFile("gcp/infrastructure.tf.tmpl", "main.tf", templateData); err != nil {
		return fmt.Errorf("failed to render infrastructure template: %w", err)
	}

	if err := orchestrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize terraform: %w", err)
	}

	if err := orchestrator.ForceUnlock(ctx, lockID); err != nil {
		return fmt.Errorf("terraform force-unlock failed: %w", err)
	}

	fmt.Printf("State lock removed successfully (Lock ID: %s)\n", lockID)
	return nil
}
