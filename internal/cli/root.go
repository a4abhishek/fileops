package cli

import (
	"context"
	"fmt"

	"github.com/a4abhishek/fileops/internal/config"
	"github.com/a4abhishek/fileops/internal/logger"
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
		NewCleanCommand(ctx, cfg, log),
		NewDedupCommand(ctx, cfg, log),
		NewConsolidateCommand(ctx, cfg, log),
		NewSimilarImagesCommand(ctx, cfg, log),
		NewOrganizeCommand(ctx, cfg, log),
		NewPipelineCommand(ctx, cfg, log),
		NewChownCommand(ctx, cfg, log),
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
