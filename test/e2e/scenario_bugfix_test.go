package e2e

import (
	"context"
	"sort"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
)

// TestScenario_Bugfix_LookupANDPermission tests Bug1: LookupEntity with AND
// permission correctly uses fallback path instead of returning false positives.
func TestScenario_Bugfix_LookupANDPermission(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Schema: access requires BOTH member AND approved
	schema := `
entity user {}

entity resource {
  relation member @user
  relation approved @user

  permission access = member and approved
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// alice has both member and approved; bob only has member
	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "approved", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "bob"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Check: alice should have access, bob should NOT
	checkAlice, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "resource", Id: "res1"},
		Permission: "access",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice failed: %v", err)
	}
	if checkAlice.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to have access (both member and approved)")
	}

	checkBob, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "resource", Id: "res1"},
		Permission: "access",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check bob failed: %v", err)
	}
	if checkBob.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected bob to be denied (only member, not approved)")
	}

	// LookupEntity: should return ONLY res1 for alice, NOT for bob
	lookupAlice, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "resource",
		Permission: "access",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity alice failed: %v", err)
	}
	if len(lookupAlice.EntityIds) != 1 || lookupAlice.EntityIds[0] != "res1" {
		t.Errorf("expected [res1] for alice, got %v", lookupAlice.EntityIds)
	}

	lookupBob, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "resource",
		Permission: "access",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("LookupEntity bob failed: %v", err)
	}
	if len(lookupBob.EntityIds) != 0 {
		t.Errorf("expected [] for bob (AND not satisfied), got %v", lookupBob.EntityIds)
	}

	t.Log("Bug1: LookupEntity with AND permission correctly uses fallback path")
}

// TestScenario_Bugfix_LookupNOTPermission tests Bug1: LookupEntity with NOT
// permission correctly uses fallback path.
func TestScenario_Bugfix_LookupNOTPermission(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity resource {
  relation viewer @user
  relation blocked @user

  permission view = viewer and not blocked
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// alice is viewer only; bob is viewer AND blocked
	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "blocked", Subject: &pb.Subject{Type: "user", Id: "bob"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Check: alice should have view, bob should NOT (blocked)
	checkAlice, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "resource", Id: "res1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice failed: %v", err)
	}
	if checkAlice.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to have view (viewer, not blocked)")
	}

	checkBob, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "resource", Id: "res1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check bob failed: %v", err)
	}
	if checkBob.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected bob to be denied (viewer but blocked)")
	}

	// LookupEntity: should return ONLY res1 for alice, NOT for bob
	lookupAlice, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "resource",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity alice failed: %v", err)
	}
	if len(lookupAlice.EntityIds) != 1 || lookupAlice.EntityIds[0] != "res1" {
		t.Errorf("expected [res1] for alice, got %v", lookupAlice.EntityIds)
	}

	lookupBob, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "resource",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("LookupEntity bob failed: %v", err)
	}
	if len(lookupBob.EntityIds) != 0 {
		t.Errorf("expected [] for bob (NOT excluded), got %v", lookupBob.EntityIds)
	}

	t.Log("Bug1: LookupEntity with NOT permission correctly uses fallback path")
}

// TestScenario_Bugfix_SubjectPermissionIncludesRelations tests Bug8:
// SubjectPermission returns both permissions and relations.
func TestScenario_Bugfix_SubjectPermissionIncludesRelations(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity document {
  relation owner @user
  relation editor @user
  relation viewer @user

  permission delete = owner
  permission edit = owner or editor
  permission view = owner or editor or viewer
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// alice is owner of doc1
	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// SubjectPermission should return relations (owner, editor, viewer) AND permissions (delete, edit, view)
	resp, err := testServer.PermissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "doc1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("SubjectPermission failed: %v", err)
	}

	// alice is owner, so:
	// Relations: owner=ALLOWED, editor=DENIED, viewer=DENIED
	// Permissions: delete=ALLOWED, edit=ALLOWED, view=ALLOWED
	expectedAllowed := []string{"owner", "delete", "edit", "view"}
	expectedDenied := []string{"editor", "viewer"}

	for _, name := range expectedAllowed {
		result, ok := resp.Results[name]
		if !ok {
			t.Errorf("expected %s in results, not found", name)
			continue
		}
		if result != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("expected %s=ALLOWED, got %v", name, result)
		}
	}

	for _, name := range expectedDenied {
		result, ok := resp.Results[name]
		if !ok {
			t.Errorf("expected %s in results, not found", name)
			continue
		}
		if result != pb.CheckResult_CHECK_RESULT_DENIED {
			t.Errorf("expected %s=DENIED, got %v", name, result)
		}
	}

	// Verify total count: 3 relations + 3 permissions = 6
	if len(resp.Results) != 6 {
		t.Errorf("expected 6 results (3 relations + 3 permissions), got %d: %v", len(resp.Results), resp.Results)
	}

	t.Log("Bug8: SubjectPermission includes both relations and permissions")
}

// TestScenario_Bugfix_CheckWithSubjectRelation tests Bug7:
// Check API correctly handles subject relations (subject set checks).
func TestScenario_Bugfix_CheckWithSubjectRelation(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity team {
  relation member @user
}

entity document {
  relation viewer @team#member

  permission view = viewer
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "team", Id: "engineering"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "viewer", Subject: &pb.Subject{Type: "team", Id: "engineering", Relation: "member"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Check: alice should have view via team membership
	checkAlice, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice failed: %v", err)
	}
	if checkAlice.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to have view (via team:engineering#member)")
	}

	// Check with subject relation: team:engineering#member should have view
	checkTeam, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "team", Id: "engineering", Relation: "member"},
	})
	if err != nil {
		t.Fatalf("Check team failed: %v", err)
	}
	if checkTeam.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected team:engineering#member to have view")
	}

	// Check with wrong subject relation: team:engineering#admin should NOT have view
	checkAdmin, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "team", Id: "engineering", Relation: "admin"},
	})
	if err != nil {
		t.Fatalf("Check admin failed: %v", err)
	}
	if checkAdmin.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected team:engineering#admin to be denied")
	}

	t.Log("Bug7: Check API correctly handles subject relations")
}

// TestScenario_Bugfix_ExpandHierarchicalRelation tests Bug4:
// Expand correctly shows hierarchical rules targeting relations (not permissions).
func TestScenario_Bugfix_ExpandHierarchicalRelation(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity folder {
  relation owner @user
}

entity document {
  relation parent @folder

  permission view = parent.owner
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "folder1"}},
			{Entity: &pb.Entity{Type: "folder", Id: "folder1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Check: alice should have view via parent.owner
	checkAlice, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if checkAlice.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to have view (via parent.owner)")
	}

	// Expand: should show the full permission tree including the relation on parent
	expandResp, err := testServer.PermissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
	})
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	if expandResp.Tree == nil {
		t.Fatal("expected non-nil tree")
	}

	// The tree should not be empty - it should contain the expanded relation
	tree := expandResp.Tree
	if tree.GetExpand() != nil && len(tree.GetExpand().GetChildren()) == 0 {
		t.Error("expected non-empty children for hierarchical rule (parent.owner should expand to folder1.owner)")
	}

	t.Log("Bug4: Expand correctly handles hierarchical rules targeting relations")
}

// TestScenario_Bugfix_LookupSubjectWithAND tests Bug1: LookupSubject with AND
// permission correctly uses fallback path.
func TestScenario_Bugfix_LookupSubjectWithAND(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity resource {
  relation member @user
  relation approved @user

  permission access = member and approved
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "approved", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "resource", Id: "res1"}, Relation: "approved", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// LookupSubject: only alice should be returned (both member AND approved)
	lookupResp, err := testServer.PermissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:           &pb.Entity{Type: "resource", Id: "res1"},
		Permission:       "access",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}

	sort.Strings(lookupResp.SubjectIds)
	if len(lookupResp.SubjectIds) != 1 || lookupResp.SubjectIds[0] != "alice" {
		t.Errorf("expected [alice] (only one with both member+approved), got %v", lookupResp.SubjectIds)
	}

	t.Log("Bug1: LookupSubject with AND permission correctly uses fallback path")
}

// TestScenario_Bugfix_CheckRelationName tests Bug8: Check API correctly handles
// relation names (not just permission names) when used via SubjectPermission path.
func TestScenario_Bugfix_CheckRelationName(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity document {
  relation owner @user
  relation editor @user

  permission edit = owner or editor
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Check with relation name "owner" (not a permission name) should work
	checkOwner, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "owner",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check owner failed: %v", err)
	}
	if checkOwner.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to have owner relation")
	}

	// Non-owner should be denied
	checkBob, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "owner",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check bob owner failed: %v", err)
	}
	if checkBob.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected bob to be denied owner relation")
	}

	t.Log("Bug8: Check API correctly handles relation names")
}

// TestScenario_Bugfix_SubjectRelationPermissionComposition tests that SubjectRelation
// is preserved through permission composition chains (manage → edit → viewer).
func TestScenario_Bugfix_SubjectRelationPermissionComposition(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity team {
    relation member @user
}

entity document {
    relation viewer @user @team#member

    permission edit = viewer
    permission manage = edit
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "viewer", Subject: &pb.Subject{Type: "team", Id: "eng", Relation: "member"}},
			{Entity: &pb.Entity{Type: "team", Id: "eng"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// SubjectRelation through permission composition: manage → edit → viewer
	check, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "manage",
		Subject:    &pb.Subject{Type: "team", Id: "eng", Relation: "member"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if check.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected team:eng#member to have manage permission via composition manage→edit→viewer")
	}

	// user:alice should also have manage via computed userset expansion
	checkAlice, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "manage",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice failed: %v", err)
	}
	if checkAlice.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected user:alice to have manage permission via team:eng#member userset expansion")
	}

	t.Log("SubjectRelation correctly preserved through permission composition")
}

// TestScenario_Bugfix_SubjectRelationHierarchical tests that SubjectRelation
// is preserved through hierarchical permission evaluation (parent.view).
func TestScenario_Bugfix_SubjectRelationHierarchical(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity team {
    relation member @user
}

entity folder {
    relation viewer @user @team#member

    permission view = viewer
}

entity document {
    relation parent @folder

    permission view = parent.view
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "f1"}},
			{Entity: &pb.Entity{Type: "folder", Id: "f1"}, Relation: "viewer", Subject: &pb.Subject{Type: "team", Id: "eng", Relation: "member"}},
			{Entity: &pb.Entity{Type: "team", Id: "eng"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// SubjectRelation through hierarchical: parent.view
	check, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "team", Id: "eng", Relation: "member"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if check.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected team:eng#member to have view on doc1 via parent.view")
	}

	// user:alice should have access through both hierarchy and userset
	checkAlice, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice failed: %v", err)
	}
	if checkAlice.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected user:alice to have view on doc1 via team:eng#member → parent.view")
	}

	// user:bob should NOT have access
	checkBob, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check bob failed: %v", err)
	}
	if checkBob.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected user:bob to be denied view on doc1")
	}

	// LookupEntity should find doc1 for alice
	lookupResp, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}

	foundDoc1 := false
	for _, id := range lookupResp.EntityIds {
		if id == "doc1" {
			foundDoc1 = true
		}
	}
	if !foundDoc1 {
		t.Errorf("expected doc1 in LookupEntity results for alice, got: %v", lookupResp.EntityIds)
	}

	t.Log("SubjectRelation correctly preserved through hierarchical evaluation")
}

// TestScenario_Bugfix_HierarchicalRelationComputedUserset tests that when a
// hierarchical rule targets a parent's RELATION (not permission), computed
// usersets are properly expanded.
func TestScenario_Bugfix_HierarchicalRelationComputedUserset(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity group {
    relation member @user
}

entity folder {
    relation viewer @user @group#member
}

entity document {
    relation parent @folder

    permission view = parent.viewer
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "f1"}},
			{Entity: &pb.Entity{Type: "folder", Id: "f1"}, Relation: "viewer", Subject: &pb.Subject{Type: "group", Id: "eng", Relation: "member"}},
			{Entity: &pb.Entity{Type: "group", Id: "eng"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// parent.viewer (relation, not permission) with computed userset
	check, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if check.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected user:alice to have view via parent.viewer → group:eng#member expansion")
	}

	// bob should NOT have access
	checkBob, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check bob failed: %v", err)
	}
	if checkBob.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Error("expected user:bob to be denied view on doc1")
	}

	t.Log("Hierarchical relation correctly expands computed usersets")
}
