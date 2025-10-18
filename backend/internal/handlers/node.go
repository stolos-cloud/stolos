package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	talos "github.com/stolos-cloud/stolos/backend/internal/services/talos"
	"gorm.io/gorm"
)

type NodeHandlers struct {
	db           *gorm.DB
	nodeService  *services.NodeService
	talosService *talos.TalosService
}

func NewNodeHandlers(db *gorm.DB, cfg *config.Config, providerManager *services.ProviderManager, talosService *talos.TalosService) *NodeHandlers {
	return &NodeHandlers{
		db:           db,
		nodeService:  services.NewNodeService(db, cfg, providerManager, talosService),
		talosService: talos.NewTalosService(db, cfg),
	}
}

// ListNodes godoc
// @Summary List nodes
// @Description Get list of nodes with optional status filter
// @Tags nodes
// @Accept json
// @Produce json
// @Param status query string false "Node status filter (pending, active, failed)"
// @Success 200 {array} models.Node
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /nodes [get]
func (h *NodeHandlers) ListNodes(c *gin.Context) {
	status := c.Query("status")

	if status != "" && !models.ValidNodeStatuses[models.NodeStatus(status)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status value. Must be one of: pending, active, failed"})
		return
	}

	nodes, err := h.nodeService.ListNodes(status, 0, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

func (h *NodeHandlers) CreateNodes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create node - TODO"})
}

func (h *NodeHandlers) GetNode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get node - TODO"})
}

// UpdateActiveNodeConfig godoc
// @Summary Update active node configuration
// @Description Update a single active node's role and labels
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path string true "Node ID (UUID)"
// @Param request body object{role=string,labels=[]string} true "Node configuration"
// @Success 200 {object} models.Node
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /nodes/{id}/config [put]
func (h *NodeHandlers) UpdateActiveNodeConfig(c *gin.Context) {
	var req struct {
		Role   string   `json:"role" binding:"required"`
		Labels []string `json:"labels"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idParam := c.Param("id")
	nodeID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node ID"})
		return
	}

	node, err := h.nodeService.UpdateActiveNodeConfig(nodeID, req.Role, req.Labels)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, node)
}

// UpdateActiveNodesConfig godoc
// @Summary Update multiple nodes labels
// @Description Update labels for multiple nodes (for active nodes only, does not change role)
// @Tags nodes
// @Accept json
// @Produce json
// @Param request body object{nodes=[]services.NodeConfigUpdate} true "Array of node label updates"
// @Success 200 {object} map[string]interface{} "Returns updated count and nodes array"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /nodes/config [put]
func (h *NodeHandlers) UpdateActiveNodesConfig(c *gin.Context) {
	var req struct {
		Nodes []services.NodeConfigUpdate `json:"nodes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: After updating labels in DB, also update Talos node config
	// This requires:
	// 1. Getting current Talos config bundle
	// 2. Applying label patches to node configs
	// 3. Re-applying configs to nodes via Talos API

	nodes, err := h.nodeService.UpdateActiveNodesConfig(req.Nodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated": len(nodes),
		"nodes":   nodes,
	})
}

// CreateSampleNodes godoc
// @Summary Create samples pending nodes
// @Description Create samples pending nodes for testing purposes
// @Tags nodes
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Message indicating success"
// @Failure 500 {object} map[string]string
// @Router /nodes/samples [post]
func (h *NodeHandlers) CreateSampleNodes(c *gin.Context) {
	err := h.nodeService.CreateSamplePendingNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Sample nodes created"})
}

// ProvisionNodes godoc
// @Summary Provision multiple on-prem nodes
// @Description Apply Talos configuration to multiple pending nodes and add them to the cluster
// @Tags nodes
// @Accept json
// @Produce json
// @Param request body models.NodeProvisionRequest true "Array of nodes to provision with role and labels"
// @Success 200 {object} map[string]interface{} "Returns provisioned count and nodes array"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /nodes/provision [post]
// @Security BearerAuth
func (h *NodeHandlers) ProvisionNodes(c *gin.Context) {
	var req models.NodeProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if len(req.Nodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one node must be provided"})
		return
	}

	nodes, err := h.nodeService.ProvisionNodes(req.Nodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Nodes provisioned successfully",
		"provisioned": len(nodes),
		"nodes":       nodes,
	})
}

// GetTalosconfig godoc
// @Summary Returns talosconfig File in TALOS_FOLDER
// @Description Returns the talosconfig file in TALOS_FOLDER, destined for operators to do manual talosctl operations.
// @Tags nodes
// @Accept json
// @Produce yaml
// @Success 200 {object} []byte "Message indicating success"
// @Failure 500 {object}
// @Router /nodes/talosconfig [get]
//func (h *NodeHandlers) GetTalosconfig(c *gin.Context) {
//	err := h.nodeService.GetTalosconfig()
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//		return
//	}
//	c.JSON(http.StatusOK, gin.H{"message": "Sample nodes created"})
//}
