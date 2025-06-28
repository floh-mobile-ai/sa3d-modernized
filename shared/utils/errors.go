package utils

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError represents an application error with additional context
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"status_code"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// Common error codes
const (
	ErrCodeValidation      = "VALIDATION_ERROR"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeUnauthorized    = "UNAUTHORIZED"
	ErrCodeForbidden       = "FORBIDDEN"
	ErrCodeConflict        = "CONFLICT"
	ErrCodeInternal        = "INTERNAL_ERROR"
	ErrCodeBadRequest      = "BAD_REQUEST"
	ErrCodeTimeout         = "TIMEOUT"
	ErrCodeRateLimit       = "RATE_LIMIT"
	ErrCodeServiceDown     = "SERVICE_UNAVAILABLE"
	ErrCodeInvalidToken    = "INVALID_TOKEN"
	ErrCodeExpiredToken    = "EXPIRED_TOKEN"
	ErrCodeDatabaseError   = "DATABASE_ERROR"
	ErrCodeExternalService = "EXTERNAL_SERVICE_ERROR"
)

// NewAppError creates a new application error
func NewAppError(code, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// NewAppErrorWithDetails creates a new application error with additional details
func NewAppErrorWithDetails(code, message string, statusCode int, err error, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    details,
		Err:        err,
	}
}

// Common error constructors

// NewValidationError creates a validation error
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return NewAppErrorWithDetails(ErrCodeValidation, message, http.StatusBadRequest, nil, details)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound, nil)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized, nil)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message string) *AppError {
	if message == "" {
		message = "Access forbidden"
	}
	return NewAppError(ErrCodeForbidden, message, http.StatusForbidden, nil)
}

// NewConflictError creates a conflict error
func NewConflictError(message string) *AppError {
	return NewAppError(ErrCodeConflict, message, http.StatusConflict, nil)
}

// NewInternalError creates an internal server error
func NewInternalError(message string, err error) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return NewAppError(ErrCodeInternal, message, http.StatusInternalServerError, err)
}

// NewBadRequestError creates a bad request error
func NewBadRequestError(message string) *AppError {
	return NewAppError(ErrCodeBadRequest, message, http.StatusBadRequest, nil)
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(message string) *AppError {
	if message == "" {
		message = "Request timeout"
	}
	return NewAppError(ErrCodeTimeout, message, http.StatusRequestTimeout, nil)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(message string) *AppError {
	if message == "" {
		message = "Rate limit exceeded"
	}
	return NewAppError(ErrCodeRateLimit, message, http.StatusTooManyRequests, nil)
}

// NewServiceUnavailableError creates a service unavailable error
func NewServiceUnavailableError(service string) *AppError {
	return NewAppError(ErrCodeServiceDown, fmt.Sprintf("Service %s is unavailable", service), http.StatusServiceUnavailable, nil)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError extracts AppError from an error
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewErrorResponse creates an error response from an AppError
func NewErrorResponse(err *AppError) ErrorResponse {
	return ErrorResponse{
		Error:   err.Error(),
		Code:    err.Code,
		Message: err.Message,
		Details: err.Details,
	}
}

// HandleError converts various error types to AppError
func HandleError(err error) *AppError {
	if err == nil {
		return nil
	}
	
	// Check if it's already an AppError
	if appErr := GetAppError(err); appErr != nil {
		return appErr
	}
	
	// Handle specific error types
	switch {
	case errors.Is(err, errors.New("not found")):
		return NewNotFoundError("Resource")
	case errors.Is(err, errors.New("unauthorized")):
		return NewUnauthorizedError("")
	case errors.Is(err, errors.New("forbidden")):
		return NewForbiddenError("")
	default:
		return NewInternalError("An unexpected error occurred", err)
	}
}