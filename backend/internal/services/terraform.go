package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
[	"time"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	gitopsservices "github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	githubpkg "github.com/stolos-cloud/stolos/backend/pkg/github"
	"gorm.io/gorm"
)

type TerraformService struct {
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

func NewTerraformService(db *gorm.DB, cfg *config.Config, providerManager *ProviderManager, gitopsService *gitopsservices.GitOpsService) *TerraformService {
	return &TerraformService{
		db:              db,
		cfg:             cfg,
		providerManager: providerManager,
		gitopsService:   gitopsService,
	}
}

// generates Terraform configuration for GCP node
func (s *TerraformService) GenerateGCPNodeConfig(nodeConfig NodeConfig) (string, error) {
	// TODO: Load template, execute with ]nodeConfig data, return generated .tf content
	// from terraform-templates/gcp/node.tf.tmpl

	return fmt.Sprintf("# Generated Terraform config for node: %s", nodeConfig.Name), nil
}

func (s *TerraformService) CommitToRepo(configContent, commitMessage string) error {
	// TODO: Use cfg.GitOps settings to commit Terraform files

	return nil
}

// sets up the base infrastructure (VPC, subnets, etc.) needed for VM provisioning
func (s *TerraformService) InitializeInfrastructure(ctx context.Context, providerName string) error {
	// Create a temporary directory for terraform files
	workDir, err := os.MkdirTemp("", "terraform-infra-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Generate infrastructure terraform configuration
	if err := s.generateInfrastructureConfig(workDir, providerName); err != nil {
		return fmt.Errorf("failed to generate infrastructure config: %w", err)
	}

	tf, err := s.initializeTerraform(workDir, providerName)
	if err != nil {
		return fmt.Errorf("failed to initialize terraform: %w", err)
	}

	//todo
	// plan, err := tf.Plan(ctx)
	
	if err := s.applyInfrastructure(ctx, tf); err != nil {
		return fmt.Errorf("failed to apply infrastructure: %w", err)
	}

	// Commit terraform files to repository
	if err := s.commitInfrastructureToRepo(workDir, providerName); err != nil {
		return fmt.Errorf("failed to commit to repository: %w", err)
	}

	return nil
}

type InfrastructureTemplateData struct {
	BucketName string
	ProjectID  string
	Region     string
}

// generateInfrastructureConfig creates terraform files for base infrastructure using templates
func (s *TerraformService) generateInfrastructureConfig(workDir string, providerName string) error {
	provider, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		return fmt.Errorf("provider %s not configured", providerName)
	}

	// Get backend configuration
	backendConfig, err := provider.GetTerraformBackendConfig()
	if err != nil {
		return fmt.Errorf("failed to get backend config: %w", err)
	}

	// Get provider-specific config (currently only GCP is supported)
	gcpService, ok := provider.(*gcpservices.GCPService)
	if !ok {
		return fmt.Errorf("unsupported provider type")
	}

	gcpConfig, err := gcpService.GetCurrentConfig()
	if err != nil {
		return fmt.Errorf("failed to get GCP config: %w", err)
	}

	// Prepare template data
	templateData := InfrastructureTemplateData{
		BucketName: backendConfig["bucket"],
		ProjectID:  gcpConfig.ProjectID,
		Region:     gcpConfig.Region,
	}

	// Load and execute template
	templatePath := "terraform-templates/gcp/infrastructure.tf.tmpl"
	tmpl, err := loadTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("failed to load infrastructure template: %w", err)
	}

	// Generate main.tf from template
	mainTfPath := filepath.Join(workDir, "main.tf")
	mainTfFile, err := os.Create(mainTfPath)
	if err != nil {
		return fmt.Errorf("failed to create main.tf: %w", err)
	}
	defer mainTfFile.Close()

	if err := tmpl.Execute(mainTfFile, templateData); err != nil {
		return fmt.Errorf("failed to execute infrastructure template: %w", err)
	}

	return nil
}

// initializeTerraform initializes terraform in the working directory
func (s *TerraformService) initializeTerraform(workDir string, providerName string) (*tfexec.Terraform, error) {
	// Find terraform binary (assuming it's in PATH)
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("terraform binary not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(workDir, terraformPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform instance: %w", err)
	}

	provider, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		return nil, fmt.Errorf("provider %s not configured", providerName)
	}

	// Set up provider credentials (currently only GCP is supported)
	gcpService, ok := provider.(*gcpservices.GCPService)
	if !ok {
		return nil, fmt.Errorf("unsupported provider type")
	}

	gcpConfig, err := gcpService.GetCurrentConfigWithCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP config: %w", err)
	}

	envVars := map[string]string{
		"GOOGLE_CREDENTIALS": gcpConfig.ServiceAccountKeyJSON,
		"GOOGLE_PROJECT":     gcpConfig.ProjectID,
	}

	// Preserve PATH
	if path := os.Getenv("PATH"); path != "" {
		envVars["PATH"] = path
	}

	tf.SetEnv(envVars)

	// Initialize terraform
	if err := tf.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("terraform init failed: %w", err)
	}

	return tf, nil
}

// applyInfrastructure plans and applies the infrastructure
func (s *TerraformService) applyInfrastructure(ctx context.Context, tf *tfexec.Terraform) error {
	// Plan the changes
	planHasChanges, err := tf.Plan(ctx)
	if err != nil {
		return fmt.Errorf("terraform plan failed: %w", err)
	}

	if !planHasChanges {
		fmt.Println("No infrastructure changes required")
		return nil
	}

	// Apply the changes
	if err := tf.Apply(ctx); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	fmt.Println("Infrastructure provisioned successfully")
	return nil
}

func (s *TerraformService) commitInfrastructureToRepo(workDir, providerName string) error {
	ctx := context.Background()

	// Get GitOps config from database or env
	gitopsConfig, err := s.gitopsService.GetConfigOrDefault()
	if err != nil {
		return fmt.Errorf("failed to get GitOps config: %w", err)
	}

	// Initialize GitHub client from centralized config (not env vars directly)
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

	owner := gitopsConfig.RepoOwner
	repo := gitopsConfig.RepoName
	branch := gitopsConfig.Branch
	// Combine working dir with provider subdirectory (e.g., "terraform/gcp")
	baseWorkingDir := filepath.Join(gitopsConfig.WorkingDir, providerName)

	// Get the latest commit SHA for the branch
	ref, _, err := ghClient.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
	if err != nil {
		return fmt.Errorf("failed to get branch ref: %w", err)
	}
	baseCommitSHA := ref.GetObject().GetSHA()

	// Get the base tree SHA
	baseCommit, _, err := ghClient.Git.GetCommit(ctx, owner, repo, baseCommitSHA)
	if err != nil {
		return fmt.Errorf("failed to get base commit: %w", err)
	}
	baseTreeSHA := baseCommit.GetTree().GetSHA()

	// Read all .tf files from workDir and create tree entries
	var treeEntries []*github.TreeEntry
	err = filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".tf" {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Get relative path from workDir
		relPath, err := filepath.Rel(workDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create blob for file content
		blob, _, err := ghClient.Git.CreateBlob(ctx, owner, repo, &github.Blob{
			Content:  github.Ptr(string(content)),
			Encoding: github.Ptr("utf-8"),
		})
		if err != nil {
			return fmt.Errorf("failed to create blob for %s: %w", relPath, err)
		}

		// Add to tree with path in configured working directory
		terraformPath := filepath.Join(baseWorkingDir, relPath)
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: github.Ptr(terraformPath),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  blob.SHA,
		})

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to process terraform files: %w", err)
	}

	if len(treeEntries) == 0 {
		return fmt.Errorf("no .tf files found in %s", workDir)
	}

	// Create a new tree with the terraform files
	tree, _, err := ghClient.Git.CreateTree(ctx, owner, repo, baseTreeSHA, treeEntries)
	if err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	// Create commit with author from GitOps config
	now := time.Now()
	author := &github.CommitAuthor{
		Name:  github.Ptr(gitopsConfig.Username),
		Email: github.Ptr(gitopsConfig.Email),
		Date:  &github.Timestamp{Time: now},
	}

	commit, _, err := ghClient.Git.CreateCommit(ctx, owner, repo, &github.Commit{
		Message:   github.Ptr("Update infrastructure terraform configuration"),
		Tree:      tree,
		Parents:   []*github.Commit{{SHA: github.Ptr(baseCommitSHA)}},
		Author:    author,
		Committer: author,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Update branch reference to point to new commit
	ref.Object.SHA = commit.SHA
	_, _, err = ghClient.Git.UpdateRef(ctx, owner, repo, ref, false)
	if err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	fmt.Printf("Committed terraform files to %s/%s (branch: %s)\n", owner, repo, branch)
	fmt.Printf("  Working directory: %s\n", baseWorkingDir)
	fmt.Printf("  Commit SHA: %s\n", commit.GetSHA())

	return nil
}


func (s *TerraformService) DestroyInfrastructure(ctx context.Context, providerName string) error {
	// temporary directory for terraform operations
	workDir, err := os.MkdirTemp("", "terraform-destroy-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Generate infrastructure configuration
	if err := s.generateInfrastructureConfig(workDir, providerName); err != nil {
		return fmt.Errorf("failed to generate infrastructure config: %w", err)
	}

	// Initialize terraform
	tf, err := s.initializeTerraform(workDir, providerName)
	if err != nil {
		return fmt.Errorf("failed to initialize terraform: %w", err)
	}

	// Destroy infrastructure
	if err := tf.Destroy(ctx); err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	fmt.Println("Infrastructure destroyed successfully")
	return nil
}

func (s *TerraformService) GetInfrastructureStatus(ctx context.Context) (map[string]any, error) {
	// Mock .. we could instead do
	// 1. Check terraform state
	// 2. Return resource status and outputs

	return map[string]any{
		"status": "provisioned",
		"vpc":    "main-vpc",
		"subnet": "main-subnet",
		"region": s.cfg.GCP.Region,
	}, nil
}

func loadTemplate(templatePath string) (*template.Template, error) {
	return template.ParseFiles(templatePath)
}

// removes a stuck Terraform state lock
func (s *TerraformService) ForceUnlockState(ctx context.Context, providerName, lockID string) error {
	// Create a temporary directory for terraform operations
	workDir, err := os.MkdirTemp("", "terraform-unlock-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Generate infrastructure configuration (needed to connect to backend)
	if err := s.generateInfrastructureConfig(workDir, providerName); err != nil {
		return fmt.Errorf("failed to generate infrastructure config: %w", err)
	}

	// Initialize terraform (to get backend connection)
	tf, err := s.initializeTerraform(workDir, providerName)
	if err != nil {
		return fmt.Errorf("failed to initialize terraform: %w", err)
	}

	// Force unlock with the provided lock ID
	if err := tf.ForceUnlock(ctx, lockID); err != nil {
		return fmt.Errorf("terraform force-unlock failed: %w", err)
	}

	fmt.Printf("State lock removed successfully (Lock ID: %s)\n", lockID)
	return nil
}
