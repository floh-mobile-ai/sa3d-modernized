package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

// Logger middleware for request logging
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		entry := logger.WithFields(logrus.Fields{
			"status_code":  statusCode,
			"latency":      latency,
			"client_ip":    clientIP,
			"method":       method,
			"path":         path,
			"request_id":   c.GetString("request_id"),
			"user_id":      c.GetString("user_id"),
		})

		if errorMessage != "" {
			entry.Error(errorMessage)
		} else if statusCode >= 500 {
			entry.Error("Internal server error")
		} else if statusCode >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request processed")
		}
	}
}

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// CORS middleware for Cross-Origin Resource Sharing
func CORS(config struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	MaxAge         int      `mapstructure:"max_age"`
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range config.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimiter middleware for rate limiting
func RateLimiter(limiter *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Auth middleware for JWT authentication
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token
		tokenString := ""
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString = parts[1]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Set user information in context
			if userID, ok := claims["user_id"].(string); ok {
				c.Set("user_id", userID)
			}
			if email, ok := claims["email"].(string); ok {
				c.Set("email", email)
			}
			if roles, ok := claims["roles"].([]interface{}); ok {
				c.Set("roles", roles)
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token claims",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Tracing middleware for distributed tracing
func Tracing(tracer trace.Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract trace context from headers
		ctx := c.Request.Context()
		
		// Start a new span
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		// Add attributes to span
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.target", c.Request.URL.Path),
			attribute.String("http.host", c.Request.Host),
			attribute.String("http.scheme", c.Request.URL.Scheme),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("request_id", c.GetString("request_id")),
		)

		// Update request context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Add response attributes
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.Int64("http.response_size", int64(c.Writer.Size())),
		)

		// Record error if any
		if len(c.Errors) > 0 {
			span.RecordError(c.Errors.Last())
		}
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "No roles found",
			})
			c.Abort()
			return
		}

		// Check if user has required role
		hasRole := false
		if roleList, ok := roles.([]interface{}); ok {
			for _, role := range roleList {
				if roleStr, ok := role.(string); ok && roleStr == requiredRole {
					hasRole = true
					break
				}
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("Required role '%s' not found", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Timeout middleware adds a timeout to requests
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Update request context
		c.Request = c.Request.WithContext(ctx)

		// Create a channel to signal completion
		finished := make(chan struct{})

		// Process request in goroutine
		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		// Wait for either completion or timeout
		select {
		case <-finished:
			// Request completed successfully
		case <-ctx.Done():
			// Timeout occurred
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error": "Request timeout",
			})
			c.Abort()
		}
	}
}