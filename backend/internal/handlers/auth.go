package handlers

import (
	"net/http"
	"strings"

	"github.com/etsmtl-pfe-cloudnative/backend/internal/api"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/middleware"
	"github.com/etsmtl-pfe-cloudnative/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandlers struct {
	db         *gorm.DB
	jwtService *middleware.JWTService
}

func NewAuthHandlers(db *gorm.DB, jwtService *middleware.JWTService) *AuthHandlers {
	return &AuthHandlers{
		db:         db,
		jwtService: jwtService,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email    string      `json:"email" binding:"required,email"`
	Password string      `json:"password" binding:"required,min=8"`
	Role     models.Role `json:"role,omitempty"`
}

type AuthResponse struct {
	Token string           `json:"token"`
	User  api.UserResponse `json:"user"`
}


func (h *AuthHandlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Preload("Teams").First(&user, "email = ?", strings.ToLower(req.Email)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := user.CheckPassword(req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := h.jwtService.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	response := AuthResponse{
		Token: token,
		User:  api.ToUserResponse(&user),
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandlers) RefreshToken(c *gin.Context) {
	user, err := middleware.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	// Reload user with teams to get latest team memberships
	h.db.Preload("Teams").First(user, user.ID)

	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	response := AuthResponse{
		Token: token,
		User:  api.ToUserResponse(user),
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandlers) GetProfile(c *gin.Context) {
	user, err := middleware.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": api.ToUserResponse(user)})
}

// Admin-only endpoint to create users with specific roles
func (h *AuthHandlers) CreateUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.db.First(&existingUser, "email = ?", strings.ToLower(req.Email)).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	user := models.User{
		Email: strings.ToLower(req.Email),
		Role:  req.Role,
	}

	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Reload user with teams
	h.db.Preload("Teams").First(&user, user.ID)

	c.JSON(http.StatusCreated, gin.H{"user": api.ToUserResponse(&user)})
}

