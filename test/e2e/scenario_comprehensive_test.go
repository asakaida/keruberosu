package e2e

import (
	"context"
	"sort"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestScenario_ReBAC_ABAC_Combined tests a schema that combines ReBAC
// (relation-based) and ABAC (attribute-based) in a single entity.
func TestScenario_ReBAC_ABAC_Combined(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Step 1: Define schema combining ReBAC and ABAC
	t.Log("Step 1: Defining combined ReBAC + ABAC schema")
	schema := `
rule check_public(resource) {
  resource.is_public == true
}

entity user {}

entity document {
  relation owner @user
  relation viewer @user
  attribute is_public boolean

  permission edit = owner
  permission view = owner or viewer or check_public(resource)
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}
	t.Log("Schema defined successfully")

	// Step 2: Write relation tuples and attributes
	t.Log("Step 2: Writing tuples and attributes")

	// doc1: owned by alice, bob is viewer, public
	// doc2: owned by alice, not public
	// doc3: no owner/viewer, public
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
		Attributes: []*pb.Attribute{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "is_public", Value: structpb.NewBoolValue(true)},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Attribute: "is_public", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Attribute: "is_public", Value: structpb.NewBoolValue(true)},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}
	t.Log("Tuples and attributes written successfully")

	// Step 3: Check permissions
	t.Log("Step 3: Testing Check API with combined ReBAC + ABAC")
	checkCases := []struct {
		name       string
		entityID   string
		permission string
		subjectID  string
		expected   pb.CheckResult
	}{
		// doc1: alice is owner
		{"alice can edit doc1 (owner)", "doc1", "edit", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"alice can view doc1 (owner)", "doc1", "view", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		// doc1: bob is viewer
		{"bob cannot edit doc1 (not owner)", "doc1", "edit", "bob", pb.CheckResult_CHECK_RESULT_DENIED},
		{"bob can view doc1 (viewer)", "doc1", "view", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},
		// doc1: charlie has no relation but doc is public
		{"charlie cannot edit doc1", "doc1", "edit", "charlie", pb.CheckResult_CHECK_RESULT_DENIED},
		{"charlie can view doc1 (public)", "doc1", "view", "charlie", pb.CheckResult_CHECK_RESULT_ALLOWED},
		// doc2: private, only alice is owner
		{"alice can view doc2 (owner)", "doc2", "view", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob cannot view doc2 (private, no relation)", "doc2", "view", "bob", pb.CheckResult_CHECK_RESULT_DENIED},
		// doc3: public, no relations
		{"charlie can view doc3 (public)", "doc3", "view", "charlie", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"charlie cannot edit doc3 (no owner)", "doc3", "edit", "charlie", pb.CheckResult_CHECK_RESULT_DENIED},
	}

	for _, tc := range checkCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
				Entity:     &pb.Entity{Type: "document", Id: tc.entityID},
				Permission: tc.permission,
				Subject:    &pb.Subject{Type: "user", Id: tc.subjectID},
			})
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}
			if resp.Can != tc.expected {
				t.Errorf("got %v, want %v", resp.Can, tc.expected)
			}
		})
	}

	// Step 4: LookupEntity for viewer
	t.Log("Step 4: Testing LookupEntity with combined ReBAC + ABAC")

	// alice can view doc1 (owner), doc2 (owner), doc3 (public)
	lookupAlice, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity alice failed: %v", err)
	}
	sort.Strings(lookupAlice.EntityIds)
	expected := []string{"doc1", "doc2", "doc3"}
	sort.Strings(expected)
	if len(lookupAlice.EntityIds) != len(expected) {
		t.Errorf("LookupEntity alice: got %v, want %v", lookupAlice.EntityIds, expected)
	} else {
		for i, id := range lookupAlice.EntityIds {
			if id != expected[i] {
				t.Errorf("LookupEntity alice: got %v, want %v", lookupAlice.EntityIds, expected)
				break
			}
		}
	}
	t.Logf("LookupEntity found %d entities for alice: %v", len(lookupAlice.EntityIds), lookupAlice.EntityIds)

	// charlie can view doc1 (public), doc3 (public), but not doc2 (private)
	lookupCharlie, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "charlie"},
	})
	if err != nil {
		t.Fatalf("LookupEntity charlie failed: %v", err)
	}
	sort.Strings(lookupCharlie.EntityIds)
	expectedCharlie := []string{"doc1", "doc3"}
	sort.Strings(expectedCharlie)
	if len(lookupCharlie.EntityIds) != len(expectedCharlie) {
		t.Errorf("LookupEntity charlie: got %v, want %v", lookupCharlie.EntityIds, expectedCharlie)
	} else {
		for i, id := range lookupCharlie.EntityIds {
			if id != expectedCharlie[i] {
				t.Errorf("LookupEntity charlie: got %v, want %v", lookupCharlie.EntityIds, expectedCharlie)
				break
			}
		}
	}
	t.Logf("LookupEntity found %d entities for charlie: %v", len(lookupCharlie.EntityIds), lookupCharlie.EntityIds)

	// Step 5: SubjectPermission
	t.Log("Step 5: Testing SubjectPermission with combined ReBAC + ABAC")
	subjResp, err := permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "doc1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("SubjectPermission failed: %v", err)
	}
	// alice is owner of doc1: edit=ALLOWED, view=ALLOWED, owner=ALLOWED, viewer=DENIED
	if subjResp.Results["edit"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected edit=ALLOWED for alice on doc1, got %v", subjResp.Results["edit"])
	}
	if subjResp.Results["view"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected view=ALLOWED for alice on doc1, got %v", subjResp.Results["view"])
	}
	t.Logf("SubjectPermission results for alice on doc1: %v", subjResp.Results)

	// bob is not owner: edit=DENIED
	subjRespBob, err := permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "doc1"},
		Subject: &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("SubjectPermission bob failed: %v", err)
	}
	if subjRespBob.Results["edit"] != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Errorf("expected edit=DENIED for bob on doc1, got %v", subjRespBob.Results["edit"])
	}
	if subjRespBob.Results["view"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected view=ALLOWED for bob on doc1, got %v", subjRespBob.Results["view"])
	}
	t.Logf("SubjectPermission results for bob on doc1: %v", subjRespBob.Results)

	t.Log("All ReBAC + ABAC combined scenario tests passed")
}

// TestScenario_DataLifecycle tests the full data lifecycle:
// write -> check -> update attribute -> check -> delete -> check
func TestScenario_DataLifecycle(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Step 1: Define schema
	t.Log("Step 1: Defining schema with rule-based permission")
	schema := `
rule is_published(resource) {
  resource.is_draft == false
}

entity user {}

entity document {
  relation editor @user
  attribute is_draft boolean

  permission view = editor or is_published(resource)
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}
	t.Log("Schema defined successfully")

	// Step 2: Write editor relation for alice
	t.Log("Step 2: Writing editor relation for alice on doc1")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Step 3: Check alice can view (as editor)
	t.Log("Step 3: Checking alice can view doc1 as editor")
	checkAlice, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice failed: %v", err)
	}
	if checkAlice.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to view doc1 (editor)")
	}
	t.Log("alice can view doc1 as editor: ALLOWED")

	// Step 4: Write is_draft=true
	t.Log("Step 4: Setting doc1 is_draft=true")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Attributes: []*pb.Attribute{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "is_draft", Value: structpb.NewBoolValue(true)},
		},
	})
	if err != nil {
		t.Fatalf("WriteAttributes failed: %v", err)
	}

	// Step 5: Check bob (non-editor) cannot view (is_draft=true means not published)
	t.Log("Step 5: Checking bob cannot view doc1 (draft, non-editor)")
	checkBob, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check bob failed: %v", err)
	}
	if checkBob.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected bob to be denied view on doc1 (draft, non-editor)")
	}
	t.Log("bob cannot view doc1 (draft): DENIED")

	// Step 6: Update is_draft=false (publish the document)
	t.Log("Step 6: Publishing doc1 (is_draft=false)")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Attributes: []*pb.Attribute{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "is_draft", Value: structpb.NewBoolValue(false)},
		},
	})
	if err != nil {
		t.Fatalf("WriteAttributes (publish) failed: %v", err)
	}

	// Step 7: Check bob can now view (published)
	t.Log("Step 7: Checking bob can view doc1 (published)")
	checkBob2, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check bob (published) failed: %v", err)
	}
	if checkBob2.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected bob to view doc1 (published)")
	}
	t.Log("bob can view doc1 (published): ALLOWED")

	// Step 8: Delete alice's editor relation
	t.Log("Step 8: Deleting alice's editor relation on doc1")
	_, err = dataClient.Delete(ctx, &pb.DataDeleteRequest{
		Filter: &pb.TupleFilter{
			Entity:   &pb.EntityFilter{Type: "document", Ids: []string{"doc1"}},
			Relation: "editor",
			Subject:  &pb.SubjectFilter{Type: "user", Ids: []string{"alice"}},
		},
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Step 9: Check alice can still view (published, even though no longer editor)
	t.Log("Step 9: Checking alice can still view doc1 (published)")
	checkAlice2, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice (after delete) failed: %v", err)
	}
	if checkAlice2.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to still view doc1 (published)")
	}
	t.Log("alice can still view doc1 (published, no longer editor): ALLOWED")

	// Step 10: Check alice no longer has editor relation
	t.Log("Step 10: Checking alice no longer has editor relation")
	checkAliceEditor, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "editor",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice editor failed: %v", err)
	}
	if checkAliceEditor.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected alice to be denied editor relation on doc1 (deleted)")
	}
	t.Log("alice no longer has editor relation on doc1: DENIED")

	t.Log("All DataLifecycle scenario tests passed")
}

// TestScenario_TenantIsolation tests that data written for one tenant
// does not leak to another tenant.
func TestScenario_TenantIsolation(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Step 1: Write schema for tenant org-a
	t.Log("Step 1: Writing schema for tenant org-a")
	schemaA := `
entity user {}

entity document {
  relation owner @user
  permission view = owner
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		TenantId: "org-a",
		Schema:   schemaA,
	})
	if err != nil {
		t.Fatalf("WriteSchema org-a failed: %v", err)
	}

	// Step 2: Write schema for tenant org-b
	t.Log("Step 2: Writing schema for tenant org-b")
	schemaB := `
entity user {}

entity document {
  relation owner @user
  relation viewer @user
  permission view = owner or viewer
}
`
	_, err = schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		TenantId: "org-b",
		Schema:   schemaB,
	})
	if err != nil {
		t.Fatalf("WriteSchema org-b failed: %v", err)
	}

	// Step 3: Write data for tenant org-a
	t.Log("Step 3: Writing data for tenant org-a")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		TenantId: "org-a",
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData org-a failed: %v", err)
	}

	// Step 4: Write data for tenant org-b
	t.Log("Step 4: Writing data for tenant org-b")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		TenantId: "org-b",
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData org-b failed: %v", err)
	}

	// Step 5: Check org-a: alice can view doc1
	t.Log("Step 5: Checking permissions in org-a")
	checkA, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		TenantId:   "org-a",
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check org-a alice failed: %v", err)
	}
	if checkA.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to view doc1 in org-a")
	}
	t.Log("org-a: alice can view doc1: ALLOWED")

	// Step 6: Check org-a: bob cannot view doc1 (bob's data is in org-b)
	checkABob, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		TenantId:   "org-a",
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check org-a bob failed: %v", err)
	}
	if checkABob.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected bob to be denied doc1 in org-a (bob's data is in org-b)")
	}
	t.Log("org-a: bob cannot view doc1: DENIED (data is in org-b)")

	// Step 7: Check org-b: bob can view doc1
	checkB, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		TenantId:   "org-b",
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check org-b bob failed: %v", err)
	}
	if checkB.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected bob to view doc1 in org-b")
	}
	t.Log("org-b: bob can view doc1: ALLOWED")

	// Step 8: Check org-b: alice cannot view doc1 (alice's data is in org-a)
	checkBAlice, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		TenantId:   "org-b",
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check org-b alice failed: %v", err)
	}
	if checkBAlice.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected alice to be denied doc1 in org-b (alice's data is in org-a)")
	}
	t.Log("org-b: alice cannot view doc1: DENIED (data is in org-a)")

	// Step 9: LookupEntity per tenant
	t.Log("Step 9: Testing LookupEntity per tenant")

	lookupA, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		TenantId:   "org-a",
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity org-a alice failed: %v", err)
	}
	if len(lookupA.EntityIds) != 1 || lookupA.EntityIds[0] != "doc1" {
		t.Errorf("org-a: expected [doc1] for alice, got %v", lookupA.EntityIds)
	}
	t.Logf("org-a LookupEntity for alice: %v", lookupA.EntityIds)

	lookupB, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		TenantId:   "org-b",
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("LookupEntity org-b bob failed: %v", err)
	}
	if len(lookupB.EntityIds) != 1 || lookupB.EntityIds[0] != "doc1" {
		t.Errorf("org-b: expected [doc1] for bob, got %v", lookupB.EntityIds)
	}
	t.Logf("org-b LookupEntity for bob: %v", lookupB.EntityIds)

	lookupBCharlie, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		TenantId:   "org-b",
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "charlie"},
	})
	if err != nil {
		t.Fatalf("LookupEntity org-b charlie failed: %v", err)
	}
	if len(lookupBCharlie.EntityIds) != 1 || lookupBCharlie.EntityIds[0] != "doc2" {
		t.Errorf("org-b: expected [doc2] for charlie, got %v", lookupBCharlie.EntityIds)
	}
	t.Logf("org-b LookupEntity for charlie: %v", lookupBCharlie.EntityIds)

	t.Log("All TenantIsolation scenario tests passed")
}

// TestScenario_ErrorHandling tests various error conditions.
func TestScenario_ErrorHandling(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Write a valid schema first so the server has something to work with
	schema := `
entity user {}

entity document {
  relation owner @user
  permission view = owner
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// Test 1: Check on non-existent entity type
	t.Log("Test 1: Check on non-existent entity type")
	_, err = permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "nonexistent", Id: "id1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err == nil {
		t.Error("expected error for Check on non-existent entity type, got nil")
	} else {
		t.Logf("Check on non-existent entity type returned error: %v", err)
	}

	// Test 2: LookupEntity with empty subject ID
	t.Log("Test 2: LookupEntity with empty subject ID")
	_, err = permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: ""},
	})
	if err == nil {
		t.Error("expected error for LookupEntity with empty subject ID, got nil")
	} else {
		t.Logf("LookupEntity with empty subject ID returned error: %v", err)
	}

	// Test 3: Write schema with invalid DSL
	t.Log("Test 3: Write schema with invalid DSL")
	_, err = schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: "this is not valid schema DSL {{{",
	})
	if err == nil {
		t.Error("expected error for invalid schema DSL, got nil")
	} else {
		t.Logf("Invalid schema DSL returned error: %v", err)
	}

	// Test 4: Delete with no filter criteria
	t.Log("Test 4: Delete with no filter criteria")
	_, err = dataClient.Delete(ctx, &pb.DataDeleteRequest{
		Filter: &pb.TupleFilter{},
	})
	if err == nil {
		t.Error("expected error for Delete with no filter criteria, got nil")
	} else {
		t.Logf("Delete with no filter returned error: %v", err)
	}

	t.Log("All ErrorHandling scenario tests passed")
}

// TestScenario_ComplexPermissionLogic tests AND/OR/NOT combinations
// in permission definitions.
func TestScenario_ComplexPermissionLogic(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Step 1: Define schema with AND/OR/NOT
	t.Log("Step 1: Defining schema with AND/OR/NOT permission logic")
	schema := `
entity user {}

entity document {
  relation owner @user
  relation editor @user
  relation banned @user

  permission edit = (owner or editor) and not banned
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}
	t.Log("Schema defined successfully")

	// Step 2: Write relations
	t.Log("Step 2: Writing relations")
	// alice: owner, not banned
	// bob: editor, not banned
	// charlie: owner AND banned
	// dave: neither owner nor editor
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// doc1 relations
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "banned", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}
	t.Log("Relations written successfully")

	// Step 3: Check permissions
	t.Log("Step 3: Testing Check API with AND/OR/NOT logic")
	checkCases := []struct {
		name     string
		subject  string
		expected pb.CheckResult
	}{
		{"alice (owner, not banned) can edit", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"bob (editor, not banned) can edit", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"charlie (owner but banned) cannot edit", "charlie", pb.CheckResult_CHECK_RESULT_DENIED},
		{"dave (neither owner nor editor) cannot edit", "dave", pb.CheckResult_CHECK_RESULT_DENIED},
	}

	for _, tc := range checkCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
				Entity:     &pb.Entity{Type: "document", Id: "doc1"},
				Permission: "edit",
				Subject:    &pb.Subject{Type: "user", Id: tc.subject},
			})
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}
			if resp.Can != tc.expected {
				t.Errorf("got %v, want %v", resp.Can, tc.expected)
			}
		})
	}

	// Step 4: LookupEntity with AND+NOT
	t.Log("Step 4: Testing LookupEntity with AND+NOT logic")

	// alice should find doc1 (owner, not banned)
	lookupAlice, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "edit",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity alice failed: %v", err)
	}
	if len(lookupAlice.EntityIds) != 1 || lookupAlice.EntityIds[0] != "doc1" {
		t.Errorf("expected [doc1] for alice, got %v", lookupAlice.EntityIds)
	}
	t.Logf("LookupEntity for alice (owner, not banned): %v", lookupAlice.EntityIds)

	// charlie should find nothing (banned)
	lookupCharlie, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "edit",
		Subject:    &pb.Subject{Type: "user", Id: "charlie"},
	})
	if err != nil {
		t.Fatalf("LookupEntity charlie failed: %v", err)
	}
	if len(lookupCharlie.EntityIds) != 0 {
		t.Errorf("expected [] for charlie (banned), got %v", lookupCharlie.EntityIds)
	}
	t.Logf("LookupEntity for charlie (owner but banned): %v", lookupCharlie.EntityIds)

	// dave should find nothing (no relation)
	lookupDave, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "edit",
		Subject:    &pb.Subject{Type: "user", Id: "dave"},
	})
	if err != nil {
		t.Fatalf("LookupEntity dave failed: %v", err)
	}
	if len(lookupDave.EntityIds) != 0 {
		t.Errorf("expected [] for dave (no relation), got %v", lookupDave.EntityIds)
	}
	t.Logf("LookupEntity for dave (no relation): %v", lookupDave.EntityIds)

	// Step 5: LookupSubject for edit permission
	t.Log("Step 5: Testing LookupSubject with AND+NOT logic")
	lookupSubj, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:           &pb.Entity{Type: "document", Id: "doc1"},
		Permission:       "edit",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}
	sort.Strings(lookupSubj.SubjectIds)
	// Only alice and bob should have edit (charlie is banned, dave has no relation)
	expectedSubjects := []string{"alice", "bob"}
	sort.Strings(expectedSubjects)
	if len(lookupSubj.SubjectIds) != len(expectedSubjects) {
		t.Errorf("expected %v, got %v", expectedSubjects, lookupSubj.SubjectIds)
	} else {
		for i, id := range lookupSubj.SubjectIds {
			if id != expectedSubjects[i] {
				t.Errorf("expected %v, got %v", expectedSubjects, lookupSubj.SubjectIds)
				break
			}
		}
	}
	t.Logf("LookupSubject for doc1.edit: %v", lookupSubj.SubjectIds)

	t.Log("All ComplexPermissionLogic scenario tests passed")
}
