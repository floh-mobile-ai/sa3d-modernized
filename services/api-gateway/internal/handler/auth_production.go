package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/sa3d-modernized/sa3d/shared/services"
)

// parseUUID parses a string to UUID
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ProductionAuthHandler handles authentication endpoints using database
type ProductionAuthHandler struct {
	authService *services.AuthService
	logger      *logrus.Logger
}

// NewProductionAuthHandler creates a new production auth handler
func NewProductionAuthHandler(authService *services.AuthService, logger *logrus.Logger) *ProductionAuthHandler {
	return &ProductionAuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register handles user registration
func (h *ProductionAuthHandler) Register(c *gin.Context) {
	var registration services.UserRegistration
	if err := c.ShouldBindJSON(&registration); err != nil {
		h.logger.WithError(err).Warn("Invalid registration request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	user, err := h.authService.Register(registration)
	if err != nil {
		h.logger.WithError(err).WithField("email", registration.Email).Error("Registration failed")
		
		switch err {
		case services.ErrUserAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		case services.ErrWeakPassword:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password does not meet security requirements"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		}
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User registered successfully")

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user,
	})
}

// Login handles user login
func (h *ProductionAuthHandler) Login(c *gin.Context) {
	var credentials services.UserLogin
	if err := c.ShouldBindJSON(&credentials); err != nil {
		h.logger.WithError(err).Warn("Invalid login request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Set client information for security logging
	credentials.IPAddress = c.ClientIP()
	credentials.UserAgent = c.GetHeader("User-Agent")

	result, err := h.authService.Login(credentials)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"email":      credentials.Email,
			"ip_address": credentials.IPAddress,
		}).Warn("Login failed")

		switch err {
		case services.ErrUserNotFound, services.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		case services.ErrAccountLocked:
			c.JSON(http.StatusLocked, gin.H{"error": "Account is locked due to too many failed login attempts"})
		case services.ErrAccountNotActive:
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
		case services.ErrAccountNotVerified:
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not verified"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		}
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    result.User.ID,
		"email":      result.User.Email,
		"ip_address": credentials.IPAddress,
	}).Info("User logged in successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":       "Login successful",
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_at":    result.ExpiresAt,
		"user":          result.User,
	})
}

// RefreshToken handles token refresh
func (h *ProductionAuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid refresh token request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	result, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		h.logger.WithError(err).Warn("Token refresh failed")

		switch err {
		case services.ErrInvalidToken, services.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		case services.ErrAccountNotActive:
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token refresh failed"})
		}
		return
	}

	h.logger.WithField("user_id", result.User.ID).Info("Token refreshed successfully")

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_at":    result.ExpiresAt,
		"user":          result.User,
	})
}

// Logout handles user logout
func (h *ProductionAuthHandler) Logout(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionToken := c.GetString("session_token")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
		return
	}

	if sessionToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session token not found"})
		return
	}

	// Parse userID to UUID
	userUUID, err := parseUUID(userID)
	if err != nil {
		h.logger.WithError(err).Error("Invalid user ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.authService.Logout(userUUID, sessionToken); err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Logout failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Logout failed"})
		return
	}

	h.logger.WithField("user_id", userID).Info("User logged out successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ValidateToken validates a token and returns user info
func (h *ProductionAuthHandler) ValidateToken(c *gin.Context) {
	// Token is already validated by middleware
	userID := c.GetString("user_id")
	email := c.GetString("email")
	role := c.GetString("role")

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"user_id": userID,
		"email":   email,
		"role":    role,
	})
}

// GetProfile returns the current user's profile
func (h *ProductionAuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse userID to UUID
	userUUID, err := parseUUID(userID)
	if err != nil {
		h.logger.WithError(err).Error("Invalid user ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.authService.GetUserByID(userUUID)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user profile")
		
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user profile"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// ChangePassword handles password change
func (h *ProductionAuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid change password request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// TODO: Implement password change functionality
	// This would involve:
	// 1. Validating current password
	// 2. Checking new password strength
	// 3. Updating password hash
	// 4. Invalidating existing sessions
	
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Password change not implemented yet"})
}