package analyzer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sa3d-modernized/sa3d/services/analysis/internal/analyzer"
)

func TestGoAnalyzer_Analyze(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected struct {
			functions int
			classes   int
			imports   int
			errors    int
		}
	}{
		{
			name: "simple function",
			code: `package main

import "fmt"

// Hello prints a greeting
func Hello(name string) {
	fmt.Printf("Hello, %s!\n", name)
}`,
			expected: struct {
				functions int
				classes   int
				imports   int
				errors    int
			}{
				functions: 1,
				classes:   0,
				imports:   1,
				errors:    0,
			},
		},
		{
			name: "struct with methods",
			code: `package main

import (
	"fmt"
	"strings"
)

// User represents a user in the system
type User struct {
	ID   int
	Name string
	Email string
}

// GetDisplayName returns the display name
func (u *User) GetDisplayName() string {
	return strings.Title(u.Name)
}

// Validate checks if the user is valid
func (u *User) Validate() error {
	if u.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}`,
			expected: struct {
				functions int
				classes   int
				imports   int
				errors    int
			}{
				functions: 0,
				classes:   1,
				imports:   2,
				errors:    0,
			},
		},
		{
			name: "interface definition",
			code: `package main

// Repository defines data access methods
type Repository interface {
	Get(id int) (*User, error)
	Save(user *User) error
	Delete(id int) error
}`,
			expected: struct {
				functions int
				classes   int
				imports   int
				errors    int
			}{
				functions: 0,
				classes:   1,
				imports:   0,
				errors:    0,
			},
		},
		{
			name: "syntax error",
			code: `package main

func broken() {
	fmt.Println("missing import"
}`,
			expected: struct {
				functions int
				classes   int
				imports   int
				errors    int
			}{
				functions: 0,
				classes:   0,
				imports:   0,
				errors:    1,
			},
		},
		{
			name: "complex function",
			code: `package main

func ComplexFunction(x int) int {
	if x < 0 {
		return -1
	} else if x == 0 {
		return 0
	} else {
		switch x {
		case 1:
			return 1
		case 2:
			return 4
		default:
			for i := 0; i < x; i++ {
				if i%2 == 0 {
					continue
				}
			}
			return x * x
		}
	}
}`,
			expected: struct {
				functions int
				classes   int
				imports   int
				errors    int
			}{
				functions: 1,
				classes:   0,
				imports:   0,
				errors:    0,
			},
		},
	}

	goAnalyzer := analyzer.NewGoAnalyzer()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := goAnalyzer.Analyze(ctx, []byte(tt.code))
			require.NoError(t, err)
			
			assert.Equal(t, analyzer.LanguageGo, result.Language)
			assert.Len(t, result.Functions, tt.expected.functions)
			assert.Len(t, result.Classes, tt.expected.classes)
			assert.Len(t, result.Imports, tt.expected.imports)
			assert.Len(t, result.Errors, tt.expected.errors)
		})
	}
}

func TestGoAnalyzer_ComplexityCalculation(t *testing.T) {
	code := `package main

func SimpleFunction() {
	// Complexity = 1
}

func ConditionalFunction(x int) int {
	// Complexity = 3 (1 base + 2 if statements)
	if x > 0 {
		return x
	} else if x < 0 {
		return -x
	}
	return 0
}

func LoopFunction(items []int) int {
	sum := 0
	// Complexity = 2 (1 base + 1 for loop)
	for _, item := range items {
		sum += item
	}
	return sum
}

func SwitchFunction(x int) string {
	// Complexity = 4 (1 base + 1 switch + 2 cases)
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	default:
		return "other"
	}
}`

	goAnalyzer := analyzer.NewGoAnalyzer()
	ctx := context.Background()

	result, err := goAnalyzer.Analyze(ctx, []byte(code))
	require.NoError(t, err)
	
	assert.Len(t, result.Functions, 4)
	
	// Find functions by name and check complexity
	functionComplexity := make(map[string]int)
	for _, fn := range result.Functions {
		functionComplexity[fn.Name] = fn.Complexity
	}
	
	assert.Equal(t, 1, functionComplexity["SimpleFunction"])
	assert.GreaterOrEqual(t, functionComplexity["ConditionalFunction"], 3)
	assert.GreaterOrEqual(t, functionComplexity["LoopFunction"], 2)
	assert.GreaterOrEqual(t, functionComplexity["SwitchFunction"], 3)
}

func TestGoAnalyzer_MethodExtraction(t *testing.T) {
	code := `package main

type Calculator struct {
	precision int
}

// Add adds two numbers
func (c *Calculator) Add(a, b float64) float64 {
	return a + b
}

// Subtract subtracts b from a
func (c Calculator) Subtract(a, b float64) float64 {
	return a - b
}

// NewCalculator creates a new calculator
func NewCalculator(precision int) *Calculator {
	return &Calculator{precision: precision}
}`

	goAnalyzer := analyzer.NewGoAnalyzer()
	ctx := context.Background()

	result, err := goAnalyzer.Analyze(ctx, []byte(code))
	require.NoError(t, err)
	
	// Should have 1 standalone function (NewCalculator)
	assert.Len(t, result.Functions, 1)
	assert.Equal(t, "NewCalculator", result.Functions[0].Name)
	
	// Should have 1 class (Calculator) with 2 methods
	require.Len(t, result.Classes, 1)
	assert.Equal(t, "Calculator", result.Classes[0].Name)
	assert.Equal(t, "struct", result.Classes[0].Type)
	assert.Len(t, result.Classes[0].Methods, 2)
	
	// Check method names
	methodNames := make(map[string]bool)
	for _, method := range result.Classes[0].Methods {
		methodNames[method.Name] = true
	}
	assert.True(t, methodNames["Add"])
	assert.True(t, methodNames["Subtract"])
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filePath string
		content  []byte
		expected analyzer.Language
	}{
		{
			filePath: "main.go",
			content:  []byte("package main"),
			expected: analyzer.LanguageGo,
		},
		{
			filePath: "App.java",
			content:  []byte("public class App {}"),
			expected: analyzer.LanguageJava,
		},
		{
			filePath: "script.py",
			content:  []byte("#!/usr/bin/env python\nimport sys"),
			expected: analyzer.LanguagePython,
		},
		{
			filePath: "app.js",
			content:  []byte("const express = require('express')"),
			expected: analyzer.LanguageJavaScript,
		},
		{
			filePath: "component.tsx",
			content:  []byte("import React from 'react'"),
			expected: analyzer.LanguageTypeScript,
		},
		{
			filePath: "Program.cs",
			content:  []byte("using System;"),
			expected: analyzer.LanguageCSharp,
		},
		{
			filePath: "unknown.txt",
			content:  []byte("Some random text"),
			expected: analyzer.LanguageUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			detected := analyzer.DetectLanguage(tt.filePath, tt.content)
			assert.Equal(t, tt.expected, detected)
		})
	}
}