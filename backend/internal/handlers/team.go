package handlers

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/api"
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"gorm.io/gorm"
)

type TeamHandlers struct {
	db *gorm.DB
}

func NewTeamHandlers(db *gorm.DB) *TeamHandlers {
	return &TeamHandlers{db: db}
}

type CreateTeamRequest struct {
	Name string `json:"name" binding:"required"`
}

type AddUserToTeamRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// CreateTeam godoc
// @Summary Create a new team
// @Description Create a new team with the provided name
// @Tags teams
// @Accept json
// @Produce json
// @Param team body CreateTeamRequest true "Team creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams [post]
// @Security BearerAuth
func (h *TeamHandlers) CreateTeam(c *gin.Context) {
	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingTeam models.Team
	if err := h.db.First(&existingTeam, "name = ?", req.Name).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Team already exists"})
		return
	}

	team := models.Team{
		Name: req.Name,
	}

	if err := h.db.Create(&team).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create team"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"team": api.ToTeamResponse(&team, false)})
}

// GetTeams godoc
// @Summary Get list of teams
// @Description Retrieve a list of all teams. Admins see all teams, users see only their teams.
// @Tags teams
// @Accept json
// @Produce json
// @Success 200 {object} map[string][]api.TeamResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams [get]
// @Security BearerAuth
func (h *TeamHandlers) GetTeams(c *gin.Context) {
	claims, err := middleware.GetClaimsFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var teams []models.Team

	// Admin can see all teams
	if claims.Role == models.RoleAdmin {
		if err := h.db.Preload("Users").Find(&teams).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}
	} else {
		// Non-admin only see their teams
		if err := h.db.Preload("Users").Where("id IN ?", claims.Teams).Find(&teams).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}
	}

	response := make([]api.TeamResponse, len(teams))
	for i, team := range teams {
		response[i] = api.ToTeamResponse(&team, true)
	}

	c.JSON(http.StatusOK, gin.H{"teams": response})
}

// GetTeam godoc
// @Summary Get team details
// @Description Retrieve details of a specific team by ID, including its users
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "Team ID (UUID)"
// @Success 200 {object} map[string]api.TeamResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams/{id} [get]
// @Security BearerAuth
func (h *TeamHandlers) GetTeam(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	claims, err := middleware.GetClaimsFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if user has access to this team
	if claims.Role != models.RoleAdmin {
		hasAccess := slices.Contains(claims.Teams, teamID)
		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to team"})
			return
		}
	}

	var team models.Team
	if err := h.db.Preload("Users").First(&team, "id = ?", teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"team": api.ToTeamResponse(&team, true)})
}

// AddUserToTeam godoc
// @Summary Add a user to a team
// @Description Add a user to a specific team by user ID
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "Team ID (UUID)"
// @Param user body AddUserToTeamRequest true "User addition request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams/{id}/users [post]
// @Security BearerAuth
func (h *TeamHandlers) AddUserToTeam(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	var req AddUserToTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if team exists
	var team models.Team
	if err := h.db.First(&team, "id = ?", teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
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

	// Check if user is already in team
	var count int64
	h.db.Model(&models.UserTeam{}).Where("user_id = ? AND team_id = ?", userID, teamID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User is already in team"})
		return
	}

	// Add user to team
	if err := h.db.Model(&team).Association("Users").Append(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to team"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User added to team successfully"})
}

// RemoveUserFromTeam godoc
// @Summary Remove a user from a team
// @Description Remove a user from a specific team by user ID
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "Team ID (UUID)"
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams/{id}/users/{user_id} [delete]
// @Security BearerAuth
func (h *TeamHandlers) RemoveUserFromTeam(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if team exists
	var team models.Team
	if err := h.db.First(&team, "id = ?", teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
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

	// Remove user from team
	if err := h.db.Model(&team).Association("Users").Delete(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from team"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User removed from team successfully"})
}

// DeleteTeam godoc
// @Summary Delete a team
// @Description Delete a specific team by ID. Team must have no associated deployments.
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "Team ID (UUID)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams/{id} [delete]
// @Security BearerAuth
func (h *TeamHandlers) DeleteTeam(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	var team models.Team
	if err := h.db.First(&team, "id = ?", teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if team has any deployments. This can be revised but well see how we want to handle this later.
	var deploymentCount int64
	h.db.Model(&models.Deployment{}).Where("team_id = ?", teamID).Count(&deploymentCount)
	if deploymentCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete team with existing deployments"})
		return
	}

	if err := h.db.Delete(&team).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete team"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team deleted successfully"})
}
