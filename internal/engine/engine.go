package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/a4abhishek/fileops/internal/logger"
	"github.com/a4abhishek/fileops/pkg/domain"
	"github.com/a4abhishek/fileops/pkg/filesystem"
	"github.com/a4abhishek/fileops/pkg/progress"
)

// Engine is the core operation engine that orchestrates file operations
type Engine struct {
	fileSystem      domain.FileSystem
	progressTracker *progress.Tracker
	logger          *logger.Logger
	operations      map[domain.OperationType]OperationFactory
	mu              sync.RWMutex
}

// OperationFactory creates specific operation implementations
type OperationFactory interface {
	Create(id string, config domain.OperationConfig) (domain.Operation, error)
	Validate(config domain.OperationConfig) error
}

// NewEngine creates a new operation engine
func NewEngine(fs domain.FileSystem, tracker *progress.Tracker, log *logger.Logger) *Engine {
	if fs == nil {
		fs = filesystem.NewOSFileSystem(64 * 1024 * 1024) // 64MB default chunk size
	}
	if tracker == nil {
		tracker = progress.NewTracker()
	}

	engine := &Engine{
		fileSystem:      fs,
		progressTracker: tracker,
		logger:          log,
		operations:      make(map[domain.OperationType]OperationFactory),
	}

	// Register built-in operation factories
	engine.RegisterOperation(domain.OperationCleanup, &CleanupFactory{engine: engine})
	engine.RegisterOperation(domain.OperationDeduplication, &DeduplicationFactory{engine: engine})
	engine.RegisterOperation(domain.OperationConsolidation, &ConsolidationFactory{engine: engine})
	engine.RegisterOperation(domain.OperationOwnership, &OwnershipFactory{engine: engine})

	return engine
}

// RegisterOperation registers an operation factory
func (e *Engine) RegisterOperation(operationType domain.OperationType, factory OperationFactory) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.operations[operationType] = factory
}

// ExecuteOperation executes an operation with the given configuration
func (e *Engine) ExecuteOperation(ctx context.Context, operationType domain.OperationType, config domain.OperationConfig) (*domain.OperationResult, error) {
	e.mu.RLock()
	factory, exists := e.operations[operationType]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("operation type %s not supported", operationType)
	}

	// Validate configuration
	if err := factory.Validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Generate operation ID
	operationID := generateOperationID(operationType)

	// Create operation
	operation, err := factory.Create(operationID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create operation: %w", err)
	}

	e.logger.Info("Starting operation", "id", operationID, "type", operationType)

	// Start progress tracking
	tracker := e.progressTracker.StartOperation(operationID, operationType, 5) // Default 5 steps

	// Execute operation
	result, err := operation.Execute(ctx, config)
	if err != nil {
		tracker.Fail(err.Error())
		e.logger.Error("Operation failed", "id", operationID, "error", err)
		return nil, err
	}

	tracker.Complete()
	e.logger.Info("Operation completed", "id", operationID, "duration", result.Duration)

	return result, nil
}

// GetSupportedOperations returns list of supported operation types
func (e *Engine) GetSupportedOperations() []domain.OperationType {
	e.mu.RLock()
	defer e.mu.RUnlock()

	operations := make([]domain.OperationType, 0, len(e.operations))
	for opType := range e.operations {
		operations = append(operations, opType)
	}
	return operations
}

// GetProgressTracker returns the progress tracker
func (e *Engine) GetProgressTracker() *progress.Tracker {
	return e.progressTracker
}

// GetFileSystem returns the file system interface
func (e *Engine) GetFileSystem() domain.FileSystem {
	return e.fileSystem
}

// GetLogger returns the logger
func (e *Engine) GetLogger() *logger.Logger {
	return e.logger
}

// generateOperationID generates a unique operation ID
func generateOperationID(operationType domain.OperationType) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s", operationType, timestamp)
}

// BaseOperation provides common functionality for all operations
type BaseOperation struct {
	id            string
	operationType domain.OperationType
	config        domain.OperationConfig
	engine        *Engine
	tracker       *progress.OperationTracker
	startTime     time.Time
	cancelled     bool
	mu            sync.RWMutex
}

// NewBaseOperation creates a new base operation
func NewBaseOperation(id string, operationType domain.OperationType, config domain.OperationConfig, engine *Engine) *BaseOperation {
	return &BaseOperation{
		id:            id,
		operationType: operationType,
		config:        config,
		engine:        engine,
		startTime:     time.Now(),
	}
}

// ID returns the operation ID
func (bo *BaseOperation) ID() string {
	return bo.id
}

// Type returns the operation type
func (bo *BaseOperation) Type() domain.OperationType {
	return bo.operationType
}

// Cancel cancels the operation
func (bo *BaseOperation) Cancel() error {
	bo.mu.Lock()
	defer bo.mu.Unlock()

	bo.cancelled = true
	if bo.tracker != nil {
		bo.tracker.Cancel()
	}
	return nil
}

// Pause pauses the operation
func (bo *BaseOperation) Pause() error {
	if bo.tracker != nil {
		bo.tracker.Pause()
	}
	return nil
}

// Resume resumes the operation
func (bo *BaseOperation) Resume() error {
	if bo.tracker != nil {
		bo.tracker.Resume()
	}
	return nil
}

// IsCancelled checks if the operation was cancelled
func (bo *BaseOperation) IsCancelled() bool {
	bo.mu.RLock()
	defer bo.mu.RUnlock()
	return bo.cancelled
}

// SetTracker sets the progress tracker
func (bo *BaseOperation) SetTracker(tracker *progress.OperationTracker) {
	bo.tracker = tracker
}

// GetTracker returns the progress tracker
func (bo *BaseOperation) GetTracker() *progress.OperationTracker {
	return bo.tracker
}

// CheckContext checks if the context is cancelled or operation is paused
func (bo *BaseOperation) CheckContext(ctx context.Context) error {
	// Check for cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check if operation was cancelled
	if bo.IsCancelled() {
		return fmt.Errorf("operation cancelled")
	}

	// Handle pause/resume
	if bo.tracker != nil && bo.tracker.IsPaused() {
		bo.tracker.WaitForResume()
	}

	return nil
}

// UpdateProgress updates the operation progress
func (bo *BaseOperation) UpdateProgress(step string, itemsProcessed, totalItems, bytesProcessed, totalBytes int64) {
	if bo.tracker != nil {
		bo.tracker.UpdateStep(step)
		bo.tracker.UpdateProgress(itemsProcessed, totalItems, bytesProcessed, totalBytes)
	}
}

// IncrementProgress increments the progress counters
func (bo *BaseOperation) IncrementProgress(items, bytes int64) {
	if bo.tracker != nil {
		bo.tracker.IncrementProgress(items, bytes)
	}
}

// SetTotals sets the total counters
func (bo *BaseOperation) SetTotals(totalItems, totalBytes int64) {
	if bo.tracker != nil {
		bo.tracker.SetTotals(totalItems, totalBytes)
	}
}

// AddError adds an error to the operation
func (bo *BaseOperation) AddError(err error) {
	if bo.tracker != nil {
		bo.tracker.AddError(err.Error())
	}
	bo.engine.logger.Error("Operation error", "id", bo.id, "error", err)
}

// CreateResult creates an operation result
func (bo *BaseOperation) CreateResult(status domain.OperationStatus, summary string, details map[string]interface{}) *domain.OperationResult {
	endTime := time.Now()
	duration := endTime.Sub(bo.startTime)

	result := &domain.OperationResult{
		ID:            bo.id,
		OperationType: bo.operationType,
		Status:        status,
		StartTime:     bo.startTime,
		EndTime:       endTime,
		Duration:      duration,
		Summary:       summary,
		Details:       details,
	}

	if bo.tracker != nil {
		progress := bo.tracker.GetProgressInfo()
		result.ItemsProcessed = progress.ItemsProcessed
		result.BytesProcessed = progress.BytesProcessed

		// Add any errors from the tracker
		if errorCount, exists := progress.Details["error_count"]; exists && errorCount.(int) > 0 {
			if allErrors, exists := progress.Details["all_errors"]; exists {
				errorStrings := allErrors.([]string)
				result.Errors = make([]domain.OperationError, len(errorStrings))
				for i, errStr := range errorStrings {
					result.Errors[i] = domain.OperationError{
						Operation:   bo.operationType.String(),
						Error:       errStr,
						Timestamp:   time.Now(),
						Recoverable: false,
					}
				}
			}
		}
	}

	return result
}

// ValidateConfig provides common configuration validation
func (bo *BaseOperation) ValidateConfig() error {
	config := bo.config

	// Validate parallelism
	if config.Parallelism < 0 {
		return fmt.Errorf("parallelism cannot be negative")
	}
	if config.Parallelism == 0 {
		// Set default based on CPU cores
		config.Parallelism = 4 // Default
	}

	// Validate file size limits
	if config.MaxFileSize > 0 && config.MinFileSize > 0 && config.MinFileSize > config.MaxFileSize {
		return fmt.Errorf("min file size cannot be greater than max file size")
	}

	// Validate thresholds
	if config.SimilarityThreshold < 0.0 || config.SimilarityThreshold > 1.0 {
		return fmt.Errorf("similarity threshold must be between 0.0 and 1.0")
	}

	// Validate hash algorithm
	validAlgorithms := []string{"md5", "sha1", "sha256", "sha512", "blake2b", "xxhash64", "crc32"}
	if config.HashAlgorithm != "" {
		valid := false
		for _, alg := range validAlgorithms {
			if config.HashAlgorithm == alg {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("unsupported hash algorithm: %s", config.HashAlgorithm)
		}
	}

	return nil
}

// OperationManager manages multiple concurrent operations
type OperationManager struct {
	engine           *Engine
	activeOperations map[string]domain.Operation
	mu               sync.RWMutex
	maxConcurrent    int
}

// NewOperationManager creates a new operation manager
func NewOperationManager(engine *Engine, maxConcurrent int) *OperationManager {
	if maxConcurrent <= 0 {
		maxConcurrent = 4 // Default
	}

	return &OperationManager{
		engine:           engine,
		activeOperations: make(map[string]domain.Operation),
		maxConcurrent:    maxConcurrent,
	}
}

// SubmitOperation submits an operation for execution
func (om *OperationManager) SubmitOperation(ctx context.Context, operationType domain.OperationType, config domain.OperationConfig) (string, error) {
	om.mu.Lock()
	defer om.mu.Unlock()

	// Check concurrent limit
	if len(om.activeOperations) >= om.maxConcurrent {
		return "", fmt.Errorf("maximum concurrent operations limit reached (%d)", om.maxConcurrent)
	}

	// Generate operation ID
	operationID := generateOperationID(operationType)

	// Start operation in background
	go func() {
		defer func() {
			om.mu.Lock()
			delete(om.activeOperations, operationID)
			om.mu.Unlock()
		}()

		result, err := om.engine.ExecuteOperation(ctx, operationType, config)
		if err != nil {
			om.engine.logger.Error("Background operation failed", "id", operationID, "error", err)
		} else {
			om.engine.logger.Info("Background operation completed", "id", operationID, "summary", result.Summary)
		}
	}()

	return operationID, nil
}

// GetActiveOperations returns list of active operation IDs
func (om *OperationManager) GetActiveOperations() []string {
	om.mu.RLock()
	defer om.mu.RUnlock()

	operations := make([]string, 0, len(om.activeOperations))
	for id := range om.activeOperations {
		operations = append(operations, id)
	}
	return operations
}

// CancelOperation cancels an active operation
func (om *OperationManager) CancelOperation(operationID string) error {
	om.mu.RLock()
	operation, exists := om.activeOperations[operationID]
	om.mu.RUnlock()

	if !exists {
		return fmt.Errorf("operation %s not found or not active", operationID)
	}

	return operation.Cancel()
}

// PauseOperation pauses an active operation
func (om *OperationManager) PauseOperation(operationID string) error {
	om.mu.RLock()
	operation, exists := om.activeOperations[operationID]
	om.mu.RUnlock()

	if !exists {
		return fmt.Errorf("operation %s not found or not active", operationID)
	}

	return operation.Pause()
}

// ResumeOperation resumes a paused operation
func (om *OperationManager) ResumeOperation(operationID string) error {
	om.mu.RLock()
	operation, exists := om.activeOperations[operationID]
	om.mu.RUnlock()

	if !exists {
		return fmt.Errorf("operation %s not found or not active", operationID)
	}

	return operation.Resume()
}
