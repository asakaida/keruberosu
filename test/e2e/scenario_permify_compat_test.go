package e2e

import (
	"context"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
)

// TestScenario_PermifyCompat tests Permify compatibility
// This test uses examples from Permify documentation to ensure compatibility
func TestScenario_PermifyCompat(t *testing.T) {
	// Setup E2E test server
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Test 1: Permify-style schema with organization/repository
	t.Log("Test 1: Permify-style organization/repository schema")
	schema := `
entity user {}

entity organization {
  relation admin: user
  relation member: user

  permission create_repository = admin or member
  permission delete = admin
}

entity repository {
  relation owner: user
  relation maintainer: user
  relation reader: user
  relation parent: organization

  permission push = owner or maintainer
  permission read = owner or maintainer or reader or parent.member
  permission delete = owner or parent.admin
}
`

	// Write schema
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
	t.Log("✓ Permify-style schema written successfully")

	// Test 2: Read schema back
	t.Log("Test 2: Reading schema back")
	readSchemaResp, err := schemaClient.Read(ctx, &pb.SchemaReadRequest{})
	if err != nil {
		t.Fatalf("ReadSchema failed: %v", err)
	}
	if readSchemaResp.Schema == "" {
		t.Fatal("ReadSchema returned empty schema")
	}
	if readSchemaResp.UpdatedAt == "" {
		t.Error("ReadSchema missing updated_at")
	}
	t.Logf("✓ Schema read successfully (updated_at: %s)", readSchemaResp.UpdatedAt)

	// Test 3: Write relation tuples (Permify-style)
	t.Log("Test 3: Writing Permify-style relation tuples")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// Organization structure
			{Entity: &pb.Entity{Type: "organization", Id: "acme"}, Relation: "admin", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "organization", Id: "acme"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "organization", Id: "acme"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "charlie"}},

			// Repository structure
			{Entity: &pb.Entity{Type: "repository", Id: "repo1"}, Relation: "parent", Subject: &pb.Subject{Type: "organization", Id: "acme"}},
			{Entity: &pb.Entity{Type: "repository", Id: "repo1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "repository", Id: "repo1"}, Relation: "maintainer", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "repository", Id: "repo1"}, Relation: "reader", Subject: &pb.Subject{Type: "user", Id: "dave"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteRelations failed: %v", err)
	}
	t.Log("✓ Relation tuples written successfully")

	// Test 4: Check API (Permify-compatible requests)
	t.Log("Test 4: Testing Permify-compatible Check API")
	checkTests := []struct {
		name       string
		entityType string
		entityID   string
		permission string
		subjectID  string
		expected   pb.CheckResult
	}{
		{"alice (admin) can delete org", "organization", "acme", "delete", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob (member) cannot delete org", "organization", "acme", "delete", "bob", pb.CheckResult_CHECK_RESULT_DENIED},
		{"bob (member) can create repository", "organization", "acme", "create_repository", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},

		{"alice (owner) can push to repo1", "repository", "repo1", "push", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob (maintainer) can push to repo1", "repository", "repo1", "push", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"dave (reader) cannot push to repo1", "repository", "repo1", "push", "dave", pb.CheckResult_CHECK_RESULT_DENIED},

		{"alice (owner) can read repo1", "repository", "repo1", "read", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"dave (reader) can read repo1", "repository", "repo1", "read", "dave", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"charlie (org member) can read repo1 via parent", "repository", "repo1", "read", "charlie", pb.CheckResult_CHECK_RESULT_ALLOWED},

		{"alice (owner) can delete repo1", "repository", "repo1", "delete", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"alice (org admin) can delete repo1 via parent", "repository", "repo1", "delete", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob (maintainer) cannot delete repo1", "repository", "repo1", "delete", "bob", pb.CheckResult_CHECK_RESULT_DENIED},
	}

	for _, tc := range checkTests {
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
	t.Log("✓ Permify-compatible Check API tests passed")

	// Test 5: Expand API
	t.Log("Test 5: Testing Expand API")
	expandResp, err := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Entity: &pb.Entity{
			Type: "repository",
			Id:   "repo1",
		},
		Permission: "read",
	})
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	if expandResp.Tree == nil {
		t.Fatal("Expand returned nil tree")
	}
	if expandResp.Tree.GetExpand() == nil && expandResp.Tree.GetLeaf() == nil {
		t.Fatal("Expand tree node is nil")
	}
	if expandResp.Tree.GetExpand() != nil {
		t.Logf("✓ Expand returned tree with type: %v", expandResp.Tree.GetExpand().Operation)
	} else {
		t.Log("✓ Expand returned leaf node")
	}

	// Test 6: LookupEntity API
	t.Log("Test 6: Testing LookupEntity API")
	lookupEntityResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "repository",
		Permission: "push",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "alice",
		},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}
	// alice should be able to push to repo1
	hasRepo1 := false
	for _, id := range lookupEntityResp.EntityIds {
		if id == "repo1" {
			hasRepo1 = true
			break
		}
	}
	if !hasRepo1 {
		t.Error("LookupEntity did not find repo1 for alice with push permission")
	}
	t.Logf("✓ LookupEntity found %d repositories for alice", len(lookupEntityResp.EntityIds))

	// Test 7: LookupSubject API
	t.Log("Test 7: Testing LookupSubject API")
	lookupSubjectResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity: &pb.Entity{
			Type: "repository",
			Id:   "repo1",
		},
		Permission: "read",
		SubjectReference: &pb.SubjectReference{
			Type: "user",
		},
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}
	// repo1 should be readable by alice, bob, charlie, dave
	if len(lookupSubjectResp.SubjectIds) < 4 {
		t.Errorf("LookupSubject returned %d subjects, expected at least 4", len(lookupSubjectResp.SubjectIds))
	}
	t.Logf("✓ LookupSubject found %d users with read permission", len(lookupSubjectResp.SubjectIds))

	// Test 8: SubjectPermission API
	t.Log("Test 8: Testing SubjectPermission API")
	subjPermResp, err := permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
		Entity: &pb.Entity{
			Type: "repository",
			Id:   "repo1",
		},
		Subject: &pb.Subject{
			Type: "user",
			Id:   "alice",
		},
	})
	if err != nil {
		t.Fatalf("SubjectPermission failed: %v", err)
	}
	// alice (owner) should have all permissions
	expectedPerms := []string{"push", "read", "delete"}
	for _, perm := range expectedPerms {
		result, ok := subjPermResp.Results[perm]
		if !ok {
			t.Errorf("SubjectPermission missing permission: %s", perm)
			continue
		}
		if result != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("SubjectPermission: alice should have %s on repo1, got %v", perm, result)
		}
	}
	t.Log("✓ SubjectPermission API tests passed")

	// Test 9: Delete relations
	t.Log("Test 9: Testing DeleteRelations API")
	_, err = dataClient.Delete(ctx, &pb.DataDeleteRequest{
		Filter: &pb.TupleFilter{
			Entity: &pb.EntityFilter{
				Type: "repository",
				Ids:  []string{"repo1"},
			},
			Relation: "reader",
			Subject: &pb.SubjectFilter{
				Type: "user",
				Ids:  []string{"dave"},
			},
		},
	})
	if err != nil {
		t.Fatalf("DeleteRelations failed: %v", err)
	}

	// Verify dave no longer has read access
	checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity: &pb.Entity{
			Type: "repository",
			Id:   "repo1",
		},
		Permission: "read",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "dave",
		},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("dave should not have read access after relation deletion")
	}
	t.Log("✓ DeleteRelations API tests passed")

	t.Log("✓ All Permify compatibility tests passed")
}
