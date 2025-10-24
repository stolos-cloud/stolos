package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"gorm.io/gorm"
)

// GetClusterInfo returns general cluster information for the dashboard
// @Summary Get cluster info
// @Description Get general cluster information including name and GitOps repository
// @Tags cluster
// @Produce json
// @Success 200 {object} map[string]interface{} "cluster_name, gitops_repo_owner, gitops_repo_name, gitops_branch"
// @Failure 500 {object} map[string]string "error"
// @Router /cluster/info [get]
// @Security BearerAuth
func (h *Handlers) GetClusterInfo(c *gin.Context) {
	// Get cluster from database
	var cluster models.Cluster
	if err := h.db.First(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cluster info"})
		return
	}

	// Get GitOps config from database
	var gitopsConfig models.GitOpsConfig
	err := h.db.Where("is_configured = ?", true).First(&gitopsConfig).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get GitOps config"})
		return
	}

	if err == gorm.ErrRecordNotFound {
		// GitOps not configured
		c.JSON(http.StatusOK, gin.H{
			"cluster_name":      cluster.Name,
			"gitops_configured": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster_name":       cluster.Name,
		"gitops_configured":  true,
		"gitops_repo_owner":  gitopsConfig.RepoOwner,
		"gitops_repo_name":   gitopsConfig.RepoName,
		"gitops_branch":      gitopsConfig.Branch,
		"gitops_working_dir": gitopsConfig.WorkingDir,
	})
}
