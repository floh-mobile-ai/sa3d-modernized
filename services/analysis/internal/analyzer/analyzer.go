package analyzer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Language represents a programming language
type Language string

const (
	LanguageGo         Language = "go"
	LanguageJava       Language = "java"
	LanguagePython     Language = "python"
	LanguageJavaScript Language = "javascript"
	LanguageTypeScript Language = "typescript"
	LanguageCSharp     Language = "csharp"
	LanguageUnknown    Language = "unknown"
)

// AnalysisResult contains the parsed AST and metadata
type AnalysisResult struct {
	Language     Language
	AST          interface{} // Language-specific AST
	Functions    []Function
	Classes      []Class
	Imports      []Import
	Comments     []Comment
	Errors       []ParseError
}

// Function represents a function/method in the code
type Function struct {
	Name           string
	StartLine      int
	EndLine        int
	Parameters     []Parameter
	ReturnType     string
	Complexity     int
	IsPublic       bool
	IsTest         bool
	Documentation  string
}

// Class represents a class/struct/interface
type Class struct {
	Name          string
	Type          string // class, struct, interface, enum
	StartLine     int
	EndLine       int
	Methods       []Function
	Properties    []Property
	IsPublic      bool
	Documentation string
}

// Property represents a class property/field
type Property struct {
	Name     string
	Type     string
	IsPublic bool
}

// Parameter represents a function parameter
type Parameter struct {
	Name string
	Type string
}

// Import represents an import statement
type Import struct {
	Package string
	Alias   string
	Line    int
}

// Comment represents a comment in the code
type Comment struct {
	Text      string
	StartLine int
	EndLine   int
	IsBlock   bool
}

// ParseError represents a parsing error
type ParseError struct {
	Message string
	Line    int
	Column  int
}

// Analyzer interface for language-specific analyzers
type Analyzer interface {
	Analyze(ctx context.Context, content []byte) (*AnalysisResult, error)
	Language() Language
}

// analyzerRegistry holds all registered analyzers
var analyzerRegistry = make(map[Language]Analyzer)

// RegisterAnalyzer registers a language analyzer
func RegisterAnalyzer(lang Language, analyzer Analyzer) {
	analyzerRegistry[lang] = analyzer
}

// GetAnalyzer returns the analyzer for a language
func GetAnalyzer(lang Language) (Analyzer, error) {
	analyzer, ok := analyzerRegistry[lang]
	if !ok {
		return nil, fmt.Errorf("no analyzer registered for language: %s", lang)
	}
	return analyzer, nil
}

// DetectLanguage detects the programming language from file path and content
func DetectLanguage(filePath string, content []byte) Language {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Check by file extension
	switch ext {
	case ".go":
		return LanguageGo
	case ".java":
		return LanguageJava
	case ".py":
		return LanguagePython
	case ".js", ".mjs", ".cjs":
		return LanguageJavaScript
	case ".ts", ".tsx":
		return LanguageTypeScript
	case ".cs":
		return LanguageCSharp
	}

	// Check by file name patterns
	baseName := filepath.Base(filePath)
	switch {
	case strings.HasSuffix(baseName, ".d.ts"):
		return LanguageTypeScript
	case baseName == "go.mod" || baseName == "go.sum":
		return LanguageGo
	case baseName == "pom.xml" || baseName == "build.gradle":
		return LanguageJava
	case baseName == "requirements.txt" || baseName == "setup.py":
		return LanguagePython
	case baseName == "package.json" || baseName == "tsconfig.json":
		// Could be JS or TS, need to check content
		if strings.Contains(string(content), "typescript") {
			return LanguageTypeScript
		}
		return LanguageJavaScript
	}

	// Try to detect from content (shebang, etc.)
	contentStr := string(content[:min(len(content), 100)])
	if strings.HasPrefix(contentStr, "#!/usr/bin/env python") || strings.HasPrefix(contentStr, "#!/usr/bin/python") {
		return LanguagePython
	}
	if strings.HasPrefix(contentStr, "#!/usr/bin/env node") {
		return LanguageJavaScript
	}

	return LanguageUnknown
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CalculateComplexity calculates cyclomatic complexity for a function
func CalculateComplexity(node interface{}) int {
	// Base complexity is 1
	complexity := 1

	// This is a simplified version - real implementation would traverse AST
	// and count decision points (if, for, while, case, catch, etc.)
	// For now, return a placeholder
	return complexity
}

// ExtractDocumentation extracts documentation/comments for a node
func ExtractDocumentation(comments []Comment, startLine int) string {
	// Look for comments immediately before the start line
	for i := len(comments) - 1; i >= 0; i-- {
		comment := comments[i]
		if comment.EndLine == startLine-1 || comment.EndLine == startLine {
			return strings.TrimSpace(comment.Text)
		}
	}
	return ""
}