package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	machineconf "github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/helpers"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	gitopsservices "github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	talosservices "github.com/stolos-cloud/stolos/backend/internal/services/talos"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	githubpkg "github.com/stolos-cloud/stolos/backend/pkg/github"
	tfpkg "github.com/stolos-cloud/stolos/backend/pkg/terraform"
	"google.golang.org/api/option"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProvisioningService struct {
	db                 *gorm.DB
	cfg                *config.Config
	talosService       *talosservices.TalosService
	gcpService         *GCPService
	gitopsService      *gitopsservices.GitOpsService
	activeProvisions   map[uuid.UUID]*ProvisionSession
}

// ProvisionSession tracks terraform state for an active provision
type ProvisionSession struct {
	WorkDir      string
	Orchestrator *tfpkg.Orchestrator
	Nodes        []NodeConfig
	GCPConfig    *models.GCPConfig
}

func NewProvisioningService(
	db *gorm.DB,
	cfg *config.Config,
	talosService *talosservices.TalosService,
	gcpService *GCPService,
	gitopsService *gitopsservices.GitOpsService,
) *ProvisioningService {
	return &ProvisioningService{
		db:               db,
		cfg:              cfg,
		talosService:     talosService,
		gcpService:       gcpService,
		gitopsService:    gitopsService,
		activeProvisions: make(map[uuid.UUID]*ProvisionSession),
	}
}

// getNextNodeNumber queries existing nodes and returns the next available number for a given prefix
func (s *ProvisioningService) getNextNodeNumber(clusterID uuid.UUID, namePrefix string) (int, error) {
	var nodes []models.Node

	// Query nodes with matching prefix in the cluster
	pattern := namePrefix + "-%"
	if err := s.db.Where("cluster_id = ? AND name LIKE ?", clusterID, pattern).Find(&nodes).Error; err != nil {
		return 0, fmt.Errorf("failed to query existing nodes: %w", err)
	}

	// If no nodes exist, start from 1
	if len(nodes) == 0 {
		return 1, nil
	}

	// Parse numbers from existing node names and find max
	maxNum := 0
	prefixLen := len(namePrefix) + 1 // +1 for the dash
	for _, node := range nodes {
		if len(node.Name) > prefixLen {
			var num int
			_, err := fmt.Sscanf(node.Name[prefixLen:], "%d", &num)
			if err == nil && num > maxNum {
				maxNum = num
			}
		}
	}

	return maxNum + 1, nil
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
	gcpConfig, err := s.gcpService.GetCurrentConfigWithCredentials()
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

	// Get Talos image information
	talosImageName, err := s.talosService.GetGCPImageName("amd64")
	if err != nil {
		return fmt.Errorf("failed to get Talos image: %w", err)
	}

	// Get next available node number
	startNum, err := s.getNextNodeNumber(clusterID, req.NamePrefix)
	if err != nil {
		return fmt.Errorf("failed to determine next node number: %w", err)
	}

	// Generate node names and configs
	session.SendLog(fmt.Sprintf("Generating configurations for %d node(s)...", req.Number))
	nodes := make([]NodeConfig, 0, req.Number)

	for i := 0; i < req.Number; i++ {
		// Get a fresh config bundle for each node
		session.SendLog("Loading Talos machine configuration bundle...")
		configBundle, err := s.talosService.GetMachineConfigBundle()
		if err != nil {
			return fmt.Errorf("failed to get machine config bundle: %w", err)
		}

		nodeName := fmt.Sprintf("%s-%d", req.NamePrefix, startNum+i)
		session.SendLog(fmt.Sprintf("Generating Talos config for node: %s", nodeName))

		// Determine machine type based on role
		var machineType machineconf.Type
		if req.Role == "control-plane" {
			machineType = machineconf.TypeControlPlane
		} else {
			machineType = machineconf.TypeWorker
		}

		// GCP always uses /dev/sda for boot disk
		diskPath := "/dev/sda"

		// Create typed config patch with hostname, disk, and network settings
		typedPatch, err := talosservices.CreateMachineConfigPatch(nodeName, diskPath)
		if err != nil {
			return fmt.Errorf("failed to create config patch: %w", err)
		}
		session.SendLog(fmt.Sprintf("Created typed config patch for hostname: %s, disk: %s", nodeName, diskPath))

		// Apply typed patch to bundle based on machine type
		isControlPlane := machineType == machineconf.TypeControlPlane
		isWorker := machineType == machineconf.TypeWorker
		if err := configBundle.ApplyPatches([]configpatcher.Patch{typedPatch}, isControlPlane, isWorker); err != nil {
			return fmt.Errorf("failed to apply typed patch to config bundle: %w", err)
		}

		// Serialize the patched config
		rendered, err := configBundle.Serialize(encoder.CommentsDocs, machineType)
		if err != nil {
			return fmt.Errorf("failed to serialize config: %w", err)
		}

		// Remove diskSelector from base config (hardware-specific busPath won't work on GCP)
		patchedBytes := helpers.RemoveDiskSelector(rendered)
		machineConfig := string(patchedBytes)
		session.SendLog("Applied typed config patch and removed diskSelector")

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

	// Upload Talos configs to GCS bucket
	session.SendLog("Uploading Talos configurations to  storage...")
	if err := s.uploadTalosConfigsToGCS(ctx, gcpConfig, nodes); err != nil {
		return fmt.Errorf("failed to upload Talos configs to GCS: %w", err)
	}
	session.SendLog("Talos configurations uploaded successfully")

	// Create terraform files
	session.SendLog("Creating Terraform configuration files...")
	if err := s.createTerraformFiles(ctx, requestID, gcpConfig, nodes); err != nil {
		return fmt.Errorf("failed to create terraform files: %w", err)
	}

	session.SendLog("Terraform files created successfully")

	// Get the provision session
	provSession, ok := s.activeProvisions[requestID]
	if !ok {
		return fmt.Errorf("provision session not found")
	}

	session.SendLog("Initializing Terraform...")
	if err := provSession.Orchestrator.Init(ctx); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	session.SendLog("Running terraform plan...")
	session.SendStatus("planning")

	// Run terraform plan with output
	hasChanges, planOutput, err := provSession.Orchestrator.PlanWithOutput(ctx)
	if err != nil {
		return fmt.Errorf("terraform plan failed: %w", err)
	}

	session.SendLog("Terraform plan executed successfully")

	// Save plan output to file
	planDir := "plans"
	if err := os.MkdirAll(planDir, 0755); err != nil {
		log.Printf("Warning: failed to create plans directory: %v", err)
	}

	planFilename := fmt.Sprintf("plan-%s.txt", requestID.String())
	planFilePath := filepath.Join(planDir, planFilename)

	if err := os.WriteFile(planFilePath, []byte(planOutput), 0644); err != nil {
		log.Printf("Warning: failed to save plan output to file: %v", err)
	} else {
		session.SendLog(fmt.Sprintf("Plan saved to file: %s", planFilename))
	}

	// Create summary
	planSummary := fmt.Sprintf("Terraform plan completed\n")
	if hasChanges {
		planSummary += fmt.Sprintf("Plan: %d node(s) to add\n", len(nodes))
	} else {
		planSummary += "No changes detected\n"
	}
	session.SendLog(planSummary)

	// Store plan output in database
	if err := s.updatePlanOutput(requestID, planSummary); err != nil {
		log.Printf("Warning: failed to store plan output: %v", err)
	}

	// Send plan file path to client
	session.SendLog("Terraform plan completed successfully")
	session.SendPlan(fmt.Sprintf("Plan file: /api/gcp/nodes/provision/%s/plan", requestID.String()))

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
	if err := s.commitTerraformFiles(ctx, requestID); err != nil {
		return fmt.Errorf("failed to commit terraform files: %w", err)
	}
	session.SendLog("Terraform files committed successfully")

	// Update status to applying
	if err := s.updateProvisionStatus(requestID, models.ProvisionStatusApplying); err != nil {
		return err
	}

	session.SendStatus("applying")
	session.SendLog("Running terraform apply...")

	// Run terraform apply
	if err := provSession.Orchestrator.Apply(ctx); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	applyOutput := fmt.Sprintf("Apply complete! Resources: %d node(s) added\n", len(nodes))

	session.SendLog("Terraform apply executed successfully")
	session.SendLog(applyOutput)
	session.SendLog("Terraform apply completed successfully")

	// Parse terraform output to get instance details
	session.SendLog("Fetching created instance information...")
	instanceDetails, err := s.getTerraformOutputs(ctx, requestID)
	if err != nil {
		log.Printf("Warning: failed to get terraform outputs: %v", err)
		// Continue anyway, we can still create node records
	}

	// Cleanup the provision session and temp directory
	defer func() {
		if provSession.WorkDir != "" {
			os.RemoveAll(provSession.WorkDir)
		}
		delete(s.activeProvisions, requestID)
	}()

	// Create or update node records in database
	session.SendLog("Creating node records in database...")
	nodeIDs := make([]uuid.UUID, 0, len(nodes))

	for i, nodeConfig := range nodes {
		// Check if node already exists
		var existingNode models.Node
		err := s.db.Where("name = ? AND cluster_id = ?", nodeConfig.Name, clusterID).First(&existingNode).Error

		if err == nil {
			if existingNode.Status == "active" {
				session.SendLog(fmt.Sprintf("Warning: Node %s already exists and is active, skipping", nodeConfig.Name))
				continue
			}
			// Allow retry for failed or stuck provisioning
			session.SendLog(fmt.Sprintf("Node %s exists with status '%s', retrying provision", nodeConfig.Name, existingNode.Status))
			existingNode.Status = "provisioning"

			if err := s.db.Save(&existingNode).Error; err != nil {
				log.Printf("Warning: failed to update node record for %s: %v", nodeConfig.Name, err)
				continue
			}

			nodeIDs = append(nodeIDs, existingNode.ID)
		} else if err == gorm.ErrRecordNotFound {
			node := models.Node{
				ID:           uuid.New(),
				Name:         nodeConfig.Name,
				Provider:     "gcp",
				Role:         nodeConfig.Role,
				Status:       "provisioning",
				ClusterID:    clusterID,
				Architecture: "amd64",
			}

			// Add labels
			if len(nodeConfig.Labels) > 0 {
				labelsJSON, _ := json.Marshal(nodeConfig.Labels)
				node.Labels = string(labelsJSON)
			}

			// Add instance details if available
			if i < len(instanceDetails) {
				node.InstanceID = instanceDetails[i].InstanceID
				node.IPAddress = instanceDetails[i].InternalIP
			}

			if err := s.db.Create(&node).Error; err != nil {
				log.Printf("Warning: failed to create node record for %s: %v", nodeConfig.Name, err)
				continue
			}

			nodeIDs = append(nodeIDs, node.ID)
			session.SendLog(fmt.Sprintf("Created node record: %s (ID: %s)", node.Name, node.ID))
		} else {
			// Database error
			log.Printf("Error checking for existing node %s: %v", nodeConfig.Name, err)
			continue
		}
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

	if len(nodeIDs) > 0 {
		session.SendLog(fmt.Sprintf("✓ Provisioning completed successfully: %d node(s) ready", len(nodeIDs)))
	} else {
		session.SendLog("⚠ Provisioning completed but no node records were created/updated")
	}

	// Send completion message with node details
	session.SendComplete(map[string]interface{}{
		"node_ids":    nodeIDs,
		"nodes_count": len(nodeIDs),
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

// createTerraformFiles creates terraform configuration files
func (s *ProvisioningService) createTerraformFiles(ctx context.Context, requestID uuid.UUID, gcpConfig *models.GCPConfig, nodes []NodeConfig) error {
	// Get GitOps config
	gitopsConfig, err := s.gitopsService.GetConfigOrDefault()
	if err != nil {
		return fmt.Errorf("failed to get GitOps config: %w", err)
	}

	// Initialize GitHub client
	ghClient, err := s.gitopsService.GetGitHubClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	existingFiles, err := s.fetchExistingNodeFiles(ctx, ghClient, gitopsConfig)
	if err != nil {
		log.Printf("Warning: failed to fetch existing node files: %v", err)
	}

	tempRoot, err := os.MkdirTemp("", "terraform-nodes-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	workDir := filepath.Join(tempRoot, "terraform", "gcp")
	modulesDir := filepath.Join(tempRoot, "modules", "node")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create gcp directory: %w", err)
	}
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create modules directory: %w", err)
	}

	var cluster models.Cluster
	if err := s.db.First(&cluster).Error; err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	talosImageName, err := s.talosService.GetGCPImageName("amd64")
	if err != nil {
		return fmt.Errorf("failed to get Talos image name: %w", err)
	}

	// Template data for node module
	moduleTemplateData := struct {
		ClusterName       string
		TalosImageProject string
		TalosImageName    string
	}{
		ClusterName:       helpers.SanitizeResourceName(cluster.Name),
		TalosImageProject: gcpConfig.ProjectID,
		TalosImageName:    talosImageName,
	}

	// todo see if we can handle better
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

	// Render module files to modules/node directory
	moduleFiles := map[string]string{
		"main.tf":      "gcp/modules/node/main.tf.tmpl",
		"variables.tf": "gcp/modules/node/variables.tf.tmpl",
		"outputs.tf":   "gcp/modules/node/outputs.tf.tmpl",
		"provider.tf":  "gcp/modules/node/provider.tf.tmpl",
	}

	for outputFile, templatePath := range moduleFiles {
		content, err := orchestrator.RenderTemplate(templatePath, moduleTemplateData)
		if err != nil {
			return fmt.Errorf("failed to render module template %s: %w", templatePath, err)
		}

		fullPath := filepath.Join(modulesDir, outputFile)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write module file %s: %w", outputFile, err)
		}
	}

	// Write existing files to work directory
	for filename, content := range existingFiles {
		if err := os.WriteFile(filepath.Join(workDir, filename), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write existing file %s: %w", filename, err)
		}
	}

	// Store session
	s.activeProvisions[requestID] = &ProvisionSession{
		WorkDir:      tempRoot, // Store root for cleanup
		Orchestrator: orchestrator,
		Nodes:        nodes,
		GCPConfig:    gcpConfig,
	}

	// Render a node.tf file for each node
	for _, node := range nodes {
		content, err := orchestrator.RenderTemplate("gcp/node.tf.tmpl", struct {
			Name         string
			Timestamp    string
			BucketName   string
			ClusterName  string
			MachineType  string
			Zone         string
			Region       string
			Role         string
			Architecture string
			DiskSizeGB   int
			DiskType     string
		}{
			Name:         helpers.SanitizeResourceName(node.Name),
			Timestamp:    time.Now().Format(time.RFC3339),
			BucketName:   gcpConfig.BucketName,
			ClusterName:  helpers.SanitizeResourceName(cluster.Name),
			MachineType:  node.MachineType,
			Zone:         node.Zone,
			Region:       gcpConfig.Region,
			Role:         node.Role,
			Architecture: "amd64",
			DiskSizeGB:   node.DiskSizeGB,
			DiskType:     node.DiskType,
		})
		if err != nil {
			return fmt.Errorf("failed to render template for node %s: %w", node.Name, err)
		}

		filename := fmt.Sprintf("node-%s.tf", helpers.SanitizeResourceName(node.Name))
		if err := os.WriteFile(filepath.Join(workDir, filename), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	log.Printf("Created terraform files for %d nodes in %s", len(nodes), workDir)
	return nil
}

// fetchExistingNodeFiles fetches existing node-.tf files from GitHub to get the latest state
func (s *ProvisioningService) fetchExistingNodeFiles(ctx context.Context, ghClient *githubpkg.Client, gitopsConfig *models.GitOpsConfig) (map[string]string, error) {
	owner, repo := ghClient.GetRepoInfo()
	path := filepath.Join(gitopsConfig.WorkingDir, "gcp")

	// List files in the gcp directory
	_, dirContent, _, err := ghClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: gitopsConfig.Branch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list directory contents: %w", err)
	}

	existingFiles := make(map[string]string)

	// Fetch each node-.tf file
	for _, item := range dirContent {
		if item.GetType() == "file" && filepath.Ext(item.GetName()) == ".tf" &&
		   (item.GetName() == "main.tf" || strings.HasPrefix(item.GetName(), "node-")) {

			fileContent, _, _, err := ghClient.Repositories.GetContents(ctx, owner, repo, item.GetPath(), &github.RepositoryContentGetOptions{
				Ref: gitopsConfig.Branch,
			})
			if err != nil {
				log.Printf("Warning: failed to fetch %s: %v", item.GetName(), err)
				continue
			}

			content, err := fileContent.GetContent()
			if err != nil {
				log.Printf("Warning: failed to decode %s: %v", item.GetName(), err)
				continue
			}

			existingFiles[item.GetName()] = content
		}
	}

	return existingFiles, nil
}

// commitTerraformFiles commits terraform files to GitOps repo using GitHub API
func (s *ProvisioningService) commitTerraformFiles(ctx context.Context, requestID uuid.UUID) error {
	provSession, ok := s.activeProvisions[requestID]
	if !ok {
		return fmt.Errorf("provision session not found")
	}

	// Get GitOps config
	gitopsConfig, err := s.gitopsService.GetConfigOrDefault()
	if err != nil {
		return fmt.Errorf("failed to get GitOps config: %w", err)
	}

	// Initialize GitHub client
	ghClient, err := s.gitopsService.GetGitHubClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Build commit message
	nodeNames := make([]string, len(provSession.Nodes))
	for i, node := range provSession.Nodes {
		nodeNames[i] = node.Name
	}
	commitMessage := fmt.Sprintf("Add node configurations: %v", nodeNames)

	// Commit to GitOps repository using the orchestrator
	if err := provSession.Orchestrator.CommitToGitOps(ctx, ghClient.Client, tfpkg.GitOpsConfig{
		Owner:    gitopsConfig.RepoOwner,
		Repo:     gitopsConfig.RepoName,
		Branch:   gitopsConfig.Branch,
		BasePath: filepath.Join(gitopsConfig.WorkingDir, "gcp"),
		Username: gitopsConfig.Username,
		Email:    gitopsConfig.Email,
	}, commitMessage); err != nil {
		return fmt.Errorf("failed to commit to repository: %w", err)
	}

	log.Printf("Committed terraform files for nodes: %v", nodeNames)
	return nil
}

// uploadTalosConfigsToGCS uploads Talos machine configs to GCS bucket
func (s *ProvisioningService) uploadTalosConfigsToGCS(ctx context.Context, gcpConfig *models.GCPConfig, nodes []NodeConfig) error {
	// Create GCS client
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(gcpConfig.ServiceAccountKeyJSON)))
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	bucket := client.Bucket(gcpConfig.BucketName)

	// Upload each node's Talos config
	for _, node := range nodes {
		objectName := fmt.Sprintf("talos-configs/%s.yaml", node.Name)
		obj := bucket.Object(objectName)

		writer := obj.NewWriter(ctx)
		writer.ContentType = "text/yaml"

		if _, err := writer.Write([]byte(node.TalosConfig)); err != nil {
			writer.Close()
			return fmt.Errorf("failed to write config for %s: %w", node.Name, err)
		}

		if err := writer.Close(); err != nil {
			return fmt.Errorf("failed to close writer for %s: %w", node.Name, err)
		}

		log.Printf("Uploaded Talos config to gs://%s/%s", gcpConfig.BucketName, objectName)
	}

	return nil
}

// getTerraformOutputs retrieves instance details from terraform outputs
func (s *ProvisioningService) getTerraformOutputs(ctx context.Context, requestID uuid.UUID) ([]InstanceDetails, error) {
	provSession, ok := s.activeProvisions[requestID]
	if !ok {
		return nil, fmt.Errorf("provision session not found")
	}

	// Get terraform outputs
	outputs, err := provSession.Orchestrator.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get terraform outputs: %w", err)
	}

	// Parse outputs for each node
	details := make([]InstanceDetails, 0, len(provSession.Nodes))

	for _, node := range provSession.Nodes {
		// Each node has an output like: "<node-name>_info"
		outputKey := fmt.Sprintf("%s_info", helpers.SanitizeResourceName(node.Name))

		if output, ok := outputs[outputKey]; ok {
			// Output -> map with instance details
			if nodeInfo, ok := output.(map[string]interface{}); ok {
				detail := InstanceDetails{}

				if instanceID, ok := nodeInfo["instance_id"].(string); ok {
					detail.InstanceID = instanceID
				}
				if internalIP, ok := nodeInfo["internal_ip"].(string); ok {
					detail.InternalIP = internalIP
				}
				if externalIP, ok := nodeInfo["external_ip"].(string); ok {
					detail.ExternalIP = externalIP
				}

				details = append(details, detail)
			}
		}
	}

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
