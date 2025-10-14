package handlers

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/asakaida/keruberosu/internal/infrastructure/config"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/asakaida/keruberosu/internal/repositories/postgres"
	"github.com/asakaida/keruberosu/internal/services"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// skipIfNotIntegration skips the test if INTEGRATION environment variable is not set
func skipIfNotIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run")
	}
}

// HandlerSet contains all three new handlers
type HandlerSet struct {
	Schema     *SchemaHandler
	Data       *DataHandler
	Permission *PermissionHandler
}

// setupIntegrationTest sets up a full integration test environment with new handlers
func setupIntegrationTest(t *testing.T) (*HandlerSet, *sql.DB) {
	t.Helper()
	skipIfNotIntegration(t)

	// Initialize test config
	if err := config.InitConfig("test"); err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	pg, err := database.NewPostgres(&cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := pg.RunMigrations("../../internal/infrastructure/database/migrations/postgres"); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	db := pg.DB

	// Initialize repositories
	schemaRepo := postgres.NewPostgresSchemaRepository(db)
	relationRepo := postgres.NewPostgresRelationRepository(db)
	attributeRepo := postgres.NewPostgresAttributeRepository(db)

	// Initialize services
	schemaService := services.NewSchemaService(schemaRepo)
	celEngine, err := authorization.NewCELEngine()
	if err != nil {
		t.Fatalf("Failed to create CEL engine: %v", err)
	}
	evaluator := authorization.NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := authorization.NewChecker(schemaService, evaluator)
	expander := authorization.NewExpander(schemaService, relationRepo)
	lookup := authorization.NewLookup(checker, schemaService, relationRepo)

	// Initialize new handlers
	handlers := &HandlerSet{
		Schema:     NewSchemaHandler(schemaService, schemaRepo),
		Data:       NewDataHandler(relationRepo, attributeRepo),
		Permission: NewPermissionHandler(checker, expander, lookup, schemaService),
	}

	return handlers, db
}

// cleanupIntegrationTest cleans up test data and closes database connection
func cleanupIntegrationTest(t *testing.T, db *sql.DB) {
	t.Helper()

	// Clean up all tables
	tables := []string{"attributes", "relations", "schemas"}
	for _, table := range tables {
		_, err := db.Exec("DELETE FROM " + table)
		if err != nil {
			t.Logf("Warning: Failed to clean up table %s: %v", table, err)
		}
	}

	if err := db.Close(); err != nil {
		t.Logf("Warning: Failed to close database: %v", err)
	}
}

func TestHandlers_Integration_FullScenario(t *testing.T) {
	handlers, db := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, db)

	ctx := context.Background()

	// Step 1: Write Schema
	t.Run("WriteSchema", func(t *testing.T) {
		req := &pb.SchemaWriteRequest{
			Schema: `
entity user {}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user

  permission edit = owner or editor
  permission view = owner or editor or viewer
}
`,
		}

		resp, err := handlers.Schema.Write(ctx, req)
		if err != nil {
			t.Fatalf("WriteSchema failed: %v", err)
		}
		// SchemaWriteResponse now only contains SchemaVersion (Permify compatible)
		// Errors are returned via gRPC error, not in response fields
		if resp.SchemaVersion == "" {
			t.Logf("schema_version is empty (expected for now)")
		}
	})

	// Step 2: Write Relations using Data.Write
	t.Run("WriteRelations", func(t *testing.T) {
		req := &pb.DataWriteRequest{
			Tuples: []*pb.Tuple{
				{
					Entity: &pb.Entity{
						Type: "document",
						Id:   "doc1",
					},
					Relation: "owner",
					Subject: &pb.Subject{
						Type: "user",
						Id:   "alice",
					},
				},
				{
					Entity: &pb.Entity{
						Type: "document",
						Id:   "doc1",
					},
					Relation: "viewer",
					Subject: &pb.Subject{
						Type: "user",
						Id:   "bob",
					},
				},
			},
		}

		_, err := handlers.Data.Write(ctx, req)
		if err != nil {
			t.Fatalf("WriteRelations failed: %v", err)
		}
	})

	// Step 3: Check - Alice can edit (owner)
	t.Run("Check_Alice_Edit", func(t *testing.T) {
		req := &pb.PermissionCheckRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   "doc1",
			},
			Permission: "edit",
			Subject: &pb.Subject{
				Type: "user",
				Id:   "alice",
			},
		}

		resp, err := handlers.Permission.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("Expected ALLOWED, got %v", resp.Can)
		}
	})

	// Step 4: Check - Bob cannot edit (only viewer)
	t.Run("Check_Bob_Edit", func(t *testing.T) {
		req := &pb.PermissionCheckRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   "doc1",
			},
			Permission: "edit",
			Subject: &pb.Subject{
				Type: "user",
				Id:   "bob",
			},
		}

		resp, err := handlers.Permission.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if resp.Can != pb.CheckResult_CHECK_RESULT_DENIED {
			t.Errorf("Expected DENIED, got %v", resp.Can)
		}
	})

	// Step 5: Check - Bob can view
	t.Run("Check_Bob_View", func(t *testing.T) {
		req := &pb.PermissionCheckRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   "doc1",
			},
			Permission: "view",
			Subject: &pb.Subject{
				Type: "user",
				Id:   "bob",
			},
		}

		resp, err := handlers.Permission.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("Expected ALLOWED, got %v", resp.Can)
		}
	})

	// Step 6: LookupEntity - Find documents Bob can view
	t.Run("LookupEntity_Bob_View", func(t *testing.T) {
		req := &pb.PermissionLookupEntityRequest{
			EntityType: "document",
			Permission: "view",
			Subject: &pb.Subject{
				Type: "user",
				Id:   "bob",
			},
		}

		resp, err := handlers.Permission.LookupEntity(ctx, req)
		if err != nil {
			t.Fatalf("LookupEntity failed: %v", err)
		}
		if len(resp.EntityIds) != 1 {
			t.Errorf("Expected 1 entity, got %d", len(resp.EntityIds))
		}
		if len(resp.EntityIds) > 0 && resp.EntityIds[0] != "doc1" {
			t.Errorf("Expected doc1, got %s", resp.EntityIds[0])
		}
	})

	// Step 7: SubjectPermission - Get all permissions for Alice on doc1
	t.Run("SubjectPermission_Alice", func(t *testing.T) {
		req := &pb.PermissionSubjectPermissionRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   "doc1",
			},
			Subject: &pb.Subject{
				Type: "user",
				Id:   "alice",
			},
		}

		resp, err := handlers.Permission.SubjectPermission(ctx, req)
		if err != nil {
			t.Fatalf("SubjectPermission failed: %v", err)
		}
		if len(resp.Results) != 2 {
			t.Errorf("Expected 2 permissions, got %d", len(resp.Results))
		}
		if resp.Results["edit"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("Expected edit=ALLOWED for Alice")
		}
		if resp.Results["view"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("Expected view=ALLOWED for Alice")
		}
	})

	// Step 8: DeleteRelations - Remove Bob's viewer relation
	t.Run("DeleteRelations", func(t *testing.T) {
		req := &pb.DataDeleteRequest{
			Filter: &pb.TupleFilter{
				Entity: &pb.EntityFilter{
					Type: "document",
					Ids:  []string{"doc1"},
				},
				Relation: "viewer",
				Subject: &pb.SubjectFilter{
					Type: "user",
					Ids:  []string{"bob"},
				},
			},
		}

		_, err := handlers.Data.Delete(ctx, req)
		if err != nil {
			t.Fatalf("DeleteRelations failed: %v", err)
		}
	})

	// Step 9: Check - Bob can no longer view after deletion
	t.Run("Check_Bob_View_After_Delete", func(t *testing.T) {
		req := &pb.PermissionCheckRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   "doc1",
			},
			Permission: "view",
			Subject: &pb.Subject{
				Type: "user",
				Id:   "bob",
			},
		}

		resp, err := handlers.Permission.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if resp.Can != pb.CheckResult_CHECK_RESULT_DENIED {
			t.Errorf("Expected DENIED after deletion, got %v", resp.Can)
		}
	})
}

func TestHandlers_Integration_ABAC(t *testing.T) {
	handlers, db := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, db)

	ctx := context.Background()

	// Step 1: Write Schema with ABAC
	t.Run("WriteSchema_ABAC", func(t *testing.T) {
		req := &pb.SchemaWriteRequest{
			Schema: `
entity user {}

entity document {
  attribute public: bool

  relation owner: user

  permission view = owner or rule(resource.public == true)
}
`,
		}

		resp, err := handlers.Schema.Write(ctx, req)
		if err != nil {
			t.Fatalf("WriteSchema failed: %v", err)
		}
		// SchemaWriteResponse now only contains SchemaVersion (Permify compatible)
		// Errors are returned via gRPC error, not in response fields
		if resp.SchemaVersion == "" {
			t.Logf("schema_version is empty (expected for now)")
		}
	})

	// Step 2: Write Relations
	t.Run("WriteRelations_ABAC", func(t *testing.T) {
		req := &pb.DataWriteRequest{
			Tuples: []*pb.Tuple{
				{
					Entity: &pb.Entity{
						Type: "document",
						Id:   "doc2",
					},
					Relation: "owner",
					Subject: &pb.Subject{
						Type: "user",
						Id:   "alice",
					},
				},
			},
		}

		_, err := handlers.Data.Write(ctx, req)
		if err != nil {
			t.Fatalf("WriteRelations failed: %v", err)
		}
	})

	// Step 3: Write Attributes - Make doc2 public using Data.Write
	t.Run("WriteAttributes_Public", func(t *testing.T) {
		req := &pb.DataWriteRequest{
			Attributes: []*pb.Attribute{
				{
					Entity:    &pb.Entity{Type: "document", Id: "doc2"},
					Attribute: "public",
					Value:     structpb.NewBoolValue(true),
				},
			},
		}

		_, err := handlers.Data.Write(ctx, req)
		if err != nil {
			t.Fatalf("WriteAttributes failed: %v", err)
		}
	})

	// Step 4: Check - Bob can view public document (via ABAC)
	t.Run("Check_Bob_View_Public_Document", func(t *testing.T) {
		req := &pb.PermissionCheckRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   "doc2",
			},
			Permission: "view",
			Subject: &pb.Subject{
				Type: "user",
				Id:   "bob",
			},
		}

		resp, err := handlers.Permission.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("Expected ALLOWED for public document, got %v", resp.Can)
		}
	})

	// Step 5: Write Attributes - Make doc2 private
	t.Run("WriteAttributes_Private", func(t *testing.T) {
		req := &pb.DataWriteRequest{
			Attributes: []*pb.Attribute{
				{
					Entity:    &pb.Entity{Type: "document", Id: "doc2"},
					Attribute: "public",
					Value:     structpb.NewBoolValue(false),
				},
			},
		}

		_, err := handlers.Data.Write(ctx, req)
		if err != nil {
			t.Fatalf("WriteAttributes failed: %v", err)
		}
	})

	// Step 6: Check - Bob cannot view private document
	t.Run("Check_Bob_View_Private_Document", func(t *testing.T) {
		req := &pb.PermissionCheckRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   "doc2",
			},
			Permission: "view",
			Subject: &pb.Subject{
				Type: "user",
				Id:   "bob",
			},
		}

		resp, err := handlers.Permission.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if resp.Can != pb.CheckResult_CHECK_RESULT_DENIED {
			t.Errorf("Expected DENIED for private document, got %v", resp.Can)
		}
	})
}
