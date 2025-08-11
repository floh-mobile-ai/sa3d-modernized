package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// DefaultJWTSecretLength is the minimum recommended length for JWT secrets
	DefaultJWTSecretLength = 32
	// MinJWTSecretLength is the absolute minimum length for JWT secrets
	MinJWTSecretLength = 16
)

// SecretManager handles secure secret management
type SecretManager struct {
	logger *logrus.Logger
}

// NewSecretManager creates a new secret manager
func NewSecretManager(logger *logrus.Logger) *SecretManager {
	return &SecretManager{
		logger: logger,
	}
}

// GetJWTSecret retrieves or generates a secure JWT secret
func (sm *SecretManager) GetJWTSecret() (string, error) {
	// Try to get from environment first
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		if err := sm.validateJWTSecret(secret); err != nil {
			sm.logger.Warnf("Invalid JWT secret from environment: %v, generating new secret", err)
			// Continue to generate new secret below
		} else {
			return secret, nil
		}
	}

	// Generate a new secure secret
	secret, err := sm.generateSecureSecret(DefaultJWTSecretLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT secret: %w", err)
	}

	sm.logger.Warn("Generated new JWT secret. Consider setting JWT_SECRET environment variable for production")
	return secret, nil
}

// GetDatabaseCredentials retrieves secure database credentials
func (sm *SecretManager) GetDatabaseCredentials() (host, port, user, password, dbname, sslmode string, err error) {
	host = sm.getEnvOrDefault("DB_HOST", "localhost")
	port = sm.getEnvOrDefault("DB_PORT", "5432")
	user = sm.getEnvOrDefault("DB_USER", "")
	password = os.Getenv("DB_PASSWORD")
	dbname = sm.getEnvOrDefault("DB_NAME", "")
	sslmode = sm.getEnvOrDefault("DB_SSL_MODE", "require")

	// Validate required fields
	if user == "" {
		err = fmt.Errorf("DB_USER environment variable is required")
		return
	}
	if password == "" {
		err = fmt.Errorf("DB_PASSWORD environment variable is required")
		return
	}
	if dbname == "" {
		err = fmt.Errorf("DB_NAME environment variable is required")
		return
	}

	// Validate SSL mode
	validSSLModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	if !validSSLModes[sslmode] {
		sm.logger.Warnf("Invalid SSL mode '%s', using 'require'", sslmode)
		sslmode = "require"
	}

	return
}

// GetRedisCredentials retrieves Redis connection details
func (sm *SecretManager) GetRedisCredentials() (addr, password string, db int, err error) {
	host := sm.getEnvOrDefault("REDIS_HOST", "localhost")
	port := sm.getEnvOrDefault("REDIS_PORT", "6379")
	addr = fmt.Sprintf("%s:%s", host, port)
	password = os.Getenv("REDIS_PASSWORD")
	
	dbStr := sm.getEnvOrDefault("REDIS_DB", "0")
	db, err = strconv.Atoi(dbStr)
	if err != nil {
		sm.logger.Warnf("Invalid REDIS_DB value '%s', using default 0", dbStr)
		db = 0
		err = nil
	}

	return
}

// validateJWTSecret ensures the JWT secret meets security requirements
func (sm *SecretManager) validateJWTSecret(secret string) error {
	if len(secret) < MinJWTSecretLength {
		return fmt.Errorf("JWT secret must be at least %d characters long", MinJWTSecretLength)
	}

	// Check for common weak secrets (only if length is sufficient)
	weakSecrets := []string{
		"secret",
		"your-secret-key",
		"your-secret-key-change-in-production",
		"your-super-secret-jwt-key-change-in-production",
		"development-secret-change-in-production",
		"12345",
		"password",
		"jwt-secret",
	}

	secretLower := strings.ToLower(secret)
	for _, weak := range weakSecrets {
		if strings.Contains(secretLower, weak) {
			return fmt.Errorf("JWT secret appears to be a common weak secret")
		}
	}

	return nil
}

// generateSecureSecret generates a cryptographically secure random secret
func (sm *SecretManager) generateSecureSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func (sm *SecretManager) getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RotateJWTSecret generates a new JWT secret for rotation
func (sm *SecretManager) RotateJWTSecret() (string, error) {
	secret, err := sm.generateSecureSecret(DefaultJWTSecretLength)
	if err != nil {
		return "", fmt.Errorf("failed to rotate JWT secret: %w", err)
	}

	sm.logger.Info("JWT secret rotated successfully")
	return secret, nil
}

// HashSecret creates a SHA256 hash of a secret for comparison
func (sm *SecretManager) HashSecret(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}

// SecretRotationInfo contains information about secret rotation
type SecretRotationInfo struct {
	SecretHash    string    `json:"secret_hash"`
	RotatedAt     time.Time `json:"rotated_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	RotationCount int       `json:"rotation_count"`
}

// GetSecretRotationInfo returns rotation information for audit purposes
func (sm *SecretManager) GetSecretRotationInfo(secret string) SecretRotationInfo {
	return SecretRotationInfo{
		SecretHash:    sm.HashSecret(secret),
		RotatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour), // 30 days
		RotationCount: 1,
	}
}