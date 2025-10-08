package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/asakaida/keruberosu/internal/handlers"
	"github.com/asakaida/keruberosu/internal/infrastructure/config"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/asakaida/keruberosu/internal/repositories/postgres"
	"github.com/asakaida/keruberosu/internal/services"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// E2ETestServer represents an E2E test server
type E2ETestServer struct {
	Server   *grpc.Server
	Client   pb.AuthorizationServiceClient
	Conn     *grpc.ClientConn
	DB       *sql.DB
	Listener *bufconn.Listener
	cancel   context.CancelFunc
}

// SetupE2ETest sets up an E2E test environment
func SetupE2ETest(t *testing.T) *E2ETestServer {
	t.Helper()

	// Initialize config for test environment
	config.InitConfig("test")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Connect to test database
	pg, err := database.NewPostgres(&cfg.Database)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Run migrations (use absolute path)
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}
	migrationsPath := projectRoot + "/internal/infrastructure/database/migrations/postgres"
	if err := pg.RunMigrations(migrationsPath); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Clean up existing data
	cleanupDatabase(t, pg.DB)

	// Initialize repositories
	schemaRepo := postgres.NewPostgresSchemaRepository(pg.DB)
	relationRepo := postgres.NewPostgresRelationRepository(pg.DB)
	attributeRepo := postgres.NewPostgresAttributeRepository(pg.DB)

	// Initialize services
	schemaService := services.NewSchemaService(schemaRepo)
	celEngine, err := authorization.NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}
	evaluator := authorization.NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := authorization.NewChecker(schemaService, evaluator)
	expander := authorization.NewExpander(schemaService, relationRepo)
	lookup := authorization.NewLookup(checker, schemaService, relationRepo)

	// Initialize handler
	handler := handlers.NewAuthorizationHandler(
		schemaService,
		relationRepo,
		attributeRepo,
		checker,
		expander,
		lookup,
		schemaRepo,
	)

	// Create in-memory gRPC server with bufconn
	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	pb.RegisterAuthorizationServiceServer(server, handler)

	// Start server in background
	_, cancel := context.WithCancel(context.Background())
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("server error: %v", err)
		}
	}()

	// Create client connection
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}

	conn, err := grpc.NewClient(
		"passthrough://bufconn",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		cancel()
		t.Fatalf("failed to create client connection: %v", err)
	}

	client := pb.NewAuthorizationServiceClient(conn)

	return &E2ETestServer{
		Server:   server,
		Client:   client,
		Conn:     conn,
		DB:       pg.DB,
		Listener: listener,
		cancel:   cancel,
	}
}

// Teardown cleans up the E2E test environment
func (e *E2ETestServer) Teardown(t *testing.T) {
	t.Helper()

	if e.Conn != nil {
		e.Conn.Close()
	}
	if e.Server != nil {
		e.Server.Stop()
	}
	if e.Listener != nil {
		e.Listener.Close()
	}
	if e.cancel != nil {
		e.cancel()
	}
	if e.DB != nil {
		cleanupDatabase(t, e.DB)
		e.DB.Close()
	}
}

// cleanupDatabase removes all data from test database
func cleanupDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Delete in correct order due to foreign key constraints
	tables := []string{"attributes", "relations", "schemas"}
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Logf("warning: failed to clean up table %s: %v", table, err)
		}
	}
}

// WaitForServer waits for the server to be ready
func (e *E2ETestServer) WaitForServer(t *testing.T, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout waiting for server to be ready")
		case <-ticker.C:
			// Try to make a simple request
			_, err := e.Client.ReadSchema(ctx, &pb.ReadSchemaRequest{})
			if err == nil || err.Error() != "context deadline exceeded" {
				return
			}
		}
	}
}

// findProjectRoot finds the project root directory by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found")
		}
		dir = parent
	}
}
