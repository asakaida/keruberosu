package postgres

import (
	"context"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
)

// TestClosure_DescendantPropagation tests that adding A→B after D→A
// correctly creates closure entry D→B via descendant propagation.
func TestClosure_DescendantPropagation(t *testing.T) {
	cluster := SetupTestDB(t)
	defer CleanupTestDB(t, cluster)

	repo := NewPostgresRelationRepository(cluster, nil)
	ctx := context.Background()
	tenantID := "closure-test-descendant"

	// Add D→A first (doc1 parent→ folderA)
	if err := repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc1",
		Relation: "parent", SubjectType: "folder", SubjectID: "folderA",
	}); err != nil {
		t.Fatalf("Failed to write D→A: %v", err)
	}

	// Then add A→B (folderA parent→ root)
	if err := repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "folder", EntityID: "folderA",
		Relation: "parent", SubjectType: "folder", SubjectID: "root",
	}); err != nil {
		t.Fatalf("Failed to write A→B: %v", err)
	}

	pgRepo := repo.(*PostgresRelationRepository)
	ancestors, err := pgRepo.LookupAncestors(ctx, tenantID, "document", "doc1", 0)
	if err != nil {
		t.Fatalf("Failed to lookup ancestors: %v", err)
	}

	hasFolderA := false
	hasRoot := false
	for _, a := range ancestors {
		if a.AncestorType == "folder" && a.AncestorID == "folderA" {
			hasFolderA = true
		}
		if a.AncestorType == "folder" && a.AncestorID == "root" {
			hasRoot = true
		}
	}

	if !hasFolderA {
		t.Error("doc1 should have folderA as ancestor")
	}
	if !hasRoot {
		t.Error("doc1 should have root as ancestor via folderA (descendant propagation)")
	}
}

// TestClosure_DeleteMultiPath tests that deleting one path in a DAG
// preserves closure entries reachable via alternate paths.
func TestClosure_DeleteMultiPath(t *testing.T) {
	cluster := SetupTestDB(t)
	defer CleanupTestDB(t, cluster)

	repo := NewPostgresRelationRepository(cluster, nil)
	ctx := context.Background()
	tenantID := "closure-test-multipath"

	// Build DAG: doc1 → folderA → root, doc1 → folderB → root
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "folder", EntityID: "folderA",
		Relation: "parent", SubjectType: "folder", SubjectID: "root",
	})
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "folder", EntityID: "folderB",
		Relation: "parent", SubjectType: "folder", SubjectID: "root",
	})
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc1",
		Relation: "parent", SubjectType: "folder", SubjectID: "folderA",
	})
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc1",
		Relation: "parent", SubjectType: "folder", SubjectID: "folderB",
	})

	// Delete one path: doc1 → folderA
	if err := repo.Delete(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc1",
		Relation: "parent", SubjectType: "folder", SubjectID: "folderA",
	}); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	pgRepo := repo.(*PostgresRelationRepository)
	ancestors, _ := pgRepo.LookupAncestors(ctx, tenantID, "document", "doc1", 0)

	hasRoot := false
	hasFolderB := false
	hasFolderA := false
	for _, a := range ancestors {
		if a.AncestorType == "folder" && a.AncestorID == "root" {
			hasRoot = true
		}
		if a.AncestorType == "folder" && a.AncestorID == "folderB" {
			hasFolderB = true
		}
		if a.AncestorType == "folder" && a.AncestorID == "folderA" {
			hasFolderA = true
		}
	}

	if hasFolderA {
		t.Error("doc1 should NOT have folderA as ancestor after deletion")
	}
	if !hasFolderB {
		t.Error("doc1 should still have folderB as ancestor")
	}
	if !hasRoot {
		t.Error("doc1 should still have root as ancestor via folderB")
	}
}

// TestClosure_DeleteDescendantCleanup tests that deleting a mid-hierarchy
// relation cleans up closure entries for descendants.
func TestClosure_DeleteDescendantCleanup(t *testing.T) {
	cluster := SetupTestDB(t)
	defer CleanupTestDB(t, cluster)

	repo := NewPostgresRelationRepository(cluster, nil)
	ctx := context.Background()
	tenantID := "closure-test-desc-cleanup"

	// Build chain: doc1 → folderA → root
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "folder", EntityID: "folderA",
		Relation: "parent", SubjectType: "folder", SubjectID: "root",
	})
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc1",
		Relation: "parent", SubjectType: "folder", SubjectID: "folderA",
	})

	// Delete mid-hierarchy: folderA → root
	if err := repo.Delete(ctx, tenantID, &entities.RelationTuple{
		EntityType: "folder", EntityID: "folderA",
		Relation: "parent", SubjectType: "folder", SubjectID: "root",
	}); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	pgRepo := repo.(*PostgresRelationRepository)
	ancestors, _ := pgRepo.LookupAncestors(ctx, tenantID, "document", "doc1", 0)

	hasRoot := false
	for _, a := range ancestors {
		if a.AncestorType == "folder" && a.AncestorID == "root" {
			hasRoot = true
		}
	}

	if hasRoot {
		t.Error("doc1 should NOT have root as ancestor after folderA→root was deleted")
	}
}

// TestClosure_RebuildOrderIndependence tests that RebuildClosure produces
// correct results regardless of relation insertion order.
func TestClosure_RebuildOrderIndependence(t *testing.T) {
	cluster := SetupTestDB(t)
	defer CleanupTestDB(t, cluster)

	repo := NewPostgresRelationRepository(cluster, nil)
	ctx := context.Background()
	tenantID := "closure-test-rebuild"

	// Add relations in "wrong" order (leaf first, then parent)
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc1",
		Relation: "parent", SubjectType: "folder", SubjectID: "folderA",
	})
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "folder", EntityID: "folderA",
		Relation: "parent", SubjectType: "folder", SubjectID: "root",
	})

	// Rebuild should fix any order-dependent issues
	if err := repo.RebuildClosure(ctx, tenantID); err != nil {
		t.Fatalf("RebuildClosure failed: %v", err)
	}

	pgRepo := repo.(*PostgresRelationRepository)
	ancestors, _ := pgRepo.LookupAncestors(ctx, tenantID, "document", "doc1", 0)

	hasFolderA := false
	hasRoot := false
	for _, a := range ancestors {
		if a.AncestorType == "folder" && a.AncestorID == "folderA" {
			hasFolderA = true
		}
		if a.AncestorType == "folder" && a.AncestorID == "root" {
			hasRoot = true
		}
	}

	if !hasFolderA {
		t.Error("doc1 → folderA missing after rebuild")
	}
	if !hasRoot {
		t.Error("doc1 → root missing after rebuild (should be order-independent)")
	}
}

// TestClosure_DeepChain tests closure correctness for a 4-level chain.
func TestClosure_DeepChain(t *testing.T) {
	cluster := SetupTestDB(t)
	defer CleanupTestDB(t, cluster)

	repo := NewPostgresRelationRepository(cluster, nil)
	ctx := context.Background()
	tenantID := "closure-test-deep"

	// Build chain: doc → sub → mid → top (root-first order)
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "folder", EntityID: "mid",
		Relation: "parent", SubjectType: "folder", SubjectID: "top",
	})
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "folder", EntityID: "sub",
		Relation: "parent", SubjectType: "folder", SubjectID: "mid",
	})
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc",
		Relation: "parent", SubjectType: "folder", SubjectID: "sub",
	})

	pgRepo := repo.(*PostgresRelationRepository)
	ancestors, _ := pgRepo.LookupAncestors(ctx, tenantID, "document", "doc", 0)

	expected := map[string]int{"sub": 1, "mid": 2, "top": 3}
	found := map[string]int{}
	for _, a := range ancestors {
		found[a.AncestorID] = a.Depth
	}

	for name, expectedDepth := range expected {
		if depth, ok := found[name]; !ok {
			t.Errorf("doc should have %s as ancestor", name)
		} else if depth != expectedDepth {
			t.Errorf("doc → %s: expected depth %d, got %d", name, expectedDepth, depth)
		}
	}
}

// TestClosure_ExcludedRelations tests that excluded relations are not tracked.
func TestClosure_ExcludedRelations(t *testing.T) {
	cluster := SetupTestDB(t)
	defer CleanupTestDB(t, cluster)

	excluded := map[string]bool{"viewer": true}
	repo := NewPostgresRelationRepository(cluster, excluded)
	ctx := context.Background()
	tenantID := "closure-test-excluded"

	// "parent" relation should be tracked
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc1",
		Relation: "parent", SubjectType: "folder", SubjectID: "folder1",
	})
	// "viewer" relation should NOT be tracked
	repo.Write(ctx, tenantID, &entities.RelationTuple{
		EntityType: "document", EntityID: "doc1",
		Relation: "viewer", SubjectType: "user", SubjectID: "alice",
	})

	pgRepo := repo.(*PostgresRelationRepository)
	ancestors, _ := pgRepo.LookupAncestors(ctx, tenantID, "document", "doc1", 0)

	hasFolder := false
	hasAlice := false
	for _, a := range ancestors {
		if a.AncestorID == "folder1" {
			hasFolder = true
		}
		if a.AncestorID == "alice" {
			hasAlice = true
		}
	}

	if !hasFolder {
		t.Error("doc1 should have folder1 as ancestor (parent relation is tracked)")
	}
	if hasAlice {
		t.Error("doc1 should NOT have alice as ancestor (viewer relation is excluded)")
	}
}
