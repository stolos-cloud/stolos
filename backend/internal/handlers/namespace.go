package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/api"
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	gitopsservices "github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	"github.com/stolos-cloud/stolos/backend/internal/services/k8s"
	"gorm.io/gorm"
)

type NamespaceHandlers struct {
	db            *gorm.DB
	gitopsService *gitopsservices.GitOpsService
	k8sClient     *k8s.K8sClient
}

func NewNamespaceHandlers(db *gorm.DB, gitopsService *gitopsservices.GitOpsService, k8sClient *k8s.K8sClient) *NamespaceHandlers {
	return &NamespaceHandlers{
		db:            db,
		gitopsService: gitopsService,
		k8sClient:     k8sClient,
	}
}

type CreateNamespaceRequest struct {
	Name string `json:"name" binding:"required"`
}

type AddUserToNamespaceRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// CreateNamespace godoc
// @Summary Create a new namespace
// @Description Create a new namespace with the provided name and initialize GitOps configuration
// @Tags namespaces
// @Accept json
// @Produce json
// @Param namespace body CreateNamespaceRequest true "Namespace creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /namespaces [post]
// @Security BearerAuth
func (h *NamespaceHandlers) CreateNamespace(c *gin.Context) {
	var req CreateNamespaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var namespaceRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	maxSize := 63 - len(k8s.K8sNamespacePrefix)
	if len(req.Name) < 1 || len(req.Name) > maxSize || !namespaceRegex.MatchString(req.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Name between 1-%d characters and can only be alphanumeric characters or \"-\"", maxSize)})
		return
	}

	fullName := k8s.K8sNamespacePrefix + req.Name

	var existingNamespace models.Namespace
	if err := h.db.First(&existingNamespace, "name = ?", fullName).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Namespace already exists"})
		return
	}

	// Get the authenticated user
	user, err := middleware.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	namespace := models.Namespace{
		Name: fullName,
	}

	if err := h.db.Create(&namespace).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create namespace"})
		return
	}

	// Add the creator to the namespace
	if err := h.db.Model(&namespace).Association("Users").Append(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to namespace"})
		return
	}

	// Create the actual Kubernetes namespace
	if err := h.k8sClient.CreateNamespace(context.Background(), fullName); err != nil {
		fmt.Printf("Warning: Failed to create Kubernetes namespace %s: %v\n", fullName, err)
	}

	// Create GitOps manifests for the namespace
	if err := h.gitopsService.CreateNamespaceDirectory(context.Background(), fullName); err != nil {
		fmt.Printf("Warning: Failed to create GitOps manifests for namespace %s: %v\n", req.Name, err)
	}

	c.JSON(http.StatusCreated, gin.H{"namespace": api.ToNamespaceResponse(&namespace, false)})
}

// GetNamespaces godoc
// @Summary Get list of namespaces
// @Description Retrieve a list of all namespaces. Admins see all namespaces, developers see only their namespaces.
// @Tags namespaces
// @Accept json
// @Produce json
// @Success 200 {object} map[string][]api.NamespaceResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /namespaces [get]
// @Security BearerAuth
func (h *NamespaceHandlers) GetNamespaces(c *gin.Context) {
	claims, err := middleware.GetClaimsFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var namespaces []models.Namespace

	// Admin can see all namespaces
	if claims.Role == models.RoleAdmin {
		if err := h.db.Preload("Users").Find(&namespaces).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch namespaces"})
			return
		}
	} else {
		// Non-admin only see their namespaces
		if err := h.db.Preload("Users").
			Joins("JOIN user_namespaces ON user_namespaces.namespace_id = namespaces.id").
			Where("user_namespaces.user_id = ?", claims.UserID).
			Find(&namespaces).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch namespaces"})
			return
		}
	}

	response := make([]api.NamespaceResponse, len(namespaces))
	for i, ns := range namespaces {
		response[i] = api.ToNamespaceResponse(&ns, true)
	}

	c.JSON(http.StatusOK, gin.H{"namespaces": response})
}

// GetNamespace godoc
// @Summary Get namespace details
// @Description Retrieve details of a specific namespace by ID, including its users
// @Tags namespaces
// @Accept json
// @Produce json
// @Param id path string true "Namespace ID (UUID)"
// @Success 200 {object} map[string]api.NamespaceResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /namespaces/{id} [get]
// @Security BearerAuth
func (h *NamespaceHandlers) GetNamespace(c *gin.Context) {
	namespaceIDStr := c.Param("id")
	namespaceID, err := uuid.Parse(namespaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid namespace ID"})
		return
	}

	claims, err := middleware.GetClaimsFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var namespace models.Namespace

	if claims.Role == models.RoleAdmin {
		if err := h.db.Preload("Users").First(&namespace, "id = ?", namespaceID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
	} else {
		if err := h.db.Preload("Users").
			Joins("JOIN user_namespaces ON user_namespaces.namespace_id = namespaces.id").
			Where("namespaces.id = ? AND user_namespaces.user_id = ?", namespaceID, claims.UserID).
			First(&namespace).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"namespace": api.ToNamespaceResponse(&namespace, true)})
}

// AddUserToNamespace godoc
// @Summary Add a user to a namespace
// @Description Add a user to a specific namespace by user ID
// @Tags namespaces
// @Accept json
// @Produce json
// @Param id path string true "Namespace ID (UUID)"
// @Param user body AddUserToNamespaceRequest true "User addition request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /namespaces/{id}/users [post]
// @Security BearerAuth
func (h *NamespaceHandlers) AddUserToNamespace(c *gin.Context) {
	namespaceIDStr := c.Param("id")
	namespaceID, err := uuid.Parse(namespaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid namespace ID"})
		return
	}

	var req AddUserToNamespaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get the authenticated user
	authUser, err := middleware.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if namespace exists and load users
	var namespace models.Namespace
	if err := h.db.Preload("Users").First(&namespace, "id = ?", namespaceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if user has permission to add users (admin or namespace member)
	if authUser.Role != models.RoleAdmin {
		isMember := false
		for _, nsUser := range namespace.Users {
			if nsUser.ID == authUser.ID {
				isMember = true
				break
			}
		}
		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to namespace"})
			return
		}
	}

	// Check if user exists
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if user is already in namespace
	var count int64
	h.db.Model(&models.UserNamespace{}).Where("user_id = ? AND namespace_id = ?", userID, namespaceID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User is already in namespace"})
		return
	}

	// Add user to namespace
	if err := h.db.Model(&namespace).Association("Users").Append(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to namespace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User added to namespace successfully"})
}

// RemoveUserFromNamespace godoc
// @Summary Remove a user from a namespace
// @Description Remove a user from a specific namespace by user ID
// @Tags namespaces
// @Accept json
// @Produce json
// @Param id path string true "Namespace ID (UUID)"
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /namespaces/{id}/users/{user_id} [delete]
// @Security BearerAuth
func (h *NamespaceHandlers) RemoveUserFromNamespace(c *gin.Context) {
	namespaceIDStr := c.Param("id")
	namespaceID, err := uuid.Parse(namespaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid namespace ID"})
		return
	}

	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get the authenticated user
	authUser, err := middleware.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if namespace exists and load users
	var namespace models.Namespace
	if err := h.db.Preload("Users").First(&namespace, "id = ?", namespaceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if user has permission to remove users (admin or namespace member)
	if authUser.Role != models.RoleAdmin {
		isMember := false
		for _, nsUser := range namespace.Users {
			if nsUser.ID == authUser.ID {
				isMember = true
				break
			}
		}
		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to namespace"})
			return
		}
	}

	// Check if user exists
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Remove user from namespace
	if err := h.db.Model(&namespace).Association("Users").Delete(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from namespace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User removed from namespace successfully"})
}

// DeleteNamespace godoc
// @Summary Delete a namespace
// @Description Delete a specific namespace by ID. Namespace must have no associated deployments.
// @Tags namespaces
// @Accept json
// @Produce json
// @Param id path string true "Namespace ID (UUID)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /namespaces/{id} [delete]
// @Security BearerAuth
func (h *NamespaceHandlers) DeleteNamespace(c *gin.Context) {
	namespaceIDStr := c.Param("id")
	namespaceID, err := uuid.Parse(namespaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid namespace ID"})
		return
	}

	// Get the authenticated user
	user, err := middleware.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var namespace models.Namespace
	if err := h.db.Preload("Users").First(&namespace, "id = ?", namespaceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if user has permission to delete (admin or namespace member)
	if user.Role != models.RoleAdmin {
		isMember := false
		for _, nsUser := range namespace.Users {
			if nsUser.ID == user.ID {
				isMember = true
				break
			}
		}
		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to namespace"})
			return
		}
	}

	// Check if namespace has any deployments
	var deploymentCount int64
	h.db.Model(&models.Deployment{}).Where("namespace_id = ?", namespaceID).Count(&deploymentCount)
	if deploymentCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete namespace with existing deployments"})
		return
	}

	if err := h.db.Delete(&namespace).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete namespace"})
		return
	}

	// Delete the actual Kubernetes namespace
	if err := h.k8sClient.DeleteNamespace(context.Background(), namespace.Name); err != nil {
		fmt.Printf("Warning: Failed to delete Kubernetes namespace %s: %v\n", namespace.Name, err)
	}

	// Delete GitOps manifests for the namespace
	if err := h.gitopsService.DeleteNamespaceManifests(context.Background(), namespace.Name); err != nil {
		fmt.Printf("Warning: Failed to delete GitOps manifests for namespace %s: %v\n", namespace.Name, err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Namespace deleted successfully"})
}
