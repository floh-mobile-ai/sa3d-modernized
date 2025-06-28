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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"

	"github.com/sa3d-modernized/sa3d/services/api-gateway/internal/handler"
	"github.com/sa3d-modernized/sa3d/services/api-gateway/internal/middleware"
	"github.com/sa3d-modernized/sa3d/services/api-gateway/internal/proxy"
)

type Config struct {
	Server struct {
		Port         string        `mapstructure:"port"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
	} `mapstructure:"server"`

	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`

	Services struct {
		Analysis struct {
			URL     string        `mapstructure:"url"`
			Timeout time.Duration `mapstructure:"timeout"`
		} `mapstructure:"analysis"`
		Visualization struct {
			URL     string        `mapstructure:"url"`
			Timeout time.Duration `mapstructure:"timeout"`
		} `mapstructure:"visualization"`
		Collaboration struct {
			URL     string        `mapstructure:"url"`
			Timeout time.Duration `mapstructure:"timeout"`
		} `mapstructure:"collaboration"`
		Metrics struct {
			URL     string        `mapstructure:"url"`
			Timeout time.Duration `mapstructure:"timeout"`
		} `mapstructure:"metrics"`
	} `mapstructure:"services"`

	Auth struct {
		JWTSecret     string        `mapstructure:"jwt_secret"`
		TokenDuration time.Duration `mapstructure:"token_duration"`
	} `mapstructure:"auth"`

	RateLimit struct {
		RequestsPerSecond int `mapstructure:"requests_per_second"`
		Burst             int `mapstructure:"burst"`
	} `mapstructure:"rate_limit"`

	CORS struct {
		AllowedOrigins []string `mapstructure:"allowed_origins"`
		AllowedMethods []string `mapstructure:"allowed_methods"`
		AllowedHeaders []string `mapstructure:"allowed_headers"`
		MaxAge         int      `mapstructure:"max_age"`
	} `mapstructure:"cors"`
}

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize tracer
	tracer := otel.Tracer("api-gateway")

	// Create rate limiter
	limiter := rate.NewLimiter(
		rate.Limit(config.RateLimit.RequestsPerSecond),
		config.RateLimit.Burst,
	)

	// Initialize service proxies
	serviceProxies := initializeServiceProxies(config, logger)

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS(config.CORS))
	router.Use(middleware.RateLimiter(limiter))
	router.Use(middleware.Tracing(tracer))

	// Initialize handlers
	authHandler := handler.NewAuthHandler(redisClient, config.Auth.JWTSecret, config.Auth.TokenDuration, logger)
	healthHandler := handler.NewHealthHandler(serviceProxies, logger)

	// Setup routes
	setupRoutes(router, authHandler, healthHandler, serviceProxies, config, logger)

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + config.Server.Port,
		Handler:      router,
		ReadTimeout:  config.Server.ReadTimeout,
		WriteTimeout: config.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Starting API Gateway on port %s", config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func loadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/api-gateway")

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("rate_limit.requests_per_second", 100)
	viper.SetDefault("rate_limit.burst", 200)
	viper.SetDefault("cors.max_age", 86400)

	// Read from environment variables
	viper.SetEnvPrefix("GATEWAY")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Validate required fields
	if config.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("JWT secret is required")
	}

	return &config, nil
}

func initializeServiceProxies(config *Config, logger *logrus.Logger) map[string]*proxy.ServiceProxy {
	proxies := make(map[string]*proxy.ServiceProxy)

	// Analysis service proxy
	if config.Services.Analysis.URL != "" {
		proxies["analysis"] = proxy.NewServiceProxy(
			"analysis",
			config.Services.Analysis.URL,
			config.Services.Analysis.Timeout,
			logger,
		)
	}

	// Visualization service proxy
	if config.Services.Visualization.URL != "" {
		proxies["visualization"] = proxy.NewServiceProxy(
			"visualization",
			config.Services.Visualization.URL,
			config.Services.Visualization.Timeout,
			logger,
		)
	}

	// Collaboration service proxy
	if config.Services.Collaboration.URL != "" {
		proxies["collaboration"] = proxy.NewServiceProxy(
			"collaboration",
			config.Services.Collaboration.URL,
			config.Services.Collaboration.Timeout,
			logger,
		)
	}

	// Metrics service proxy
	if config.Services.Metrics.URL != "" {
		proxies["metrics"] = proxy.NewServiceProxy(
			"metrics",
			config.Services.Metrics.URL,
			config.Services.Metrics.Timeout,
			logger,
		)
	}

	return proxies
}

func setupRoutes(
	router *gin.Engine,
	authHandler *handler.AuthHandler,
	healthHandler *handler.HealthHandler,
	serviceProxies map[string]*proxy.ServiceProxy,
	config *Config,
	logger *logrus.Logger,
) {
	// Health check
	router.GET("/health", healthHandler.Health)
	router.GET("/health/ready", healthHandler.Ready)
	router.GET("/health/live", healthHandler.Live)

	// Auth routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.GET("/validate", authHandler.ValidateToken)
	}

	// API routes with authentication
	api := router.Group("/api/v1")
	api.Use(middleware.Auth(config.Auth.JWTSecret))
	{
		// Analysis routes
		if analysisProxy, ok := serviceProxies["analysis"]; ok {
			analysis := api.Group("/analysis")
			{
				analysis.POST("/start/:projectId", createProxyHandler(analysisProxy, "POST", "/analysis/start"))
				analysis.GET("/status/:analysisId", createProxyHandler(analysisProxy, "GET", "/analysis/status"))
				analysis.DELETE("/cancel/:analysisId", createProxyHandler(analysisProxy, "DELETE", "/analysis/cancel"))
				analysis.GET("/results/:analysisId", createProxyHandler(analysisProxy, "GET", "/analysis/results"))
			}
		}

		// Visualization routes
		if vizProxy, ok := serviceProxies["visualization"]; ok {
			viz := api.Group("/visualization")
			{
				viz.GET("/project/:projectId", createProxyHandler(vizProxy, "GET", "/visualization/project"))
				viz.POST("/render", createProxyHandler(vizProxy, "POST", "/visualization/render"))
				viz.GET("/layouts", createProxyHandler(vizProxy, "GET", "/visualization/layouts"))
				viz.PUT("/layout/:projectId", createProxyHandler(vizProxy, "PUT", "/visualization/layout"))
			}
		}

		// Collaboration routes
		if collabProxy, ok := serviceProxies["collaboration"]; ok {
			collab := api.Group("/collaboration")
			{
				collab.GET("/session/:projectId", createProxyHandler(collabProxy, "GET", "/collaboration/session"))
				collab.POST("/session/join", createProxyHandler(collabProxy, "POST", "/collaboration/session/join"))
				collab.POST("/session/leave", createProxyHandler(collabProxy, "POST", "/collaboration/session/leave"))
				collab.GET("/annotations/:projectId", createProxyHandler(collabProxy, "GET", "/collaboration/annotations"))
				collab.POST("/annotation", createProxyHandler(collabProxy, "POST", "/collaboration/annotation"))
				collab.PUT("/annotation/:id", createProxyHandler(collabProxy, "PUT", "/collaboration/annotation"))
				collab.DELETE("/annotation/:id", createProxyHandler(collabProxy, "DELETE", "/collaboration/annotation"))
			}
		}

		// Metrics routes
		if metricsProxy, ok := serviceProxies["metrics"]; ok {
			metrics := api.Group("/metrics")
			{
				metrics.GET("/project/:projectId", createProxyHandler(metricsProxy, "GET", "/metrics/project"))
				metrics.GET("/file/:projectId/:filePath", createProxyHandler(metricsProxy, "GET", "/metrics/file"))
				metrics.GET("/trends/:projectId", createProxyHandler(metricsProxy, "GET", "/metrics/trends"))
				metrics.GET("/compare", createProxyHandler(metricsProxy, "GET", "/metrics/compare"))
			}
		}

		// Project routes (handled by API Gateway directly)
		projects := api.Group("/projects")
		{
			projectHandler := handler.NewProjectHandler(logger)
			projects.GET("", projectHandler.ListProjects)
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/:id", projectHandler.GetProject)
			projects.PUT("/:id", projectHandler.UpdateProject)
			projects.DELETE("/:id", projectHandler.DeleteProject)
		}
	}

	// WebSocket endpoint for real-time updates
	router.GET("/ws", middleware.Auth(config.Auth.JWTSecret), createWebSocketHandler(serviceProxies, logger))
}

func createProxyHandler(serviceProxy *proxy.ServiceProxy, method, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceProxy.ProxyRequest(c, method, path)
	}
}

func createWebSocketHandler(serviceProxies map[string]*proxy.ServiceProxy, logger *logrus.Logger) gin.HandlerFunc {
	wsHandler := handler.NewWebSocketHandler(serviceProxies, logger)
	return wsHandler.Handle
}