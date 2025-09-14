package domain

import (
	"context"
	"time"
)

// FileInfo represents metadata about a file or directory
type FileInfo struct {
	Path        string            `json:"path"`
	Name        string            `json:"name"`
	Size        int64             `json:"size"`
	ModTime     time.Time         `json:"mod_time"`
	IsDir       bool              `json:"is_dir"`
	Mode        uint32            `json:"mode"`
	Hash        string            `json:"hash,omitempty"`
	HashType    string            `json:"hash_type,omitempty"`
	MimeType    string            `json:"mime_type,omitempty"`
	Checksum    string            `json:"checksum,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ContentHash string            `json:"content_hash,omitempty"`
	PartialHash string            `json:"partial_hash,omitempty"`
}

// OperationType defines the type of operation being performed
type OperationType string

const (
	OperationCleanup       OperationType = "cleanup"
	OperationDeduplication OperationType = "deduplication"
	OperationConsolidation OperationType = "consolidation"
	OperationSimilarity    OperationType = "similarity"
	OperationOrganization  OperationType = "organization"
	OperationOwnership     OperationType = "ownership"
	OperationPipeline      OperationType = "pipeline"
)

// String returns the string representation of the operation type
func (ot OperationType) String() string {
	return string(ot)
}

// OperationStatus represents the current status of an operation
type OperationStatus string

const (
	StatusPending   OperationStatus = "pending"
	StatusRunning   OperationStatus = "running"
	StatusCompleted OperationStatus = "completed"
	StatusFailed    OperationStatus = "failed"
	StatusCancelled OperationStatus = "cancelled"
	StatusPaused    OperationStatus = "paused"
)

// Priority defines the priority level of an operation
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// ProgressInfo represents progress information for an operation
type ProgressInfo struct {
	ID             string                 `json:"id"`
	OperationType  OperationType          `json:"operation_type"`
	Status         OperationStatus        `json:"status"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        *time.Time             `json:"end_time,omitempty"`
	CurrentStep    string                 `json:"current_step"`
	StepsCompleted int                    `json:"steps_completed"`
	TotalSteps     int                    `json:"total_steps"`
	ItemsProcessed int64                  `json:"items_processed"`
	TotalItems     int64                  `json:"total_items"`
	BytesProcessed int64                  `json:"bytes_processed"`
	TotalBytes     int64                  `json:"total_bytes"`
	Speed          int64                  `json:"speed"` // items per second
	EstimatedETA   *time.Duration         `json:"estimated_eta,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Details        map[string]interface{} `json:"details,omitempty"`
}

// OperationResult represents the result of an operation
type OperationResult struct {
	ID             string                 `json:"id"`
	OperationType  OperationType          `json:"operation_type"`
	Status         OperationStatus        `json:"status"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Duration       time.Duration          `json:"duration"`
	ItemsProcessed int64                  `json:"items_processed"`
	BytesProcessed int64                  `json:"bytes_processed"`
	FilesAffected  []string               `json:"files_affected"`
	Summary        string                 `json:"summary"`
	Details        map[string]interface{} `json:"details"`
	Errors         []OperationError       `json:"errors,omitempty"`
	Warnings       []string               `json:"warnings,omitempty"`
}

// OperationError represents an error that occurred during an operation
type OperationError struct {
	File        string    `json:"file"`
	Operation   string    `json:"operation"`
	Error       string    `json:"error"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

// DuplicateGroup represents a group of duplicate files
type DuplicateGroup struct {
	ID          string     `json:"id"`
	Files       []FileInfo `json:"files"`
	TotalSize   int64      `json:"total_size"`
	SaveablSize int64      `json:"saveable_size"`
	HashType    string     `json:"hash_type"`
	Confidence  float64    `json:"confidence"`
}

// SimilarityGroup represents a group of similar files (mainly images)
type SimilarityGroup struct {
	ID         string     `json:"id"`
	Files      []FileInfo `json:"files"`
	Similarity float64    `json:"similarity"`
	Method     string     `json:"method"`
	Confidence float64    `json:"confidence"`
}

// OrganizationSuggestion represents a suggestion for file organization
type OrganizationSuggestion struct {
	File          FileInfo `json:"file"`
	SuggestedPath string   `json:"suggested_path"`
	Reason        string   `json:"reason"`
	Confidence    float64  `json:"confidence"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	ConflictsWith []string `json:"conflicts_with,omitempty"`
}

// ConsolidationPlan represents a plan for consolidating files
type ConsolidationPlan struct {
	ID          string                   `json:"id"`
	Strategy    string                   `json:"strategy"`
	Destination string                   `json:"destination"`
	Operations  []ConsolidationOperation `json:"operations"`
	TotalFiles  int                      `json:"total_files"`
	TotalSize   int64                    `json:"total_size"`
	Conflicts   []ConflictResolution     `json:"conflicts"`
}

// ConsolidationOperation represents a single file operation in consolidation
type ConsolidationOperation struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Operation  string `json:"operation"` // move, copy, link
	Reason     string `json:"reason"`
}

// ConflictResolution represents how to handle file conflicts
type ConflictResolution struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Resolution string `json:"resolution"` // skip, rename, overwrite, merge
	NewName    string `json:"new_name,omitempty"`
}

// OperationConfig represents configuration for an operation
type OperationConfig struct {
	DryRun              bool                   `json:"dry_run"`
	Recursive           bool                   `json:"recursive"`
	FollowSymlinks      bool                   `json:"follow_symlinks"`
	ExcludePatterns     []string               `json:"exclude_patterns"`
	IncludePatterns     []string               `json:"include_patterns"`
	MaxDepth            int                    `json:"max_depth"`
	MaxFileSize         int64                  `json:"max_file_size"`
	MinFileSize         int64                  `json:"min_file_size"`
	BackupBeforeDelete  bool                   `json:"backup_before_delete"`
	BackupDirectory     string                 `json:"backup_directory"`
	Parallelism         int                    `json:"parallelism"`
	ChunkSize           int64                  `json:"chunk_size"`
	HashAlgorithm       string                 `json:"hash_algorithm"`
	SimilarityThreshold float64                `json:"similarity_threshold"`
	Extensions          map[string]string      `json:"extensions,omitempty"`
	CustomSettings      map[string]interface{} `json:"custom_settings,omitempty"`
}

// Operation represents a file operation that can be executed
type Operation interface {
	// ID returns the unique identifier for this operation
	ID() string

	// Type returns the type of operation
	Type() OperationType

	// Execute runs the operation with the given context and configuration
	Execute(ctx context.Context, config OperationConfig) (*OperationResult, error)

	// Validate checks if the operation can be executed with the given configuration
	Validate(config OperationConfig) error

	// EstimateProgress provides an estimate of the operation's scope
	EstimateProgress(config OperationConfig) (*ProgressInfo, error)

	// Cancel cancels the operation if it's currently running
	Cancel() error

	// Pause pauses the operation if supported
	Pause() error

	// Resume resumes a paused operation
	Resume() error
}

// ProgressReporter allows operations to report their progress
type ProgressReporter interface {
	// ReportProgress updates the current progress
	ReportProgress(progress ProgressInfo) error

	// Subscribe allows clients to subscribe to progress updates
	Subscribe(operationID string) (<-chan ProgressInfo, error)

	// Unsubscribe removes a progress subscription
	Unsubscribe(operationID string) error
}

// FileSystem provides an abstraction over file system operations
type FileSystem interface {
	// Walk traverses the file system starting from the given path
	Walk(ctx context.Context, path string, fn WalkFunc) error

	// Stat returns file information for the given path
	Stat(path string) (*FileInfo, error)

	// Remove removes the file or directory at the given path
	Remove(path string) error

	// RemoveAll removes the directory and all its contents
	RemoveAll(path string) error

	// Move moves a file or directory from source to destination
	Move(source, destination string) error

	// Copy copies a file or directory from source to destination
	Copy(source, destination string) error

	// CreateDir creates a directory at the given path
	CreateDir(path string) error

	// IsEmpty checks if a directory is empty
	IsEmpty(path string) (bool, error)

	// Exists checks if a file or directory exists
	Exists(path string) bool

	// ComputeHash computes the hash of a file
	ComputeHash(path string, algorithm string) (string, error)
}

// WalkFunc is the function signature for file system traversal
type WalkFunc func(path string, info *FileInfo, err error) error

// HashAlgorithm represents supported hash algorithms
type HashAlgorithm string

const (
	HashMD5      HashAlgorithm = "md5"
	HashSHA1     HashAlgorithm = "sha1"
	HashSHA256   HashAlgorithm = "sha256"
	HashSHA512   HashAlgorithm = "sha512"
	HashBlake2b  HashAlgorithm = "blake2b"
	HashXXHash64 HashAlgorithm = "xxhash64"
	HashCRC32    HashAlgorithm = "crc32"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// OperationFactory creates operations based on type and configuration
type OperationFactory interface {
	CreateOperation(operationType OperationType, config OperationConfig) (Operation, error)
	SupportedOperations() []OperationType
}

// Repository provides persistence for operation results and metadata
type Repository interface {
	// SaveResult saves an operation result
	SaveResult(result *OperationResult) error

	// GetResult retrieves an operation result by ID
	GetResult(id string) (*OperationResult, error)

	// ListResults lists all operation results with optional filtering
	ListResults(filter map[string]interface{}) ([]*OperationResult, error)

	// SaveFileInfo saves file metadata
	SaveFileInfo(info *FileInfo) error

	// GetFileInfo retrieves file metadata by path
	GetFileInfo(path string) (*FileInfo, error)

	// SaveDuplicateGroup saves a duplicate group
	SaveDuplicateGroup(group *DuplicateGroup) error

	// GetDuplicateGroups retrieves all duplicate groups
	GetDuplicateGroups() ([]*DuplicateGroup, error)

	// Close closes the repository
	Close() error
}
