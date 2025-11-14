package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	"github.com/stolos-cloud/stolos/backend/internal/services/k8s"
	"gorm.io/gorm"
)

// ScaffoldsHandler manages scaffold-related API endpoints.
type ScaffoldsHandler struct {
	k8sClient *k8s.K8sClient
	gitOps    *gitops.GitOpsService
	db        *gorm.DB
}

func NewScaffoldsHandler(k8s *k8s.K8sClient, gitOps *gitops.GitOpsService, db *gorm.DB) *ScaffoldsHandler {
	return &ScaffoldsHandler{
		k8sClient: k8s,
		gitOps:    gitOps,
		db:        db,
	}
}

// GetScaffoldsList godoc
// @Summary Get template scaffolds list
// @Description returns a list of available template scaffolds in the GitOps repository
// @Tags scaffolds
// @Produce json
// @Success 200 {object} []*github.RepositoryContent
// @Failure 500 {object} string "error"
// @Router /scaffolds [get]
// @Security BearerAuth
func (h *ScaffoldsHandler) GetScaffoldsList(c *gin.Context) {

	bob, _ := h.gitOps.GetTemplateScaffolds()

	c.JSON(http.StatusOK, bob)
}

// Future: Adding, removing, etc of Templates Scaffolds
