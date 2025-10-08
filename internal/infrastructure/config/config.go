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
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host string
	Port int
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
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 15432)
	viper.SetDefault("DB_USER", "keruberosu")
	viper.SetDefault("DB_NAME", "keruberosu_dev")
	viper.SetDefault("DB_SSLMODE", "disable")

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
			Host: viper.GetString("SERVER_HOST"),
			Port: viper.GetInt("SERVER_PORT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: dbPassword,
			Database: viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
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
