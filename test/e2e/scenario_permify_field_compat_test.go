package e2e

import (
	"context"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
)

// TestScenario_PermifyFieldCompat tests Permify field-level compatibility
// This test explicitly verifies new fields like tenant_id, scope, page_size
func TestScenario_PermifyFieldCompat(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Setup: Define schema
	schema := `
entity user {}

entity document {
  relation owner: user
  relation viewer: user

  permission view = owner or viewer
  permission edit = owner
}
`

	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// Setup: Write test data
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteRelations failed: %v", err)
	}

	// Test 1: tenant_id field - empty should use "default"
	t.Log("Test 1: tenant_id field compatibility (empty = default)")
	checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		TenantId:   "", // Empty tenant_id should default to "default"
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check with empty tenant_id failed: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("Check with empty tenant_id should work")
	}
	t.Log("✓ Empty tenant_id defaults to 'default'")

	// Test 2: tenant_id field - explicit "default"
	t.Log("Test 2: tenant_id field with explicit 'default'")
	checkResp, err = permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		TenantId:   "default", // Explicit tenant_id
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check with explicit tenant_id failed: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("Check with explicit tenant_id='default' should work")
	}
	t.Log("✓ Explicit tenant_id='default' works")

	// Test 3: LookupEntity with page_size (uint32 type)
	t.Log("Test 3: LookupEntity with page_size (uint32)")
	lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		TenantId:   "default",
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
		PageSize:   10, // uint32 type
	})
	if err != nil {
		t.Fatalf("LookupEntity with page_size failed: %v", err)
	}
	if len(lookupResp.EntityIds) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(lookupResp.EntityIds))
	}
	t.Logf("✓ LookupEntity with page_size=10 returned %d entities", len(lookupResp.EntityIds))

	// Test 4: LookupEntity with scope parameter (currently not used in implementation but should not error)
	t.Log("Test 4: LookupEntity with scope parameter")
	lookupResp, err = permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		TenantId:   "default",
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
		Scope:      map[string]*pb.StringArrayValue{
			"document": {Data: []string{"doc1", "doc2"}},
		},
		PageSize: 5,
	})
	if err != nil {
		t.Fatalf("LookupEntity with scope failed: %v", err)
	}
	// Scope may not be implemented yet, but API should accept it without error
	t.Logf("✓ LookupEntity accepts scope parameter (returned %d entities)", len(lookupResp.EntityIds))

	// Test 5: LookupSubject with tenant_id and page_size
	t.Log("Test 5: LookupSubject with tenant_id and page_size")
	lookupSubjectResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		TenantId:         "default",
		Entity:           &pb.Entity{Type: "document", Id: "doc1"},
		Permission:       "view",
		SubjectReference: &pb.SubjectReference{Type: "user"},
		PageSize:         10,
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}
	if len(lookupSubjectResp.SubjectIds) != 2 {
		t.Errorf("Expected 2 subjects for doc1, got %d", len(lookupSubjectResp.SubjectIds))
	}
	t.Logf("✓ LookupSubject with tenant_id and page_size returned %d subjects", len(lookupSubjectResp.SubjectIds))

	// Test 6: SubjectPermission with tenant_id
	t.Log("Test 6: SubjectPermission with tenant_id")
	subjPermResp, err := permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
		TenantId: "default",
		Entity:   &pb.Entity{Type: "document", Id: "doc1"},
		Subject:  &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("SubjectPermission failed: %v", err)
	}
	if len(subjPermResp.Results) != 2 {
		t.Errorf("Expected 2 permissions (view, edit), got %d", len(subjPermResp.Results))
	}
	if subjPermResp.Results["view"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("alice should have view permission")
	}
	if subjPermResp.Results["edit"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("alice should have edit permission")
	}
	t.Log("✓ SubjectPermission with tenant_id works correctly")

	// Test 7: Expand with tenant_id
	t.Log("Test 7: Expand with tenant_id")
	expandResp, err := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		TenantId:   "default",
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
	})
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	if expandResp.Tree == nil {
		t.Fatal("Expand returned nil tree")
	}
	if expandResp.Tree.GetExpand() == nil && expandResp.Tree.GetLeaf() == nil {
		t.Fatal("Expand tree has no expand or leaf node")
	}
	t.Log("✓ Expand with tenant_id works correctly")

	// Test 8: Verify all Permission APIs work without tenant_id (backwards compatibility)
	t.Log("Test 8: Backwards compatibility - all APIs work without tenant_id")

	// Check without tenant_id
	_, err = permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Errorf("Check without tenant_id failed: %v", err)
	}

	// LookupEntity without tenant_id
	_, err = permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Errorf("LookupEntity without tenant_id failed: %v", err)
	}

	// LookupSubject without tenant_id
	_, err = permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:           &pb.Entity{Type: "document", Id: "doc1"},
		Permission:       "view",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	})
	if err != nil {
		t.Errorf("LookupSubject without tenant_id failed: %v", err)
	}

	// SubjectPermission without tenant_id
	_, err = permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "doc1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Errorf("SubjectPermission without tenant_id failed: %v", err)
	}

	// Expand without tenant_id
	_, err = permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
	})
	if err != nil {
		t.Errorf("Expand without tenant_id failed: %v", err)
	}

	t.Log("✓ All APIs work without tenant_id (backwards compatibility)")

	t.Log("✓ All Permify field compatibility tests passed")
}
