package e2e

import (
	"context"
	"sort"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
)

// TestScenario_ClosureLookup_OptimizedPath tests the Lookup API's optimized
// closure table path. This uses a 2-level hierarchy where the target entity's
// permission resolves to a simple relation (hasUnresolvable=false).
//
// Schema: project.access = team.member (team.member is a relation, not a permission)
// This makes extractRelationsFromRuleWithContext produce parentRelations=["team.member"]
// without setting hasUnresolvable, so LookupAccessibleEntitiesComplex is used.
func TestScenario_ClosureLookup_OptimizedPath(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	schema := `
entity user {}

entity team {
  relation member @user
}

entity project {
  relation owner @user
  relation team @team

  permission manage = owner
  permission access = owner or team.member
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// alice is member of team-alpha
	// team-alpha is team of project-x and project-y
	// bob owns project-z
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "team", Id: "team-alpha"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "project", Id: "project-x"}, Relation: "team", Subject: &pb.Subject{Type: "team", Id: "team-alpha"}},
			{Entity: &pb.Entity{Type: "project", Id: "project-y"}, Relation: "team", Subject: &pb.Subject{Type: "team", Id: "team-alpha"}},
			{Entity: &pb.Entity{Type: "project", Id: "project-z"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "bob"}},
		},
	})
	if err != nil {
		t.Fatalf("Write tuples failed: %v", err)
	}

	// Verify Check correctness first
	for _, tc := range []struct {
		project  string
		user     string
		expected pb.CheckResult
	}{
		{"project-x", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"project-y", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED},
		{"project-z", "alice", pb.CheckResult_CHECK_RESULT_DENIED},
		{"project-z", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED},
	} {
		resp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
			Entity:     &pb.Entity{Type: "project", Id: tc.project},
			Permission: "access",
			Subject:    &pb.Subject{Type: "user", Id: tc.user},
		})
		if err != nil {
			t.Fatalf("Check(%s, %s) failed: %v", tc.project, tc.user, err)
		}
		if resp.Can != tc.expected {
			t.Errorf("Check(%s, %s) = %v, want %v", tc.project, tc.user, resp.Can, tc.expected)
		}
	}

	// LookupEntity: find projects alice can access
	lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "project",
		Permission: "access",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}

	sort.Strings(lookupResp.EntityIds)
	expected := []string{"project-x", "project-y"}
	if len(lookupResp.EntityIds) != len(expected) {
		t.Errorf("LookupEntity: got %v, want %v", lookupResp.EntityIds, expected)
	} else {
		for i, id := range lookupResp.EntityIds {
			if id != expected[i] {
				t.Errorf("LookupEntity[%d]: got %s, want %s", i, id, expected[i])
			}
		}
	}
	t.Logf("LookupEntity for alice: %v", lookupResp.EntityIds)

	// LookupSubject: find users who can access project-x
	subjectResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:           &pb.Entity{Type: "project", Id: "project-x"},
		Permission:       "access",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}

	foundAlice := false
	for _, id := range subjectResp.SubjectIds {
		if id == "alice" {
			foundAlice = true
		}
	}
	if !foundAlice {
		t.Errorf("LookupSubject for project-x: expected alice, got %v", subjectResp.SubjectIds)
	}
	t.Logf("LookupSubject for project-x: %v", subjectResp.SubjectIds)
}

// TestScenario_ClosureLookup_InsertionOrder tests that Lookup returns
// correct results regardless of relation insertion order.
// Relations are added leaf-first (project→team before team→user).
func TestScenario_ClosureLookup_InsertionOrder(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	schema := `
entity user {}

entity department {
  relation head @user
}

entity project {
  relation dept @department

  permission view = dept.head
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// Add project→department FIRST (before department→user)
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "project", Id: "proj1"}, Relation: "dept", Subject: &pb.Subject{Type: "department", Id: "eng"}},
		},
	})
	if err != nil {
		t.Fatalf("Write project→dept failed: %v", err)
	}

	// Add department→user SECOND
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "department", Id: "eng"}, Relation: "head", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("Write dept→user failed: %v", err)
	}

	// Check should work (uses recursive evaluation, not closure)
	checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "project", Id: "proj1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("Check: expected ALLOWED, got %v", checkResp.Can)
	}

	// LookupEntity should also work (uses closure table)
	lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "project",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}

	found := false
	for _, id := range lookupResp.EntityIds {
		if id == "proj1" {
			found = true
		}
	}
	if !found {
		t.Errorf("LookupEntity: expected proj1, got %v", lookupResp.EntityIds)
	}
	t.Logf("LookupEntity for alice (leaf-first order): %v", lookupResp.EntityIds)
}

// TestScenario_ClosureLookup_DeletionPreservesAlternatePaths tests that
// deleting one path in a DAG preserves Lookup results via alternate paths.
func TestScenario_ClosureLookup_DeletionPreservesAlternatePaths(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	schema := `
entity user {}

entity team {
  relation lead @user
}

entity project {
  relation team_a @team
  relation team_b @team

  permission review = team_a.lead or team_b.lead
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// alice leads both team-x and team-y
	// project-p has team_a=team-x and team_b=team-y
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "team", Id: "team-x"}, Relation: "lead", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "team", Id: "team-y"}, Relation: "lead", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "project", Id: "project-p"}, Relation: "team_a", Subject: &pb.Subject{Type: "team", Id: "team-x"}},
			{Entity: &pb.Entity{Type: "project", Id: "project-p"}, Relation: "team_b", Subject: &pb.Subject{Type: "team", Id: "team-y"}},
		},
	})
	if err != nil {
		t.Fatalf("Write tuples failed: %v", err)
	}

	// Verify alice can review project-p via both paths
	checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "project", Id: "project-p"},
		Permission: "review",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Fatalf("Check: expected ALLOWED before delete, got %v", checkResp.Can)
	}

	// Delete one path: project-p team_a team-x
	_, err = dataClient.Delete(ctx, &pb.DataDeleteRequest{
		Filter: &pb.TupleFilter{
			Entity:   &pb.EntityFilter{Type: "project", Ids: []string{"project-p"}},
			Relation: "team_a",
			Subject:  &pb.SubjectFilter{Type: "team", Ids: []string{"team-x"}},
		},
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// alice should still have access via team_b=team-y
	checkResp, err = permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "project", Id: "project-p"},
		Permission: "review",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check after delete failed: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("Check after delete: expected ALLOWED via alternate path, got %v", checkResp.Can)
	}

	// LookupEntity should still find project-p
	lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "project",
		Permission: "review",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity after delete failed: %v", err)
	}

	found := false
	for _, id := range lookupResp.EntityIds {
		if id == "project-p" {
			found = true
		}
	}
	if !found {
		t.Errorf("LookupEntity after delete: expected project-p via alternate path, got %v", lookupResp.EntityIds)
	}
	t.Logf("LookupEntity after delete: %v", lookupResp.EntityIds)
}

// TestScenario_ClosureLookup_FallbackPath verifies that the fallback path
// (Check loop) works correctly for deep self-referential hierarchies
// where the optimized path cannot be used.
func TestScenario_ClosureLookup_FallbackPath(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	schema := `
entity user {}

entity folder {
  relation owner @user
  relation parent @folder

  permission view = owner or parent.view
}
`
	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// folder hierarchy: sub → mid → top
	// alice owns top
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "folder", Id: "top"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "folder", Id: "mid"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "top"}},
			{Entity: &pb.Entity{Type: "folder", Id: "sub"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "mid"}},
		},
	})
	if err != nil {
		t.Fatalf("Write tuples failed: %v", err)
	}

	// Check: alice can view all 3 folders
	for _, folderID := range []string{"top", "mid", "sub"} {
		checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
			Entity:     &pb.Entity{Type: "folder", Id: folderID},
			Permission: "view",
			Subject:    &pb.Subject{Type: "user", Id: "alice"},
		})
		if err != nil {
			t.Fatalf("Check(%s) failed: %v", folderID, err)
		}
		if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
			t.Errorf("Check(%s): expected ALLOWED, got %v", folderID, checkResp.Can)
		}
	}

	// LookupEntity should find all 3 folders (uses fallback path for self-referential)
	lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "folder",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}

	sort.Strings(lookupResp.EntityIds)
	expected := []string{"mid", "sub", "top"}
	if len(lookupResp.EntityIds) != len(expected) {
		t.Errorf("LookupEntity: got %v, want %v", lookupResp.EntityIds, expected)
	} else {
		for i, id := range lookupResp.EntityIds {
			if id != expected[i] {
				t.Errorf("LookupEntity[%d]: got %s, want %s", i, id, expected[i])
			}
		}
	}
	t.Logf("LookupEntity (fallback path): %v", lookupResp.EntityIds)
}
