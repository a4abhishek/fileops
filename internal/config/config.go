package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Performance Performance `mapstructure:"performance"`
	Operations  Operations  `mapstructure:"operations"`
	AI          AI          `mapstructure:"ai"`
	Logging     Logging     `mapstructure:"logging"`
	Plugins     Plugins     `mapstructure:"plugins"`
}

type Performance struct {
	MaxWorkers  int    `mapstructure:"max_workers"`
	MemoryLimit string `mapstructure:"memory_limit"`
	ChunkSize   string `mapstructure:"chunk_size"`
	CacheSize   string `mapstructure:"cache_size"`
}

type Operations struct {
	HashAlgorithm       string  `mapstructure:"hash_algorithm"`
	DuplicateThreshold  float64 `mapstructure:"duplicate_threshold"`
	SimilarityThreshold float64 `mapstructure:"similarity_threshold"`
	EnableProgressBar   bool    `mapstructure:"enable_progress_bar"`
	BackupBeforeDelete  bool    `mapstructure:"backup_before_delete"`
}

type AI struct {
	Enabled          bool   `mapstructure:"enabled"`
	ModelCache       string `mapstructure:"model_cache"`
	PythonServiceURL string `mapstructure:"python_service_url"`
	AutoStartService bool   `mapstructure:"auto_start_service"`
}

type Logging struct {
	Level   string `mapstructure:"level"`
	File    string `mapstructure:"file"`
	MaxSize string `mapstructure:"max_size"`
	Format  string `mapstructure:"format"`
	Console bool   `mapstructure:"console"`
}

type Plugins struct {
	Enabled          []string `mapstructure:"enabled"`
	CustomPluginsDir string   `mapstructure:"custom_plugins_dir"`
}

// Default configuration values
func defaultConfig() *Config {
	return &Config{
		Performance: Performance{
			MaxWorkers:  0, // Auto-detect
			MemoryLimit: "80%",
			ChunkSize:   "64MB",
			CacheSize:   "1GB",
		},
		Operations: Operations{
			HashAlgorithm:       "blake2b",
			DuplicateThreshold:  0.99,
			SimilarityThreshold: 0.85,
			EnableProgressBar:   true,
			BackupBeforeDelete:  true,
		},
		AI: AI{
			Enabled:          true,
			ModelCache:       "./models",
			PythonServiceURL: "http://localhost:8001",
			AutoStartService: true,
		},
		Logging: Logging{
			Level:   "info",
			File:    "fileops.log",
			MaxSize: "100MB",
			Format:  "json",
			Console: true,
		},
		Plugins: Plugins{
			Enabled:          []string{"dedup", "cleanup", "organize"},
			CustomPluginsDir: "./plugins",
		},
	}
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set defaults
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config paths
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.fileops")
	viper.AddConfigPath("/etc/fileops")

	// Environment variable support
	viper.SetEnvPrefix("FILEOPS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	cfg := defaultConfig()
	setDefaults(cfg)

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is okay, we'll use defaults
	}

	// Unmarshal into struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Post-process and validate
	if err := postProcess(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values in viper
func setDefaults(cfg *Config) {
	viper.SetDefault("performance.max_workers", cfg.Performance.MaxWorkers)
	viper.SetDefault("performance.memory_limit", cfg.Performance.MemoryLimit)
	viper.SetDefault("performance.chunk_size", cfg.Performance.ChunkSize)
	viper.SetDefault("performance.cache_size", cfg.Performance.CacheSize)

	viper.SetDefault("operations.hash_algorithm", cfg.Operations.HashAlgorithm)
	viper.SetDefault("operations.duplicate_threshold", cfg.Operations.DuplicateThreshold)
	viper.SetDefault("operations.similarity_threshold", cfg.Operations.SimilarityThreshold)
	viper.SetDefault("operations.enable_progress_bar", cfg.Operations.EnableProgressBar)
	viper.SetDefault("operations.backup_before_delete", cfg.Operations.BackupBeforeDelete)

	viper.SetDefault("ai.enabled", cfg.AI.Enabled)
	viper.SetDefault("ai.model_cache", cfg.AI.ModelCache)
	viper.SetDefault("ai.python_service_url", cfg.AI.PythonServiceURL)
	viper.SetDefault("ai.auto_start_service", cfg.AI.AutoStartService)

	viper.SetDefault("logging.level", cfg.Logging.Level)
	viper.SetDefault("logging.file", cfg.Logging.File)
	viper.SetDefault("logging.max_size", cfg.Logging.MaxSize)
	viper.SetDefault("logging.format", cfg.Logging.Format)
	viper.SetDefault("logging.console", cfg.Logging.Console)

	viper.SetDefault("plugins.enabled", cfg.Plugins.Enabled)
	viper.SetDefault("plugins.custom_plugins_dir", cfg.Plugins.CustomPluginsDir)
}

// postProcess handles post-processing and validation
func postProcess(cfg *Config) error {
	// Auto-detect CPU cores if not specified
	if cfg.Performance.MaxWorkers == 0 {
		cfg.Performance.MaxWorkers = runtime.NumCPU()
	}

	// Expand paths
	if cfg.AI.ModelCache != "" {
		if expanded, err := expandPath(cfg.AI.ModelCache); err == nil {
			cfg.AI.ModelCache = expanded
		}
	}

	if cfg.Plugins.CustomPluginsDir != "" {
		if expanded, err := expandPath(cfg.Plugins.CustomPluginsDir); err == nil {
			cfg.Plugins.CustomPluginsDir = expanded
		}
	}

	// Validate hash algorithm
	validHashAlgorithms := []string{"blake2b", "sha256", "xxhash64", "crc32"}
	if !contains(validHashAlgorithms, cfg.Operations.HashAlgorithm) {
		return fmt.Errorf("invalid hash algorithm: %s, must be one of %v",
			cfg.Operations.HashAlgorithm, validHashAlgorithms)
	}

	// Validate thresholds
	if cfg.Operations.DuplicateThreshold < 0.0 || cfg.Operations.DuplicateThreshold > 1.0 {
		return fmt.Errorf("duplicate_threshold must be between 0.0 and 1.0")
	}

	if cfg.Operations.SimilarityThreshold < 0.0 || cfg.Operations.SimilarityThreshold > 1.0 {
		return fmt.Errorf("similarity_threshold must be between 0.0 and 1.0")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLogLevels, strings.ToLower(cfg.Logging.Level)) {
		return fmt.Errorf("invalid log level: %s, must be one of %v",
			cfg.Logging.Level, validLogLevels)
	}

	return nil
}

// expandPath expands ~ and environment variables in paths
func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	return os.ExpandEnv(path), nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetMaxWorkers returns the optimal number of workers based on configuration
func (c *Config) GetMaxWorkers() int {
	if c.Performance.MaxWorkers <= 0 {
		return runtime.NumCPU()
	}
	return c.Performance.MaxWorkers
}
