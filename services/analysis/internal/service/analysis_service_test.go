package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/sa3d-modernized/sa3d/services/analysis/internal/repository"
	"github.com/sa3d-modernized/sa3d/services/analysis/internal/service"
)

// Mock repositories
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) GetByID(ctx context.Context, id string) (*repository.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Project), args.Error(1)
}

func (m *MockProjectRepository) GetProjectFiles(ctx context.Context, projectID string) ([]*repository.ProjectFile, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.ProjectFile), args.Error(1)
}

type MockAnalysisRepository struct {
	mock.Mock
}

func (m *MockAnalysisRepository) CreateJob(ctx context.Context, job *service.AnalysisJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockAnalysisRepository) GetJob(ctx context.Context, jobID string) (*service.AnalysisJob, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AnalysisJob), args.Error(1)
}

func (m *MockAnalysisRepository) UpdateJob(ctx context.Context, job *service.AnalysisJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

type MockMetricsRepository struct {
	mock.Mock
}

func (m *MockMetricsRepository) SaveAnalysisResults(ctx context.Context, analysisID string, results []*service.FileAnalysisResult, aggregateMetrics map[string]interface{}) error {
	args := m.Called(ctx, analysisID, results, aggregateMetrics)
	return args.Error(0)
}

// Test AnalysisService
func TestAnalysisService_StartAnalysis(t *testing.T) {
	// Setup
	mockProjectRepo := new(MockProjectRepository)
	mockAnalysisRepo := new(MockAnalysisRepository)
	mockMetricsRepo := new(MockMetricsRepository)
	
	// Create a test Redis client (you might want to use miniredis for testing)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	// Create a test Kafka writer
	kafkaWriter := &kafka.Writer{
		Addr:  kafka.TCP("localhost:9092"),
		Topic: "test-topic",
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	analysisService := service.NewAnalysisService(
		mockProjectRepo,
		mockAnalysisRepo,
		mockMetricsRepo,
		redisClient,
		kafkaWriter,
		logger,
	)

	// Test data
	projectID := "test-project-123"
	project := &repository.Project{
		ID:   projectID,
		Name: "Test Project",
	}

	// Mock expectations
	mockProjectRepo.On("GetByID", mock.Anything, projectID).Return(project, nil)
	mockAnalysisRepo.On("CreateJob", mock.Anything, mock.AnythingOfType("*service.AnalysisJob")).Return(nil)

	// Execute
	ctx := context.Background()
	job, err := analysisService.StartAnalysis(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, projectID, job.ProjectID)
	assert.Equal(t, service.StatusPending, job.Status)
	assert.NotEmpty(t, job.ID)

	// Verify mocks
	mockProjectRepo.AssertExpectations(t)
	mockAnalysisRepo.AssertExpectations(t)
}

func TestAnalysisService_StartAnalysis_ProjectNotFound(t *testing.T) {
	// Setup
	mockProjectRepo := new(MockProjectRepository)
	mockAnalysisRepo := new(MockAnalysisRepository)
	mockMetricsRepo := new(MockMetricsRepository)
	
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	kafkaWriter := &kafka.Writer{
		Addr:  kafka.TCP("localhost:9092"),
		Topic: "test-topic",
	}
	
	logger := logrus.New()
	
	analysisService := service.NewAnalysisService(
		mockProjectRepo,
		mockAnalysisRepo,
		mockMetricsRepo,
		redisClient,
		kafkaWriter,
		logger,
	)

	// Test data
	projectID := "non-existent-project"

	// Mock expectations
	mockProjectRepo.On("GetByID", mock.Anything, projectID).Return(nil, nil)

	// Execute
	ctx := context.Background()
	job, err := analysisService.StartAnalysis(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "project not found")

	// Verify mocks
	mockProjectRepo.AssertExpectations(t)
}

func TestAnalysisService_GetAnalysis(t *testing.T) {
	// Setup
	mockProjectRepo := new(MockProjectRepository)
	mockAnalysisRepo := new(MockAnalysisRepository)
	mockMetricsRepo := new(MockMetricsRepository)
	
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	kafkaWriter := &kafka.Writer{
		Addr:  kafka.TCP("localhost:9092"),
		Topic: "test-topic",
	}
	
	logger := logrus.New()
	
	analysisService := service.NewAnalysisService(
		mockProjectRepo,
		mockAnalysisRepo,
		mockMetricsRepo,
		redisClient,
		kafkaWriter,
		logger,
	)

	// Test data
	analysisID := "test-analysis-123"
	expectedJob := &service.AnalysisJob{
		ID:        analysisID,
		ProjectID: "test-project",
		Status:    service.StatusCompleted,
		StartedAt: time.Now().Add(-10 * time.Minute),
		Progress:  100,
	}

	// Mock expectations
	mockAnalysisRepo.On("GetJob", mock.Anything, analysisID).Return(expectedJob, nil)

	// Execute
	ctx := context.Background()
	job, err := analysisService.GetAnalysis(ctx, analysisID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, analysisID, job.ID)
	assert.Equal(t, service.StatusCompleted, job.Status)

	// Verify mocks
	mockAnalysisRepo.AssertExpectations(t)
}

func TestAnalysisService_CancelAnalysis(t *testing.T) {
	// Setup
	mockProjectRepo := new(MockProjectRepository)
	mockAnalysisRepo := new(MockAnalysisRepository)
	mockMetricsRepo := new(MockMetricsRepository)
	
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	kafkaWriter := &kafka.Writer{
		Addr:  kafka.TCP("localhost:9092"),
		Topic: "test-topic",
	}
	
	logger := logrus.New()
	
	analysisService := service.NewAnalysisService(
		mockProjectRepo,
		mockAnalysisRepo,
		mockMetricsRepo,
		redisClient,
		kafkaWriter,
		logger,
	)

	// Test data
	analysisID := "test-analysis-123"
	runningJob := &service.AnalysisJob{
		ID:        analysisID,
		ProjectID: "test-project",
		Status:    service.StatusRunning,
		StartedAt: time.Now().Add(-5 * time.Minute),
	}

	// Mock expectations
	mockAnalysisRepo.On("GetJob", mock.Anything, analysisID).Return(runningJob, nil)
	mockAnalysisRepo.On("UpdateJob", mock.Anything, mock.AnythingOfType("*service.AnalysisJob")).Return(nil)

	// Execute
	ctx := context.Background()
	err := analysisService.CancelAnalysis(ctx, analysisID)

	// Assert
	assert.NoError(t, err)

	// Verify mocks
	mockAnalysisRepo.AssertExpectations(t)
}