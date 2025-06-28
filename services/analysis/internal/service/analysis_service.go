package service

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/sa3d-modernized/sa3d/services/analysis/internal/analyzer"
	"github.com/sa3d-modernized/sa3d/services/analysis/internal/metrics"
	"github.com/sa3d-modernized/sa3d/services/analysis/internal/repository"
)

// AnalysisStatus represents the status of an analysis job
type AnalysisStatus string

const (
	StatusPending   AnalysisStatus = "PENDING"
	StatusRunning   AnalysisStatus = "RUNNING"
	StatusCompleted AnalysisStatus = "COMPLETED"
	StatusFailed    AnalysisStatus = "FAILED"
	StatusCancelled AnalysisStatus = "CANCELLED"
)

// AnalysisJob represents an analysis job
type AnalysisJob struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	Status      AnalysisStatus `json:"status"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Error       string         `json:"error,omitempty"`
	Progress    int            `json:"progress"`
	TotalFiles  int            `json:"total_files"`
}

// FileAnalysisResult represents the analysis result for a single file
type FileAnalysisResult struct {
	FilePath   string                 `json:"file_path"`
	Language   string                 `json:"language"`
	LOC        int                    `json:"loc"`
	Complexity int                    `json:"complexity"`
	Metrics    map[string]interface{} `json:"metrics"`
	Error      string                 `json:"error,omitempty"`
}

// AnalysisService handles code analysis operations
type AnalysisService struct {
	projectRepo  repository.ProjectRepository
	analysisRepo repository.AnalysisRepository
	metricsRepo  repository.MetricsRepository
	redisClient  *redis.Client
	kafkaWriter  *kafka.Writer
	logger       *logrus.Logger
	workerPool   int
	cancelFuncs  sync.Map // map[analysisID]context.CancelFunc
}

// NewAnalysisService creates a new analysis service
func NewAnalysisService(
	projectRepo repository.ProjectRepository,
	analysisRepo repository.AnalysisRepository,
	metricsRepo repository.MetricsRepository,
	redisClient *redis.Client,
	kafkaWriter *kafka.Writer,
	logger *logrus.Logger,
) *AnalysisService {
	workerPool := runtime.NumCPU() * 2
	if workerPool < 4 {
		workerPool = 4
	}

	return &AnalysisService{
		projectRepo:  projectRepo,
		analysisRepo: analysisRepo,
		metricsRepo:  metricsRepo,
		redisClient:  redisClient,
		kafkaWriter:  kafkaWriter,
		logger:       logger,
		workerPool:   workerPool,
	}
}

// StartAnalysis starts a new analysis job for a project
func (s *AnalysisService) StartAnalysis(ctx context.Context, projectID string) (*AnalysisJob, error) {
	// Verify project exists
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	// Create analysis job
	job := &AnalysisJob{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Status:    StatusPending,
		StartedAt: time.Now(),
		Progress:  0,
	}

	// Save job to database
	if err := s.analysisRepo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create analysis job: %w", err)
	}

	// Cache job status in Redis
	if err := s.cacheJobStatus(ctx, job); err != nil {
		s.logger.Warnf("Failed to cache job status: %v", err)
	}

	// Start analysis in background
	analysisCtx, cancel := context.WithCancel(context.Background())
	s.cancelFuncs.Store(job.ID, cancel)

	go s.runAnalysis(analysisCtx, job, project)

	return job, nil
}

// runAnalysis performs the actual analysis
func (s *AnalysisService) runAnalysis(ctx context.Context, job *AnalysisJob, project *repository.Project) {
	defer func() {
		s.cancelFuncs.Delete(job.ID)
		if r := recover(); r != nil {
			s.logger.Errorf("Analysis panic recovered: %v", r)
			s.updateJobStatus(context.Background(), job.ID, StatusFailed, fmt.Sprintf("Analysis panic: %v", r))
		}
	}()

	// Update status to running
	job.Status = StatusRunning
	if err := s.updateJobStatus(ctx, job.ID, StatusRunning, ""); err != nil {
		s.logger.Errorf("Failed to update job status: %v", err)
		return
	}

	// Get project files
	files, err := s.projectRepo.GetProjectFiles(ctx, project.ID)
	if err != nil {
		s.updateJobStatus(ctx, job.ID, StatusFailed, fmt.Sprintf("Failed to get project files: %v", err))
		return
	}

	job.TotalFiles = len(files)
	s.cacheJobStatus(ctx, job)

	// Create channels for worker pool
	fileChan := make(chan *repository.ProjectFile, len(files))
	resultChan := make(chan *FileAnalysisResult, len(files))

	// Start worker pool
	g, ctx := errgroup.WithContext(ctx)
	
	// Producer: send files to channel
	g.Go(func() error {
		defer close(fileChan)
		for _, file := range files {
			select {
			case fileChan <- file:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Workers: analyze files
	for i := 0; i < s.workerPool; i++ {
		g.Go(func() error {
			for file := range fileChan {
				result := s.analyzeFile(ctx, file)
				select {
				case resultChan <- result:
					// Update progress
					job.Progress++
					s.cacheJobStatus(ctx, job)
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	// Collector: collect results
	var results []*FileAnalysisResult
	g.Go(func() error {
		for i := 0; i < len(files); i++ {
			select {
			case result := <-resultChan:
				results = append(results, result)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		s.updateJobStatus(ctx, job.ID, StatusFailed, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	// Process and save results
	if err := s.processResults(ctx, job, results); err != nil {
		s.updateJobStatus(ctx, job.ID, StatusFailed, fmt.Sprintf("Failed to process results: %v", err))
		return
	}

	// Update job status to completed
	s.updateJobStatus(ctx, job.ID, StatusCompleted, "")

	// Publish completion event
	s.publishAnalysisEvent(job.ID, "analysis.completed", map[string]interface{}{
		"project_id":   project.ID,
		"analysis_id":  job.ID,
		"total_files":  job.TotalFiles,
		"completed_at": time.Now(),
	})
}

// analyzeFile analyzes a single file
func (s *AnalysisService) analyzeFile(ctx context.Context, file *repository.ProjectFile) *FileAnalysisResult {
	result := &FileAnalysisResult{
		FilePath: file.Path,
		Metrics:  make(map[string]interface{}),
	}

	// Detect language
	language := analyzer.DetectLanguage(file.Path, file.Content)
	result.Language = language

	// Get appropriate analyzer
	fileAnalyzer, err := analyzer.GetAnalyzer(language)
	if err != nil {
		result.Error = fmt.Sprintf("No analyzer available for language: %s", language)
		return result
	}

	// Parse and analyze file
	analysisResult, err := fileAnalyzer.Analyze(ctx, file.Content)
	if err != nil {
		result.Error = fmt.Sprintf("Analysis failed: %v", err)
		return result
	}

	// Calculate metrics
	metricsCalculator := metrics.NewCalculator()
	fileMetrics := metricsCalculator.Calculate(analysisResult)

	result.LOC = fileMetrics.LOC
	result.Complexity = fileMetrics.CyclomaticComplexity
	result.Metrics = map[string]interface{}{
		"functions":           fileMetrics.FunctionCount,
		"classes":             fileMetrics.ClassCount,
		"imports":             fileMetrics.ImportCount,
		"comment_lines":       fileMetrics.CommentLines,
		"code_lines":          fileMetrics.CodeLines,
		"blank_lines":         fileMetrics.BlankLines,
		"average_complexity":  fileMetrics.AverageComplexity,
		"max_complexity":      fileMetrics.MaxComplexity,
		"maintainability":     fileMetrics.MaintainabilityIndex,
		"technical_debt":      fileMetrics.TechnicalDebt,
		"code_smells":         fileMetrics.CodeSmells,
		"duplication_ratio":   fileMetrics.DuplicationRatio,
		"test_coverage":       fileMetrics.TestCoverage,
	}

	return result
}

// processResults processes and saves analysis results
func (s *AnalysisService) processResults(ctx context.Context, job *AnalysisJob, results []*FileAnalysisResult) error {
	// Calculate aggregate metrics
	aggregateMetrics := s.calculateAggregateMetrics(results)

	// Save results to database
	if err := s.metricsRepo.SaveAnalysisResults(ctx, job.ID, results, aggregateMetrics); err != nil {
		return fmt.Errorf("failed to save analysis results: %w", err)
	}

	// Cache summary in Redis for quick access
	summaryKey := fmt.Sprintf("analysis:summary:%s", job.ID)
	summaryData, _ := json.Marshal(aggregateMetrics)
	s.redisClient.Set(ctx, summaryKey, summaryData, 24*time.Hour)

	return nil
}

// calculateAggregateMetrics calculates aggregate metrics from file results
func (s *AnalysisService) calculateAggregateMetrics(results []*FileAnalysisResult) map[string]interface{} {
	totalLOC := 0
	totalComplexity := 0
	totalFiles := len(results)
	languageDistribution := make(map[string]int)
	errorCount := 0

	for _, result := range results {
		if result.Error != "" {
			errorCount++
			continue
		}
		totalLOC += result.LOC
		totalComplexity += result.Complexity
		languageDistribution[result.Language]++
	}

	avgComplexity := 0.0
	if totalFiles-errorCount > 0 {
		avgComplexity = float64(totalComplexity) / float64(totalFiles-errorCount)
	}

	return map[string]interface{}{
		"total_files":           totalFiles,
		"total_loc":             totalLOC,
		"total_complexity":      totalComplexity,
		"average_complexity":    avgComplexity,
		"language_distribution": languageDistribution,
		"error_count":           errorCount,
		"analysis_timestamp":    time.Now(),
	}
}

// updateJobStatus updates the job status in database and cache
func (s *AnalysisService) updateJobStatus(ctx context.Context, jobID string, status AnalysisStatus, errorMsg string) error {
	job, err := s.analysisRepo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = status
	if errorMsg != "" {
		job.Error = errorMsg
	}
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		now := time.Now()
		job.CompletedAt = &now
	}

	if err := s.analysisRepo.UpdateJob(ctx, job); err != nil {
		return err
	}

	return s.cacheJobStatus(ctx, job)
}

// cacheJobStatus caches job status in Redis
func (s *AnalysisService) cacheJobStatus(ctx context.Context, job *AnalysisJob) error {
	key := fmt.Sprintf("analysis:job:%s", job.ID)
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return s.redisClient.Set(ctx, key, data, 24*time.Hour).Err()
}

// publishAnalysisEvent publishes an event to Kafka
func (s *AnalysisService) publishAnalysisEvent(analysisID, eventType string, data map[string]interface{}) {
	event := map[string]interface{}{
		"analysis_id": analysisID,
		"event_type":  eventType,
		"timestamp":   time.Now(),
		"data":        data,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		s.logger.Errorf("Failed to marshal event: %v", err)
		return
	}

	msg := kafka.Message{
		Key:   []byte(analysisID),
		Value: eventData,
	}

	if err := s.kafkaWriter.WriteMessages(context.Background(), msg); err != nil {
		s.logger.Errorf("Failed to publish event: %v", err)
	}
}

// GetAnalysis retrieves analysis job details
func (s *AnalysisService) GetAnalysis(ctx context.Context, analysisID string) (*AnalysisJob, error) {
	// Try cache first
	key := fmt.Sprintf("analysis:job:%s", analysisID)
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err == nil {
		var job AnalysisJob
		if err := json.Unmarshal(data, &job); err == nil {
			return &job, nil
		}
	}

	// Fallback to database
	return s.analysisRepo.GetJob(ctx, analysisID)
}

// CancelAnalysis cancels a running analysis
func (s *AnalysisService) CancelAnalysis(ctx context.Context, analysisID string) error {
	// Get cancel function
	if cancel, ok := s.cancelFuncs.Load(analysisID); ok {
		if cancelFunc, ok := cancel.(context.CancelFunc); ok {
			cancelFunc()
		}
	}

	// Update status
	return s.updateJobStatus(ctx, analysisID, StatusCancelled, "Analysis cancelled by user")
}