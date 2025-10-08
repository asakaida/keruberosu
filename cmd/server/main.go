package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asakaida/keruberosu/internal/handlers"
	"github.com/asakaida/keruberosu/internal/infrastructure/config"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/asakaida/keruberosu/internal/repositories/postgres"
	"github.com/asakaida/keruberosu/internal/services"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	defaultEnv  = "dev"
	defaultPort = "50051"
)

func main() {
	// Get environment from ENV variable or use default
	env := os.Getenv("ENV")
	if env == "" {
		env = defaultEnv
	}

	// Get port from PORT variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
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

	// Initialize repositories
	schemaRepo := postgres.NewPostgresSchemaRepository(pg.DB)
	relationRepo := postgres.NewPostgresRelationRepository(pg.DB)
	attributeRepo := postgres.NewPostgresAttributeRepository(pg.DB)

	// Initialize services
	schemaService := services.NewSchemaService(schemaRepo)
	celEngine, err := authorization.NewCELEngine()
	if err != nil {
		log.Fatalf("Failed to create CEL engine: %v", err)
	}
	evaluator := authorization.NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := authorization.NewChecker(schemaService, evaluator)
	expander := authorization.NewExpander(schemaService, relationRepo)
	lookup := authorization.NewLookup(checker, schemaService, relationRepo)

	// Initialize unified authorization handler
	authHandler := handlers.NewAuthorizationHandler(
		schemaService,
		relationRepo,
		attributeRepo,
		checker,
		expander,
		lookup,
		schemaRepo,
	)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterAuthorizationServiceServer(grpcServer, authHandler)

	// Register reflection service (for grpcurl, etc.)
	reflection.Register(grpcServer)

	// Start listening
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("gRPC server listening on :%s", port)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			serverErrors <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		log.Println("Initiating graceful shutdown...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Channel to notify when graceful stop completes
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		// Wait for graceful stop or timeout
		select {
		case <-stopped:
			log.Println("Server stopped gracefully")
		case <-shutdownCtx.Done():
			log.Println("Shutdown timeout exceeded, forcing stop")
			grpcServer.Stop()
		}

		// Close database connection
		if err := pg.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}

		log.Println("Shutdown complete")
	}
}
