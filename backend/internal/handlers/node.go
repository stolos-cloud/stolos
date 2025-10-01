package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/internal/services"
	"gorm.io/gorm"
)

type NodeHandlers struct {
	db          *gorm.DB
	nodeService *services.NodeService
}

func NewNodeHandlers(db *gorm.DB, cfg *config.Config) *NodeHandlers {
	return &NodeHandlers{
		db:          db,
		nodeService: services.NewNodeService(db, cfg),
	}
}

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

func (h *NodeHandlers) UpdateNodeConfig(c *gin.Context) {
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

	node, err := h.nodeService.UpdateNodeConfig(nodeID, req.Role, req.Labels)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, node)
}

func (h *NodeHandlers) UpdateNodesConfig(c *gin.Context) {
	var req struct {
		Nodes []services.NodeConfigUpdate `json:"nodes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nodes, err := h.nodeService.UpdateNodesConfig(req.Nodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated": len(nodes),
		"nodes":   nodes,
	})
}

func (h *NodeHandlers) CreateSampleNodes(c *gin.Context) {
	err := h.nodeService.CreateSamplePendingNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Sample nodes created"})
}