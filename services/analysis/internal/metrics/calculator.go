package metrics

import (
	"math"
	"strings"

	"github.com/sa3d-modernized/sa3d/services/analysis/internal/analyzer"
)

// FileMetrics represents metrics for a single file
type FileMetrics struct {
	LOC                  int     // Lines of Code
	CodeLines            int     // Actual code lines (excluding comments and blanks)
	CommentLines         int     // Comment lines
	BlankLines           int     // Blank lines
	CyclomaticComplexity int     // Total cyclomatic complexity
	FunctionCount        int     // Number of functions
	ClassCount           int     // Number of classes
	ImportCount          int     // Number of imports
	AverageComplexity    float64 // Average complexity per function
	MaxComplexity        int     // Maximum complexity in any function
	MaintainabilityIndex float64 // Maintainability index (0-100)
	TechnicalDebt        float64 // Technical debt in hours
	CodeSmells           int     // Number of code smells detected
	DuplicationRatio     float64 // Code duplication ratio (0-1)
	TestCoverage         float64 // Test coverage percentage (0-100)
}

// Calculator calculates metrics from analysis results
type Calculator struct {
	// Configuration for metric calculations
	complexityThreshold int
	locThreshold        int
	duplicationWindow   int
}

// NewCalculator creates a new metrics calculator
func NewCalculator() *Calculator {
	return &Calculator{
		complexityThreshold: 10,  // Functions with complexity > 10 are considered complex
		locThreshold:        500, // Files with > 500 LOC are considered large
		duplicationWindow:   6,   // Minimum lines for duplication detection
	}
}

// Calculate calculates metrics from analysis result
func (c *Calculator) Calculate(result *analyzer.AnalysisResult) *FileMetrics {
	metrics := &FileMetrics{
		FunctionCount: len(result.Functions),
		ClassCount:    len(result.Classes),
		ImportCount:   len(result.Imports),
	}

	// Count lines
	c.countLines(result, metrics)

	// Calculate complexity metrics
	c.calculateComplexityMetrics(result, metrics)

	// Calculate maintainability index
	metrics.MaintainabilityIndex = c.calculateMaintainabilityIndex(metrics)

	// Estimate technical debt
	metrics.TechnicalDebt = c.estimateTechnicalDebt(result, metrics)

	// Count code smells
	metrics.CodeSmells = c.countCodeSmells(result, metrics)

	// Calculate duplication ratio (simplified)
	metrics.DuplicationRatio = c.calculateDuplicationRatio(result)

	// Calculate test coverage (would need actual coverage data)
	metrics.TestCoverage = c.estimateTestCoverage(result)

	return metrics
}

// countLines counts different types of lines
func (c *Calculator) countLines(result *analyzer.AnalysisResult, metrics *FileMetrics) {
	// This is a simplified implementation
	// In a real implementation, we would parse the actual content
	
	// Estimate based on function and class definitions
	for _, fn := range result.Functions {
		lines := fn.EndLine - fn.StartLine + 1
		metrics.LOC += lines
		metrics.CodeLines += int(float64(lines) * 0.7) // Assume 70% are code lines
	}

	for _, class := range result.Classes {
		lines := class.EndLine - class.StartLine + 1
		metrics.LOC += lines
		metrics.CodeLines += int(float64(lines) * 0.7)
		
		// Add method lines
		for _, method := range class.Methods {
			methodLines := method.EndLine - method.StartLine + 1
			metrics.FunctionCount++
			metrics.CodeLines += int(float64(methodLines) * 0.7)
		}
	}

	// Count comment lines
	for _, comment := range result.Comments {
		metrics.CommentLines += comment.EndLine - comment.StartLine + 1
	}

	// Estimate blank lines
	metrics.BlankLines = int(float64(metrics.LOC) * 0.15) // Assume 15% blank lines
}

// calculateComplexityMetrics calculates complexity-related metrics
func (c *Calculator) calculateComplexityMetrics(result *analyzer.AnalysisResult, metrics *FileMetrics) {
	totalComplexity := 0
	maxComplexity := 0
	functionCount := 0

	// Process standalone functions
	for _, fn := range result.Functions {
		complexity := fn.Complexity
		if complexity == 0 {
			complexity = 1 // Default complexity
		}
		totalComplexity += complexity
		if complexity > maxComplexity {
			maxComplexity = complexity
		}
		functionCount++
	}

	// Process methods in classes
	for _, class := range result.Classes {
		for _, method := range class.Methods {
			complexity := method.Complexity
			if complexity == 0 {
				complexity = 1
			}
			totalComplexity += complexity
			if complexity > maxComplexity {
				maxComplexity = complexity
			}
			functionCount++
		}
	}

	metrics.CyclomaticComplexity = totalComplexity
	metrics.MaxComplexity = maxComplexity
	
	if functionCount > 0 {
		metrics.AverageComplexity = float64(totalComplexity) / float64(functionCount)
	}
}

// calculateMaintainabilityIndex calculates the maintainability index
// Based on the formula: MI = 171 - 5.2 * ln(V) - 0.23 * CC - 16.2 * ln(LOC)
// Where V = Halstead Volume, CC = Cyclomatic Complexity, LOC = Lines of Code
func (c *Calculator) calculateMaintainabilityIndex(metrics *FileMetrics) float64 {
	if metrics.LOC == 0 {
		return 100.0
	}

	// Simplified calculation without Halstead Volume
	// Using average complexity as a proxy
	mi := 171.0 - 
		0.23*float64(metrics.CyclomaticComplexity) - 
		16.2*math.Log(float64(metrics.LOC))

	// Normalize to 0-100 range
	mi = math.Max(0, math.Min(100, mi))

	// Apply comment ratio bonus
	commentRatio := float64(metrics.CommentLines) / float64(metrics.LOC)
	if commentRatio > 0.1 {
		mi += 5.0 // Bonus for well-documented code
	}

	return math.Round(mi*100) / 100
}

// estimateTechnicalDebt estimates technical debt in hours
func (c *Calculator) estimateTechnicalDebt(result *analyzer.AnalysisResult, metrics *FileMetrics) float64 {
	debt := 0.0

	// High complexity functions
	for _, fn := range result.Functions {
		if fn.Complexity > c.complexityThreshold {
			debt += float64(fn.Complexity-c.complexityThreshold) * 0.5 // 0.5 hours per complexity point
		}
	}

	// Large file penalty
	if metrics.LOC > c.locThreshold {
		debt += float64(metrics.LOC-c.locThreshold) * 0.01 // 0.01 hours per line over threshold
	}

	// Missing documentation
	undocumentedFunctions := 0
	for _, fn := range result.Functions {
		if fn.IsPublic && fn.Documentation == "" {
			undocumentedFunctions++
		}
	}
	debt += float64(undocumentedFunctions) * 0.25 // 0.25 hours per undocumented public function

	// Code smells
	debt += float64(metrics.CodeSmells) * 0.5 // 0.5 hours per code smell

	return math.Round(debt*100) / 100
}

// countCodeSmells counts various code smells
func (c *Calculator) countCodeSmells(result *analyzer.AnalysisResult, metrics *FileMetrics) int {
	smells := 0

	// Long functions
	for _, fn := range result.Functions {
		if fn.EndLine-fn.StartLine > 50 {
			smells++
		}
		// Too many parameters
		if len(fn.Parameters) > 5 {
			smells++
		}
		// High complexity
		if fn.Complexity > c.complexityThreshold {
			smells++
		}
	}

	// Large classes
	for _, class := range result.Classes {
		if len(class.Methods) > 20 {
			smells++
		}
		if len(class.Properties) > 15 {
			smells++
		}
	}

	// Too many imports (potential feature envy)
	if metrics.ImportCount > 20 {
		smells++
	}

	// Low comment ratio
	if metrics.LOC > 0 {
		commentRatio := float64(metrics.CommentLines) / float64(metrics.LOC)
		if commentRatio < 0.05 {
			smells++
		}
	}

	return smells
}

// calculateDuplicationRatio calculates code duplication ratio
func (c *Calculator) calculateDuplicationRatio(result *analyzer.AnalysisResult) float64 {
	// This is a simplified implementation
	// Real implementation would use suffix trees or other algorithms
	
	// For now, return a low duplication ratio
	// In production, this would analyze actual code patterns
	return 0.05
}

// estimateTestCoverage estimates test coverage based on test functions
func (c *Calculator) estimateTestCoverage(result *analyzer.AnalysisResult) float64 {
	testFunctions := 0
	totalFunctions := len(result.Functions)

	for _, fn := range result.Functions {
		if fn.IsTest {
			testFunctions++
		}
	}

	// Count methods
	for _, class := range result.Classes {
		totalFunctions += len(class.Methods)
		for _, method := range class.Methods {
			if method.IsTest {
				testFunctions++
			}
		}
	}

	if totalFunctions == 0 {
		return 0.0
	}

	// Rough estimate: assume each test covers 2 functions
	coverage := float64(testFunctions*2) / float64(totalFunctions) * 100
	return math.Min(100, math.Round(coverage*100)/100)
}

// AggregateMetrics aggregates metrics from multiple files
func AggregateMetrics(fileMetrics []*FileMetrics) map[string]interface{} {
	totalLOC := 0
	totalComplexity := 0
	totalFunctions := 0
	totalClasses := 0
	totalDebt := 0.0
	totalSmells := 0
	avgMaintainability := 0.0
	avgCoverage := 0.0

	for _, m := range fileMetrics {
		totalLOC += m.LOC
		totalComplexity += m.CyclomaticComplexity
		totalFunctions += m.FunctionCount
		totalClasses += m.ClassCount
		totalDebt += m.TechnicalDebt
		totalSmells += m.CodeSmells
		avgMaintainability += m.MaintainabilityIndex
		avgCoverage += m.TestCoverage
	}

	fileCount := len(fileMetrics)
	if fileCount > 0 {
		avgMaintainability /= float64(fileCount)
		avgCoverage /= float64(fileCount)
	}

	return map[string]interface{}{
		"total_loc":               totalLOC,
		"total_complexity":        totalComplexity,
		"total_functions":         totalFunctions,
		"total_classes":           totalClasses,
		"total_technical_debt":    totalDebt,
		"total_code_smells":       totalSmells,
		"average_maintainability": avgMaintainability,
		"average_test_coverage":   avgCoverage,
		"file_count":              fileCount,
	}
}