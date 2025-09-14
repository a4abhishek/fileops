package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/a4abhishek/fileops/internal/config"
	"github.com/a4abhishek/fileops/internal/engine"
	"github.com/a4abhishek/fileops/internal/logger"
	"github.com/a4abhishek/fileops/pkg/domain"
	"github.com/a4abhishek/fileops/pkg/filesystem"
	"github.com/a4abhishek/fileops/pkg/progress"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for the CLI
func NewRootCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "fileops",
		Short: "Advanced file operations toolkit",
		Long: `FileOps is a high-performance, AI-powered file operations toolkit
for advanced file management, deduplication, and intelligent organization.

Features:
  • Smart cleanup of empty directories
  • Advanced file deduplication with multiple algorithms
  • AI-powered image similarity detection
  • Intelligent file organization
  • Pipeline support for chaining operations
  • High-performance parallel processing`,
		Example: `  # Clean empty directories
  fileops clean /path/to/directory --dry-run

  # Deduplicate files
  fileops dedup /path/to/files --algorithm blake2b

  # Find similar images
  fileops similar-images /photos --threshold 0.85

  # Run a pipeline
  fileops pipeline run cleanup-and-organize.yaml`,
		SilenceUsage: true,
	}

	// Global flags
	rootCmd.PersistentFlags().String("config", "", "config file path")
	rootCmd.PersistentFlags().String("log-level", cfg.Logging.Level, "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
	rootCmd.PersistentFlags().Bool("quiet", false, "quiet output (errors only)")

	// Add subcommands
	rootCmd.AddCommand(
		newCleanCommand(ctx, cfg, log),
		newDedupCommand(ctx, cfg, log),
		newConsolidateCommand(ctx, cfg, log),
		newSimilarImagesCommand(ctx, cfg, log),
		newOrganizeCommand(ctx, cfg, log),
		newPipelineCommand(ctx, cfg, log),
		newVersionCommand(),
	)

	return rootCmd
}

// newVersionCommand creates the version command
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("FileOps v1.0.0")
			fmt.Println("Build: development")
			fmt.Println("Go version:", "go1.21+")
		},
	}
}

// Command implementations will be added in separate files
func newCleanCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
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
			chunkSize := parseSize(cfg.Performance.ChunkSize, 64*1024*1024) // Default 64MB
			fs := filesystem.NewOSFileSystem(chunkSize)
			tracker := progress.NewTracker()
			operationEngine := engine.NewEngine(fs, tracker, log)

			log.Info("🧹 Starting directory cleanup",
				"paths", validPaths,
				"dry_run", dryRun,
				"exclude_patterns", excludePatterns)

			// Execute operation
			result, err := operationEngine.ExecuteOperation(ctx, domain.OperationCleanup, config)
			if err != nil {
				return fmt.Errorf("cleanup operation failed: %w", err)
			}

			// Display results
			log.Info("✅ Cleanup completed", "summary", result.Summary)

			if removedDirs, ok := result.Details["removed_directories"].([]string); ok && len(removedDirs) > 0 {
				fmt.Printf("\n📁 Directories processed:\n")
				for _, dir := range removedDirs {
					if dryRun {
						fmt.Printf("  [DRY RUN] Would remove: %s\n", dir)
					} else {
						fmt.Printf("  ✓ Removed: %s\n", dir)
					}
				}
			}

			if skippedDirs, ok := result.Details["skipped_directories"].([]string); ok && len(skippedDirs) > 0 {
				fmt.Printf("\n⚠️  Skipped directories:\n")
				for _, dir := range skippedDirs {
					fmt.Printf("  - %s\n", dir)
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

func newDedupCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dedup [path...]",
		Short: "Find and remove duplicate files",
		Long: `Find and optionally remove duplicate files using advanced hashing algorithms.

This command uses a multi-stage approach for fast and accurate duplicate detection:
1. Group files by size (instant)
2. Compute fast hash for size-matching files (xxHash64)
3. Compute cryptographic hash for verification (Blake2b/SHA256)
4. Optional byte-by-byte comparison for absolute certainty`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flags
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			algorithm, _ := cmd.Flags().GetString("algorithm")
			threshold, _ := cmd.Flags().GetFloat64("threshold")
			excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
			minSize, _ := cmd.Flags().GetInt64("min-size")
			maxSize, _ := cmd.Flags().GetInt64("max-size")
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
				DryRun:              dryRun,
				Recursive:           true,
				ExcludePatterns:     excludePatterns,
				IncludePatterns:     validPaths,
				HashAlgorithm:       algorithm,
				SimilarityThreshold: threshold,
				MinFileSize:         minSize,
				MaxFileSize:         maxSize,
				Parallelism:         parallelism,
			}

			// Create engine
			chunkSize := parseSize(cfg.Performance.ChunkSize, 64*1024*1024)
			fs := filesystem.NewOSFileSystem(chunkSize)
			tracker := progress.NewTracker()
			operationEngine := engine.NewEngine(fs, tracker, log)

			log.Info("🔍 Starting deduplication",
				"paths", validPaths,
				"algorithm", algorithm,
				"threshold", threshold,
				"dry_run", dryRun)

			// Execute operation
			result, err := operationEngine.ExecuteOperation(ctx, domain.OperationDeduplication, config)
			if err != nil {
				return fmt.Errorf("deduplication operation failed: %w", err)
			}

			// Display results
			log.Info("✅ Deduplication completed", "summary", result.Summary)
			fmt.Printf("\n📊 Deduplication Results:\n")
			fmt.Printf("  Algorithm: %s\n", algorithm)
			fmt.Printf("  Threshold: %.2f\n", threshold)

			if duplicateGroups, ok := result.Details["duplicate_groups"].(int); ok {
				fmt.Printf("  Duplicate groups found: %d\n", duplicateGroups)
			}

			if totalSize, ok := result.Details["total_size"].(int64); ok {
				fmt.Printf("  Total size processed: %s\n", formatBytes(totalSize))
			}

			if saveableSize, ok := result.Details["saveable_size"].(int64); ok {
				fmt.Printf("  Space that can be saved: %s\n", formatBytes(saveableSize))
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().Bool("dry-run", false, "Preview changes without executing them")
	cmd.Flags().String("algorithm", "blake2b", "Hash algorithm (md5, sha1, sha256, sha512, blake2b, xxhash64, crc32)")
	cmd.Flags().Float64("threshold", 0.99, "Similarity threshold for duplicate detection (0.0-1.0)")
	cmd.Flags().StringSlice("exclude", []string{"*.tmp", "*.log", ".DS_Store"}, "Patterns to exclude")
	cmd.Flags().Int64("min-size", 0, "Minimum file size to process (bytes)")
	cmd.Flags().Int64("max-size", 0, "Maximum file size to process (0 = no limit)")
	cmd.Flags().Int("parallelism", runtime.NumCPU(), "Number of parallel workers")

	return cmd
}

func newConsolidateCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "consolidate [sources...] --dest [destination]",
		Short: "Consolidate files from multiple sources",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("📦 Starting file consolidation")
			// Implementation will be added
			return fmt.Errorf("not implemented yet")
		},
	}
}

func newSimilarImagesCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "similar-images [path]",
		Short: "Find similar images using AI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("🖼️ Starting image similarity detection", "path", args[0])
			// Implementation will be added
			return fmt.Errorf("not implemented yet")
		},
	}
}

func newOrganizeCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "organize [path]",
		Short: "Intelligently organize files using AI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("🤖 Starting intelligent organization", "path", args[0])
			// Implementation will be added
			return fmt.Errorf("not implemented yet")
		},
	}
}

func newPipelineCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
	pipelineCmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Manage and run operation pipelines",
	}

	pipelineCmd.AddCommand(
		&cobra.Command{
			Use:   "run [pipeline-file]",
			Short: "Run a pipeline from file",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				log.Info("⚙️ Starting pipeline execution", "file", args[0])
				// Implementation will be added
				return fmt.Errorf("not implemented yet")
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List available pipelines",
			RunE: func(cmd *cobra.Command, args []string) error {
				log.Info("📋 Listing available pipelines")
				// Implementation will be added
				return fmt.Errorf("not implemented yet")
			},
		},
	)

	return pipelineCmd
}

// parseSize parses a size string (e.g., "64MB", "1GB") to bytes
func parseSize(sizeStr string, defaultSize int64) int64 {
	if sizeStr == "" {
		return defaultSize
	}

	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))

	// Handle numeric-only values as bytes
	if val, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return val
	}

	// Parse with units
	var multiplier int64 = 1
	var numStr string

	if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		numStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "B") {
		multiplier = 1
		numStr = strings.TrimSuffix(sizeStr, "B")
	} else {
		// Try to parse as-is
		numStr = sizeStr
	}

	if val, err := strconv.ParseInt(numStr, 10, 64); err == nil {
		return val * multiplier
	}

	return defaultSize
}

// formatBytes formats a byte count into a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
