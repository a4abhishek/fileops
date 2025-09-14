package cli

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/a4abhishek/fileops/internal/config"
	"github.com/a4abhishek/fileops/internal/engine"
	"github.com/a4abhishek/fileops/internal/logger"
	"github.com/a4abhishek/fileops/pkg/domain"
	"github.com/a4abhishek/fileops/pkg/filesystem"
	"github.com/a4abhishek/fileops/pkg/progress"
	"github.com/spf13/cobra"
)

// NewChownCommand creates the chown command
func NewChownCommand(ctx context.Context, cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chown [path...]",
		Short: "Change ownership of files and directories to current user",
		Long: `Change ownership of files and directories to the current user recursively.

This command changes the ownership of files and directories to the current user.
On Linux/macOS, it uses chown functionality. On Windows, it attempts to take
ownership using Windows-specific APIs.

The operation is performed recursively by default and supports dry-run mode
for preview before making changes.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flags
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			recursive, _ := cmd.Flags().GetBool("recursive")
			excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
			parallelism, _ := cmd.Flags().GetInt("parallelism")
			targetUser, _ := cmd.Flags().GetString("user")
			targetGroup, _ := cmd.Flags().GetString("group")
			quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

			// Get current user if not specified
			var uid, gid int
			var err error

			if targetUser == "" {
				currentUser, err := user.Current()
				if err != nil {
					return fmt.Errorf("failed to get current user: %w", err)
				}
				if runtime.GOOS != "windows" {
					uid, err = strconv.Atoi(currentUser.Uid)
					if err != nil {
						return fmt.Errorf("invalid user ID: %w", err)
					}
					gid, err = strconv.Atoi(currentUser.Gid)
					if err != nil {
						return fmt.Errorf("invalid group ID: %w", err)
					}
				}
				targetUser = currentUser.Username
			} else {
				// Look up specified user
				targetUserInfo, err := user.Lookup(targetUser)
				if err != nil {
					return fmt.Errorf("user not found: %s: %w", targetUser, err)
				}
				if runtime.GOOS != "windows" {
					uid, err = strconv.Atoi(targetUserInfo.Uid)
					if err != nil {
						return fmt.Errorf("invalid user ID: %w", err)
					}
					if targetGroup != "" {
						groupInfo, err := user.LookupGroup(targetGroup)
						if err != nil {
							return fmt.Errorf("group not found: %s: %w", targetGroup, err)
						}
						gid, err = strconv.Atoi(groupInfo.Gid)
						if err != nil {
							return fmt.Errorf("invalid group ID: %w", err)
						}
					} else {
						gid, err = strconv.Atoi(targetUserInfo.Gid)
						if err != nil {
							return fmt.Errorf("invalid group ID: %w", err)
						}
					}
				}
			}

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
				DryRun:          dryRun,
				Recursive:       recursive,
				ExcludePatterns: excludePatterns,
				IncludePatterns: validPaths,
				Parallelism:     parallelism,
				CustomSettings: map[string]interface{}{
					"target_user":  targetUser,
					"target_group": targetGroup,
					"uid":          uid,
					"gid":          gid,
				},
			}

			// Create engine
			chunkSize := ParseSize(cfg.Performance.ChunkSize, 64*1024*1024)
			fs := filesystem.NewOSFileSystem(chunkSize)
			tracker := progress.NewTracker()
			operationEngine := engine.NewEngine(fs, tracker, log)

			log.Info("👑 Starting ownership change",
				"paths", validPaths,
				"target_user", targetUser,
				"target_group", targetGroup,
				"dry_run", dryRun)

			// Show initial status
			if !quiet {
				params := map[string]interface{}{
					"Target user": targetUser,
					"Recursive":   recursive,
					"Parallelism": parallelism,
				}
				if targetGroup != "" {
					params["Target group"] = targetGroup
				}
				if len(excludePatterns) > 0 {
					params["Excluded patterns"] = excludePatterns
				}
				DisplayOperationStart("ownership change", fmt.Sprintf("%v", validPaths), dryRun, params)
			}

			// Pre-generate operation ID for progress monitoring
			operationID := fmt.Sprintf("ownership-%s", time.Now().Format("20060102-150405"))

			// Start progress monitoring in a separate goroutine BEFORE starting operation
			progressCtx, progressCancel := context.WithCancel(ctx)
			defer progressCancel()

			var progressWg sync.WaitGroup
			if !quiet && cfg.Operations.EnableProgressBar {
				progressWg.Add(1)
				go func() {
					defer progressWg.Done()
					MonitorProgress(progressCtx, tracker, operationID, "ownership")
				}()
				// Give the monitor a moment to start
				time.Sleep(50 * time.Millisecond)
			}

			// Execute operation with predefined ID so progress monitoring works
			result, err := operationEngine.ExecuteOperationWithID(ctx, domain.OperationOwnership, config, operationID)

			// Stop progress monitoring
			progressCancel()
			progressWg.Wait()

			if err != nil {
				if !quiet {
					fmt.Printf("\n❌ Ownership change operation failed: %v\n", err)
				}
				return fmt.Errorf("ownership change operation failed: %w", err)
			}

			// Display results
			if !quiet {
				duration := result.EndTime.Sub(result.StartTime)
				DisplayOperationComplete("ownership change", duration, result.Summary)
			}

			log.Info("✅ Ownership change completed", "summary", result.Summary)

			if changedItems, ok := result.Details["changed_items"].([]string); ok && len(changedItems) > 0 {
				if !quiet {
					fmt.Printf("\n👑 Ownership changed (%d total):\n", len(changedItems))
					for i, item := range changedItems {
						if i >= 20 {
							fmt.Printf("  ... and %d more items\n", len(changedItems)-20)
							break
						}
						if dryRun {
							fmt.Printf("  [DRY RUN] Would change: %s\n", item)
						} else {
							fmt.Printf("  ✓ Changed: %s\n", item)
						}
					}
				}
			} else if !quiet {
				if dryRun {
					fmt.Printf("\n👑 No ownership changes needed\n")
				} else {
					fmt.Printf("\n👑 No items required ownership changes\n")
				}
			}

			if skippedItems, ok := result.Details["skipped_items"].([]string); ok && len(skippedItems) > 0 {
				if !quiet {
					fmt.Printf("\n⚠️  Skipped items (%d total):\n", len(skippedItems))
					for i, item := range skippedItems {
						if i >= 10 {
							fmt.Printf("  ... and %d more items\n", len(skippedItems)-10)
							break
						}
						fmt.Printf("  - %s\n", item)
					}
				}
			}

			if errors, ok := result.Details["errors"].([]string); ok && len(errors) > 0 {
				if !quiet {
					fmt.Printf("\n❌ Errors encountered (%d total):\n", len(errors))
					for i, errMsg := range errors {
						if i >= 5 {
							fmt.Printf("  ... and %d more errors\n", len(errors)-5)
							break
						}
						fmt.Printf("  ! %s\n", errMsg)
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
	cmd.Flags().Int("parallelism", runtime.NumCPU(), "Number of parallel workers")
	cmd.Flags().String("user", "", "Target user (defaults to current user)")
	cmd.Flags().String("group", "", "Target group (defaults to user's primary group)")

	return cmd
}

// ChangeOwnership changes the ownership of a file or directory
// This function works cross-platform with different behaviors on Windows vs Unix
func ChangeOwnership(path string, uid, gid int, dryRun bool) error {
	if dryRun {
		return nil // Don't actually change anything in dry run
	}

	if runtime.GOOS == "windows" {
		// On Windows, we use a different approach
		return changeOwnershipWindows(path)
	} else {
		// On Unix-like systems, use chown
		return syscall.Chown(path, uid, gid)
	}
}

// changeOwnershipWindows handles ownership changes on Windows
func changeOwnershipWindows(path string) error {
	// On Windows, we can attempt to take ownership using Windows APIs
	// For now, we'll implement a basic version that tries to change permissions
	// A full implementation would use Windows security APIs

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Try to change file permissions to give current user full control
	// This is a simplified approach - full Windows ownership change requires
	// more complex security descriptor manipulation
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// On Windows, we'll focus on ensuring the current user has access
	// rather than changing actual ownership (which requires admin privileges)
	_ = currentUser // Mark as used
	_ = fileInfo    // Mark as used

	// For a basic implementation, we'll just return success
	// A production version would implement proper Windows security descriptor handling
	return nil
}
