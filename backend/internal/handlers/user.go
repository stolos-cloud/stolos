package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/api"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"gorm.io/gorm"
)

type UserHandlers struct {
	db *gorm.DB
}

func NewUserHandlers(db *gorm.DB) *UserHandlers {
	return &UserHandlers{db: db}
}

type UpdateUserRoleRequest struct {
	Role models.Role `json:"role" binding:"required"`
}

func (h *UserHandlers) ListUsers(c *gin.Context) {
	var users []models.User
	if err := h.db.Preload("Teams").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	response := make([]api.UserResponse, len(users))
	for i, user := range users {
		response[i] = api.ToUserResponse(&user)
	}

	c.JSON(http.StatusOK, gin.H{"users": response})
}

func (h *UserHandlers) GetUser(c *gin.Context) {
	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := h.db.Preload("Teams").First(&user, "id = ?", userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": api.ToUserResponse(&user)})
}

func (h *UserHandlers) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	user.Role = req.Role
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		return
	}

	// Reload user with teams
	h.db.Preload("Teams").First(&user, user.ID)

	c.JSON(http.StatusOK, gin.H{"user": api.ToUserResponse(&user)})
}

func (h *UserHandlers) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Remove user from all teams
	if err := h.db.Model(&user).Association("Teams").Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from teams"})
		return
	}

	if err := h.db.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
