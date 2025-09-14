package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/a4abhishek/fileops/pkg/domain"
)

// CleanupFactory creates cleanup operations
type CleanupFactory struct {
	engine *Engine
}

// Create creates a new cleanup operation
func (cf *CleanupFactory) Create(id string, config domain.OperationConfig) (domain.Operation, error) {
	return NewCleanupOperation(id, config, cf.engine), nil
}

// Validate validates the cleanup configuration
func (cf *CleanupFactory) Validate(config domain.OperationConfig) error {
	// Cleanup-specific validation can be added here
	return nil
}

// CleanupOperation implements directory cleanup functionality
type CleanupOperation struct {
	*BaseOperation
	removedDirs   []string
	skippedDirs   []string
	totalDirs     int64
	processedDirs int64
}

// NewCleanupOperation creates a new cleanup operation
func NewCleanupOperation(id string, config domain.OperationConfig, engine *Engine) *CleanupOperation {
	base := NewBaseOperation(id, domain.OperationCleanup, config, engine)
	return &CleanupOperation{
		BaseOperation: base,
		removedDirs:   make([]string, 0),
		skippedDirs:   make([]string, 0),
	}
}

// Execute performs the cleanup operation
func (co *CleanupOperation) Execute(ctx context.Context, config domain.OperationConfig) (*domain.OperationResult, error) {
	// Start tracking progress
	tracker := co.engine.progressTracker.StartOperation(co.id, domain.OperationCleanup, 4)
	co.SetTracker(tracker)

	tracker.UpdateStep("Scanning directories")

	// First pass: count total directories for progress tracking
	if err := co.countDirectories(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to count directories: %w", err)
	}

	tracker.UpdateStep("Identifying empty directories")
	tracker.SetTotals(co.totalDirs, 0)

	// Find empty directories (bottom-up approach)
	emptyDirs, err := co.findEmptyDirectories(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to find empty directories: %w", err)
	}

	tracker.UpdateStep("Processing empty directories")

	// Process empty directories
	if err := co.processEmptyDirectories(ctx, config, emptyDirs); err != nil {
		return nil, fmt.Errorf("failed to process empty directories: %w", err)
	}

	tracker.UpdateStep("Completing cleanup")

	// Create result
	details := map[string]interface{}{
		"removed_directories": co.removedDirs,
		"skipped_directories": co.skippedDirs,
		"total_directories":   co.totalDirs,
		"dry_run":             config.DryRun,
	}

	summary := fmt.Sprintf("Cleanup completed: %d directories removed, %d skipped",
		len(co.removedDirs), len(co.skippedDirs))

	if config.DryRun {
		summary = fmt.Sprintf("Cleanup (dry run): %d directories would be removed, %d skipped",
			len(co.removedDirs), len(co.skippedDirs))
	}

	return co.CreateResult(domain.StatusCompleted, summary, details), nil
}

// Validate validates the cleanup operation configuration
func (co *CleanupOperation) Validate(config domain.OperationConfig) error {
	return co.ValidateConfig()
}

// EstimateProgress provides an estimate of the operation's scope
func (co *CleanupOperation) EstimateProgress(config domain.OperationConfig) (*domain.ProgressInfo, error) {
	// This is a simplified estimate - in a real implementation, we might do a quick scan
	return &domain.ProgressInfo{
		ID:            co.id,
		OperationType: domain.OperationCleanup,
		Status:        domain.StatusPending,
		TotalSteps:    4,
		TotalItems:    100, // Estimated
	}, nil
}

// countDirectories counts the total number of directories for progress tracking
func (co *CleanupOperation) countDirectories(ctx context.Context, config domain.OperationConfig) error {
	pathsToProcess := config.IncludePatterns
	if len(pathsToProcess) == 0 {
		// If no paths specified, we can't proceed
		return fmt.Errorf("no paths specified for cleanup")
	}

	var processedItems int64

	for _, rootPath := range pathsToProcess {
		if err := co.CheckContext(ctx); err != nil {
			return err
		}

		err := co.engine.fileSystem.Walk(ctx, rootPath, func(path string, info *domain.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors during counting
			}

			if info != nil {
				if info.IsDir && co.shouldProcessDirectory(path, config) {
					co.totalDirs++
				}

				// Update progress for every item scanned (files and directories)
				processedItems++
				co.IncrementProgress(1, 0)
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// findEmptyDirectories finds all empty directories using bottom-up traversal
func (co *CleanupOperation) findEmptyDirectories(ctx context.Context, config domain.OperationConfig) ([]string, error) {
	emptyDirs := make([]string, 0)
	pathsToProcess := config.IncludePatterns

	for _, rootPath := range pathsToProcess {
		rootEmptyDirs, err := co.findEmptyDirectoriesInPath(ctx, rootPath, config)
		if err != nil {
			return nil, err
		}
		emptyDirs = append(emptyDirs, rootEmptyDirs...)
	}

	return emptyDirs, nil
}

// findEmptyDirectoriesInPath finds empty directories in a specific path
func (co *CleanupOperation) findEmptyDirectoriesInPath(ctx context.Context, rootPath string, config domain.OperationConfig) ([]string, error) {
	emptyDirs := make([]string, 0)
	dirContents := make(map[string][]string)

	// First, build a map of directory contents
	err := co.engine.fileSystem.Walk(ctx, rootPath, func(path string, info *domain.FileInfo, err error) error {
		if err != nil {
			co.AddError(fmt.Errorf("error accessing %s: %w", path, err))
			return nil // Continue processing
		}

		if err := co.CheckContext(ctx); err != nil {
			return err
		}

		if info != nil {
			dir := filepath.Dir(path)
			if _, exists := dirContents[dir]; !exists {
				dirContents[dir] = make([]string, 0)
			}

			// Only add to parent if it's not the path itself
			if path != dir {
				dirContents[dir] = append(dirContents[dir], path)
			}

			// If it's a directory, initialize its entry
			if info.IsDir {
				if _, exists := dirContents[path]; !exists {
					dirContents[path] = make([]string, 0)
				}
			}
		}

		co.processedDirs++
		co.IncrementProgress(1, 0)

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Now identify empty directories (bottom-up)
	for dir, contents := range dirContents {
		if len(contents) == 0 && co.shouldProcessDirectory(dir, config) {
			// Check if directory actually exists and is empty
			if exists := co.engine.fileSystem.Exists(dir); exists {
				if isEmpty, err := co.engine.fileSystem.IsEmpty(dir); err == nil && isEmpty {
					emptyDirs = append(emptyDirs, dir)
				}
			}
		}
	}

	return emptyDirs, nil
}

// processEmptyDirectories processes the identified empty directories
func (co *CleanupOperation) processEmptyDirectories(ctx context.Context, config domain.OperationConfig, emptyDirs []string) error {
	for _, dir := range emptyDirs {
		if err := co.CheckContext(ctx); err != nil {
			return err
		}

		if co.shouldProcessDirectory(dir, config) {
			if config.DryRun {
				co.removedDirs = append(co.removedDirs, dir)
				co.engine.logger.Info("Would remove empty directory", "path", dir)
			} else {
				// Create backup if requested
				if config.BackupBeforeDelete && config.BackupDirectory != "" {
					// For empty directories, we might just log the action
					co.engine.logger.Info("Empty directory marked for removal", "path", dir)
				}

				// Remove the directory
				if err := co.engine.fileSystem.Remove(dir); err != nil {
					co.AddError(fmt.Errorf("failed to remove directory %s: %w", dir, err))
					co.skippedDirs = append(co.skippedDirs, dir)
				} else {
					co.removedDirs = append(co.removedDirs, dir)
					co.engine.logger.Info("Removed empty directory", "path", dir)
				}
			}
		} else {
			co.skippedDirs = append(co.skippedDirs, dir)
		}

		co.IncrementProgress(1, 0)
	}

	return nil
}

// shouldProcessDirectory checks if a directory should be processed based on configuration
func (co *CleanupOperation) shouldProcessDirectory(dir string, config domain.OperationConfig) bool {
	// Check exclude patterns
	for _, pattern := range config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(dir)); matched {
			return false
		}
		// Also check if the pattern matches any part of the path
		if strings.Contains(strings.ToLower(dir), strings.ToLower(pattern)) {
			return false
		}
	}

	// Skip system directories and hidden directories by default
	dirName := filepath.Base(dir)
	systemDirs := []string{".git", ".svn", ".hg", "node_modules", "__pycache__", ".DS_Store"}
	for _, sysDir := range systemDirs {
		if dirName == sysDir {
			return false
		}
	}

	// Skip hidden directories (starting with .) unless explicitly included
	if strings.HasPrefix(dirName, ".") && dirName != "." && dirName != ".." {
		// Check if explicitly included
		included := false
		for _, pattern := range config.IncludePatterns {
			if matched, _ := filepath.Match(pattern, dirName); matched {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	return true
}
