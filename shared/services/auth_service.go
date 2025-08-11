package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sa3d-modernized/sa3d/shared/models"
	"github.com/sa3d-modernized/sa3d/shared/utils"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrAccountLocked        = errors.New("account is locked due to too many failed login attempts")
	ErrAccountNotActive     = errors.New("account is not active")
	ErrAccountNotVerified   = errors.New("account is not verified")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrInvalidToken         = errors.New("invalid token")
	ErrTokenExpired         = errors.New("token has expired")
	ErrWeakPassword         = errors.New("password does not meet security requirements")
)

// AuthService handles user authentication and management
type AuthService struct {
	db     *DatabaseService
	logger *logrus.Logger
}

// LoginAttempt represents a login attempt record
type LoginAttempt struct {
	Email         string
	IPAddress     string
	UserAgent     string
	Success       bool
	FailureReason string
	AttemptedAt   time.Time
}

// UserRegistration represents user registration data
type UserRegistration struct {
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,min=3,max=100"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required,min=1,max=100"`
	LastName  string `json:"last_name" binding:"required,min=1,max=100"`
}

// UserLogin represents login credentials
type UserLogin struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

// AuthResult represents authentication result
type AuthResult struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// NewAuthService creates a new authentication service
func NewAuthService(db *DatabaseService, logger *logrus.Logger) *AuthService {
	return &AuthService{
		db:     db,
		logger: logger,
	}
}

// Register creates a new user account
func (as *AuthService) Register(registration UserRegistration) (*models.User, error) {
	// Validate password strength
	if !utils.IsValidPassword(registration.Password) {
		return nil, ErrWeakPassword
	}

	// Check if user already exists
	var existingUser models.User
	err := as.db.DB.Where("email = ? OR username = ?", registration.Email, registration.Username).First(&existingUser).Error
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	hashedPassword, err := as.hashPassword(registration.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:           registration.Email,
		Username:        registration.Username,
		Password:        hashedPassword,
		FirstName:       registration.FirstName,
		LastName:        registration.LastName,
		Role:            "user",
		IsActive:        true,
		IsVerified:      false, // Require email verification
		PasswordChangedAt: time.Now(),
	}

	// Set system context for creation
	if err := as.db.SetUserContext("system", "system"); err != nil {
		return nil, fmt.Errorf("failed to set system context: %w", err)
	}
	defer as.db.ClearUserContext()

	if err := as.db.DB.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Remove password from response
	user.Password = ""

	as.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"email":    user.Email,
		"username": user.Username,
	}).Info("User registered successfully")

	return user, nil
}

// Login authenticates a user
func (as *AuthService) Login(credentials UserLogin) (*AuthResult, error) {
	// Find user by email
	var user models.User
	err := as.db.DB.Where("email = ? AND deleted_at IS NULL", credentials.Email).First(&user).Error
	if err != nil {
		// Log failed attempt
		as.logLoginAttempt(LoginAttempt{
			Email:         credentials.Email,
			IPAddress:     credentials.IPAddress,
			UserAgent:     credentials.UserAgent,
			Success:       false,
			FailureReason: "user not found",
			AttemptedAt:   time.Now(),
		})
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if account is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		as.logLoginAttempt(LoginAttempt{
			Email:         credentials.Email,
			IPAddress:     credentials.IPAddress,
			UserAgent:     credentials.UserAgent,
			Success:       false,
			FailureReason: "account locked",
			AttemptedAt:   time.Now(),
		})
		return nil, ErrAccountLocked
	}

	// Check if account is active
	if !user.IsActive {
		as.logLoginAttempt(LoginAttempt{
			Email:         credentials.Email,
			IPAddress:     credentials.IPAddress,
			UserAgent:     credentials.UserAgent,
			Success:       false,
			FailureReason: "account not active",
			AttemptedAt:   time.Now(),
		})
		return nil, ErrAccountNotActive
	}

	// Check if account is verified (optional - can be disabled for development)
	if !user.IsVerified && as.requireEmailVerification() {
		as.logLoginAttempt(LoginAttempt{
			Email:         credentials.Email,
			IPAddress:     credentials.IPAddress,
			UserAgent:     credentials.UserAgent,
			Success:       false,
			FailureReason: "account not verified",
			AttemptedAt:   time.Now(),
		})
		return nil, ErrAccountNotVerified
	}

	// Verify password
	if !as.verifyPassword(credentials.Password, user.Password) {
		// Handle failed login
		if err := as.handleFailedLogin(&user); err != nil {
			as.logger.WithError(err).Error("Failed to handle failed login")
		}

		as.logLoginAttempt(LoginAttempt{
			Email:         credentials.Email,
			IPAddress:     credentials.IPAddress,
			UserAgent:     credentials.UserAgent,
			Success:       false,
			FailureReason: "invalid password",
			AttemptedAt:   time.Now(),
		})
		return nil, ErrInvalidCredentials
	}

	// Handle successful login
	if err := as.handleSuccessfulLogin(&user); err != nil {
		return nil, fmt.Errorf("failed to handle successful login: %w", err)
	}

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := as.generateTokens(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session record
	if err := as.createUserSession(&user, accessToken, refreshToken, credentials.IPAddress, credentials.UserAgent, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to create user session: %w", err)
	}

	// Log successful attempt
	as.logLoginAttempt(LoginAttempt{
		Email:       credentials.Email,
		IPAddress:   credentials.IPAddress,
		UserAgent:   credentials.UserAgent,
		Success:     true,
		AttemptedAt: time.Now(),
	})

	// Remove password from response
	user.Password = ""

	as.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User logged in successfully")

	return &AuthResult{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshToken generates a new access token using a refresh token
func (as *AuthService) RefreshToken(refreshToken string) (*AuthResult, error) {
	// Find session by refresh token
	var session models.UserSession
	err := as.db.DB.Where("refresh_token = ? AND is_active = ? AND expires_at > ?", 
		refreshToken, true, time.Now()).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	// Get user
	var user models.User
	err = as.db.DB.Where("id = ?", session.UserID).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if user is still active
	if !user.IsActive {
		// Deactivate session
		session.IsActive = false
		as.db.DB.Save(&session)
		return nil, ErrAccountNotActive
	}

	// Generate new tokens
	accessToken, newRefreshToken, expiresAt, err := as.generateTokens(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update session with new tokens
	session.SessionToken = accessToken
	session.RefreshToken = newRefreshToken
	session.ExpiresAt = expiresAt
	session.UpdatedAt = time.Now()

	if err := as.db.DB.Save(&session).Error; err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Remove password from response
	user.Password = ""

	return &AuthResult{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Logout invalidates a user session
func (as *AuthService) Logout(userID uuid.UUID, sessionToken string) error {
	// Find and deactivate session
	result := as.db.DB.Model(&models.UserSession{}).
		Where("user_id = ? AND session_token = ?", userID, sessionToken).
		Update("is_active", false)

	if result.Error != nil {
		return fmt.Errorf("failed to logout user: %w", result.Error)
	}

	as.logger.WithField("user_id", userID).Info("User logged out successfully")
	return nil
}

// ValidateToken validates a JWT token and returns user information
func (as *AuthService) ValidateToken(token string) (*models.User, error) {
	// Find active session with token
	var session models.UserSession
	err := as.db.DB.Where("session_token = ? AND is_active = ? AND expires_at > ?", 
		token, true, time.Now()).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	// Get user
	var user models.User
	err = as.db.DB.Where("id = ?", session.UserID).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if user is still active
	if !user.IsActive {
		// Deactivate session
		session.IsActive = false
		as.db.DB.Save(&session)
		return nil, ErrAccountNotActive
	}

	// Remove password from response
	user.Password = ""
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (as *AuthService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := as.db.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Remove password from response
	user.Password = ""
	return &user, nil
}

// hashPassword hashes a password using bcrypt
func (as *AuthService) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// verifyPassword verifies a password against its hash
func (as *AuthService) verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateTokens generates access and refresh tokens
func (as *AuthService) generateTokens(user *models.User) (string, string, time.Time, error) {
	// For now, generate simple tokens. In production, use proper JWT
	accessToken, err := as.generateSecureToken(32)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := as.generateSecureToken(32)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour) // 24 hours

	return accessToken, refreshToken, expiresAt, nil
}

// generateSecureToken generates a cryptographically secure random token
func (as *AuthService) generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// createUserSession creates a new user session record
func (as *AuthService) createUserSession(user *models.User, accessToken, refreshToken, ipAddress, userAgent string, expiresAt time.Time) error {
	session := &models.UserSession{
		UserID:       user.ID,
		SessionToken: accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsActive:     true,
	}

	return as.db.DB.Create(session).Error
}

// handleSuccessfulLogin updates user after successful login
func (as *AuthService) handleSuccessfulLogin(user *models.User) error {
	now := time.Now()
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.LastLogin = &now
	user.UpdatedAt = now

	return as.db.DB.Save(user).Error
}

// handleFailedLogin handles failed login attempt
func (as *AuthService) handleFailedLogin(user *models.User) error {
	const maxAttempts = 5
	const lockoutDuration = 15 * time.Minute

	user.FailedLoginAttempts++
	
	if user.FailedLoginAttempts >= maxAttempts {
		lockUntil := time.Now().Add(lockoutDuration)
		user.LockedUntil = &lockUntil
	}

	user.UpdatedAt = time.Now()
	return as.db.DB.Save(user).Error
}

// logLoginAttempt logs login attempt for security monitoring
func (as *AuthService) logLoginAttempt(attempt LoginAttempt) {
	// This would normally be stored in the login_attempts table
	// For now, just log it
	fields := logrus.Fields{
		"email":          attempt.Email,
		"ip_address":     attempt.IPAddress,
		"success":        attempt.Success,
		"failure_reason": attempt.FailureReason,
	}

	if attempt.Success {
		as.logger.WithFields(fields).Info("Login attempt successful")
	} else {
		as.logger.WithFields(fields).Warn("Login attempt failed")
	}
}

// requireEmailVerification returns whether email verification is required
func (as *AuthService) requireEmailVerification() bool {
	// In development, email verification might be disabled
	// In production, this should always return true
	return false // TODO: Make this configurable
}