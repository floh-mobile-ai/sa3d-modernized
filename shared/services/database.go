package services

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sa3d-modernized/sa3d/shared/utils"
)

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DatabaseService handles database connections and operations
type DatabaseService struct {
	DB     *gorm.DB
	config DatabaseConfig
	logger *logrus.Logger
}

// NewDatabaseService creates a new database service
func NewDatabaseService(secretManager *utils.SecretManager, logger *logrus.Logger) (*DatabaseService, error) {
	// Get database credentials securely
	host, port, user, password, dbname, sslmode, err := secretManager.GetDatabaseCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get database credentials: %w", err)
	}

	config := DatabaseConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbname,
		SSLMode:  sslmode,
	}

	service := &DatabaseService{
		config: config,
		logger: logger,
	}

	if err := service.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return service, nil
}

// Connect establishes connection to the database
func (ds *DatabaseService) Connect() error {
	// Build DSN (Data Source Name)
	dsn := ds.buildDSN()

	// Configure GORM logger
	gormLogger := logger.New(
		ds.logger,
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level (Silent in production)
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,         // Disable color
		},
	)

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)                  // Maximum number of idle connections
	sqlDB.SetMaxOpenConns(100)                 // Maximum number of open connections
	sqlDB.SetConnMaxLifetime(time.Hour)        // Maximum amount of time a connection may be reused
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Maximum amount of time a connection may be idle

	ds.DB = db
	ds.logger.Info("Database connection established successfully")

	return nil
}

// buildDSN constructs the database connection string
func (ds *DatabaseService) buildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		ds.config.Host,
		ds.config.Port,
		ds.config.User,
		ds.config.Password,
		ds.config.DBName,
		ds.config.SSLMode,
	)
}

// Close closes the database connection
func (ds *DatabaseService) Close() error {
	if ds.DB != nil {
		sqlDB, err := ds.DB.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying database connection: %w", err)
		}
		
		if err := sqlDB.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
		
		ds.logger.Info("Database connection closed")
	}
	return nil
}

// Health checks the database health
func (ds *DatabaseService) Health() error {
	if ds.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := ds.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// SetUserContext sets the user context for RLS policies
func (ds *DatabaseService) SetUserContext(userID, userRole string) error {
	if ds.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Set PostgreSQL session variables for RLS
	if err := ds.DB.Exec("SELECT set_config('app.current_user_id', ?, true)", userID).Error; err != nil {
		return fmt.Errorf("failed to set user ID context: %w", err)
	}

	if err := ds.DB.Exec("SELECT set_config('app.current_user_role', ?, true)", userRole).Error; err != nil {
		return fmt.Errorf("failed to set user role context: %w", err)
	}

	return nil
}

// ClearUserContext clears the user context
func (ds *DatabaseService) ClearUserContext() error {
	if ds.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Clear PostgreSQL session variables
	if err := ds.DB.Exec("SELECT set_config('app.current_user_id', '', true)").Error; err != nil {
		return fmt.Errorf("failed to clear user ID context: %w", err)
	}

	if err := ds.DB.Exec("SELECT set_config('app.current_user_role', '', true)").Error; err != nil {
		return fmt.Errorf("failed to clear user role context: %w", err)
	}

	return nil
}

// Transaction executes a function within a database transaction
func (ds *DatabaseService) Transaction(fn func(*gorm.DB) error) error {
	return ds.DB.Transaction(fn)
}

// GetDB returns the GORM database instance
func (ds *DatabaseService) GetDB() *gorm.DB {
	return ds.DB
}

// Stats returns database connection statistics
func (ds *DatabaseService) Stats() (map[string]interface{}, error) {
	if ds.DB == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	sqlDB, err := ds.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	stats := sqlDB.Stats()
	
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}, nil
}