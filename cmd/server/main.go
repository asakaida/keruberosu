package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asakaida/keruberosu/internal/handlers"
	"github.com/asakaida/keruberosu/internal/infrastructure/cache"
	"github.com/asakaida/keruberosu/internal/infrastructure/config"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/asakaida/keruberosu/internal/infrastructure/metrics"
	"github.com/asakaida/keruberosu/internal/repositories/postgres"
	"github.com/asakaida/keruberosu/internal/services"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	pkgcache "github.com/asakaida/keruberosu/pkg/cache"
	"github.com/asakaida/keruberosu/pkg/cache/memorycache"
	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Initialize cache and snapshot manager if enabled
	var checkCache pkgcache.Cache
	var snapshotMgr *cache.SnapshotManager

	if cfg.Cache.Enabled {
		// Initialize memory cache
		checkCache, err = memorycache.New(&memorycache.Config{
			MaxSizeBytes:  cfg.Cache.MaxMemoryBytes,
			DefaultTTL:    time.Duration(cfg.Cache.TTLMinutes) * time.Minute,
			EnableMetrics: cfg.Cache.Metrics,
		})
		if err != nil {
			log.Fatalf("Failed to create cache: %v", err)
		}
		log.Printf("Cache enabled: maxSize=%dMB, TTL=%dm",
			cfg.Cache.MaxMemoryBytes/(1024*1024),
			cfg.Cache.TTLMinutes)

		// Initialize snapshot manager for cache consistency
		connStr := cfg.Database.ConnectionString()
		snapshotMgr = cache.NewSnapshotManager(pg.DB, connStr, 5*time.Minute)
		if err := snapshotMgr.Start(context.Background()); err != nil {
			log.Printf("Warning: Failed to start snapshot manager: %v (cache will use TTL-only mode)", err)
			snapshotMgr = nil
		} else {
			log.Println("Snapshot manager started (LISTEN/NOTIFY enabled)")
		}
	}

	// Initialize checker (with or without cache)
	var checker authorization.CheckerInterface
	if cfg.Cache.Enabled && checkCache != nil {
		checker = authorization.NewCheckerWithCache(
			schemaService,
			evaluator,
			checkCache,
			snapshotMgr,
			time.Duration(cfg.Cache.TTLMinutes)*time.Minute,
		)
	} else {
		checker = authorization.NewChecker(schemaService, evaluator)
	}

	expander := authorization.NewExpander(schemaService, relationRepo)
	lookup := authorization.NewLookup(checker, schemaService, relationRepo)

	// Initialize metrics collector and Prometheus exporter
	metricsCollector := metrics.NewCollector()
	if checkCache != nil {
		metricsCollector.SetCache(checkCache)
	}
	prometheusExporter := metrics.NewPrometheusExporter(metricsCollector)

	// Initialize token generator for Data API snapshot tokens
	tokenGenerator := postgres.NewSnapshotManager(pg.DB)

	// Initialize service handlers
	permissionHandler := handlers.NewPermissionHandler(
		checker,
		expander,
		lookup,
		schemaService,
	)
	dataHandler := handlers.NewDataHandlerWithTokenGenerator(
		relationRepo,
		attributeRepo,
		tokenGenerator,
	)
	schemaHandler := handlers.NewSchemaHandler(
		schemaService,
		schemaRepo,
	)

	// Create gRPC server with metrics interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.UnaryServerInterceptor(metricsCollector, prometheusExporter)),
	)
	pb.RegisterPermissionServer(grpcServer, permissionHandler)
	pb.RegisterDataServer(grpcServer, dataHandler)
	pb.RegisterSchemaServer(grpcServer, schemaHandler)

	// Register reflection service (for grpcurl, etc.)
	reflection.Register(grpcServer)

	// Start Prometheus metrics HTTP server
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.MetricsPort),
		Handler: promhttp.Handler(),
	}
	go func() {
		log.Printf("Prometheus metrics server listening on :%d", cfg.Server.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Start periodic metrics update goroutine
	metricsStopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				prometheusExporter.Update()
			case <-metricsStopCh:
				return
			}
		}
	}()

	// Start gRPC server listening
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

		// Stop metrics update goroutine
		close(metricsStopCh)

		// Shutdown metrics server
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down metrics server: %v", err)
		}

		// Channel to notify when graceful stop completes
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		// Wait for graceful stop or timeout
		select {
		case <-stopped:
			log.Println("gRPC server stopped gracefully")
		case <-shutdownCtx.Done():
			log.Println("Shutdown timeout exceeded, forcing stop")
			grpcServer.Stop()
		}

		// Stop snapshot manager
		if snapshotMgr != nil {
			if err := snapshotMgr.Stop(); err != nil {
				log.Printf("Error stopping snapshot manager: %v", err)
			}
		}

		// Close cache
		if checkCache != nil {
			if err := checkCache.Close(); err != nil {
				log.Printf("Error closing cache: %v", err)
			}
		}

		// Close database connection
		if err := pg.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}

		log.Println("Shutdown complete")
	}
}
