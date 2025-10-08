package postgres

import (
	"context"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

func TestRelationRepository_Write(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresRelationRepository(db)
	ctx := context.Background()

	t.Run("正常系: リレーション作成成功", func(t *testing.T) {
		tuple := &entities.RelationTuple{
			EntityType:  "document",
			EntityID:    "doc1",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "alice",
		}

		err := repo.Write(ctx, "tenant1", tuple)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("正常系: subject_relation付きリレーション作成", func(t *testing.T) {
		tuple := &entities.RelationTuple{
			EntityType:      "document",
			EntityID:        "doc2",
			Relation:        "viewer",
			SubjectType:     "organization",
			SubjectID:       "org1",
			SubjectRelation: "member",
		}

		err := repo.Write(ctx, "tenant1", tuple)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("正常系: 重複リレーション（冪等性）", func(t *testing.T) {
		tuple := &entities.RelationTuple{
			EntityType:  "document",
			EntityID:    "doc3",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "bob",
		}

		// 1回目
		err := repo.Write(ctx, "tenant1", tuple)
		if err != nil {
			t.Fatalf("Expected no error on first write, got: %v", err)
		}

		// 2回目（エラーなし、ON CONFLICT DO NOTHING）
		err = repo.Write(ctx, "tenant1", tuple)
		if err != nil {
			t.Fatalf("Expected no error on duplicate write, got: %v", err)
		}
	})

	t.Run("異常系: 無効なリレーション（entity_type が空）", func(t *testing.T) {
		tuple := &entities.RelationTuple{
			EntityType:  "",
			EntityID:    "doc1",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "alice",
		}

		err := repo.Write(ctx, "tenant1", tuple)
		if err == nil {
			t.Fatal("Expected validation error, got nil")
		}
	})
}

func TestRelationRepository_Read(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresRelationRepository(db)
	ctx := context.Background()
	tenantID := "tenant2"

	// テストデータの準備
	tuples := []*entities.RelationTuple{
		{EntityType: "document", EntityID: "doc1", Relation: "owner", SubjectType: "user", SubjectID: "alice"},
		{EntityType: "document", EntityID: "doc1", Relation: "editor", SubjectType: "user", SubjectID: "bob"},
		{EntityType: "document", EntityID: "doc2", Relation: "owner", SubjectType: "user", SubjectID: "alice"},
		{EntityType: "folder", EntityID: "folder1", Relation: "owner", SubjectType: "user", SubjectID: "charlie"},
	}

	for _, tuple := range tuples {
		if err := repo.Write(ctx, tenantID, tuple); err != nil {
			t.Fatalf("Failed to write tuple: %v", err)
		}
	}

	t.Run("正常系: フィルタなしで全件取得", func(t *testing.T) {
		results, err := repo.Read(ctx, tenantID, nil)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(results) != 4 {
			t.Errorf("Expected 4 tuples, got %d", len(results))
		}
	})

	t.Run("正常系: entity_typeでフィルタ", func(t *testing.T) {
		filter := &repositories.RelationFilter{
			EntityType: "document",
		}

		results, err := repo.Read(ctx, tenantID, filter)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 tuples, got %d", len(results))
		}
	})

	t.Run("正常系: entity_idでフィルタ", func(t *testing.T) {
		filter := &repositories.RelationFilter{
			EntityType: "document",
			EntityID:   "doc1",
		}

		results, err := repo.Read(ctx, tenantID, filter)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 tuples, got %d", len(results))
		}
	})

	t.Run("正常系: subject_idでフィルタ", func(t *testing.T) {
		filter := &repositories.RelationFilter{
			SubjectID: "alice",
		}

		results, err := repo.Read(ctx, tenantID, filter)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 tuples, got %d", len(results))
		}
	})

	t.Run("正常系: 複合条件でフィルタ", func(t *testing.T) {
		filter := &repositories.RelationFilter{
			EntityType:  "document",
			EntityID:    "doc1",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "alice",
		}

		results, err := repo.Read(ctx, tenantID, filter)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 tuple, got %d", len(results))
		}

		if results[0].SubjectID != "alice" {
			t.Errorf("Expected subject_id alice, got %s", results[0].SubjectID)
		}
	})

	t.Run("正常系: 該当なしの場合", func(t *testing.T) {
		filter := &repositories.RelationFilter{
			EntityID: "nonexistent",
		}

		results, err := repo.Read(ctx, tenantID, filter)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 tuples, got %d", len(results))
		}
	})
}

func TestRelationRepository_CheckExists(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresRelationRepository(db)
	ctx := context.Background()
	tenantID := "tenant3"

	tuple := &entities.RelationTuple{
		EntityType:  "document",
		EntityID:    "doc1",
		Relation:    "owner",
		SubjectType: "user",
		SubjectID:   "alice",
	}

	// リレーションを作成
	if err := repo.Write(ctx, tenantID, tuple); err != nil {
		t.Fatalf("Failed to write tuple: %v", err)
	}

	t.Run("正常系: 存在するリレーション", func(t *testing.T) {
		exists, err := repo.CheckExists(ctx, tenantID, tuple)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !exists {
			t.Error("Expected tuple to exist")
		}
	})

	t.Run("正常系: 存在しないリレーション", func(t *testing.T) {
		nonExistentTuple := &entities.RelationTuple{
			EntityType:  "document",
			EntityID:    "doc1",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "bob",
		}

		exists, err := repo.CheckExists(ctx, tenantID, nonExistentTuple)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if exists {
			t.Error("Expected tuple to not exist")
		}
	})

	t.Run("異常系: 無効なリレーション", func(t *testing.T) {
		invalidTuple := &entities.RelationTuple{
			EntityType: "",
			EntityID:   "doc1",
		}

		_, err := repo.CheckExists(ctx, tenantID, invalidTuple)
		if err == nil {
			t.Fatal("Expected validation error, got nil")
		}
	})
}

func TestRelationRepository_Delete(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresRelationRepository(db)
	ctx := context.Background()
	tenantID := "tenant4"

	t.Run("正常系: リレーション削除成功", func(t *testing.T) {
		tuple := &entities.RelationTuple{
			EntityType:  "document",
			EntityID:    "doc1",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "alice",
		}

		// 作成
		if err := repo.Write(ctx, tenantID, tuple); err != nil {
			t.Fatalf("Failed to write tuple: %v", err)
		}

		// 削除
		err := repo.Delete(ctx, tenantID, tuple)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 削除されたことを確認
		exists, err := repo.CheckExists(ctx, tenantID, tuple)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		if exists {
			t.Error("Expected tuple to be deleted")
		}
	})

	t.Run("正常系: 存在しないリレーションの削除（エラーなし）", func(t *testing.T) {
		tuple := &entities.RelationTuple{
			EntityType:  "document",
			EntityID:    "nonexistent",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "nobody",
		}

		err := repo.Delete(ctx, tenantID, tuple)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("異常系: 無効なリレーション", func(t *testing.T) {
		invalidTuple := &entities.RelationTuple{
			EntityType: "",
		}

		err := repo.Delete(ctx, tenantID, invalidTuple)
		if err == nil {
			t.Fatal("Expected validation error, got nil")
		}
	})
}

func TestRelationRepository_BatchWrite(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresRelationRepository(db)
	ctx := context.Background()
	tenantID := "tenant5"

	t.Run("正常系: 複数リレーション一括作成", func(t *testing.T) {
		tuples := []*entities.RelationTuple{
			{EntityType: "document", EntityID: "doc1", Relation: "owner", SubjectType: "user", SubjectID: "alice"},
			{EntityType: "document", EntityID: "doc1", Relation: "editor", SubjectType: "user", SubjectID: "bob"},
			{EntityType: "document", EntityID: "doc2", Relation: "owner", SubjectType: "user", SubjectID: "charlie"},
		}

		err := repo.BatchWrite(ctx, tenantID, tuples)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 全て作成されたことを確認
		for _, tuple := range tuples {
			exists, err := repo.CheckExists(ctx, tenantID, tuple)
			if err != nil {
				t.Fatalf("Failed to check existence: %v", err)
			}
			if !exists {
				t.Errorf("Expected tuple %v to exist", tuple)
			}
		}
	})

	t.Run("正常系: 空のスライス", func(t *testing.T) {
		err := repo.BatchWrite(ctx, tenantID, []*entities.RelationTuple{})
		if err != nil {
			t.Fatalf("Expected no error for empty slice, got: %v", err)
		}
	})

	t.Run("異常系: バリデーションエラー", func(t *testing.T) {
		tuples := []*entities.RelationTuple{
			{EntityType: "document", EntityID: "doc1", Relation: "owner", SubjectType: "user", SubjectID: "alice"},
			{EntityType: "", EntityID: "doc2", Relation: "owner", SubjectType: "user", SubjectID: "bob"},
		}

		err := repo.BatchWrite(ctx, tenantID, tuples)
		if err == nil {
			t.Fatal("Expected validation error, got nil")
		}
	})
}

func TestRelationRepository_BatchDelete(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresRelationRepository(db)
	ctx := context.Background()
	tenantID := "tenant6"

	t.Run("正常系: 複数リレーション一括削除", func(t *testing.T) {
		tuples := []*entities.RelationTuple{
			{EntityType: "document", EntityID: "doc1", Relation: "owner", SubjectType: "user", SubjectID: "alice"},
			{EntityType: "document", EntityID: "doc1", Relation: "editor", SubjectType: "user", SubjectID: "bob"},
			{EntityType: "document", EntityID: "doc2", Relation: "owner", SubjectType: "user", SubjectID: "charlie"},
		}

		// 作成
		if err := repo.BatchWrite(ctx, tenantID, tuples); err != nil {
			t.Fatalf("Failed to batch write: %v", err)
		}

		// 削除
		err := repo.BatchDelete(ctx, tenantID, tuples)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 全て削除されたことを確認
		for _, tuple := range tuples {
			exists, err := repo.CheckExists(ctx, tenantID, tuple)
			if err != nil {
				t.Fatalf("Failed to check existence: %v", err)
			}
			if exists {
				t.Errorf("Expected tuple %v to be deleted", tuple)
			}
		}
	})

	t.Run("正常系: 空のスライス", func(t *testing.T) {
		err := repo.BatchDelete(ctx, tenantID, []*entities.RelationTuple{})
		if err != nil {
			t.Fatalf("Expected no error for empty slice, got: %v", err)
		}
	})
}
