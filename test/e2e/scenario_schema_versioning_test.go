package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
)

// TestScenario_SchemaVersioning tests schema versioning functionality
func TestScenario_SchemaVersioning(t *testing.T) {
	// Setup E2E test server
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Test 1: Write initial schema (v1)
	t.Log("Test 1: Writing initial schema (v1)")
	schemaV1 := `
entity user {}

entity document {
  relation owner @user
  relation viewer @user

  permission view = owner or viewer
}
`

	v1Resp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schemaV1,
	})
	if err != nil {
		t.Fatalf("Write schema v1 failed: %v", err)
	}
	if v1Resp.SchemaVersion == "" {
		t.Fatal("Schema v1 version is empty")
	}
	v1Version := v1Resp.SchemaVersion
	t.Logf("✓ Schema v1 written successfully (version: %s)", v1Version)

	// Test 2: Write updated schema (v2) - add editor role
	t.Log("Test 2: Writing updated schema (v2) with editor role")
	time.Sleep(100 * time.Millisecond) // Ensure ULID uniqueness
	schemaV2 := `
entity user {}

entity document {
  relation owner @user
  relation editor @user
  relation viewer @user

  permission edit = owner or editor
  permission view = owner or editor or viewer
}
`

	v2Resp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schemaV2,
	})
	if err != nil {
		t.Fatalf("Write schema v2 failed: %v", err)
	}
	if v2Resp.SchemaVersion == "" {
		t.Fatal("Schema v2 version is empty")
	}
	v2Version := v2Resp.SchemaVersion
	if v2Version == v1Version {
		t.Fatal("v2 version should be different from v1")
	}
	t.Logf("✓ Schema v2 written successfully (version: %s)", v2Version)

	// Test 3: Write another updated schema (v3) - add admin role
	t.Log("Test 3: Writing updated schema (v3) with admin role")
	time.Sleep(100 * time.Millisecond)
	schemaV3 := `
entity user {}

entity document {
  relation owner @user
  relation admin @user
  relation editor @user
  relation viewer @user

  permission delete = owner or admin
  permission edit = owner or admin or editor
  permission view = owner or admin or editor or viewer
}
`

	v3Resp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schemaV3,
	})
	if err != nil {
		t.Fatalf("Write schema v3 failed: %v", err)
	}
	if v3Resp.SchemaVersion == "" {
		t.Fatal("Schema v3 version is empty")
	}
	v3Version := v3Resp.SchemaVersion
	if v3Version == v2Version || v3Version == v1Version {
		t.Fatal("v3 version should be different from v1 and v2")
	}
	t.Logf("✓ Schema v3 written successfully (version: %s)", v3Version)

	// Test 4: List all schema versions
	t.Log("Test 4: Listing all schema versions")
	listResp, err := schemaClient.List(ctx, &pb.SchemaListRequest{
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("List schemas failed: %v", err)
	}
	if len(listResp.Schemas) < 3 {
		t.Fatalf("Expected at least 3 schema versions, got %d", len(listResp.Schemas))
	}
	if listResp.Head == "" {
		t.Fatal("Head version is empty")
	}
	if listResp.Head != v3Version {
		t.Errorf("Head version should be v3 (%s), got %s", v3Version, listResp.Head)
	}
	t.Logf("✓ Found %d schema versions, HEAD: %s", len(listResp.Schemas), listResp.Head)

	// Verify versions are sorted newest first
	if listResp.Schemas[0].Version != v3Version {
		t.Errorf("First schema should be v3 (%s), got %s", v3Version, listResp.Schemas[0].Version)
	}

	// Verify created_at timestamps
	for i, schema := range listResp.Schemas {
		if schema.CreatedAt == "" {
			t.Errorf("Schema %d has empty created_at", i)
		}
	}
	t.Log("✓ Schema versions are properly sorted with timestamps")

	// Test 5: Read specific version (v1)
	t.Log("Test 5: Reading specific version (v1)")
	readV1Resp, err := schemaClient.Read(ctx, &pb.SchemaReadRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v1Version,
		},
	})
	if err != nil {
		t.Fatalf("Read schema v1 failed: %v", err)
	}
	if !strings.Contains(readV1Resp.Schema, "relation viewer") {
		t.Error("v1 schema should contain 'relation viewer'")
	}
	if strings.Contains(readV1Resp.Schema, "relation editor") {
		t.Error("v1 schema should not contain 'relation editor'")
	}
	t.Log("✓ Successfully read v1 schema")

	// Test 6: Read latest version (without specifying version)
	t.Log("Test 6: Reading latest version (no version specified)")
	readLatestResp, err := schemaClient.Read(ctx, &pb.SchemaReadRequest{})
	if err != nil {
		t.Fatalf("Read latest schema failed: %v", err)
	}
	if !strings.Contains(readLatestResp.Schema, "relation admin") {
		t.Error("Latest schema should contain 'relation admin' (v3)")
	}
	if !strings.Contains(readLatestResp.Schema, "permission delete") {
		t.Error("Latest schema should contain 'permission delete' (v3)")
	}
	t.Log("✓ Successfully read latest schema (v3)")

	// Test 7: Setup test data
	t.Log("Test 7: Setting up test data")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "admin", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "dave"}},
		},
	})
	if err != nil {
		t.Fatalf("Write relations failed: %v", err)
	}
	t.Log("✓ Test data written successfully")

	// Test 8: Check permission with v1 schema (edit permission doesn't exist)
	t.Log("Test 8: Checking permission with v1 schema (edit permission should not exist)")
	_, err = permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v1Version,
		},
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "edit",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "charlie",
		},
	})
	if err == nil {
		t.Error("Check with v1 schema should fail for 'edit' permission (doesn't exist in v1)")
	}
	if err != nil && !strings.Contains(err.Error(), "permission edit not found") {
		t.Errorf("Expected 'permission not found' error, got: %v", err)
	}
	t.Log("✓ v1 schema correctly rejects non-existent 'edit' permission")

	// Test 9: Check permission with v1 schema (view permission exists)
	t.Log("Test 9: Checking view permission with v1 schema")
	checkV1Resp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v1Version,
		},
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "view",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "dave",
		},
	})
	if err != nil {
		t.Fatalf("Check with v1 schema failed: %v", err)
	}
	if checkV1Resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("dave should be allowed to view with v1 schema (is viewer)")
	}
	t.Log("✓ v1 schema correctly allows view permission for viewer")

	// Test 10: Check permission with v2 schema (edit permission exists)
	t.Log("Test 10: Checking edit permission with v2 schema")
	checkV2Resp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v2Version,
		},
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "edit",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "charlie",
		},
	})
	if err != nil {
		t.Fatalf("Check with v2 schema failed: %v", err)
	}
	if checkV2Resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("charlie should be allowed to edit with v2 schema (is editor)")
	}
	t.Log("✓ v2 schema correctly allows edit permission for editor")

	// Test 11: Check permission with v2 schema (delete permission doesn't exist)
	t.Log("Test 11: Checking delete permission with v2 schema (should not exist)")
	_, err = permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v2Version,
		},
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "delete",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "bob",
		},
	})
	if err == nil {
		t.Error("Check with v2 schema should fail for 'delete' permission (doesn't exist in v2)")
	}
	t.Log("✓ v2 schema correctly rejects non-existent 'delete' permission")

	// Test 12: Check permission with v3 schema (delete permission exists)
	t.Log("Test 12: Checking delete permission with v3 schema")
	checkV3Resp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v3Version,
		},
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "delete",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "bob",
		},
	})
	if err != nil {
		t.Fatalf("Check with v3 schema failed: %v", err)
	}
	if checkV3Resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("bob should be allowed to delete with v3 schema (is admin)")
	}
	t.Log("✓ v3 schema correctly allows delete permission for admin")

	// Test 13: Check without version specified (should use latest = v3)
	t.Log("Test 13: Checking without version specified (should use v3)")
	checkLatestResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "delete",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "bob",
		},
	})
	if err != nil {
		t.Fatalf("Check without version failed: %v", err)
	}
	if checkLatestResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("bob should be allowed to delete without version (defaults to v3)")
	}
	t.Log("✓ Default behavior uses latest schema version")

	// Test 14: Expand API with specific version
	t.Log("Test 14: Testing Expand API with v2 schema")
	expandResp, err := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v2Version,
		},
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "edit",
	})
	if err != nil {
		t.Fatalf("Expand with v2 schema failed: %v", err)
	}
	if expandResp.Tree == nil {
		t.Fatal("Expand returned nil tree")
	}
	t.Log("✓ Expand API works with specific schema version")

	// Test 15: LookupEntity API with specific version
	t.Log("Test 15: Testing LookupEntity API with v2 schema")
	lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v2Version,
		},
		EntityType: "document",
		Permission: "edit",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "charlie",
		},
	})
	if err != nil {
		t.Fatalf("LookupEntity with v2 schema failed: %v", err)
	}
	hasDoc1 := false
	for _, id := range lookupResp.EntityIds {
		if id == "doc1" {
			hasDoc1 = true
			break
		}
	}
	if !hasDoc1 {
		t.Error("charlie should have edit permission on doc1 in v2 schema")
	}
	t.Log("✓ LookupEntity API works with specific schema version")

	t.Log("✓ All schema versioning tests passed")
}
