package e2e

import (
	"context"
	"sort"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
)

// TestScenario_Lookup_NestedHierarchy tests lookup across a three-level hierarchy:
// organization -> folder -> document, where admin on org grants view on documents
// via transitive parent.view permissions.
func TestScenario_Lookup_NestedHierarchy(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Step 1: Define schema with three-level hierarchy
	schema := `
entity user {}

entity organization {
  relation admin @user

  permission view = admin
}

entity folder {
  relation owner @user
  relation parent @organization

  permission view = owner or parent.view
}

entity document {
  relation owner @user
  relation parent @folder

  permission view = owner or parent.view
}
`

	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// Step 2: Write relation tuples
	// alice is admin of org1
	// org1 is parent of folder1
	// folder1 is parent of doc1
	// bob owns doc2
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "organization", Id: "org1"}, Relation: "admin", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "folder", Id: "folder1"}, Relation: "parent", Subject: &pb.Subject{Type: "organization", Id: "org1"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "folder1"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "bob"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteRelations failed: %v", err)
	}

	// Step 3: Verify Check - alice can view doc1 via org1 -> folder1 -> doc1
	checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected alice to be ALLOWED to view doc1, got %v", checkResp.Can)
	}

	// Step 4: LookupEntity - find documents alice can view
	lookupEntityResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}

	// alice should be able to view doc1 (via org1 -> folder1 -> doc1)
	foundDoc1 := false
	for _, id := range lookupEntityResp.EntityIds {
		if id == "doc1" {
			foundDoc1 = true
		}
	}
	if !foundDoc1 {
		t.Errorf("expected alice to see doc1 via nested hierarchy, got entities: %v", lookupEntityResp.EntityIds)
	}
	t.Logf("LookupEntity found %d documents for alice: %v", len(lookupEntityResp.EntityIds), lookupEntityResp.EntityIds)

	// Step 5: LookupSubject - find users who can view doc1
	lookupSubjectResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:     &pb.Entity{Type: "document", Id: "doc1"},
		Permission: "view",
		SubjectReference: &pb.SubjectReference{
			Type: "user",
		},
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}

	foundAlice := false
	for _, id := range lookupSubjectResp.SubjectIds {
		if id == "alice" {
			foundAlice = true
		}
	}
	if !foundAlice {
		t.Errorf("expected alice in subjects who can view doc1, got: %v", lookupSubjectResp.SubjectIds)
	}
	t.Logf("LookupSubject found %d users for doc1.view: %v", len(lookupSubjectResp.SubjectIds), lookupSubjectResp.SubjectIds)
}

// TestScenario_Lookup_SubjectRelation tests lookup with subject relation (computed userset).
// team-a#member is contributor of repo1 → alice (member of team-a) can push repo1.
func TestScenario_Lookup_SubjectRelation(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Step 1: Define schema with subject relation
	schema := `
entity user {}

entity team {
  relation member @user
}

entity repository {
  relation contributor @user @team#member

  permission push = contributor
}
`

	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// Step 2: Write tuples
	// alice is member of team-a
	// team-a#member is contributor of repo1
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "team", Id: "team-a"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "repository", Id: "repo1"}, Relation: "contributor", Subject: &pb.Subject{Type: "team", Id: "team-a", Relation: "member"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteRelations failed: %v", err)
	}

	// Step 3: Verify Check - alice can push repo1
	checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "repository", Id: "repo1"},
		Permission: "push",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected alice to be ALLOWED to push repo1, got %v", checkResp.Can)
	}

	// Step 4: LookupEntity - find repos alice can push
	lookupEntityResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "repository",
		Permission: "push",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}

	foundRepo1 := false
	for _, id := range lookupEntityResp.EntityIds {
		if id == "repo1" {
			foundRepo1 = true
		}
	}
	if !foundRepo1 {
		t.Errorf("expected repo1 in lookup results, got: %v", lookupEntityResp.EntityIds)
	}
	t.Logf("LookupEntity found %d repositories for alice: %v", len(lookupEntityResp.EntityIds), lookupEntityResp.EntityIds)

	// Step 5: LookupSubject - find users who can push repo1
	lookupSubjectResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:     &pb.Entity{Type: "repository", Id: "repo1"},
		Permission: "push",
		SubjectReference: &pb.SubjectReference{
			Type: "user",
		},
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}

	foundAlice := false
	for _, id := range lookupSubjectResp.SubjectIds {
		if id == "alice" {
			foundAlice = true
		}
	}
	if !foundAlice {
		t.Errorf("expected alice in subjects who can push repo1, got: %v", lookupSubjectResp.SubjectIds)
	}
	t.Logf("LookupSubject found %d users for repo1.push: %v", len(lookupSubjectResp.SubjectIds), lookupSubjectResp.SubjectIds)
}

// TestScenario_Lookup_Pagination tests cursor-based pagination for LookupEntity.
// Creates 5 documents owned by alice, iterates pages of size 2, and verifies
// all 5 are found with no duplicates.
func TestScenario_Lookup_Pagination(t *testing.T) {
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaClient := testServer.SchemaClient
	dataClient := testServer.DataClient
	permissionClient := testServer.PermissionClient

	// Step 1: Define schema
	schema := `
entity user {}

entity document {
  relation owner @user

  permission view = owner
}
`

	_, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}

	// Step 2: Create 5 documents owned by alice
	tuples := make([]*pb.Tuple, 5)
	expectedIDs := make(map[string]bool)
	for i := 0; i < 5; i++ {
		docID := "doc" + string(rune('1'+i))
		tuples[i] = &pb.Tuple{
			Entity:   &pb.Entity{Type: "document", Id: docID},
			Relation: "owner",
			Subject:  &pb.Subject{Type: "user", Id: "alice"},
		}
		expectedIDs[docID] = true
	}

	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: tuples,
	})
	if err != nil {
		t.Fatalf("WriteRelations failed: %v", err)
	}

	// Step 3: Iterate pages of size 2
	var allEntityIDs []string
	continuousToken := ""
	pageCount := 0
	maxPages := 10 // safety limit

	for pageCount < maxPages {
		pageCount++

		lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
			EntityType:      "document",
			Permission:      "view",
			Subject:         &pb.Subject{Type: "user", Id: "alice"},
			PageSize:        2,
			ContinuousToken: continuousToken,
		})
		if err != nil {
			t.Fatalf("LookupEntity page %d failed: %v", pageCount, err)
		}

		allEntityIDs = append(allEntityIDs, lookupResp.EntityIds...)
		t.Logf("Page %d: %v (token=%q)", pageCount, lookupResp.EntityIds, lookupResp.ContinuousToken)

		if lookupResp.ContinuousToken == "" {
			break
		}
		continuousToken = lookupResp.ContinuousToken
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, id := range allEntityIDs {
		if seen[id] {
			t.Errorf("duplicate entity ID found: %s", id)
		}
		seen[id] = true
	}

	// Verify all 5 found
	if len(seen) != 5 {
		sort.Strings(allEntityIDs)
		t.Errorf("expected 5 unique entities, got %d: %v", len(seen), allEntityIDs)
	}

	t.Logf("Pagination completed in %d pages, found %d entities", pageCount, len(seen))
}
