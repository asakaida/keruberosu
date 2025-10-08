package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/asakaida/keruberosu/internal/infrastructure/config"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/golang-migrate/migrate/v4"
)

const (
	defaultEnv           = "dev"
	migrationsPathSuffix = "internal/infrastructure/database/migrations/postgres"
)

func main() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Get environment from ENV variable or use default
	env := os.Getenv("ENV")
	if env == "" {
		env = defaultEnv
	}

	// Initialize configuration
	if err := config.InitConfig(env); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	pg, err := database.NewPostgres(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pg.Close()

	log.Printf("Connected to database: %s@%s:%d/%s",
		cfg.Database.User,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database)

	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("Failed to find project root: %v", err)
	}

	migrationsPath := filepath.Join(projectRoot, migrationsPathSuffix)
	log.Printf("Using migrations path: %s", migrationsPath)

	// Execute command
	switch command {
	case "up":
		if err := runUp(pg, migrationsPath); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("Migration up completed successfully")

	case "down":
		steps := 1 // Default: rollback 1 migration
		if len(os.Args) > 2 {
			steps, err = strconv.Atoi(os.Args[2])
			if err != nil {
				log.Fatalf("Invalid steps argument: %v", err)
			}
		}
		if err := runDown(pg, migrationsPath, steps); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Printf("Migration down completed successfully (rolled back %d migration(s))", steps)

	case "goto":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate goto <version>")
		}
		version, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err != nil {
			log.Fatalf("Invalid version argument: %v", err)
		}
		if err := runGoto(pg, migrationsPath, uint(version)); err != nil {
			log.Fatalf("Migration goto failed: %v", err)
		}
		log.Printf("Migration goto %d completed successfully", version)

	case "version":
		version, dirty, err := getVersion(pg, migrationsPath)
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		if dirty {
			log.Printf("Current version: %d (dirty)", version)
		} else {
			log.Printf("Current version: %d", version)
		}

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate force <version>")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Invalid version argument: %v", err)
		}
		if err := runForce(pg, migrationsPath, version); err != nil {
			log.Fatalf("Migration force failed: %v", err)
		}
		log.Printf("Migration forced to version %d", version)

	default:
		log.Printf("Unknown command: %s", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up                 Apply all pending migrations")
	fmt.Println("  down [steps]       Rollback migrations (default: 1 step)")
	fmt.Println("  goto <version>     Migrate to a specific version")
	fmt.Println("  version            Show current migration version")
	fmt.Println("  force <version>    Force set migration version (use with caution)")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  ENV=dev|test|prod  Set environment (default: dev)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  migrate up")
	fmt.Println("  migrate down")
	fmt.Println("  migrate down 2")
	fmt.Println("  migrate goto 1")
	fmt.Println("  migrate version")
	fmt.Println("  ENV=test migrate up")
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

func runUp(pg *database.Postgres, migrationsPath string) error {
	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("No migrations to apply")
	}

	return nil
}

func runDown(pg *database.Postgres, migrationsPath string, steps int) error {
	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("No migrations to rollback")
	}

	return nil
}

func runGoto(pg *database.Postgres, migrationsPath string, version uint) error {
	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration goto failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Printf("Already at version %d", version)
	}

	return nil
}

func getVersion(pg *database.Postgres, migrationsPath string) (version uint, dirty bool, err error) {
	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err = m.Version()
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}

	return version, dirty, nil
}

func runForce(pg *database.Postgres, migrationsPath string, version int) error {
	m, err := createMigrate(pg, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("migration force failed: %w", err)
	}

	return nil
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
