package utils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"valid email with plus", "user+tag@example.com", true},
		{"invalid - no @", "testexample.com", false},
		{"invalid - no domain", "test@", false},
		{"invalid - no user", "@example.com", false},
		{"invalid - spaces", "test @example.com", false},
		{"invalid - double @", "test@@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "Test123!@#", false},
		{"too short", "Test1!", true},
		{"no uppercase", "test123!@#", true},
		{"no lowercase", "TEST123!@#", true},
		{"no number", "TestTest!@#", true},
		{"no special char", "TestTest123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	lengths := []int{8, 16, 32}
	
	for _, length := range lengths {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			str, err := GenerateRandomString(length)
			require.NoError(t, err)
			assert.Len(t, str, length)
			
			// Generate another one and ensure they're different
			str2, err := GenerateRandomString(length)
			require.NoError(t, err)
			assert.NotEqual(t, str, str2)
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal string", "Hello World", "Hello World"},
		{"with null bytes", "Hello\x00World", "HelloWorld"},
		{"with control chars", "Hello\x01\x02World", "HelloWorld"},
		{"with whitespace", "  Hello World  ", "Hello World"},
		{"with newlines", "Hello\nWorld", "HelloWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"no truncation needed", "Hello", 10, "Hello"},
		{"exact length", "Hello", 5, "Hello"},
		{"truncate with ellipsis", "Hello World", 8, "Hello..."},
		{"very short max", "Hello", 2, "He"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with duplicates", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all duplicates", []string{"a", "a", "a"}, []string{"a"}},
		{"empty slice", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveDuplicates(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple text", "Hello World", "hello-world"},
		{"with special chars", "Hello, World!", "hello-world"},
		{"with numbers", "Test 123 Project", "test-123-project"},
		{"multiple spaces", "Test   Multiple   Spaces", "test-multiple-spaces"},
		{"leading/trailing spaces", "  Test  ", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		total float64
		want  float64
	}{
		{"normal calculation", 25, 100, 25},
		{"zero total", 25, 0, 0},
		{"decimal result", 33, 100, 33},
		{"over 100%", 150, 100, 150},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePercentage(tt.value, tt.total)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMinMaxClamp(t *testing.T) {
	t.Run("MinInt", func(t *testing.T) {
		assert.Equal(t, 5, MinInt(5, 10))
		assert.Equal(t, 5, MinInt(10, 5))
		assert.Equal(t, -5, MinInt(-5, 5))
	})

	t.Run("MaxInt", func(t *testing.T) {
		assert.Equal(t, 10, MaxInt(5, 10))
		assert.Equal(t, 10, MaxInt(10, 5))
		assert.Equal(t, 5, MaxInt(-5, 5))
	})

	t.Run("ClampInt", func(t *testing.T) {
		assert.Equal(t, 5, ClampInt(5, 0, 10))
		assert.Equal(t, 0, ClampInt(-5, 0, 10))
		assert.Equal(t, 10, ClampInt(15, 0, 10))
	})
}

func TestAppError(t *testing.T) {
	t.Run("NewAppError", func(t *testing.T) {
		err := NewAppError("TEST_ERROR", "Test error message", 400, nil)
		assert.Equal(t, "TEST_ERROR", err.Code)
		assert.Equal(t, "Test error message", err.Message)
		assert.Equal(t, 400, err.StatusCode)
		assert.Equal(t, "Test error message", err.Error())
	})

	t.Run("NewAppErrorWithWrappedError", func(t *testing.T) {
		wrappedErr := errors.New("wrapped error")
		err := NewAppError("TEST_ERROR", "Test error message", 500, wrappedErr)
		assert.Equal(t, "Test error message: wrapped error", err.Error())
		assert.Equal(t, wrappedErr, err.Unwrap())
	})

	t.Run("NewValidationError", func(t *testing.T) {
		details := map[string]interface{}{
			"field": "email",
			"error": "invalid format",
		}
		err := NewValidationError("Validation failed", details)
		assert.Equal(t, ErrCodeValidation, err.Code)
		assert.Equal(t, 400, err.StatusCode)
		assert.Equal(t, details, err.Details)
	})

	t.Run("IsAppError", func(t *testing.T) {
		appErr := NewBadRequestError("bad request")
		normalErr := errors.New("normal error")
		
		assert.True(t, IsAppError(appErr))
		assert.False(t, IsAppError(normalErr))
	})

	t.Run("GetAppError", func(t *testing.T) {
		appErr := NewNotFoundError("user")
		normalErr := errors.New("normal error")
		
		assert.Equal(t, appErr, GetAppError(appErr))
		assert.Nil(t, GetAppError(normalErr))
	})
}