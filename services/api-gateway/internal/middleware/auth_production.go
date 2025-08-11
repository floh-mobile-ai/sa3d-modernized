package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/sa3d-modernized/sa3d/shared/services"
)

// ProductionAuth creates a production authentication middleware
func ProductionAuth(authService *services.AuthService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Check for Bearer token format
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			logger.Warn("Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must use Bearer token"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, bearerPrefix)
		if token == "" {
			logger.Warn("Empty token in Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is required"})
			c.Abort()
			return
		}

		// Validate token and get user
		user, err := authService.ValidateToken(token)
		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"token_prefix": token[:min(10, len(token))] + "...",
				"ip_address":   c.ClientIP(),
			}).Warn("Token validation failed")

			switch err {
			case services.ErrInvalidToken:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			case services.ErrTokenExpired:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			case services.ErrAccountNotActive:
				c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
			default:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
			}
			c.Abort()
			return
		}

		// Set user context in Gin context
		c.Set("user_id", user.ID.String())
		c.Set("user", user)
		c.Set("email", user.Email)
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Set("session_token", token)

		// Log successful authentication
		logger.WithFields(logrus.Fields{
			"user_id":    user.ID,
			"email":      user.Email,
			"role":       user.Role,
			"ip_address": c.ClientIP(),
		}).Debug("User authenticated successfully")

		c.Next()
	}
}

// ProductionRequireRole creates middleware that requires specific user roles
func ProductionRequireRole(authService *services.AuthService, logger *logrus.Logger, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First run authentication
		ProductionAuth(authService, logger)(c)
		if c.IsAborted() {
			return
		}

		// Check user role
		userRole := c.GetString("role")
		if userRole == "" {
			logger.Error("User role not found in context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization failed"})
			c.Abort()
			return
		}

		// Check if user role is allowed
		roleAllowed := false
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			logger.WithFields(logrus.Fields{
				"user_id":       c.GetString("user_id"),
				"user_role":     userRole,
				"allowed_roles": allowedRoles,
			}).Warn("User role not authorized for this endpoint")

			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ProductionRequireAdmin creates middleware that requires admin role
func ProductionRequireAdmin(authService *services.AuthService, logger *logrus.Logger) gin.HandlerFunc {
	return ProductionRequireRole(authService, logger, "admin", "super_admin")
}

// ProductionOptionalAuth creates middleware that optionally authenticates users
// If authentication fails, the request continues without user context
func ProductionOptionalAuth(authService *services.AuthService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No authorization header, continue without authentication
			c.Next()
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			// Invalid format, continue without authentication
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, bearerPrefix)
		if token == "" {
			// Empty token, continue without authentication
			c.Next()
			return
		}

		// Try to validate token
		user, err := authService.ValidateToken(token)
		if err != nil {
			// Token validation failed, continue without authentication
			logger.WithError(err).Debug("Optional authentication failed, continuing without user context")
			c.Next()
			return
		}

		// Set user context if authentication succeeded
		c.Set("user_id", user.ID.String())
		c.Set("user", user)
		c.Set("email", user.Email)
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Set("session_token", token)

		logger.WithField("user_id", user.ID).Debug("User optionally authenticated")
		c.Next()
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}