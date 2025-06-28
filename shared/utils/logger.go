package utils

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// LoggerConfig contains logger configuration
type LoggerConfig struct {
	Level      string
	Format     string
	Output     string
	TimeFormat string
}

// NewLogger creates a new configured logger
func NewLogger(config LoggerConfig) *logrus.Logger {
	logger := logrus.New()
	
	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	
	// Set formatter
	switch strings.ToLower(config.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.TimeFormat,
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: config.TimeFormat,
			FullTimestamp:   true,
		})
	}
	
	// Set output
	switch strings.ToLower(config.Output) {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		// Could add file output here
		logger.SetOutput(os.Stdout)
	}
	
	return logger
}

// LoggerWithFields creates a logger entry with common fields
func LoggerWithFields(logger *logrus.Logger, fields map[string]interface{}) *logrus.Entry {
	return logger.WithFields(fields)
}

// LogError logs an error with additional context
func LogError(logger *logrus.Logger, err error, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["error"] = err.Error()
	logger.WithFields(fields).Error(message)
}

// LogRequest logs HTTP request information
func LogRequest(logger *logrus.Logger, method, path string, statusCode int, duration float64, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["method"] = method
	fields["path"] = path
	fields["status_code"] = statusCode
	fields["duration_ms"] = duration
	
	entry := logger.WithFields(fields)
	
	if statusCode >= 500 {
		entry.Error("Request completed with error")
	} else if statusCode >= 400 {
		entry.Warn("Request completed with client error")
	} else {
		entry.Info("Request completed successfully")
	}
}

// LogDatabaseQuery logs database query information
func LogDatabaseQuery(logger *logrus.Logger, query string, duration float64, rowsAffected int64, err error) {
	fields := logrus.Fields{
		"query":         query,
		"duration_ms":   duration,
		"rows_affected": rowsAffected,
	}
	
	if err != nil {
		fields["error"] = err.Error()
		logger.WithFields(fields).Error("Database query failed")
	} else {
		logger.WithFields(fields).Debug("Database query executed")
	}
}

// LogServiceCall logs external service call information
func LogServiceCall(logger *logrus.Logger, service, method string, duration float64, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["service"] = service
	fields["method"] = method
	fields["duration_ms"] = duration
	
	if err != nil {
		fields["error"] = err.Error()
		logger.WithFields(fields).Error("Service call failed")
	} else {
		logger.WithFields(fields).Info("Service call completed")
	}
}

// LogMetric logs a metric value
func LogMetric(logger *logrus.Logger, metric string, value interface{}, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["metric"] = metric
	fields["value"] = value
	
	logger.WithFields(fields).Info("Metric recorded")
}

// CreateRequestLogger creates a logger for HTTP requests with request ID
func CreateRequestLogger(logger *logrus.Logger, requestID string) *logrus.Entry {
	return logger.WithField("request_id", requestID)
}

// CreateServiceLogger creates a logger for a specific service
func CreateServiceLogger(logger *logrus.Logger, serviceName string) *logrus.Entry {
	return logger.WithField("service", serviceName)
}