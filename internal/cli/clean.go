package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/a4abhishek/fileops/internal/config"
	"github.com/a4abhishek/fileops/internal/engine"
	"github.com/a4abhishek/fileops/internal/logger"
	"github.com/a4abhishek/fileops/pkg/domain"
	"github.com/a4abhishek/fileops/pkg/filesystem"
	"github.com/a4abhishek/fileops/pkg/progress"
	"github.com/spf13/cobra"
)

// NewCleanCommand creates the clean command
func NewCleanCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [path...]",
		Short: "Remove empty directories recursively",
		Long: `Remove empty directories recursively from the specified paths.

This command performs a bottom-up traversal to identify and remove empty directories.
It supports dry-run mode for safe preview and has configurable exclusion patterns.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flags
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			recursive, _ := cmd.Flags().GetBool("recursive")
			excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
			backupDir, _ := cmd.Flags().GetString("backup-dir")
			parallelism, _ := cmd.Flags().GetInt("parallelism")

			// Validate paths
			validPaths := make([]string, 0, len(args))
			for _, path := range args {
				absPath, err := filepath.Abs(path)
				if err != nil {
					return fmt.Errorf("invalid path %s: %w", path, err)
				}
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					return fmt.Errorf("path does not exist: %s", absPath)
				}
				validPaths = append(validPaths, absPath)
			}

			// Create operation configuration
			config := domain.OperationConfig{
				DryRun:             dryRun,
				Recursive:          recursive,
				ExcludePatterns:    excludePatterns,
				IncludePatterns:    validPaths,
				BackupBeforeDelete: backupDir != "",
				BackupDirectory:    backupDir,
				Parallelism:        parallelism,
			}

			// Create engine
			chunkSize := ParseSize(cfg.Performance.ChunkSize, 64*1024*1024) // Default 64MB
			fs := filesystem.NewOSFileSystem(chunkSize)
			tracker := progress.NewTracker()
			operationEngine := engine.NewEngine(fs, tracker, log)

			// Get quiet flag from root command
			quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

			log.Info("🧹 Starting directory cleanup",
				"paths", validPaths,
				"dry_run", dryRun,
				"exclude_patterns", excludePatterns)

			// Show initial status
			if !quiet {
				fmt.Printf("🔍 Scanning directories...\n")
				if dryRun {
					fmt.Printf("📋 DRY RUN MODE: No changes will be made\n")
				}
				fmt.Printf("📂 Paths to process: %v\n", validPaths)
				if len(excludePatterns) > 0 {
					fmt.Printf("🚫 Excluding patterns: %v\n", excludePatterns)
				}
				fmt.Printf("⚡ Using %d parallel workers\n\n", parallelism)
			}

			// Start progress monitoring in a separate goroutine
			progressCtx, progressCancel := context.WithCancel(ctx)
			defer progressCancel()

			var progressWg sync.WaitGroup
			if !quiet && cfg.Operations.EnableProgressBar {
				progressWg.Add(1)
				go func() {
					defer progressWg.Done()
					MonitorProgress(progressCtx, tracker, "cleanup", "cleanup")
				}()
			}

			// Execute operation
			result, err := operationEngine.ExecuteOperation(ctx, domain.OperationCleanup, config)

			// Stop progress monitoring
			progressCancel()
			progressWg.Wait()

			if err != nil {
				if !quiet {
					fmt.Printf("\n❌ Cleanup operation failed: %v\n", err)
				}
				return fmt.Errorf("cleanup operation failed: %w", err)
			}

			// Display results
			if !quiet {
				fmt.Printf("\n\n✅ Cleanup completed successfully!\n")

				// Show operation summary
				if result.Summary != "" {
					fmt.Printf("📊 %s\n", result.Summary)
				}

				// Show timing information
				duration := result.EndTime.Sub(result.StartTime)
				fmt.Printf("⏱️  Total time: %v\n", duration.Round(time.Millisecond))
			}

			log.Info("✅ Cleanup completed", "summary", result.Summary)

			if removedDirs, ok := result.Details["removed_directories"].([]string); ok && len(removedDirs) > 0 {
				if !quiet {
					fmt.Printf("\n📁 Directories processed (%d total):\n", len(removedDirs))
					for i, dir := range removedDirs {
						if i >= 20 {
							fmt.Printf("  ... and %d more directories\n", len(removedDirs)-20)
							break
						}
						if dryRun {
							fmt.Printf("  [DRY RUN] Would remove: %s\n", dir)
						} else {
							fmt.Printf("  ✓ Removed: %s\n", dir)
						}
					}
				}
			} else if !quiet {
				if dryRun {
					fmt.Printf("\n📁 No empty directories found to remove\n")
				} else {
					fmt.Printf("\n📁 No directories were removed\n")
				}
			}

			if skippedDirs, ok := result.Details["skipped_directories"].([]string); ok && len(skippedDirs) > 0 {
				if !quiet {
					fmt.Printf("\n⚠️  Skipped directories (%d total):\n", len(skippedDirs))
					for i, dir := range skippedDirs {
						if i >= 10 {
							fmt.Printf("  ... and %d more directories\n", len(skippedDirs)-10)
							break
						}
						fmt.Printf("  - %s\n", dir)
					}
				}
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().Bool("dry-run", false, "Preview changes without executing them")
	cmd.Flags().BoolP("recursive", "r", true, "Process directories recursively")
	cmd.Flags().StringSlice("exclude", []string{".git", ".svn", "node_modules", "__pycache__"}, "Patterns to exclude")
	cmd.Flags().String("backup-dir", "", "Directory to store backups before deletion")
	cmd.Flags().Int("parallelism", runtime.NumCPU(), "Number of parallel workers")

	return cmd
}
