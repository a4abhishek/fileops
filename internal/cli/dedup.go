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

// NewDedupCommand creates the dedup command
func NewDedupCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
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
			chunkSize := ParseSize(cfg.Performance.ChunkSize, 64*1024*1024)
			fs := filesystem.NewOSFileSystem(chunkSize)
			tracker := progress.NewTracker()
			operationEngine := engine.NewEngine(fs, tracker, log)

			// Get quiet flag from root command
			quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

			log.Info("🔍 Starting deduplication",
				"paths", validPaths,
				"algorithm", algorithm,
				"threshold", threshold,
				"dry_run", dryRun)

			// Show initial status
			if !quiet {
				fmt.Printf("🔍 Starting file deduplication...\n")
				if dryRun {
					fmt.Printf("📋 DRY RUN MODE: No files will be deleted\n")
				}
				fmt.Printf("📂 Paths to scan: %v\n", validPaths)
				fmt.Printf("🔢 Hash algorithm: %s\n", algorithm)
				fmt.Printf("📊 Similarity threshold: %.2f\n", threshold)
				if len(excludePatterns) > 0 {
					fmt.Printf("🚫 Excluding patterns: %v\n", excludePatterns)
				}
				if minSize > 0 {
					fmt.Printf("📏 Minimum file size: %s\n", FormatBytes(minSize))
				}
				if maxSize > 0 {
					fmt.Printf("📏 Maximum file size: %s\n", FormatBytes(maxSize))
				}
				fmt.Printf("⚡ Using %d parallel workers\n\n", parallelism)
			}

			// Pre-generate operation ID for progress monitoring
			operationID := fmt.Sprintf("deduplication-%s", time.Now().Format("20060102-150405"))

			// Start progress monitoring in a separate goroutine BEFORE starting operation
			progressCtx, progressCancel := context.WithCancel(ctx)
			defer progressCancel()

			var progressWg sync.WaitGroup
			if !quiet && cfg.Operations.EnableProgressBar {
				progressWg.Add(1)
				go func() {
					defer progressWg.Done()
					MonitorProgress(progressCtx, tracker, operationID, "deduplication")
				}()
				// Give the monitor a moment to start
				time.Sleep(50 * time.Millisecond)
			}

			// Execute operation with predefined ID so progress monitoring works
			result, err := operationEngine.ExecuteOperationWithID(ctx, domain.OperationDeduplication, config, operationID)

			// Stop progress monitoring
			progressCancel()
			progressWg.Wait()

			if err != nil {
				if !quiet {
					fmt.Printf("\n❌ Deduplication operation failed: %v\n", err)
				}
				return fmt.Errorf("deduplication operation failed: %w", err)
			}

			// Display results
			if !quiet {
				fmt.Printf("\n\n✅ Deduplication completed successfully!\n")

				// Show timing information
				duration := result.EndTime.Sub(result.StartTime)
				fmt.Printf("⏱️  Total time: %v\n\n", duration.Round(time.Millisecond))

				fmt.Printf("📊 Deduplication Results:\n")
				fmt.Printf("  🔢 Algorithm: %s\n", algorithm)
				fmt.Printf("  📊 Threshold: %.2f\n", threshold)
			}

			log.Info("✅ Deduplication completed", "summary", result.Summary)

			if duplicateGroups, ok := result.Details["duplicate_groups"].(int); ok && !quiet {
				fmt.Printf("  🔍 Duplicate groups found: %d\n", duplicateGroups)
			}

			if totalSize, ok := result.Details["total_size"].(int64); ok && !quiet {
				fmt.Printf("  📦 Total size processed: %s\n", FormatBytes(totalSize))
			}

			if saveableSize, ok := result.Details["saveable_size"].(int64); ok {
				fmt.Printf("  Space that can be saved: %s\n", FormatBytes(saveableSize))
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
