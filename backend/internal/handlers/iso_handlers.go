package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) generateISO(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Generate ISO - TODO"})
}