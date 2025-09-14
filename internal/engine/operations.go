package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/a4abhishek/fileops/pkg/domain"
)

// DeduplicationFactory creates deduplication operations
type DeduplicationFactory struct {
	engine *Engine
}

// Create creates a new deduplication operation
func (df *DeduplicationFactory) Create(id string, config domain.OperationConfig) (domain.Operation, error) {
	return NewDeduplicationOperation(id, config, df.engine), nil
}

// Validate validates the deduplication configuration
func (df *DeduplicationFactory) Validate(config domain.OperationConfig) error {
	// Add deduplication-specific validation
	if config.HashAlgorithm == "" {
		config.HashAlgorithm = "blake2b" // Default
	}
	return nil
}

// DeduplicationOperation implements file deduplication functionality
type DeduplicationOperation struct {
	*BaseOperation
	duplicateGroups []domain.DuplicateGroup
	totalSize       int64
	saveableSize    int64
}

// NewDeduplicationOperation creates a new deduplication operation
func NewDeduplicationOperation(id string, config domain.OperationConfig, engine *Engine) *DeduplicationOperation {
	base := NewBaseOperation(id, domain.OperationDeduplication, config, engine)
	return &DeduplicationOperation{
		BaseOperation:   base,
		duplicateGroups: make([]domain.DuplicateGroup, 0),
	}
}

// Execute performs the deduplication operation
func (do *DeduplicationOperation) Execute(ctx context.Context, config domain.OperationConfig) (*domain.OperationResult, error) {
	// Start tracking progress
	tracker := do.engine.progressTracker.StartOperation(do.id, domain.OperationDeduplication, 5)
	do.SetTracker(tracker)

	tracker.UpdateStep("Scanning files")

	// Basic implementation - TODO: Add full deduplication logic
	details := map[string]interface{}{
		"duplicate_groups": len(do.duplicateGroups),
		"total_size":       do.totalSize,
		"saveable_size":    do.saveableSize,
		"hash_algorithm":   config.HashAlgorithm,
		"dry_run":          config.DryRun,
	}

	summary := "Deduplication operation completed (basic implementation)"
	tracker.UpdateStep("Completed")
	return do.CreateResult(domain.StatusCompleted, summary, details), nil
}

// Validate validates the deduplication operation configuration
func (do *DeduplicationOperation) Validate(config domain.OperationConfig) error {
	return do.ValidateConfig()
}

// EstimateProgress provides an estimate of the operation's scope
func (do *DeduplicationOperation) EstimateProgress(config domain.OperationConfig) (*domain.ProgressInfo, error) {
	return &domain.ProgressInfo{
		ID:            do.id,
		OperationType: domain.OperationDeduplication,
		Status:        domain.StatusPending,
		TotalSteps:    5,
		TotalItems:    1000, // Estimated
	}, nil
}

// ConsolidationFactory creates consolidation operations
type ConsolidationFactory struct {
	engine *Engine
}

// Create creates a new consolidation operation
func (cf *ConsolidationFactory) Create(id string, config domain.OperationConfig) (domain.Operation, error) {
	return NewConsolidationOperation(id, config, cf.engine), nil
}

// Validate validates the consolidation configuration
func (cf *ConsolidationFactory) Validate(config domain.OperationConfig) error {
	// Add consolidation-specific validation
	return nil
}

// ConsolidationOperation implements file consolidation functionality
type ConsolidationOperation struct {
	*BaseOperation
	movedFiles  []string
	copiedFiles []string
}

// NewConsolidationOperation creates a new consolidation operation
func NewConsolidationOperation(id string, config domain.OperationConfig, engine *Engine) *ConsolidationOperation {
	base := NewBaseOperation(id, domain.OperationConsolidation, config, engine)
	return &ConsolidationOperation{
		BaseOperation: base,
		movedFiles:    make([]string, 0),
		copiedFiles:   make([]string, 0),
	}
}

// Execute performs the consolidation operation
func (co *ConsolidationOperation) Execute(ctx context.Context, config domain.OperationConfig) (*domain.OperationResult, error) {
	// Start tracking progress
	tracker := co.engine.progressTracker.StartOperation(co.id, domain.OperationConsolidation, 4)
	co.SetTracker(tracker)

	tracker.UpdateStep("Planning consolidation")

	// Implementation will be added in the next phase
	details := map[string]interface{}{
		"moved_files":  co.movedFiles,
		"copied_files": co.copiedFiles,
		"dry_run":      config.DryRun,
	}

	summary := "Consolidation operation completed (placeholder implementation)"
	return co.CreateResult(domain.StatusCompleted, summary, details), nil
}

// Validate validates the consolidation operation configuration
func (co *ConsolidationOperation) Validate(config domain.OperationConfig) error {
	return co.ValidateConfig()
}

// EstimateProgress provides an estimate of the operation's scope
func (co *ConsolidationOperation) EstimateProgress(config domain.OperationConfig) (*domain.ProgressInfo, error) {
	return &domain.ProgressInfo{
		ID:            co.id,
		OperationType: domain.OperationConsolidation,
		Status:        domain.StatusPending,
		TotalSteps:    4,
		TotalItems:    500, // Estimated
	}, nil
}

// OwnershipFactory creates ownership change operations
type OwnershipFactory struct {
	engine *Engine
}

// Create creates a new ownership operation
func (of *OwnershipFactory) Create(id string, config domain.OperationConfig) (domain.Operation, error) {
	return NewOwnershipOperation(id, config, of.engine), nil
}

// Validate validates the ownership configuration
func (of *OwnershipFactory) Validate(config domain.OperationConfig) error {
	// Add ownership-specific validation
	if config.CustomSettings == nil {
		return fmt.Errorf("ownership operation requires custom settings")
	}

	if _, ok := config.CustomSettings["target_user"]; !ok {
		return fmt.Errorf("target_user parameter is required")
	}

	return nil
}

// OwnershipOperation implements file/directory ownership change functionality
type OwnershipOperation struct {
	*BaseOperation
	changedItems []string
	skippedItems []string
	errors       []string
}

// NewOwnershipOperation creates a new ownership operation
func NewOwnershipOperation(id string, config domain.OperationConfig, engine *Engine) *OwnershipOperation {
	base := NewBaseOperation(id, domain.OperationOwnership, config, engine)
	return &OwnershipOperation{
		BaseOperation: base,
		changedItems:  make([]string, 0),
		skippedItems:  make([]string, 0),
		errors:        make([]string, 0),
	}
}

// Execute performs the ownership change operation
func (oo *OwnershipOperation) Execute(ctx context.Context, config domain.OperationConfig) (*domain.OperationResult, error) {
	// Start tracking progress
	tracker := oo.engine.progressTracker.StartOperation(oo.id, domain.OperationOwnership, 3)
	oo.SetTracker(tracker)

	defer func() {
		tracker.Complete()
	}()

	// Step 1: Scan files/directories
	tracker.UpdateStep("Scanning paths...")

	var filesToProcess []string
	var scanErrors []error
	var scannedCount int64

	for _, pattern := range config.IncludePatterns {
		err := oo.engine.fileSystem.Walk(ctx, pattern, func(path string, info *domain.FileInfo, err error) error {
			if err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("error walking %s: %w", path, err))
				return nil // Continue walking
			}

			// Apply exclude patterns
			excluded := false
			for _, excludePattern := range config.ExcludePatterns {
				if matched, _ := filepath.Match(excludePattern, path); matched {
					excluded = true
					break
				}
			}

			if !excluded {
				// If not recursive, only include direct children
				if !config.Recursive && len(strings.Split(path, string(filepath.Separator))) > len(strings.Split(pattern, string(filepath.Separator)))+1 {
					return nil
				}

				filesToProcess = append(filesToProcess, path)
			}

			// Update progress in real-time during scanning
			scannedCount++
			if scannedCount%100 == 0 || scannedCount < 100 {
				tracker.UpdateProgress(scannedCount, 0, 0, 0) // TotalItems unknown during scanning
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk path %s: %w", pattern, err)
		}
	}

	// Report any scan errors but continue
	if len(scanErrors) > 0 {
		for _, err := range scanErrors {
			oo.errors = append(oo.errors, err.Error())
		}
	}

	// Update progress with final scan count and total items
	tracker.UpdateProgress(int64(len(filesToProcess)), int64(len(filesToProcess)), 0, 0)

	// Step 2: Change ownership
	tracker.UpdateStep("Changing ownership...")

	// Get ownership parameters
	uid, _ := config.CustomSettings["uid"].(int)
	gid, _ := config.CustomSettings["gid"].(int)

	processed := int64(0)
	for _, file := range filesToProcess {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		err := oo.changeFileOwnership(file, uid, gid, config.DryRun)
		if err != nil {
			oo.errors = append(oo.errors, fmt.Sprintf("%s: %v", file, err))
			oo.skippedItems = append(oo.skippedItems, file)
		} else {
			oo.changedItems = append(oo.changedItems, file)
		}

		processed++
		tracker.UpdateProgress(processed, int64(len(filesToProcess)), 0, 0)
	}

	// Step 3: Complete
	tracker.UpdateStep("Finalizing...")

	// Create result
	summary := fmt.Sprintf("Ownership change (%s): %d items changed, %d skipped, %d errors",
		map[bool]string{true: "dry run", false: "completed"}[config.DryRun],
		len(oo.changedItems), len(oo.skippedItems), len(oo.errors))

	result := &domain.OperationResult{
		ID:            oo.id,
		OperationType: domain.OperationOwnership,
		Status:        domain.StatusCompleted,
		StartTime:     oo.startTime,
		EndTime:       time.Now(),
		Summary:       summary,
		Details: map[string]interface{}{
			"changed_items": oo.changedItems,
			"skipped_items": oo.skippedItems,
			"errors":        oo.errors,
		},
	}

	return result, nil
}

// Validate validates the ownership operation configuration
func (oo *OwnershipOperation) Validate(config domain.OperationConfig) error {
	if config.CustomSettings == nil {
		return fmt.Errorf("ownership operation requires custom settings")
	}

	if _, ok := config.CustomSettings["target_user"]; !ok {
		return fmt.Errorf("target_user parameter is required")
	}

	return nil
}

// changeFileOwnership changes ownership of a single file/directory
func (oo *OwnershipOperation) changeFileOwnership(path string, uid, gid int, dryRun bool) error {
	if dryRun {
		return nil // Don't actually change anything in dry run
	}

	if runtime.GOOS == "windows" {
		// On Windows, use simplified ownership change
		return oo.changeOwnershipWindows(path)
	} else {
		// On Unix-like systems, use chown
		return syscall.Chown(path, uid, gid)
	}
}

// changeOwnershipWindows handles ownership changes on Windows
func (oo *OwnershipOperation) changeOwnershipWindows(path string) error {
	// On Windows, ownership changes require different handling
	// For now, we'll implement a basic version

	// Try to ensure current user has access
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// For a basic implementation on Windows, we'll just verify access
	// A full implementation would use Windows security APIs
	_ = fileInfo
	return nil
}

// EstimateProgress provides an estimate of the operation's scope
func (oo *OwnershipOperation) EstimateProgress(config domain.OperationConfig) (*domain.ProgressInfo, error) {
	return &domain.ProgressInfo{
		ID:            oo.id,
		OperationType: domain.OperationOwnership,
		Status:        domain.StatusPending,
		TotalSteps:    3,
		TotalItems:    100, // Estimated
	}, nil
}
