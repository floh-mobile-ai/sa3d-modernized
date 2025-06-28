package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ServiceProxy handles proxying requests to backend services
type ServiceProxy struct {
	name    string
	baseURL string
	timeout time.Duration
	client  *http.Client
	logger  *logrus.Logger
}

// NewServiceProxy creates a new service proxy
func NewServiceProxy(name, baseURL string, timeout time.Duration, logger *logrus.Logger) *ServiceProxy {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ServiceProxy{
		name:    name,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

// ProxyRequest proxies a request to the backend service
func (p *ServiceProxy) ProxyRequest(c *gin.Context, method, path string) {
	// Build target URL
	targetURL := p.buildTargetURL(path, c.Request.URL.Query())

	// Create request body
	var body io.Reader
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			p.logger.WithError(err).Error("Failed to read request body")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}
		body = bytes.NewReader(bodyBytes)
	}

	// Create new request
	req, err := http.NewRequestWithContext(c.Request.Context(), method, targetURL, body)
	if err != nil {
		p.logger.WithError(err).Error("Failed to create request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Copy headers
	p.copyHeaders(c.Request.Header, req.Header)

	// Add custom headers
	req.Header.Set("X-Forwarded-For", c.ClientIP())
	req.Header.Set("X-Request-ID", c.GetString("request_id"))
	req.Header.Set("X-User-ID", c.GetString("user_id"))

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.WithError(err).WithFields(logrus.Fields{
			"service": p.name,
			"url":     targetURL,
			"method":  method,
		}).Error("Failed to execute request")
		
		if err == context.DeadlineExceeded {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Service timeout"})
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Service unavailable"})
		}
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.WithError(err).Error("Failed to read response body")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Write response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// HealthCheck checks if the service is healthy
func (p *ServiceProxy) HealthCheck(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/health", p.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// buildTargetURL builds the target URL for the backend service
func (p *ServiceProxy) buildTargetURL(path string, query url.Values) string {
	// Replace path parameters
	path = p.replacePathParams(path)
	
	// Build URL
	targetURL := fmt.Sprintf("%s%s", p.baseURL, path)
	
	// Add query parameters
	if len(query) > 0 {
		targetURL = fmt.Sprintf("%s?%s", targetURL, query.Encode())
	}
	
	return targetURL
}

// replacePathParams replaces Gin path parameters with actual values
func (p *ServiceProxy) replacePathParams(path string) string {
	// This is a simplified implementation
	// In production, you'd want more sophisticated path parameter handling
	return path
}

// copyHeaders copies headers from source to destination
func (p *ServiceProxy) copyHeaders(src, dst http.Header) {
	for key, values := range src {
		// Skip hop-by-hop headers
		if p.isHopByHopHeader(key) {
			continue
		}
		
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// isHopByHopHeader checks if a header is a hop-by-hop header
func (p *ServiceProxy) isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
	
	header = strings.ToLower(header)
	for _, h := range hopByHopHeaders {
		if strings.ToLower(h) == header {
			return true
		}
	}
	
	return false
}

// ProxyWebSocket proxies WebSocket connections
func (p *ServiceProxy) ProxyWebSocket(c *gin.Context, targetPath string) {
	// WebSocket proxying would require additional implementation
	// using gorilla/websocket or similar library
	c.JSON(http.StatusNotImplemented, gin.H{"error": "WebSocket proxy not implemented"})
}

// CircuitBreaker wraps the proxy with circuit breaker functionality
type CircuitBreaker struct {
	proxy           *ServiceProxy
	failureThreshold int
	resetTimeout     time.Duration
	failures         int
	lastFailureTime  time.Time
	state            string // "closed", "open", "half-open"
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(proxy *ServiceProxy, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		proxy:            proxy,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            "closed",
	}
}

// Call executes the request with circuit breaker protection
func (cb *CircuitBreaker) Call(c *gin.Context, method, path string) {
	// Check circuit breaker state
	if cb.state == "open" {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = "half-open"
			cb.failures = 0
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service circuit breaker is open"})
			return
		}
	}

	// Create a channel to capture the result
	done := make(chan bool, 1)
	
	// Execute request in goroutine
	go func() {
		cb.proxy.ProxyRequest(c, method, path)
		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Request completed
		if c.Writer.Status() >= 500 {
			cb.recordFailure()
		} else {
			cb.recordSuccess()
		}
	case <-time.After(cb.proxy.timeout):
		// Timeout
		cb.recordFailure()
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Service timeout"})
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	
	if cb.failures >= cb.failureThreshold {
		cb.state = "open"
		cb.proxy.logger.Warnf("Circuit breaker opened for service %s", cb.proxy.name)
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == "half-open" {
		cb.state = "closed"
		cb.failures = 0
		cb.proxy.logger.Infof("Circuit breaker closed for service %s", cb.proxy.name)
	}
}