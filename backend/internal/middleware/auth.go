package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"gorm.io/gorm"
)

type Claims struct {
	UserID     uuid.UUID   `json:"user_id"`
	Email      string      `json:"email"`
	Role       models.Role `json:"role"`
	Namespaces []uuid.UUID `json:"namespaces"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey     string
	issuer        string
	expiryMinutes int
}

func NewJWTService(cfg *config.Config) *JWTService {
	return &JWTService{
		secretKey:     cfg.JWT.SecretKey,
		issuer:        cfg.JWT.Issuer,
		expiryMinutes: cfg.JWT.ExpiryMinutes,
	}
}

func (j *JWTService) GenerateToken(user *models.User) (string, error) {
	namespaceIDs := make([]uuid.UUID, len(user.Namespaces))
	for i, ns := range user.Namespaces {
		namespaceIDs[i] = ns.ID
	}

	claims := Claims{
		UserID:     user.ID,
		Email:      user.Email,
		Role:       user.Role,
		Namespaces: namespaceIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(j.expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func JWTAuthMiddleware(jwtService *JWTService, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.Preload("Namespaces").First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		c.Set("user", &user)
		c.Set("claims", claims)
		c.Next()
	}
}

func RequireRole(role models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := GetUserFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Admin can access everything
		if user.Role == models.RoleAdmin {
			c.Next()
			return
		}

		// Check if user has required role
		if user.Role != role {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func RequireNamespaceAccess(namespaceID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := GetUserFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Admin can access everything
		if user.Role == models.RoleAdmin {
			c.Next()
			return
		}

		// Check if user belongs to the namespace
		for _, ns := range user.Namespaces {
			if ns.ID == namespaceID {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to namespace resources"})
		c.Abort()
	}
}

func GetUserFromContext(c *gin.Context) (*models.User, error) {
	user, exists := c.Get("user")
	if !exists {
		return nil, fmt.Errorf("user not found in context")
	}

	userModel, ok := user.(*models.User)
	if !ok {
		return nil, fmt.Errorf("invalid user type in context")
	}

	return userModel, nil
}

func GetClaimsFromContext(c *gin.Context) (*Claims, error) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, fmt.Errorf("claims not found in context")
	}

	claimsModel, ok := claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type in context")
	}

	return claimsModel, nil
}
