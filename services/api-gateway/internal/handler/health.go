package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/sa3d-modernized/sa3d/services/api-gateway/internal/proxy"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	services map[string]*proxy.ServiceProxy
	logger   *logrus.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(services map[string]*proxy.ServiceProxy, logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		services: services,
		logger:   logger,
	}
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status   string                   `json:"status"`
	Version  string                   `json:"version"`
	Services map[string]ServiceHealth `json:"services"`
}

// ServiceHealth represents the health of a service
type ServiceHealth struct {
	Status       string `json:"status"`
	ResponseTime int64  `json:"response_time_ms"`
	Error        string `json:"error,omitempty"`
}

// Health returns the overall health status
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:   "healthy",
		Version:  "1.0.0", // TODO: Get from build info
		Services: make(map[string]ServiceHealth),
	}

	// Check all services concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	allHealthy := true

	for name, service := range h.services {
		wg.Add(1)
		go func(serviceName string, svc *proxy.ServiceProxy) {
			defer wg.Done()

			start := time.Now()
			err := svc.HealthCheck(ctx)
			responseTime := time.Since(start).Milliseconds()

			health := ServiceHealth{
				Status:       "healthy",
				ResponseTime: responseTime,
			}

			if err != nil {
				health.Status = "unhealthy"
				health.Error = err.Error()
				mu.Lock()
				allHealthy = false
				mu.Unlock()
			}

			mu.Lock()
			response.Services[serviceName] = health
			mu.Unlock()
		}(name, service)
	}

	wg.Wait()

	if !allHealthy {
		response.Status = "degraded"
	}

	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// Ready checks if the service is ready to accept requests
func (h *HealthHandler) Ready(c *gin.Context) {
	// Check critical dependencies
	ready := true
	errors := []string{}

	// Check if we have at least one service configured
	if len(h.services) == 0 {
		ready = false
		errors = append(errors, "No backend services configured")
	}

	// TODO: Add more readiness checks (database, cache, etc.)

	if ready {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"errors": errors,
		})
	}
}

// Live checks if the service is alive
func (h *HealthHandler) Live(c *gin.Context) {
	// Simple liveness check
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"timestamp": time.Now().Unix(),
	})
}