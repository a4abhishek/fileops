package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/a4abhishek/fileops/pkg/domain"
	"github.com/a4abhishek/fileops/pkg/progress"
)

// ParseSize parses a size string (e.g., "64MB", "1GB") to bytes
func ParseSize(sizeStr string, defaultSize int64) int64 {
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

// FormatBytes formats a byte count into a human-readable string
func FormatBytes(bytes int64) string {
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

// DisplayOperationStart shows initial operation information
func DisplayOperationStart(operation, paths string, dryRun bool, params map[string]interface{}) {
	operationIcon := map[string]string{
		"cleanup":       "🧹",
		"deduplication": "🔍",
		"consolidation": "📦",
		"organization":  "📁",
		"similarity":    "🖼️",
	}

	icon := operationIcon[operation]
	if icon == "" {
		icon = "⚙️"
	}

	fmt.Printf("%s Starting %s...\n", icon, operation)
	if dryRun {
		fmt.Printf("📋 DRY RUN MODE: No changes will be made\n")
	}
	fmt.Printf("📂 Target paths: %s\n", paths)

	// Display operation-specific parameters
	for key, value := range params {
		fmt.Printf("📊 %s: %v\n", capitalizeFirst(strings.ReplaceAll(key, "_", " ")), value)
	}
	fmt.Println()
}

// DisplayOperationComplete shows completion summary
func DisplayOperationComplete(operation string, duration time.Duration, summary string) {
	operationIcon := map[string]string{
		"cleanup":       "🧹",
		"deduplication": "🔍",
		"consolidation": "📦",
		"organization":  "📁",
		"similarity":    "🖼️",
	}

	icon := operationIcon[operation]
	if icon == "" {
		icon = "⚙️"
	}

	fmt.Printf("\n\n%s ✅ %s completed successfully!\n", icon, capitalizeFirst(operation))
	if summary != "" {
		fmt.Printf("📊 %s\n", summary)
	}
	fmt.Printf("⏱️  Total time: %v\n", duration.Round(time.Millisecond))
}

// MonitorProgress displays generic real-time progress updates
func MonitorProgress(ctx context.Context, tracker *progress.Tracker, operationID, operationType string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastItemsProcessed int64
	var lastBytesProcessed int64
	var lastUpdate time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if info := tracker.GetProgress(operationID); info != nil {
				// Calculate processing speeds
				var itemsPerSec, bytesPerSec float64
				if !lastUpdate.IsZero() {
					duration := time.Since(lastUpdate).Seconds()
					if duration > 0 {
						if info.ItemsProcessed > lastItemsProcessed {
							itemsPerSec = float64(info.ItemsProcessed-lastItemsProcessed) / duration
						}
						if info.BytesProcessed > lastBytesProcessed {
							bytesPerSec = float64(info.BytesProcessed-lastBytesProcessed) / duration
						}
					}
				}

				// Operation-specific progress display
				switch operationType {
				case "cleanup":
					displayCleanupProgress(info, itemsPerSec)
				case "deduplication":
					displayDedupProgress(info, itemsPerSec, bytesPerSec)
				default:
					displayGenericProgress(info, itemsPerSec, bytesPerSec)
				}

				lastItemsProcessed = info.ItemsProcessed
				lastBytesProcessed = info.BytesProcessed
				lastUpdate = time.Now()
			}
		}
	}
}

func displayCleanupProgress(info *domain.ProgressInfo, itemsPerSec float64) {
	if info.TotalItems > 0 {
		percentage := float64(info.ItemsProcessed) / float64(info.TotalItems) * 100
		fmt.Printf("\r🔄 Scanning: %.1f%% (%d/%d directories",
			percentage, info.ItemsProcessed, info.TotalItems)
	} else {
		fmt.Printf("\r🔄 Processing: %d directories", info.ItemsProcessed)
	}

	if itemsPerSec > 0 {
		fmt.Printf(", %.0f dirs/sec", itemsPerSec)
	}

	if info.CurrentStep != "" {
		fmt.Printf(" - %s", info.CurrentStep)
	}

	if info.EstimatedETA != nil && *info.EstimatedETA > 0 {
		fmt.Printf(", ETA: %v", info.EstimatedETA.Round(time.Second))
	}

	fmt.Print(")")
}

func displayDedupProgress(info *domain.ProgressInfo, itemsPerSec, bytesPerSec float64) {
	if info.TotalItems > 0 {
		percentage := float64(info.ItemsProcessed) / float64(info.TotalItems) * 100
		fmt.Printf("\r🔍 Scanning: %.1f%% (%d/%d files",
			percentage, info.ItemsProcessed, info.TotalItems)
	} else {
		fmt.Printf("\r🔍 Processing: %d files", info.ItemsProcessed)
	}

	if info.BytesProcessed > 0 {
		fmt.Printf(", %s processed", FormatBytes(info.BytesProcessed))
	}

	if itemsPerSec > 0 {
		fmt.Printf(", %.0f files/sec", itemsPerSec)
	}
	if bytesPerSec > 0 {
		fmt.Printf(", %s/sec", FormatBytes(int64(bytesPerSec)))
	}

	if info.CurrentStep != "" {
		fmt.Printf(" - %s", info.CurrentStep)
	}

	if info.EstimatedETA != nil && *info.EstimatedETA > 0 {
		fmt.Printf(", ETA: %v", info.EstimatedETA.Round(time.Second))
	}

	fmt.Print(")")
}

func displayGenericProgress(info *domain.ProgressInfo, itemsPerSec, bytesPerSec float64) {
	if info.TotalItems > 0 {
		percentage := float64(info.ItemsProcessed) / float64(info.TotalItems) * 100
		fmt.Printf("\r⚙️  Progress: %.1f%% (%d/%d items",
			percentage, info.ItemsProcessed, info.TotalItems)
	} else {
		fmt.Printf("\r⚙️  Processing: %d items", info.ItemsProcessed)
	}

	if info.BytesProcessed > 0 {
		fmt.Printf(", %s processed", FormatBytes(info.BytesProcessed))
	}

	if itemsPerSec > 0 {
		fmt.Printf(", %.0f items/sec", itemsPerSec)
	}
	if bytesPerSec > 0 {
		fmt.Printf(", %s/sec", FormatBytes(int64(bytesPerSec)))
	}

	if info.CurrentStep != "" {
		fmt.Printf(" - %s", info.CurrentStep)
	}

	if info.EstimatedETA != nil && *info.EstimatedETA > 0 {
		fmt.Printf(", ETA: %v", info.EstimatedETA.Round(time.Second))
	}

	fmt.Print(")")
}
