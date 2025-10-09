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
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	envFlag  string
	portFlag int
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Keruberosu gRPC server",
	Long: `Keruberosu is a Permify-compatible ReBAC/ABAC authorization microservice.
This command starts the gRPC server.`,
	Run: runServer,
}

func init() {
	rootCmd.Flags().StringVarP(&envFlag, "env", "e", "dev", "Environment to use (dev, test, prod)")
	rootCmd.Flags().IntVarP(&portFlag, "port", "p", 50051, "Port to listen on")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	log.Printf("Starting Keruberosu server (environment: %s)", envFlag)

	// Initialize configuration from .env.{env} file
	if err := config.InitConfig(envFlag); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override port if specified via flag
	if cmd.Flags().Changed("port") {
		cfg.Server.Port = portFlag
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
	port := cfg.Server.Port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("gRPC server listening on :%d", port)

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
