package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/asakaida/keruberosu/internal/infrastructure/config"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
)

const (
	migrationsPathSuffix = "internal/infrastructure/database/migrations/postgres"
)

var (
	envFlag string
	pg      *database.Postgres
)

var rootCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration tool for Keruberosu",
	Long: `Database migration tool for Keruberosu.
Manages PostgreSQL schema migrations using golang-migrate.`,
	PersistentPreRun: setupDatabase,
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	Long:  `Apply all pending migrations to the database.`,
	Run:   runUp,
}

var downCmd = &cobra.Command{
	Use:   "down [steps]",
	Short: "Rollback migrations",
	Long:  `Rollback the specified number of migrations (default: 1).`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runDown,
}

var gotoCmd = &cobra.Command{
	Use:   "goto <version>",
	Short: "Migrate to a specific version",
	Long:  `Migrate to a specific version number.`,
	Args:  cobra.ExactArgs(1),
	Run:   runGoto,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current migration version",
	Long:  `Display the current migration version of the database.`,
	Run:   runVersion,
}

var forceCmd = &cobra.Command{
	Use:   "force <version>",
	Short: "Force set migration version (use with caution)",
	Long:  `Force set the migration version without running migrations. Use with caution.`,
	Args:  cobra.ExactArgs(1),
	Run:   runForce,
}

func init() {
	// Add global --env flag to all commands
	rootCmd.PersistentFlags().StringVarP(&envFlag, "env", "e", "dev", "Environment to use (dev, test, prod)")

	// Add subcommands
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(gotoCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(forceCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}
}

func setupDatabase(cmd *cobra.Command, args []string) {
	log.Printf("Using environment: %s", envFlag)

	// Initialize configuration from .env.{env} file
	if err := config.InitConfig(envFlag); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	pg, err = database.NewPostgres(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Printf("Connected to database: %s@%s:%d/%s",
		cfg.Database.User,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database)
}

func getMigrationsPath() (string, error) {
	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}

	migrationsPath := filepath.Join(projectRoot, migrationsPathSuffix)
	log.Printf("Using migrations path: %s", migrationsPath)
	return migrationsPath, nil
}

func runUp(cmd *cobra.Command, args []string) {
	migrationsPath, err := getMigrationsPath()
	if err != nil {
		log.Fatalf("Failed to get migrations path: %v", err)
	}

	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration up failed: %v", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("No migrations to apply")
	} else {
		log.Println("Migration up completed successfully")
	}
}

func runDown(cmd *cobra.Command, args []string) {
	steps := 1 // Default: rollback 1 migration
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &steps)
	}

	migrationsPath, err := getMigrationsPath()
	if err != nil {
		log.Fatalf("Failed to get migrations path: %v", err)
	}

	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration down failed: %v", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("No migrations to rollback")
	} else {
		log.Printf("Migration down completed successfully (rolled back %d migration(s))", steps)
	}
}

func runGoto(cmd *cobra.Command, args []string) {
	var version uint
	fmt.Sscanf(args[0], "%d", &version)

	migrationsPath, err := getMigrationsPath()
	if err != nil {
		log.Fatalf("Failed to get migrations path: %v", err)
	}

	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration goto failed: %v", err)
	}

	if err == migrate.ErrNoChange {
		log.Printf("Already at version %d", version)
	} else {
		log.Printf("Migration goto %d completed successfully", version)
	}
}

func runVersion(cmd *cobra.Command, args []string) {
	migrationsPath, err := getMigrationsPath()
	if err != nil {
		log.Fatalf("Failed to get migrations path: %v", err)
	}

	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		log.Println("Current version: No migrations applied yet")
		return
	}
	if err != nil {
		log.Fatalf("Failed to get version: %v", err)
	}

	if dirty {
		log.Printf("Current version: %d (dirty - migration may have failed)", version)
	} else {
		log.Printf("Current version: %d", version)
	}
}

func runForce(cmd *cobra.Command, args []string) {
	var version int
	fmt.Sscanf(args[0], "%d", &version)

	migrationsPath, err := getMigrationsPath()
	if err != nil {
		log.Fatalf("Failed to get migrations path: %v", err)
	}

	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		log.Fatalf("Migration force failed: %v", err)
	}

	log.Printf("Migration forced to version %d", version)
}

func createMigrate(pg *database.Postgres, migrationsPath string) (*migrate.Migrate, error) {
	driver, err := database.NewMigrateDriver(pg.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	return m, nil
}

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
