package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	redis         *redis.Client
	jwtSecret     string
	tokenDuration time.Duration
	logger        *logrus.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(redis *redis.Client, jwtSecret string, tokenDuration time.Duration, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		redis:         redis,
		jwtSecret:     jwtSecret,
		tokenDuration: tokenDuration,
		logger:        logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}

// User represents a user
type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: In production, fetch user from database
	// For now, using a mock user
	user := &User{
		ID:    uuid.New().String(),
		Email: req.Email,
		Name:  "Test User",
		Roles: []string{"user"},
	}

	// TODO: Verify password against stored hash
	// For now, accepting any password
	
	// Generate tokens
	token, expiresAt, err := h.generateToken(user)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken := uuid.New().String()
	
	// Store refresh token in Redis
	ctx := context.Background()
	err = h.redis.Set(ctx, "refresh:"+refreshToken, user.ID, 7*24*time.Hour).Err()
	if err != nil {
		h.logger.WithError(err).Error("Failed to store refresh token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store refresh token"})
		return
	}

	// Store user session
	sessionKey := "session:" + user.ID
	err = h.redis.HSet(ctx, sessionKey, map[string]interface{}{
		"email": user.Email,
		"name":  user.Name,
		"roles": user.Roles[0], // Simplified for now
	}).Err()
	if err != nil {
		h.logger.WithError(err).Error("Failed to store session")
	}
	h.redis.Expire(ctx, sessionKey, h.tokenDuration)

	c.JSON(http.StatusOK, LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         *user,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
		return
	}

	// Remove session from Redis
	ctx := context.Background()
	sessionKey := "session:" + userID
	err := h.redis.Del(ctx, sessionKey).Err()
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete session")
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from refresh token
	ctx := context.Background()
	userID, err := h.redis.Get(ctx, "refresh:"+req.RefreshToken).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		} else {
			h.logger.WithError(err).Error("Failed to get refresh token")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate refresh token"})
		}
		return
	}

	// Get user session
	sessionKey := "session:" + userID
	userData, err := h.redis.HGetAll(ctx, sessionKey).Result()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user session"})
		return
	}

	// Create user object
	user := &User{
		ID:    userID,
		Email: userData["email"],
		Name:  userData["name"],
		Roles: []string{userData["roles"]},
	}

	// Generate new token
	token, expiresAt, err := h.generateToken(user)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
	})
}

// ValidateToken validates a token
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	// Token is already validated by middleware
	userID := c.GetString("user_id")
	email := c.GetString("email")
	
	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"user_id": userID,
		"email":   email,
	})
}

// generateToken generates a JWT token for a user
func (h *AuthHandler) generateToken(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(h.tokenDuration)
	
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"roles":   user.Roles,
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPassword checks if a password matches a hash
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}