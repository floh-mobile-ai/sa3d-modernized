package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/sa3d-modernized/sa3d/services/analysis/internal/handler"
	"github.com/sa3d-modernized/sa3d/services/analysis/internal/repository"
	"github.com/sa3d-modernized/sa3d/services/analysis/internal/service"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	// Load configuration
	if err := loadConfig(); err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Set log level
	logLevel, err := logrus.ParseLevel(viper.GetString("LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Initialize database connection
	db, err := repository.NewPostgresDB(
		viper.GetString("DB_HOST"),
		viper.GetInt("DB_PORT"),
		viper.GetString("DB_USER"),
		viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_NAME"),
		viper.GetString("DB_SSL_MODE"),
	)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis client
	redisClient := repository.NewRedisClient(
		viper.GetString("REDIS_HOST"),
		viper.GetInt("REDIS_PORT"),
		viper.GetString("REDIS_PASSWORD"),
		viper.GetInt("REDIS_DB"),
	)
	defer redisClient.Close()

	// Initialize Kafka producer
	kafkaProducer, err := repository.NewKafkaProducer(
		viper.GetStringSlice("KAFKA_BROKERS"),
		viper.GetString("KAFKA_TOPIC_ANALYSIS"),
	)
	if err != nil {
		logger.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Initialize repositories
	projectRepo := repository.NewProjectRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	metricsRepo := repository.NewMetricsRepository(db)

	// Initialize services
	analysisService := service.NewAnalysisService(
		projectRepo,
		analysisRepo,
		metricsRepo,
		redisClient,
		kafkaProducer,
		logger,
	)

	// Initialize Gin router
	router := setupRouter(analysisService, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", viper.GetInt("ANALYSIS_SERVICE_PORT")),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting Analysis Service on port %d", viper.GetInt("ANALYSIS_SERVICE_PORT"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func loadConfig() error {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../..")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ANALYSIS_SERVICE_PORT", 8081)
	viper.SetDefault("WORKER_POOL_SIZE", 0)
	viper.SetDefault("ANALYSIS_TIMEOUT_MINUTES", 30)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

func setupRouter(analysisService *service.AnalysisService, logger *logrus.Logger) *gin.Engine {
	// Set Gin mode based on environment
	if viper.GetString("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(handler.LoggerMiddleware(logger))
	router.Use(handler.CORSMiddleware())
	router.Use(handler.RequestIDMiddleware())

	// Health check endpoint
	router.GET("/health", handler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Analysis endpoints
		analysis := v1.Group("/analysis")
		{
			analysisHandler := handler.NewAnalysisHandler(analysisService, logger)
			analysis.POST("/projects/:projectId/analyze", analysisHandler.AnalyzeProject)
			analysis.GET("/projects/:projectId/analysis/:analysisId", analysisHandler.GetAnalysis)
			analysis.GET("/projects/:projectId/analyses", analysisHandler.ListAnalyses)
			analysis.DELETE("/projects/:projectId/analysis/:analysisId", analysisHandler.CancelAnalysis)
		}

		// Metrics endpoints
		metrics := v1.Group("/metrics")
		{
			metricsHandler := handler.NewMetricsHandler(analysisService, logger)
			metrics.GET("/projects/:projectId/summary", metricsHandler.GetProjectMetricsSummary)
			metrics.GET("/projects/:projectId/files/:fileId", metricsHandler.GetFileMetrics)
			metrics.GET("/projects/:projectId/trends", metricsHandler.GetMetricsTrends)
		}
	}

	// Prometheus metrics endpoint
	router.GET("/metrics", handler.PrometheusHandler())

	return router
}