package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/asakaida/keruberosu/internal/infrastructure/config"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/asakaida/keruberosu/internal/repositories/postgres"
	"github.com/spf13/cobra"
)

var envFlag string

var rootCmd = &cobra.Command{
	Use:   "admin",
	Short: "Keruberosu admin CLI",
	Long:  "Administration tool for Keruberosu authorization service.",
}

var rebuildClosuresCmd = &cobra.Command{
	Use:   "rebuild-closures",
	Short: "Rebuild closure table for all tenants",
	Long: `Rebuild the entity_closure table from scratch for all tenants.
Use this command to fix stale closure entries or after bulk imports
that skipped closure updates. Safe to run at any time.`,
	Run: runRebuildClosures,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&envFlag, "env", "e", "dev", "Environment to use (dev, test, prod)")
	rootCmd.AddCommand(rebuildClosuresCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}
}

func runRebuildClosures(cmd *cobra.Command, args []string) {
	log.Println("Starting closure table rebuild...")

	if err := config.InitConfig(envFlag); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cluster, err := database.NewDBCluster(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer cluster.Close()

	log.Printf("Connected to database: %s@%s:%d/%s",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)

	// Get all tenant IDs
	ctx := context.Background()
	rows, err := cluster.PrimaryDB().QueryContext(ctx, "SELECT DISTINCT tenant_id FROM schemas ORDER BY tenant_id")
	if err != nil {
		log.Fatalf("Failed to query tenant IDs: %v", err)
	}
	defer rows.Close()

	var tenantIDs []string
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			log.Fatalf("Failed to scan tenant ID: %v", err)
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating tenant IDs: %v", err)
	}

	if len(tenantIDs) == 0 {
		log.Println("No tenants found. Nothing to rebuild.")
		return
	}

	log.Printf("Found %d tenant(s): %v", len(tenantIDs), tenantIDs)

	closureExcluded := cfg.Database.ParseClosureExcludedRelations()
	relationRepo := postgres.NewPostgresRelationRepository(cluster, closureExcluded)

	totalStart := time.Now()
	for _, tenantID := range tenantIDs {
		// Count relations and closures before rebuild
		var relationCount, closureBefore int
		cluster.PrimaryDB().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM relations WHERE tenant_id = $1", tenantID).Scan(&relationCount)
		cluster.PrimaryDB().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM entity_closure WHERE tenant_id = $1", tenantID).Scan(&closureBefore)

		log.Printf("  Rebuilding tenant %s (relations: %d, closures before: %d)...",
			tenantID, relationCount, closureBefore)

		start := time.Now()
		if err := relationRepo.RebuildClosure(ctx, tenantID); err != nil {
			log.Printf("  ERROR rebuilding tenant %s: %v", tenantID, err)
			continue
		}

		var closureAfter int
		cluster.PrimaryDB().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM entity_closure WHERE tenant_id = $1", tenantID).Scan(&closureAfter)

		log.Printf("  Done tenant %s in %v (closures: %d -> %d, delta: %+d)",
			tenantID, time.Since(start).Round(time.Millisecond),
			closureBefore, closureAfter, closureAfter-closureBefore)
	}

	fmt.Printf("\nAll tenants rebuilt in %v\n", time.Since(totalStart).Round(time.Millisecond))
}
