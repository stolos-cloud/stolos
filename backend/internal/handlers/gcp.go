package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	gcpservices "github.com/stolos-cloud/stolos/backend/internal/services/gcp"
	gitopsservices "github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	"github.com/stolos-cloud/stolos/backend/internal/services/node"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Configure allowed origins properly
	},
}

type GCPHandlers struct {
	db                    *gorm.DB
	gcpService            *gcpservices.GCPService
	gitopsService         *gitopsservices.GitOpsService
	nodeService           *node.NodeService
	infrastructureService *services.InfrastructureService
	gcpResourcesService   *gcpservices.GCPResourcesService
	provisioningService   *gcpservices.ProvisioningService
	wsManager             *wsservices.Manager
}

func NewGCPHandlers(
	db *gorm.DB,
	gcpService *gcpservices.GCPService,
	gitopsService *gitopsservices.GitOpsService,
	nodeService *node.NodeService,
	infrastructureService *services.InfrastructureService,
	gcpResourcesService *gcpservices.GCPResourcesService,
	provisioningService *gcpservices.ProvisioningService,
	wsManager *wsservices.Manager,
) *GCPHandlers {
	return &GCPHandlers{
		db:                    db,
		gcpService:            gcpService,
		gitopsService:         gitopsService,
		nodeService:           nodeService,
		infrastructureService: infrastructureService,
		gcpResourcesService:   gcpResourcesService,
		provisioningService:   provisioningService,
		wsManager:             wsManager,
	}
}

// GetGCPStatus godoc
// @Summary Get GCP configuration status
// @Description Retrieve the current GCP configuration status
// @Tags gcp
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /gcp/status [get]
// @Security BearerAuth
func (h *GCPHandlers) GetGCPStatus(c *gin.Context) {
	// Check GCP configuration
	gcpConfig, err := h.gcpService.GetCurrentConfig()
	gcpConfigured := err == nil && gcpConfig != nil

	// Check GitOps configuration
	gitopsConfig, err := h.gitopsService.GetCurrentConfig()
	gitopsConfigured := err == nil && gitopsConfig != nil

	// Build response structure
	response := gin.H{
		"gcp": gin.H{
			"configured": gcpConfigured,
		},
		"gitops": gin.H{
			"configured": gitopsConfigured,
		},
	}

	// Add GCP details if configured
	if gcpConfigured {
		// Check if Talos images are configured
		talosImagesConfigured := gcpConfig.TalosImageAMD64 != ""

		response["gcp"] = gin.H{
			"configured":              true,
			"project_id":              gcpConfig.ProjectID,
			"region":                  gcpConfig.Region,
			"bucket_name":             gcpConfig.BucketName,
			"service_account_email":   gcpConfig.ServiceAccountEmail,
			"infrastructure_status":   gcpConfig.InfrastructureStatus,
			"talos_version":           gcpConfig.TalosVersion,
			"talos_images_configured": talosImagesConfigured,
		}

		// Add Talos image details if configured
		if talosImagesConfigured {
			talosImages := gin.H{}
			if gcpConfig.TalosImageAMD64 != "" {
				talosImages["amd64"] = gcpConfig.TalosImageAMD64
			}
			if gcpConfig.TalosImageARM64 != "" {
				talosImages["arm64"] = gcpConfig.TalosImageARM64
			}
			response["gcp"].(gin.H)["talos_images"] = talosImages
		}

		// Get infrastructure details if infrastructure is set up
		if gcpConfig.InfrastructureStatus != "unconfigured" {
			infraStatus, err := h.infrastructureService.GetInfrastructureStatus(c.Request.Context(), "gcp")
			if err != nil {
				log.Printf("Warning: Failed to get infrastructure status: %v", err)
			} else {
				response["infrastructure"] = infraStatus
			}
		}
	}

	// Add GitOps details if configured
	if gitopsConfigured {
		response["gitops"] = gin.H{
			"configured":  true,
			"repo_owner":  gitopsConfig.RepoOwner,
			"repo_name":   gitopsConfig.RepoName,
			"branch":      gitopsConfig.Branch,
			"working_dir": gitopsConfig.WorkingDir,
		}
	}

	c.JSON(http.StatusOK, response)
}

// ConfigureGCP godoc
// @Summary Configure GCP
// @Description Configure GCP with provided project ID, region, and service account JSON
// @Tags gcp
// @Accept json
// @Produce json
// @Param config body object{region=string,service_account_json=string} true "GCP configuration"
// @Success 200 {object} models.GCPConfig
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gcp/configure [post]
// @Security BearerAuth
func (h *GCPHandlers) ConfigureGCP(c *gin.Context) {
	var req struct {
		Region             string `json:"region" binding:"required"`
		ServiceAccountJSON string `json:"service_account_json" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ProjectID, err := h.gcpService.ExtractProjectID([]byte(req.ServiceAccountJSON))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service account JSON"})
		return
	}

	gcpConfig, err := h.gcpService.ConfigureGCP(c.Request.Context(), ProjectID, req.Region, req.ServiceAccountJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gcpConfig)
}

// ConfigureGCPUpload godoc
// @Summary Configure GCP with file upload
// @Description Configure GCP by uploading service account JSON file
// @Tags gcp
// @Accept multipart/form-data
// @Produce json
// @Param region formData string true "GCP Region"
// @Param service_account_file formData file true "Service Account JSON file"
// @Success 200 {object} models.GCPConfig
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gcp/configure/upload [post]
// @Security BearerAuth
func (h *GCPHandlers) ConfigureGCPUpload(c *gin.Context) {
	region := c.PostForm("region")

	if region == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "region is required"})
		return
	}

	file, err := c.FormFile("service_account_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_account_file is required"})
		return
	}

	// Read file contents
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open file"})
		return
	}
	defer f.Close()

	serviceAccountJSON, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file contents"})
		return
	}

	projectID, err := h.gcpService.ExtractProjectID(serviceAccountJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service account JSON"})
		return
	}

	gcpConfig, err := h.gcpService.ConfigureGCP(c.Request.Context(), projectID, region, string(serviceAccountJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gcpConfig)
}

// CreateTerraformBucket godoc
// @Summary Create GCP Terraform bucket
// @Description Create a GCP bucket for storing Terraform state files
// @Tags gcp
// @Accept json
// @Produce json
// @Param config body object{project_id=string,region=string} true "GCP project ID and region"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gcp/bucket [post]
// @Security BearerAuth
func (h *GCPHandlers) CreateTerraformBucket(c *gin.Context) {
	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
		Region    string `json:"region" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get cluster name from database
	var cluster models.Cluster
	if err := h.db.First(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cluster info"})
		return
	}

	bucketName, err := h.gcpService.CreateTerraformBucket(c.Request.Context(), req.ProjectID, req.Region, cluster.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bucket_name": bucketName,
		"message":     "Terraform bucket created successfully",
	})
}

// QueryGCPInstances godoc
// @Summary Query GCP instances
// @Description Query and store GCP instances in the database
// @Tags gcp
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gcp/query-instances [post]
// @Security BearerAuth
func (h *GCPHandlers) QueryGCPInstances(c *gin.Context) {
	err := h.nodeService.QueryGCPInstances(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sample : successfully queried GCP instances"})
}

// InitInfra godoc
// @Summary Initialize Terraform infrastructure
// @Description Initialize the Terraform infrastructure on GCP
// @Tags gcp
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gcp/init-infra [post]
// @Security BearerAuth
func (h *GCPHandlers) InitInfra(c *gin.Context) {
	err := h.infrastructureService.InitializeInfrastructure(c.Request.Context(), "gcp")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Terraform infrastructure initialized successfully"})
}

// DeleteInfra godoc
// @Summary Destroy Terraform infrastructure
// @Description Destroy the Terraform infrastructure on GCP
// @Tags gcp
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gcp/delete-infra [post]
// @Security BearerAuth
func (h *GCPHandlers) DeleteInfra(c *gin.Context) {
	err := h.infrastructureService.DestroyInfrastructure(c.Request.Context(), "gcp")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Terraform infrastructure destroyed successfully"})
}

// GetGCPResources godoc
// @Summary Get available GCP resources
// @Description Get list of available zones and machine types for VM provisioning forms
// @Tags gcp
// @Accept json
// @Produce json
// @Success 200 {object} config.GCPResources
// @Failure 500 {object} map[string]string
// @Router /gcp/resources [get]
func (h *GCPHandlers) GetGCPResources(c *gin.Context) {
	resources, err := h.gcpResourcesService.GetResources()
	if err != nil {
		if err.Error() == "record not found" {
			// Return empty resources if none cached
			c.JSON(http.StatusOK, gin.H{
				"last_updated":          "",
				"zones":                 []string{},
				"machine_types_by_zone": map[string][]string{},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resources)
}

// RefreshGCPResources godoc
// @Summary Refresh GCP resources cache
// @Description Fetch and update the cached list of zones and machine types from GCP
// @Tags gcp
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /gcp/resources/refresh [post]
// @Security BearerAuth
func (h *GCPHandlers) RefreshGCPResources(c *gin.Context) {
	resources, err := h.gcpResourcesService.RefreshFromGCP(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "GCP resources refreshed successfully",
		"last_updated": resources.LastUpdated,
		"zones":        len(resources.Zones),
	})
}

// ForceUnlockTerraformState godoc
// @Summary Force unlock Terraform state
// @Description Remove a stuck Terraform state lock. WARNING: Only use when certain no operations are running
// @Tags gcp
// @Accept json
// @Produce json
// @Param body body object{lock_id=string} true "Lock ID from the error message"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gcp/terraform/force-unlock [post]
// @Security BearerAuth
func (h *GCPHandlers) ForceUnlockTerraformState(c *gin.Context) {
	var req struct {
		LockID string `json:"lock_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lock_id is required"})
		return
	}

	err := h.infrastructureService.ForceUnlockState(c.Request.Context(), "gcp", req.LockID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Terraform state lock removed successfully",
		"lock_id": req.LockID,
	})
}

// ProvisionGCPNodes godoc
// @Summary Provision GCP nodes with Talos
// @Description Create a provision request and return request_id for WebSocket connection
// @Tags gcp
// @Accept json
// @Produce json
// @Param request body models.GCPNodeProvisionRequest true "Node provision request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gcp/nodes/provision [post]
// @Security BearerAuth
func (h *GCPHandlers) ProvisionGCPNodes(c *gin.Context) {
	var req models.GCPNodeProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role
	if req.Role != "worker" && req.Role != "control-plane" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'worker' or 'control-plane'"})
		return
	}

	// Create provision request record
	requestID := uuid.New()
	requestJSON, _ := json.Marshal(req)

	provisionRequest := models.ProvisionRequest{
		ID:       requestID,
		Provider: "gcp",
		Status:   models.ProvisionStatusPending,
		Request:  requestJSON,
	}

	if err := h.db.Create(&provisionRequest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create provision request"})
		return
	}

	// Return request ID for WebSocket connection
	c.JSON(http.StatusOK, gin.H{
		"request_id": requestID.String(),
		"message":    "Provision request created. Connect to WebSocket to monitor progress.",
	})
}

// GetProvisionPlan godoc
// @Summary Download terraform plan output
// @Description Download the terraform plan output as a text file
// @Tags gcp
// @Param request_id path string true "Provision request ID"
// @Produce text/plain
// @Success 200 {file} string
// @Failure 404 {object} map[string]string
// @Router /gcp/nodes/provision/{request_id}/plan [get]
// @Security BearerAuth
func (h *GCPHandlers) GetProvisionPlan(c *gin.Context) {
	requestID := c.Param("request_id")

	// Validate request ID
	parsedUUID, err := uuid.Parse(requestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request_id"})
		return
	}
	canonicalRequestID := parsedUUID.String()

	// Check if provision request exists
	var provisionRequest models.ProvisionRequest
	if err := h.db.Where("id = ?", canonicalRequestID).First(&provisionRequest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provision request not found"})
		return
	}

	// Serve the plan file
	plansDir := "plans"
	planFileName := fmt.Sprintf("plan-%s.txt", canonicalRequestID)
	planFilePath := filepath.Join(plansDir, planFileName)

	// Defensive: make sure the resolved path is within the plans directory
	absPlansDir, err := filepath.Abs(plansDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	absPlanFilePath, err := filepath.Abs(planFilePath)
	if err != nil || len(absPlanFilePath) < len(absPlansDir) || absPlanFilePath[:len(absPlansDir)] != absPlansDir {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", planFileName))
	c.File(planFilePath)
}

// GetProvisionApply godoc
// @Summary Download terraform apply logs
// @Description Download the terraform apply JSON logs
// @Tags gcp
// @Param request_id path string true "Provision request ID"
// @Produce application/json
// @Success 200 {file} string
// @Failure 404 {object} map[string]string
// @Router /gcp/nodes/provision/{request_id}/apply [get]
// @Security BearerAuth
func (h *GCPHandlers) GetProvisionApply(c *gin.Context) {
	requestID := c.Param("request_id")

	// Validate request ID
	parsedUUID, err := uuid.Parse(requestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request_id"})
		return
	}
	canonicalRequestID := parsedUUID.String()

	// Check if provision request exists
	var provisionRequest models.ProvisionRequest
	if err := h.db.Where("id = ?", canonicalRequestID).First(&provisionRequest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provision request not found"})
		return
	}

	// Serve the apply log file
	appliesDir := "applies"
	applyFileName := fmt.Sprintf("apply-%s.json", canonicalRequestID)
	applyFilePath := filepath.Join(appliesDir, applyFileName)

	// Defensive: make sure the resolved path is within the applies directory
	absAppliesDir, err := filepath.Abs(appliesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	absApplyFilePath, err := filepath.Abs(applyFilePath)
	if err != nil || len(absApplyFilePath) < len(absAppliesDir) || absApplyFilePath[:len(absAppliesDir)] != absAppliesDir {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", applyFileName))
	c.File(applyFilePath)
}

// ProvisionGCPNodesStream godoc
// @Summary WebSocket stream for GCP node provisioning
// @Description Connect to this WebSocket endpoint to receive real-time logs and approval requests
// @Tags gcp
// @Param request_id path string true "Provision request ID"
// @Param token query string true "JWT token"
// @Router /gcp/nodes/provision/{request_id}/stream [get]
func (h *GCPHandlers) ProvisionGCPNodesStream(c *gin.Context) {
	requestID := c.Param("request_id")

	// Validate request ID
	if _, err := uuid.Parse(requestID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request_id"})
		return
	}

	// Check if provision request exists
	var provisionRequest models.ProvisionRequest
	if err := h.db.Where("id = ?", requestID).First(&provisionRequest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provision request not found"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade to websocket"})
		return
	}

	// Register WebSocket client
	client := h.wsManager.RegisterClient(requestID, conn, nil)

	// Create approval session for GCP provisioning workflow
	session := wsservices.NewApprovalSession(requestID, client)

	// Update client's session so it can route incoming messages
	client.SetSession(session)

	// Start provisioning in a goroutine
	go func() {
		// Give write pump time to start
		time.Sleep(100 * time.Millisecond)

		// Parse the provision request
		var req models.GCPNodeProvisionRequest
		if err := json.Unmarshal(provisionRequest.Request, &req); err != nil {
			session.SendErrorString(fmt.Sprintf("Failed to parse provision request: %v", err))
			session.SendStatus("failed")
			return
		}

		// Run the provisioning workflow with a background context
		// We use context.Background() instead of c.Request.Context() because the HTTP context
		// is canceled after the WebSocket upgrade completes
		requestUUID, _ := uuid.Parse(requestID)
		if err := h.provisioningService.ProvisionNodes(context.Background(), requestUUID, req, session); err != nil {
			session.SendErrorString(fmt.Sprintf("Provisioning failed: %v", err))
			session.SendStatus("failed")

			// Update provision request status
			h.db.Model(&models.ProvisionRequest{}).
				Where("id = ?", requestID).
				Update("status", models.ProvisionStatusFailed)
		}
	}()
}
