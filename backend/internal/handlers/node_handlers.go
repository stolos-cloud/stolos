package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) listNodes(c *gin.Context) {
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

func (h *Handlers) createNodes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create node - TODO"})
}

func (h *Handlers) getNode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get node - TODO"})
}