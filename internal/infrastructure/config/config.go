package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host        string
	Port        int
	MetricsPort int // Port for Prometheus metrics HTTP server
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled        bool
	NumCounters    int64
	MaxMemoryBytes int64 // Maximum memory usage in bytes (e.g., 104857600 = 100MB)
	BufferItems    int64
	Metrics        bool
	TTLMinutes     int // Time-to-live for cache entries in minutes
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// findProjectRoot finds the project root directory by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree until we find go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parent
	}
}

// InitConfig initializes viper configuration
// env: environment name (dev, test, prod)
func InitConfig(env string) error {
	if env == "" {
		env = "dev"
	}

	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Set config file name based on environment
	viper.SetConfigName(fmt.Sprintf(".env.%s", env))
	viper.SetConfigType("env")
	viper.AddConfigPath(projectRoot) // Project root

	// Read config file (optional, ignore error if not found)
	_ = viper.ReadInConfig()

	// Environment variables take precedence over config file
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("SERVER_HOST", "0.0.0.0")
	viper.SetDefault("SERVER_PORT", 50051)
	viper.SetDefault("METRICS_PORT", 9090)
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 15432)
	viper.SetDefault("DB_USER", "keruberosu")
	viper.SetDefault("DB_NAME", "keruberosu_dev")
	viper.SetDefault("DB_SSLMODE", "disable")

	// Cache defaults
	viper.SetDefault("CACHE_ENABLED", true)
	viper.SetDefault("CACHE_NUM_COUNTERS", 100000)            // ~10k items expected
	viper.SetDefault("CACHE_MAX_MEMORY_BYTES", 100*1024*1024) // 100MB
	viper.SetDefault("CACHE_BUFFER_ITEMS", 64)
	viper.SetDefault("CACHE_METRICS", true)
	viper.SetDefault("CACHE_TTL_MINUTES", 5) // 5 minutes TTL

	return nil
}

// Load loads configuration from viper
func Load() (*Config, error) {
	// DB_PASSWORD is required for security
	dbPassword := viper.GetString("DB_PASSWORD")
	if dbPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required (set via environment variable or .env file)")
	}

	config := &Config{
		Server: ServerConfig{
			Host:        viper.GetString("SERVER_HOST"),
			Port:        viper.GetInt("SERVER_PORT"),
			MetricsPort: viper.GetInt("METRICS_PORT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: dbPassword,
			Database: viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
		},
		Cache: CacheConfig{
			Enabled:        viper.GetBool("CACHE_ENABLED"),
			NumCounters:    viper.GetInt64("CACHE_NUM_COUNTERS"),
			MaxMemoryBytes: viper.GetInt64("CACHE_MAX_MEMORY_BYTES"),
			BufferItems:    viper.GetInt64("CACHE_BUFFER_ITEMS"),
			Metrics:        viper.GetBool("CACHE_METRICS"),
			TTLMinutes:     viper.GetInt("CACHE_TTL_MINUTES"),
		},
	}

	return config, nil
}

// ConnectionString returns PostgreSQL connection string
func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Database,
		c.SSLMode,
	)
}
