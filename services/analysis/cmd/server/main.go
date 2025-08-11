package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Initialize configuration
	viper.SetDefault("ANALYSIS_SERVER_PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.AutomaticEnv()

	// Set log level from config
	if level, err := logrus.ParseLevel(viper.GetString("LOG_LEVEL")); err == nil {
		logger.SetLevel(level)
	}

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Add basic middleware for logging
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.WithFields(logrus.Fields{
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"status":    c.Writer.Status(),
			"duration":  time.Since(start),
			"client_ip": c.ClientIP(),
		}).Info("Request processed")
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "analysis-service",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Basic info endpoint
	router.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "analysis-service",
			"version": "1.0.0",
			"status": "running",
		})
	})

	// Placeholder analysis endpoint
	router.POST("/analyze", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "Analysis service implementation in progress",
			"message": "This endpoint will be implemented in the next development phase",
		})
	})

	// Start server
	port := viper.GetString("ANALYSIS_SERVER_PORT")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		logger.Infof("Starting analysis service on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server shutdown complete")
}