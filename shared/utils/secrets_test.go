package utils

import (
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretManager_GetJWTSecret(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests
	sm := NewSecretManager(logger)

	t.Run("uses environment variable when valid", func(t *testing.T) {
		validSecret := "a-very-strong-jwt-token-that-is-long-enough-for-testing"
		os.Setenv("JWT_SECRET", validSecret)
		defer os.Unsetenv("JWT_SECRET")

		secret, err := sm.GetJWTSecret()
		require.NoError(t, err)
		assert.Equal(t, validSecret, secret)
	})

	t.Run("generates new secret when env var is missing", func(t *testing.T) {
		os.Unsetenv("JWT_SECRET")

		secret, err := sm.GetJWTSecret()
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.GreaterOrEqual(t, len(secret), MinJWTSecretLength)
	})

	t.Run("generates new secret when env var is weak", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "your-secret-key-change-in-production")
		defer os.Unsetenv("JWT_SECRET")

		secret, err := sm.GetJWTSecret()
		require.NoError(t, err)
		assert.NotEqual(t, "your-secret-key-change-in-production", secret)
		assert.GreaterOrEqual(t, len(secret), MinJWTSecretLength)
	})
}

func TestSecretManager_GetDatabaseCredentials(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewSecretManager(logger)

	t.Run("returns error when required fields missing", func(t *testing.T) {
		os.Clearenv()
		_, _, _, _, _, _, err := sm.GetDatabaseCredentials()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DB_USER")
	})

	t.Run("returns valid credentials when all required fields present", func(t *testing.T) {
		os.Setenv("DB_HOST", "testhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "testuser")
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("DB_NAME", "testdb")
		os.Setenv("DB_SSL_MODE", "require")
		defer func() {
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_PASSWORD")
			os.Unsetenv("DB_NAME")
			os.Unsetenv("DB_SSL_MODE")
		}()

		host, port, user, password, dbname, sslmode, err := sm.GetDatabaseCredentials()
		require.NoError(t, err)
		assert.Equal(t, "testhost", host)
		assert.Equal(t, "5432", port)
		assert.Equal(t, "testuser", user)
		assert.Equal(t, "testpass", password)
		assert.Equal(t, "testdb", dbname)
		assert.Equal(t, "require", sslmode)
	})

	t.Run("uses defaults for optional fields", func(t *testing.T) {
		os.Setenv("DB_USER", "testuser")
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("DB_NAME", "testdb")
		defer func() {
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_PASSWORD")
			os.Unsetenv("DB_NAME")
		}()

		host, port, _, _, _, sslmode, err := sm.GetDatabaseCredentials()
		require.NoError(t, err)
		assert.Equal(t, "localhost", host)
		assert.Equal(t, "5432", port)
		assert.Equal(t, "require", sslmode)
	})
}

func TestSecretManager_GetRedisCredentials(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewSecretManager(logger)

	t.Run("returns default values when env vars not set", func(t *testing.T) {
		os.Clearenv()

		addr, password, db, err := sm.GetRedisCredentials()
		require.NoError(t, err)
		assert.Equal(t, "localhost:6379", addr)
		assert.Equal(t, "", password)
		assert.Equal(t, 0, db)
	})

	t.Run("uses environment variables when set", func(t *testing.T) {
		os.Setenv("REDIS_HOST", "redishost")
		os.Setenv("REDIS_PORT", "6380")
		os.Setenv("REDIS_PASSWORD", "redispass")
		os.Setenv("REDIS_DB", "2")
		defer func() {
			os.Unsetenv("REDIS_HOST")
			os.Unsetenv("REDIS_PORT")
			os.Unsetenv("REDIS_PASSWORD")
			os.Unsetenv("REDIS_DB")
		}()

		addr, password, db, err := sm.GetRedisCredentials()
		require.NoError(t, err)
		assert.Equal(t, "redishost:6380", addr)
		assert.Equal(t, "redispass", password)
		assert.Equal(t, 2, db)
	})

	t.Run("handles invalid DB number gracefully", func(t *testing.T) {
		os.Setenv("REDIS_DB", "invalid")
		defer os.Unsetenv("REDIS_DB")

		_, _, db, err := sm.GetRedisCredentials()
		require.NoError(t, err)
		assert.Equal(t, 0, db)
	})
}

func TestSecretManager_validateJWTSecret(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewSecretManager(logger)

	t.Run("rejects secrets that are too short", func(t *testing.T) {
		err := sm.validateJWTSecret("short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least")
	})

	t.Run("rejects common weak secrets", func(t *testing.T) {
		weakSecrets := []string{
			"your-secret-key-change-in-production",
			"thissecretcontainsthewordsecret", // should be case insensitive
			"password123456789", // meets length but contains "password"
			"jwt-secret-that-is-long-enough",
		}

		for _, secret := range weakSecrets {
			err := sm.validateJWTSecret(secret)
			assert.Error(t, err, "should reject weak secret: %s", secret)
			assert.Contains(t, err.Error(), "weak secret")
		}
	})

	t.Run("accepts strong secrets", func(t *testing.T) {
		strongSecret := "a-very-strong-and-unique-jwt-token-that-meets-requirements"
		err := sm.validateJWTSecret(strongSecret)
		assert.NoError(t, err)
	})
}

func TestSecretManager_generateSecureSecret(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewSecretManager(logger)

	t.Run("generates secret of correct length", func(t *testing.T) {
		secret, err := sm.generateSecureSecret(32)
		require.NoError(t, err)
		assert.Equal(t, 32, len(secret))
	})

	t.Run("generates unique secrets", func(t *testing.T) {
		secret1, err1 := sm.generateSecureSecret(32)
		secret2, err2 := sm.generateSecureSecret(32)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, secret1, secret2)
	})

	t.Run("generates secrets without special characters that could cause issues", func(t *testing.T) {
		secret, err := sm.generateSecureSecret(32)
		require.NoError(t, err)
		// Should not contain characters that could cause issues in URLs or configs
		assert.False(t, strings.ContainsAny(secret, " \t\n\r\"'`\\"))
	})
}

func TestSecretManager_RotateJWTSecret(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewSecretManager(logger)

	t.Run("generates new secret on rotation", func(t *testing.T) {
		secret1, err1 := sm.RotateJWTSecret()
		secret2, err2 := sm.RotateJWTSecret()
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, secret1, secret2)
		assert.GreaterOrEqual(t, len(secret1), DefaultJWTSecretLength)
		assert.GreaterOrEqual(t, len(secret2), DefaultJWTSecretLength)
	})
}

func TestSecretManager_HashSecret(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewSecretManager(logger)

	t.Run("generates consistent hash for same input", func(t *testing.T) {
		secret := "test-secret"
		hash1 := sm.HashSecret(secret)
		hash2 := sm.HashSecret(secret)
		assert.Equal(t, hash1, hash2)
		assert.NotEqual(t, secret, hash1) // Hash should be different from input
	})

	t.Run("generates different hashes for different inputs", func(t *testing.T) {
		hash1 := sm.HashSecret("secret1")
		hash2 := sm.HashSecret("secret2")
		assert.NotEqual(t, hash1, hash2)
	})
}

func TestSecretManager_GetSecretRotationInfo(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewSecretManager(logger)

	t.Run("returns valid rotation info", func(t *testing.T) {
		secret := "test-secret"
		info := sm.GetSecretRotationInfo(secret)
		
		assert.NotEmpty(t, info.SecretHash)
		assert.False(t, info.RotatedAt.IsZero())
		assert.False(t, info.ExpiresAt.IsZero())
		assert.True(t, info.ExpiresAt.After(info.RotatedAt))
		assert.Equal(t, 1, info.RotationCount)
	})
}