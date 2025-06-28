package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel contains common fields for all models
type BaseModel struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// BeforeCreate hook to set UUID
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// User represents a user in the system
type User struct {
	BaseModel
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role" gorm:"default:'user'"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	LastLogin time.Time `json:"last_login,omitempty"`
	Projects  []Project `json:"projects,omitempty" gorm:"many2many:user_projects;"`
}

// Project represents a software project
type Project struct {
	BaseModel
	Name         string     `json:"name" gorm:"not null"`
	Description  string     `json:"description"`
	Language     string     `json:"language" gorm:"not null"`
	Repository   string     `json:"repository"`
	Branch       string     `json:"branch" gorm:"default:'main'"`
	CreatedBy    uuid.UUID  `json:"created_by" gorm:"not null"`
	Creator      *User      `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Users        []User     `json:"users,omitempty" gorm:"many2many:user_projects;"`
	Analyses     []Analysis `json:"analyses,omitempty"`
	LastAnalysis *Analysis  `json:"last_analysis,omitempty"`
	Settings     ProjectSettings `json:"settings" gorm:"embedded"`
}

// ProjectSettings contains project-specific settings
type ProjectSettings struct {
	AutoAnalyze      bool   `json:"auto_analyze" gorm:"default:false"`
	AnalyzeFrequency string `json:"analyze_frequency" gorm:"default:'daily'"`
	IgnorePatterns   string `json:"ignore_patterns"`
	MaxFileSize      int64  `json:"max_file_size" gorm:"default:10485760"` // 10MB
}

// Analysis represents a code analysis run
type Analysis struct {
	BaseModel
	ProjectID   uuid.UUID       `json:"project_id" gorm:"not null"`
	Project     *Project        `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Status      AnalysisStatus  `json:"status" gorm:"not null"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Error       string          `json:"error,omitempty"`
	Results     AnalysisResults `json:"results" gorm:"type:jsonb"`
	Metrics     ProjectMetrics  `json:"metrics" gorm:"embedded"`
}

// AnalysisStatus represents the status of an analysis
type AnalysisStatus string

const (
	AnalysisStatusPending   AnalysisStatus = "pending"
	AnalysisStatusRunning   AnalysisStatus = "running"
	AnalysisStatusCompleted AnalysisStatus = "completed"
	AnalysisStatusFailed    AnalysisStatus = "failed"
	AnalysisStatusCancelled AnalysisStatus = "cancelled"
)

// AnalysisResults contains the results of code analysis
type AnalysisResults struct {
	Files         []FileInfo         `json:"files"`
	Dependencies  []Dependency       `json:"dependencies"`
	Components    []Component        `json:"components"`
	Relationships []Relationship     `json:"relationships"`
	Issues        []Issue            `json:"issues"`
	Statistics    AnalysisStatistics `json:"statistics"`
}

// FileInfo represents information about a source file
type FileInfo struct {
	Path         string         `json:"path"`
	Language     string         `json:"language"`
	Size         int64          `json:"size"`
	Lines        int            `json:"lines"`
	Functions    []FunctionInfo `json:"functions"`
	Classes      []ClassInfo    `json:"classes"`
	Imports      []string       `json:"imports"`
	Complexity   int            `json:"complexity"`
	Dependencies []string       `json:"dependencies"`
}

// FunctionInfo represents information about a function
type FunctionInfo struct {
	Name       string   `json:"name"`
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	Parameters []string `json:"parameters"`
	Returns    []string `json:"returns"`
	Complexity int      `json:"complexity"`
	Calls      []string `json:"calls"`
}

// ClassInfo represents information about a class
type ClassInfo struct {
	Name       string         `json:"name"`
	StartLine  int            `json:"start_line"`
	EndLine    int            `json:"end_line"`
	Methods    []FunctionInfo `json:"methods"`
	Properties []string       `json:"properties"`
	Extends    string         `json:"extends,omitempty"`
	Implements []string       `json:"implements,omitempty"`
}

// Dependency represents a project dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // internal, external, system
	Source  string `json:"source"`
}

// Component represents a high-level component in the architecture
type Component struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"` // service, library, module, package
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
	Size        int64    `json:"size"`
	Complexity  int      `json:"complexity"`
}

// Relationship represents a relationship between components
type Relationship struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"` // depends_on, calls, implements, extends
	Strength int    `json:"strength"`
}

// Issue represents a code quality issue
type Issue struct {
	Type        string `json:"type"` // bug, vulnerability, code_smell, duplication
	Severity    string `json:"severity"` // critical, major, minor, info
	File        string `json:"file"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Message     string `json:"message"`
	Rule        string `json:"rule"`
	Effort      string `json:"effort"` // time to fix
}

// AnalysisStatistics contains overall statistics
type AnalysisStatistics struct {
	TotalFiles      int            `json:"total_files"`
	TotalLines      int            `json:"total_lines"`
	TotalFunctions  int            `json:"total_functions"`
	TotalClasses    int            `json:"total_classes"`
	TotalComponents int            `json:"total_components"`
	Languages       map[string]int `json:"languages"`
	FileTypes       map[string]int `json:"file_types"`
}

// ProjectMetrics contains calculated metrics for a project
type ProjectMetrics struct {
	LinesOfCode         int     `json:"lines_of_code"`
	CyclomaticComplexity int     `json:"cyclomatic_complexity"`
	MaintainabilityIndex float64 `json:"maintainability_index"`
	TechnicalDebt        float64 `json:"technical_debt"`
	CodeSmells           int     `json:"code_smells"`
	Bugs                 int     `json:"bugs"`
	Vulnerabilities      int     `json:"vulnerabilities"`
	SecurityHotspots     int     `json:"security_hotspots"`
	Coverage             float64 `json:"coverage"`
	DuplicationRatio     float64 `json:"duplication_ratio"`
}

// Visualization represents a 3D visualization configuration
type Visualization struct {
	BaseModel
	ProjectID    uuid.UUID           `json:"project_id" gorm:"not null"`
	Project      *Project            `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Name         string              `json:"name" gorm:"not null"`
	Description  string              `json:"description"`
	Layout       string              `json:"layout" gorm:"default:'force-directed'"`
	Settings     VisualizationSettings `json:"settings" gorm:"type:jsonb"`
	IsDefault    bool                `json:"is_default" gorm:"default:false"`
	SharedWith   []User              `json:"shared_with,omitempty" gorm:"many2many:visualization_shares;"`
}

// VisualizationSettings contains visualization-specific settings
type VisualizationSettings struct {
	ColorScheme      string                 `json:"color_scheme"`
	NodeSize         string                 `json:"node_size"` // loc, complexity, dependencies
	EdgeThickness    string                 `json:"edge_thickness"` // calls, dependencies
	ShowLabels       bool                   `json:"show_labels"`
	ShowMetrics      bool                   `json:"show_metrics"`
	FilterComponents []string               `json:"filter_components"`
	HighlightIssues  bool                   `json:"highlight_issues"`
	CustomColors     map[string]string      `json:"custom_colors"`
	CameraPosition   map[string]float64     `json:"camera_position"`
}

// Session represents a collaboration session
type Session struct {
	BaseModel
	ProjectID    uuid.UUID      `json:"project_id" gorm:"not null"`
	Project      *Project       `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	HostID       uuid.UUID      `json:"host_id" gorm:"not null"`
	Host         *User          `json:"host,omitempty" gorm:"foreignKey:HostID"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	Participants []Participant  `json:"participants,omitempty"`
	Annotations  []Annotation   `json:"annotations,omitempty"`
}

// Participant represents a user in a collaboration session
type Participant struct {
	BaseModel
	SessionID  uuid.UUID  `json:"session_id" gorm:"not null"`
	Session    *Session   `json:"session,omitempty" gorm:"foreignKey:SessionID"`
	UserID     uuid.UUID  `json:"user_id" gorm:"not null"`
	User       *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	JoinedAt   time.Time  `json:"joined_at"`
	LeftAt     *time.Time `json:"left_at,omitempty"`
	IsActive   bool       `json:"is_active" gorm:"default:true"`
	CursorData string     `json:"cursor_data" gorm:"type:jsonb"`
}

// Annotation represents a comment or note in a visualization
type Annotation struct {
	BaseModel
	SessionID   uuid.UUID  `json:"session_id" gorm:"not null"`
	Session     *Session   `json:"session,omitempty" gorm:"foreignKey:SessionID"`
	UserID      uuid.UUID  `json:"user_id" gorm:"not null"`
	User        *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	ComponentID string     `json:"component_id"`
	Position    string     `json:"position" gorm:"type:jsonb"`
	Content     string     `json:"content" gorm:"not null"`
	Type        string     `json:"type"` // comment, issue, suggestion
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy  *uuid.UUID `json:"resolved_by,omitempty"`
}