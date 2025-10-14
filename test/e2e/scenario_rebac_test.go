package e2e

import (
	"context"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
)

// TestScenario_ReBAC_GoogleDocs tests a Google Docs-like ReBAC scenario
func TestScenario_ReBAC_GoogleDocs(t *testing.T) {
	// Setup E2E test server
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Step 1: Define schema
	t.Log("Step 1: Defining schema (document, folder, user)")
	schema := `
entity user {}

entity folder {
  relation owner: user
  relation editor: user
  relation viewer: user

  permission edit = owner or editor
  permission view = owner or editor or viewer
}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user
  relation parent: folder

  permission delete = owner
  permission edit = owner or editor
  permission view = owner or editor or viewer or parent.view
}
`

	writeSchemaResp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}
	// WriteSchemaResponse now only contains SchemaVersion (Permify compatible)
	// Errors are returned via gRPC error, not in response fields
	if writeSchemaResp.SchemaVersion == "" {
		t.Logf("WriteSchema returned empty schema_version (expected for now - TODO: implement schema versioning)")
	}
	t.Log("✓ Schema defined successfully")

	// Step 2: Write relation tuples
	t.Log("Step 2: Writing relation tuples")
	// Folder hierarchy:
	// - folder1: owned by alice, bob is editor
	// - folder2: owned by charlie
	// Document hierarchy:
	// - doc1: owned by alice, parent is folder1
	// - doc2: bob is editor, parent is folder1
	// - doc3: owned by charlie, parent is folder2

	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// folder1
			{Entity: &pb.Entity{Type: "folder", Id: "folder1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "folder", Id: "folder1"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "bob"}},

			// folder2
			{Entity: &pb.Entity{Type: "folder", Id: "folder2"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "charlie"}},

			// doc1
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "folder1"}},

			// doc2
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "folder1"}},

			// doc3
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "folder2"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteRelations failed: %v", err)
	}
	// WriteRelationsResponse now only contains snap_token (Permify compatible)
	// WrittenCount field no longer exists
	t.Logf("✓ Relation tuples written successfully")


	// Step 3: Check permissions
	t.Log("Step 3: Testing Check API")

	testCases := []struct {
		name       string
		entityType string
		entityID   string
		permission string
		subjectID  string
		expected   pb.CheckResult
	}{
		// doc1: owned by alice
		{"alice can view doc1", "document", "doc1", "view", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"alice can edit doc1", "document", "doc1", "edit", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"alice can delete doc1", "document", "doc1", "delete", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},

		// doc1: bob is editor of parent folder1
		{"bob can view doc1 (via parent folder)", "document", "doc1", "view", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob cannot edit doc1 directly", "document", "doc1", "edit", "bob", pb.CheckResult_CHECK_RESULT_DENIED},
		{"bob cannot delete doc1", "document", "doc1", "delete", "bob", pb.CheckResult_CHECK_RESULT_DENIED},

		// doc2: bob is editor
		{"bob can view doc2", "document", "doc2", "view", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob can edit doc2", "document", "doc2", "edit", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob cannot delete doc2", "document", "doc2", "delete", "bob", pb.CheckResult_CHECK_RESULT_DENIED},

		// doc3: owned by charlie
		{"charlie can delete doc3", "document", "doc3", "delete", "charlie", pb.CheckResult_CHECK_RESULT_ALLOWED},

		// alice should not have access to doc3
		{"alice cannot view doc3", "document", "doc3", "view", "alice", pb.CheckResult_CHECK_RESULT_DENIED},

		// folder permissions
		{"alice can edit folder1", "folder", "folder1", "edit", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob can edit folder1", "folder", "folder1", "edit", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"charlie cannot view folder1", "folder", "folder1", "view", "charlie", pb.CheckResult_CHECK_RESULT_DENIED},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
				Entity: &pb.Entity{
					Type: tc.entityType,
					Id:   tc.entityID,
				},
				Permission: tc.permission,
				Subject: &pb.Subject{
					Type: "user",
					Id:   tc.subjectID,
				},
			})
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}
			if checkResp.Can != tc.expected {
				t.Errorf("Check result mismatch: got %v, want %v", checkResp.Can, tc.expected)
			}
		})
	}
	t.Log("✓ Check API tests passed")

	// Step 4: Test Expand API
	t.Log("Step 4: Testing Expand API")
	expandResp, err := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "view",
	})
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	if expandResp.Tree == nil {
		t.Fatal("Expand returned nil tree")
	}
	// Verify tree structure (should be a union node with multiple children)
	if expandResp.Tree.GetExpand() == nil {
		t.Fatal("Expand tree node is nil")
	}
	if expandResp.Tree.GetExpand().Operation != pb.ExpandTreeNode_OPERATION_UNION {
		t.Errorf("Expected union node, got %v", expandResp.Tree.GetExpand().Operation)
	}
	t.Logf("✓ Expand API returned tree with type: %v", expandResp.Tree.GetExpand().Operation)

	// Step 5: Test LookupEntity API
	t.Log("Step 5: Testing LookupEntity API")
	lookupEntityResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "alice",
		},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}
	// alice should be able to view doc1 (owned by alice)
	if len(lookupEntityResp.EntityIds) == 0 {
		t.Error("LookupEntity returned no entities for alice")
	}
	t.Logf("✓ LookupEntity found %d entities for alice", len(lookupEntityResp.EntityIds))

	// Step 6: Test LookupSubject API
	t.Log("Step 6: Testing LookupSubject API")
	lookupSubjectResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Permission: "view",
		SubjectReference: &pb.SubjectReference{
			Type: "user",
		},
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}
	// doc1 should be viewable by alice (owner) and bob (via parent folder)
	if len(lookupSubjectResp.SubjectIds) < 2 {
		t.Errorf("LookupSubject returned %d subjects, expected at least 2", len(lookupSubjectResp.SubjectIds))
	}
	t.Logf("✓ LookupSubject found %d subjects for doc1.view", len(lookupSubjectResp.SubjectIds))

	// Step 7: Test SubjectPermission API
	t.Log("Step 7: Testing SubjectPermission API")
	subjPermResp, err := permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc1",
		},
		Subject: &pb.Subject{
			Type: "user",
			Id:   "alice",
		},
	})
	if err != nil {
		t.Fatalf("SubjectPermission failed: %v", err)
	}
	// alice (owner of doc1) should have all permissions
	expectedPerms := []string{"view", "edit", "delete"}
	for _, perm := range expectedPerms {
		result, ok := subjPermResp.Results[perm]
		if !ok {
			t.Errorf("SubjectPermission missing permission: %s", perm)
			continue
		}
		if result != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("SubjectPermission: alice should have %s on doc1, got %v", perm, result)
		}
	}
	t.Log("✓ SubjectPermission API tests passed")

	t.Log("✓ All ReBAC scenario tests passed")
}
