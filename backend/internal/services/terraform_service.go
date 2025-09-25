package services

import (
	"fmt"
	"text/template"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/config"
	"github.com/etsmtl-pfe-cloudnative/backend/pkg/gcp"
	"github.com/etsmtl-pfe-cloudnative/backend/pkg/github"
	"gorm.io/gorm"
)

type TerraformService struct {
	db         *gorm.DB
	cfg        *config.Config
	gcpClient  *gcp.Client
	githubClient *github.Client
}

type NodeConfig struct {
	Name         string
	Zone         string
	MachineType  string
	Architecture string
}

func NewTerraformService(db *gorm.DB, cfg *config.Config) *TerraformService {
	return &TerraformService{db: db, cfg: cfg}
}

// initialize the service with GCP and GitHub clients from kubeconfig
func (s *TerraformService) WithCredentials(kubeconfig []byte) error {
	// Initialize GCP client from environment variables
	gcpClient, err := gcp.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	// Initialize GitHub client from environment variables
	githubClient, err := github.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	s.gcpClient = gcpClient
	s.githubClient = githubClient
	return nil
}

// GenerateGCPNodeConfig generates Terraform configuration for GCP node
func (s *TerraformService) GenerateGCPNodeConfig(nodeConfig NodeConfig) (string, error) {
	// TODO: Load template, execute with nodeConfig data, return generated .tf content
	// from terraform-templates/gcp/node.tf.tmpl

	return fmt.Sprintf("# Generated Terraform config for node: %s", nodeConfig.Name), nil
}

// commits generated tf to repository
func (s *TerraformService) CommitToRepo(configContent, commitMessage string) error {
	// TODO: Use cfg.GitOps settings to commit Terraform files

	return nil
}

func loadTemplate(templatePath string) (*template.Template, error) {
	return template.ParseFiles(templatePath)
}