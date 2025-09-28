package handlers

import (
	"net/http"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/config"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/services"
	"github.com/gin-gonic/gin"
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

	if status == "pending" {
		if err := h.nodeService.CreateSamplePendingNodes(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		nodes, err := h.nodeService.ListPendingNodes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, nodes)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "List nodes - TODO"})
}

func (h *NodeHandlers) CreateNodes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create node - TODO"})
}

func (h *NodeHandlers) GetNode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get node - TODO"})
}