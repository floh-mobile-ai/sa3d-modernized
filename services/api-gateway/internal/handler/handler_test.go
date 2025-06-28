package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sa3d-modernized/sa3d/services/api-gateway/internal/handler"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthHandler_Login(t *testing.T) {
	// Setup
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	authHandler := handler.NewAuthHandler(
		redisClient,
		"test-secret",
		24*time.Hour,
		logger,
	)

	router := setupTestRouter()
	router.POST("/login", authHandler.Login)

	// Test cases
	tests := []struct {
		name       string
		payload    handler.LoginRequest
		wantStatus int
	}{
		{
			name: "valid login",
			payload: handler.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "missing email",
			payload: handler.LoginRequest{
				Password: "password123",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			payload: handler.LoginRequest{
				Email: "test@example.com",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid email format",
			payload: handler.LoginRequest{
				Email:    "invalid-email",
				Password: "password123",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			body, err := json.Marshal(tt.payload)
			require.NoError(t, err)
			
			req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Assert
			assert.Equal(t, tt.wantStatus, w.Code)
			
			if tt.wantStatus == http.StatusOK {
				var resp handler.LoginResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				
				assert.NotEmpty(t, resp.Token)
				assert.NotEmpty(t, resp.RefreshToken)
				assert.NotZero(t, resp.ExpiresAt)
				assert.Equal(t, tt.payload.Email, resp.User.Email)
			}
		})
	}
}

func TestAuthHandler_ValidateToken(t *testing.T) {
	// Setup
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	logger := logrus.New()
	
	authHandler := handler.NewAuthHandler(
		redisClient,
		"test-secret",
		24*time.Hour,
		logger,
	)

	router := setupTestRouter()
	
	// Add middleware to simulate authenticated request
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Set("email", "test@example.com")
		c.Next()
	})
	
	router.GET("/validate", authHandler.ValidateToken)

	// Test
	req := httptest.NewRequest("GET", "/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	
	assert.True(t, resp["valid"].(bool))
	assert.Equal(t, "test-user-123", resp["user_id"])
	assert.Equal(t, "test@example.com", resp["email"])
}

func TestProjectHandler_CreateProject(t *testing.T) {
	// Setup
	logger := logrus.New()
	projectHandler := handler.NewProjectHandler(logger)
	
	router := setupTestRouter()
	
	// Add middleware to simulate authenticated request
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Next()
	})
	
	router.POST("/projects", projectHandler.CreateProject)

	// Test cases
	tests := []struct {
		name       string
		payload    handler.CreateProjectRequest
		wantStatus int
	}{
		{
			name: "valid project",
			payload: handler.CreateProjectRequest{
				Name:        "Test Project",
				Description: "A test project",
				Language:    "go",
				Repository:  "https://github.com/test/project",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing name",
			payload: handler.CreateProjectRequest{
				Description: "A test project",
				Language:    "go",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing language",
			payload: handler.CreateProjectRequest{
				Name:        "Test Project",
				Description: "A test project",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			body, err := json.Marshal(tt.payload)
			require.NoError(t, err)
			
			req := httptest.NewRequest("POST", "/projects", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Assert
			assert.Equal(t, tt.wantStatus, w.Code)
			
			if tt.wantStatus == http.StatusCreated {
				var project handler.Project
				err := json.Unmarshal(w.Body.Bytes(), &project)
				require.NoError(t, err)
				
				assert.NotEmpty(t, project.ID)
				assert.Equal(t, tt.payload.Name, project.Name)
				assert.Equal(t, tt.payload.Description, project.Description)
				assert.Equal(t, tt.payload.Language, project.Language)
				assert.Equal(t, "test-user-123", project.CreatedBy)
			}
		})
	}
}

func TestProjectHandler_ListProjects(t *testing.T) {
	// Setup
	logger := logrus.New()
	projectHandler := handler.NewProjectHandler(logger)
	
	router := setupTestRouter()
	
	// Add middleware to simulate authenticated request
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Next()
	})
	
	router.GET("/projects", projectHandler.ListProjects)

	// Test
	req := httptest.NewRequest("GET", "/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp struct {
		Projects []handler.Project `json:"projects"`
		Total    int               `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	
	assert.Greater(t, len(resp.Projects), 0)
	assert.Equal(t, len(resp.Projects), resp.Total)
}