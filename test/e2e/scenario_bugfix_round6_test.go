package e2e

import (
	"context"
	"sort"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
)

// TestScenario_Bugfix_LookupEntityWithRelationName tests that LookupEntity
// accepts relation names (not just permission names), consistent with Check API.
func TestScenario_Bugfix_LookupEntityWithRelationName(t *testing.T) {
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
  permission view = owner or editor
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Check with relation name works
	checkOwner, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "owner",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check owner failed: %v", err)
	}
	if checkOwner.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected Check(owner)=ALLOWED for alice on doc1")
	}

	// LookupEntity with relation name "owner" should also work
	lookupOwner, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "owner",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity(owner) failed: %v", err)
	}
	if len(lookupOwner.EntityIds) != 1 || lookupOwner.EntityIds[0] != "doc1" {
		t.Errorf("expected [doc1] for LookupEntity(owner), got %v", lookupOwner.EntityIds)
	}

	// LookupSubject with relation name "owner" should also work
	lookupSubject, err := testServer.PermissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:           &pb.Entity{Type: "document", Id: "doc1"},
		Permission:       "owner",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	})
	if err != nil {
		t.Fatalf("LookupSubject(owner) failed: %v", err)
	}
	if len(lookupSubject.SubjectIds) != 1 || lookupSubject.SubjectIds[0] != "alice" {
		t.Errorf("expected [alice] for LookupSubject(owner), got %v", lookupSubject.SubjectIds)
	}

	// Expand with relation name "owner" should also work
	expandOwner, err := testServer.PermissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "owner",
	})
	if err != nil {
		t.Fatalf("Expand(owner) failed: %v", err)
	}
	if expandOwner.Tree == nil {
		t.Error("expected non-nil expand tree for relation name")
	}

	t.Log("LookupEntity/LookupSubject/Expand correctly accept relation names")
}

// TestScenario_Bugfix_LookupEntityMultiTypeHierarchy tests that LookupEntity
// correctly handles hierarchical rules with multiple target types.
func TestScenario_Bugfix_LookupEntityMultiTypeHierarchy(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity folder {
  relation owner @user

  permission view = owner
}

entity organization {
  relation admin @user

  permission view = admin
}

entity document {
  relation parent @folder @organization

  permission view = parent.view
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// doc1 has folder parent with alice as owner
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "f1"}},
			{Entity: &pb.Entity{Type: "folder", Id: "f1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			// doc2 has organization parent with bob as admin
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "parent", Subject: &pb.Subject{Type: "organization", Id: "org1"}},
			{Entity: &pb.Entity{Type: "organization", Id: "org1"}, Relation: "admin", Subject: &pb.Subject{Type: "user", Id: "bob"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Check: alice has view on doc1 via folder parent
	checkAlice, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check alice doc1 failed: %v", err)
	}
	if checkAlice.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected alice to have view on doc1 via folder parent")
	}

	// Check: bob has view on doc2 via organization parent
	checkBob, err := testServer.PermissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc2"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check bob doc2 failed: %v", err)
	}
	if checkBob.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Error("expected bob to have view on doc2 via organization parent")
	}

	// LookupEntity for alice should find doc1
	lookupAlice, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity alice failed: %v", err)
	}
	aliceIDs := lookupAlice.EntityIds
	sort.Strings(aliceIDs)
	if len(aliceIDs) != 1 || aliceIDs[0] != "doc1" {
		t.Errorf("expected [doc1] for alice, got %v", aliceIDs)
	}

	// LookupEntity for bob should find doc2
	lookupBob, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("LookupEntity bob failed: %v", err)
	}
	bobIDs := lookupBob.EntityIds
	sort.Strings(bobIDs)
	if len(bobIDs) != 1 || bobIDs[0] != "doc2" {
		t.Errorf("expected [doc2] for bob, got %v", bobIDs)
	}

	// LookupSubject for doc2 should find bob
	lookupSubject, err := testServer.PermissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:           &pb.Entity{Type: "document", Id: "doc2"},
		Permission:       "view",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	})
	if err != nil {
		t.Fatalf("LookupSubject doc2 failed: %v", err)
	}
	subjectIDs := lookupSubject.SubjectIds
	sort.Strings(subjectIDs)
	if len(subjectIDs) != 1 || subjectIDs[0] != "bob" {
		t.Errorf("expected [bob] for doc2 view subjects, got %v", subjectIDs)
	}

	t.Log("Multi-type hierarchical LookupEntity/LookupSubject correctly finds all target types")
}

// TestScenario_Bugfix_ContextualTuplePaginationNoDuplicates tests that
// contextual tuples don't cause duplicate results across pages.
func TestScenario_Bugfix_ContextualTuplePaginationNoDuplicates(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
entity user {}

entity document {
  relation viewer @user

  permission view = viewer
}
`
	_, err := testServer.SchemaClient.Write(ctx, &pb.SchemaWriteRequest{Schema: schema})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// Write some DB tuples
	_, err = testServer.DataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	// Contextual tuple adds doc2
	ctxTuples := &pb.Context{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	}

	// Page through with small page size to test pagination
	allIDs := make(map[string]bool)
	var pageToken string
	pageCount := 0

	for {
		resp, err := testServer.PermissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
			EntityType: "document",
			Permission: "view",
			Subject:    &pb.Subject{Type: "user", Id: "alice"},
			Context:    ctxTuples,
			PageSize:   2,
			ContinuousToken: pageToken,
		})
		if err != nil {
			t.Fatalf("LookupEntity page %d failed: %v", pageCount, err)
		}

		for _, id := range resp.EntityIds {
			if allIDs[id] {
				t.Errorf("duplicate entity ID across pages: %s", id)
			}
			allIDs[id] = true
		}

		pageCount++
		if resp.ContinuousToken == "" || pageCount > 10 {
			break
		}
		pageToken = resp.ContinuousToken
	}

	// Should find doc1, doc2 (contextual), doc3, doc5
	expectedCount := 4
	if len(allIDs) != expectedCount {
		t.Errorf("expected %d unique entities, got %d: %v", expectedCount, len(allIDs), allIDs)
	}

	for _, expected := range []string{"doc1", "doc2", "doc3", "doc5"} {
		if !allIDs[expected] {
			t.Errorf("expected %s in results, not found", expected)
		}
	}

	t.Log("Contextual tuple pagination produces no duplicates across pages")
}
