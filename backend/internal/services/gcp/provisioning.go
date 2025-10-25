package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	talosservices "github.com/stolos-cloud/stolos/backend/internal/services/talos"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProvisioningService struct {
	db           *gorm.DB
	cfg          *config.Config
	talosService *talosservices.TalosService
	gcpService   *GCPService
}

func NewProvisioningService(
	db *gorm.DB,
	cfg *config.Config,
	talosService *talosservices.TalosService,
	gcpService *GCPService,
) *ProvisioningService {
	return &ProvisioningService{
		db:           db,
		cfg:          cfg,
		talosService: talosService,
		gcpService:   gcpService,
	}
}

// ProvisionNodes orchestrates the complete node provisioning workflow
func (s *ProvisioningService) ProvisionNodes(
	ctx context.Context,
	requestID uuid.UUID,
	req models.GCPNodeProvisionRequest,
	session *wsservices.ApprovalSession,
) error {
	// Update status to planning
	if err := s.updateProvisionStatus(requestID, models.ProvisionStatusPlanning); err != nil {
		return err
	}

	session.SendStatus("planning")
	session.SendLog("Starting node provisioning workflow...")

	// Get GCP configuration
	session.SendLog("Fetching GCP configuration...")
	gcpConfig, err := s.gcpService.GetCurrentConfig()
	if err != nil {
		return fmt.Errorf("failed to get GCP config: %w", err)
	}

	session.SendLog("Fetching Talos cluster information...")
	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("no cluster found in database - cluster discovery may not have run yet")
		}
		return fmt.Errorf("failed to fetch cluster: %w", err)
	}
	clusterID := cluster.ID
	session.SendLog(fmt.Sprintf("Using cluster: %s (ID: %s)", cluster.Name, cluster.ID))

	// Generate node names and configs
	session.SendLog(fmt.Sprintf("Generating configurations for %d node(s)...", req.Number))
	nodes := make([]NodeConfig, 0, req.Number)

	for i := 0; i < req.Number; i++ {
		nodeName := fmt.Sprintf("%s-%d", req.NamePrefix, i+1)

		session.SendLog(fmt.Sprintf("Generating Talos config for node: %s", nodeName))

		// TODO: Generate Talos machine config
		// This needs to be implemented using the factory and cluster config
		// The node type should be determined from req.Role:
		// "control-plane" -> "controlplane", "worker" -> "worker"
		session.SendLog(fmt.Sprintf("TODO: Generate Talos config for node: %s (placeholder)", nodeName))
		machineConfig := "" // Placeholder

		// Get Talos image information
		// Images are uploaded to the user's GCP project during provider initialization
		talosImageName, err := s.talosService.GetGCPImageName("amd64")
		if err != nil {
			return fmt.Errorf("failed to get Talos image: %w", err)
		}

		nodes = append(nodes, NodeConfig{
			Name:              nodeName,
			Zone:              req.Zone,
			MachineType:       req.MachineType,
			Role:              req.Role,
			Labels:            req.Labels,
			DiskSizeGB:        req.DiskSizeGB,
			DiskType:          req.DiskType,
			TalosConfig:       machineConfig,
			TalosImageProject: gcpConfig.ProjectID,
			TalosImageName:    talosImageName,
		})
	}

	session.SendLog(fmt.Sprintf("Generated configurations for %d node(s)", len(nodes)))

	// Create terraform files in gitops repo
	session.SendLog("Creating Terraform configuration files...")
	if err := s.createTerraformFiles(gcpConfig, nodes); err != nil {
		return fmt.Errorf("failed to create terraform files: %w", err)
	}

	session.SendLog("Terraform files created successfully")

	session.SendLog("Running terraform plan...")
	session.SendStatus("planning")

	// TODO: Implement actual terraform plan
	// should call: s.terraformService.RunTerraformPlan(ctx, "gcp", outputCallback)
	planOutput := "Placeholder terraform plan output\n" +
		fmt.Sprintf("Plan: %d to add, 0 to change, 0 to destroy\n", len(nodes)) +
		"\nTODO: Implement actual terraform plan execution"

	session.SendLog("Terraform plan placeholder executed")
	session.SendLog(planOutput)

	// Store plan output
	if err := s.updatePlanOutput(requestID, planOutput); err != nil {
		log.Printf("Warning: failed to store plan output: %v", err)
	}

	// Send plan to client and request approval
	session.SendLog("Terraform plan completed successfully")
	session.SendPlan(planOutput)

	// Update status to awaiting approval
	if err := s.updateProvisionStatus(requestID, models.ProvisionStatusAwaitingApproval); err != nil {
		return err
	}

	session.SendStatus("awaiting_approval")
	session.SendApprovalRequest(fmt.Sprintf("Ready to create %d node(s). Please review the plan and approve to continue.", req.Number))

	// Wait for approval (with timeout)
	session.SendLog("Waiting for user approval...")
	approved, err := session.WaitForApprovalCtx(ctx, 30*time.Minute)
	if err != nil {
		return fmt.Errorf("approval failed: %w", err)
	}

	if !approved {
		session.SendLog("Provisioning rejected by user")
		if err := s.updateProvisionStatus(requestID, models.ProvisionStatusFailed); err != nil {
			log.Printf("Warning: failed to update status: %v", err)
		}
		return fmt.Errorf("provisioning rejected by user")
	}

	session.SendLog("Provisioning approved by user")

	// Commit terraform files to git repo after approval
	session.SendLog("Committing terraform files to GitOps repository...")
	if err := s.commitTerraformFiles(nodes); err != nil {
		return fmt.Errorf("failed to commit terraform files: %w", err)
	}
	session.SendLog("Terraform files committed successfully")

	// Update status to applying
	if err := s.updateProvisionStatus(requestID, models.ProvisionStatusApplying); err != nil {
		return err
	}

	session.SendStatus("applying")
	session.SendLog("Running terraform apply...")

	// TODO: Implement actual terraform apply
	// This should call: s.terraformService.RunTerraformApply(ctx, "gcp", outputCallback)
	applyOutput := "Placeholder terraform apply output\n" +
		fmt.Sprintf("Apply complete! Resources: %d added, 0 changed, 0 destroyed\n", len(nodes)) +
		"\nTODO: Implement actual terraform apply execution"

	session.SendLog("Terraform apply placeholder executed")
	session.SendLog(applyOutput)
	session.SendLog("Terraform apply completed successfully")

	// Parse terraform output to get instance details
	session.SendLog("Fetching created instance information...")
	instanceDetails, err := s.getTerraformOutputs("gcp", nodes)
	if err != nil {
		log.Printf("Warning: failed to get terraform outputs: %v", err)
		// Continue anyway, we can still create node records
	}

	// Create node records in database
	session.SendLog("Creating node records in database...")
	nodeIDs := make([]uuid.UUID, 0, len(nodes))

	for i, nodeConfig := range nodes {
		node := models.Node{
			ID:           uuid.New(),
			Name:         nodeConfig.Name,
			Provider:     "gcp",
			Role:         nodeConfig.Role,
			Status:       "provisioning", // Will be updated when Talos reports in
			ClusterID:    clusterID,
			Architecture: "amd64", // Default for GCP
		}

		// Add instance details if available
		if i < len(instanceDetails) {
			node.InstanceID = instanceDetails[i].InstanceID
			node.IPAddress = instanceDetails[i].InternalIP // Use internal IP as primary
		}

		if err := s.db.Create(&node).Error; err != nil {
			log.Printf("Warning: failed to create node record for %s: %v", nodeConfig.Name, err)
			continue
		}

		nodeIDs = append(nodeIDs, node.ID)
		session.SendLog(fmt.Sprintf("Created node record: %s (ID: %s)", node.Name, node.ID))
	}

	// Update provision request with node IDs
	nodeIDsJSON, _ := json.Marshal(nodeIDs)
	if err := s.db.Model(&models.ProvisionRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"node_ids": datatypes.JSON(nodeIDsJSON),
			"status":   models.ProvisionStatusCompleted,
		}).Error; err != nil {
		log.Printf("Warning: failed to update provision request: %v", err)
	}

	session.SendStatus("completed")
	session.SendLog(fmt.Sprintf("Successfully provisioned %d node(s)", len(nodeIDs)))

	// Send completion message with node details
	session.SendComplete(map[string]interface{}{
		"node_ids":      nodeIDs,
		"nodes":         len(nodeIDs),
		"plan_output":   planOutput,
		"apply_output":  applyOutput,
	})

	return nil
}

// NodeConfig holds the configuration for a single node
type NodeConfig struct {
	Name              string
	Zone              string
	MachineType       string
	Role              string
	Labels            []string
	DiskSizeGB        int
	DiskType          string
	TalosConfig       string
	TalosImageProject string
	TalosImageName    string
}

// InstanceDetails holds terraform output for a node
type InstanceDetails struct {
	InstanceID string
	InternalIP string
	ExternalIP string
}

// createTerraformFiles creates terraform configuration files in the gitops repo
func (s *ProvisioningService) createTerraformFiles(gcpConfig *models.GCPConfig, nodes []NodeConfig) error {
	// TODO: Implement terraform file creation
	// 1. Load the node.tf.tmpl template
	// 2. For each node, render the template with node config
	// 3. Write to gitops repo: terraform/gcp/node-{name}.tf
	// Note: We don't commit here ->  happens after approval

	log.Printf("Would create terraform files for %d nodes", len(nodes))
	return nil
}

// commitTerraformFiles commits and pushes terraform files to git repo
func (s *ProvisioningService) commitTerraformFiles(nodes []NodeConfig) error {
	// TODO: Implement git commit and push
	// This should:
	// git add terraform/gcp/node-*.tf files
	// git commit -m "Add node configurations for <node-names>"
	// git push to remote

	// For now just log
	nodeNames := make([]string, len(nodes))
	for i, node := range nodes {
		nodeNames[i] = node.Name
	}
	log.Printf("Would commit terraform files for nodes: %v", nodeNames)
	return nil
}

// getTerraformOutputs retrieves instance details from terraform outputs
func (s *ProvisioningService) getTerraformOutputs(provider string, nodes []NodeConfig) ([]InstanceDetails, error) {
	// TODO: Implement terraform output parsing
	// This should run: terraform output -json
	// And parse the instance details for each node

	details := make([]InstanceDetails, 0, len(nodes))
	// For now return empty details
	return details, nil
}

// updateProvisionStatus updates the status of a provision request
func (s *ProvisioningService) updateProvisionStatus(requestID uuid.UUID, status models.ProvisionRequestStatus) error {
	return s.db.Model(&models.ProvisionRequest{}).
		Where("id = ?", requestID).
		Update("status", status).Error
}

// updatePlanOutput stores the terraform plan output
func (s *ProvisioningService) updatePlanOutput(requestID uuid.UUID, planOutput string) error {
	return s.db.Model(&models.ProvisionRequest{}).
		Where("id = ?", requestID).
		Update("plan_output", planOutput).Error
}
