package postgres

import (
	"context"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
)

func TestAttributeRepository_Write(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresAttributeRepository(db)
	ctx := context.Background()
	tenantID := "tenant1"

	t.Run("正常系: 文字列属性の作成", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc1",
			Name:       "title",
			Value:      "My Document",
		}

		err := repo.Write(ctx, tenantID, attr)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("正常系: 真偽値属性の作成", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc1",
			Name:       "public",
			Value:      true,
		}

		err := repo.Write(ctx, tenantID, attr)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("正常系: 数値属性の作成", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc1",
			Name:       "version",
			Value:      42,
		}

		err := repo.Write(ctx, tenantID, attr)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("正常系: 配列属性の作成", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc1",
			Name:       "tags",
			Value:      []interface{}{"important", "urgent"},
		}

		err := repo.Write(ctx, tenantID, attr)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("正常系: 既存属性の更新（Upsert）", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc2",
			Name:       "status",
			Value:      "draft",
		}

		// 1回目の作成
		err := repo.Write(ctx, tenantID, attr)
		if err != nil {
			t.Fatalf("Expected no error on first write, got: %v", err)
		}

		// 2回目の更新
		attr.Value = "published"
		err = repo.Write(ctx, tenantID, attr)
		if err != nil {
			t.Fatalf("Expected no error on update, got: %v", err)
		}

		// 更新されたことを確認
		value, err := repo.GetValue(ctx, tenantID, "document", "doc2", "status")
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}

		if value != "published" {
			t.Errorf("Expected value 'published', got %v", value)
		}
	})

	t.Run("異常系: 無効な属性（entity_typeが空）", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "",
			EntityID:   "doc1",
			Name:       "title",
			Value:      "Test",
		}

		err := repo.Write(ctx, tenantID, attr)
		if err == nil {
			t.Fatal("Expected validation error, got nil")
		}
	})

	t.Run("異常系: 無効な属性（valueがnil）", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc1",
			Name:       "title",
			Value:      nil,
		}

		err := repo.Write(ctx, tenantID, attr)
		if err == nil {
			t.Fatal("Expected validation error, got nil")
		}
	})
}

func TestAttributeRepository_Read(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresAttributeRepository(db)
	ctx := context.Background()
	tenantID := "tenant2"

	// テストデータの準備
	attributes := []*entities.Attribute{
		{EntityType: "document", EntityID: "doc1", Name: "title", Value: "My Document"},
		{EntityType: "document", EntityID: "doc1", Name: "public", Value: true},
		{EntityType: "document", EntityID: "doc1", Name: "version", Value: float64(1)}, // JSONでは数値はfloat64になる
		{EntityType: "document", EntityID: "doc2", Name: "title", Value: "Another Doc"},
	}

	for _, attr := range attributes {
		if err := repo.Write(ctx, tenantID, attr); err != nil {
			t.Fatalf("Failed to write attribute: %v", err)
		}
	}

	t.Run("正常系: エンティティの全属性を取得", func(t *testing.T) {
		attrs, err := repo.Read(ctx, tenantID, "document", "doc1")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(attrs) != 3 {
			t.Errorf("Expected 3 attributes, got %d", len(attrs))
		}

		if attrs["title"] != "My Document" {
			t.Errorf("Expected title 'My Document', got %v", attrs["title"])
		}
		if attrs["public"] != true {
			t.Errorf("Expected public true, got %v", attrs["public"])
		}
		if attrs["version"] != float64(1) {
			t.Errorf("Expected version 1, got %v", attrs["version"])
		}
	})

	t.Run("正常系: 属性がないエンティティ", func(t *testing.T) {
		attrs, err := repo.Read(ctx, tenantID, "document", "nonexistent")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(attrs) != 0 {
			t.Errorf("Expected 0 attributes, got %d", len(attrs))
		}
	})
}

func TestAttributeRepository_GetValue(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresAttributeRepository(db)
	ctx := context.Background()
	tenantID := "tenant3"

	// テストデータの準備
	attr := &entities.Attribute{
		EntityType: "document",
		EntityID:   "doc1",
		Name:       "title",
		Value:      "Test Document",
	}
	if err := repo.Write(ctx, tenantID, attr); err != nil {
		t.Fatalf("Failed to write attribute: %v", err)
	}

	t.Run("正常系: 特定の属性値を取得", func(t *testing.T) {
		value, err := repo.GetValue(ctx, tenantID, "document", "doc1", "title")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if value != "Test Document" {
			t.Errorf("Expected 'Test Document', got %v", value)
		}
	})

	t.Run("異常系: 存在しない属性", func(t *testing.T) {
		_, err := repo.GetValue(ctx, tenantID, "document", "doc1", "nonexistent")
		if err == nil {
			t.Fatal("Expected error for nonexistent attribute, got nil")
		}
	})

	t.Run("異常系: 存在しないエンティティ", func(t *testing.T) {
		_, err := repo.GetValue(ctx, tenantID, "document", "nonexistent", "title")
		if err == nil {
			t.Fatal("Expected error for nonexistent entity, got nil")
		}
	})
}

func TestAttributeRepository_Delete(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresAttributeRepository(db)
	ctx := context.Background()
	tenantID := "tenant4"

	t.Run("正常系: 属性削除成功", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc1",
			Name:       "title",
			Value:      "Test",
		}

		// 作成
		if err := repo.Write(ctx, tenantID, attr); err != nil {
			t.Fatalf("Failed to write attribute: %v", err)
		}

		// 削除
		err := repo.Delete(ctx, tenantID, "document", "doc1", "title")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 削除されたことを確認
		_, err = repo.GetValue(ctx, tenantID, "document", "doc1", "title")
		if err == nil {
			t.Fatal("Expected error for deleted attribute, got nil")
		}
	})

	t.Run("正常系: 存在しない属性の削除（エラーなし）", func(t *testing.T) {
		err := repo.Delete(ctx, tenantID, "document", "nonexistent", "title")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("正常系: 複数属性のうち1つだけ削除", func(t *testing.T) {
		// 2つの属性を作成
		attrs := []*entities.Attribute{
			{EntityType: "document", EntityID: "doc2", Name: "title", Value: "Test"},
			{EntityType: "document", EntityID: "doc2", Name: "public", Value: true},
		}

		for _, attr := range attrs {
			if err := repo.Write(ctx, tenantID, attr); err != nil {
				t.Fatalf("Failed to write attribute: %v", err)
			}
		}

		// 1つ削除
		err := repo.Delete(ctx, tenantID, "document", "doc2", "title")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// title は削除されている
		_, err = repo.GetValue(ctx, tenantID, "document", "doc2", "title")
		if err == nil {
			t.Fatal("Expected error for deleted attribute, got nil")
		}

		// public は残っている
		value, err := repo.GetValue(ctx, tenantID, "document", "doc2", "public")
		if err != nil {
			t.Fatalf("Expected no error for remaining attribute, got: %v", err)
		}
		if value != true {
			t.Errorf("Expected public true, got %v", value)
		}
	})
}

func TestAttributeRepository_ComplexTypes(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresAttributeRepository(db)
	ctx := context.Background()
	tenantID := "tenant5"

	t.Run("正常系: マップ型の属性", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc1",
			Name:       "metadata",
			Value: map[string]interface{}{
				"author": "Alice",
				"year":   float64(2025),
			},
		}

		// 作成
		err := repo.Write(ctx, tenantID, attr)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 取得
		value, err := repo.GetValue(ctx, tenantID, "document", "doc1", "metadata")
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}

		metadata, ok := value.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map[string]interface{}, got %T", value)
		}

		if metadata["author"] != "Alice" {
			t.Errorf("Expected author 'Alice', got %v", metadata["author"])
		}
		if metadata["year"] != float64(2025) {
			t.Errorf("Expected year 2025, got %v", metadata["year"])
		}
	})

	t.Run("正常系: ネストした配列", func(t *testing.T) {
		attr := &entities.Attribute{
			EntityType: "document",
			EntityID:   "doc2",
			Name:       "matrix",
			Value: []interface{}{
				[]interface{}{1, 2, 3},
				[]interface{}{4, 5, 6},
			},
		}

		// 作成
		err := repo.Write(ctx, tenantID, attr)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 取得
		value, err := repo.GetValue(ctx, tenantID, "document", "doc2", "matrix")
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}

		matrix, ok := value.([]interface{})
		if !ok {
			t.Fatalf("Expected []interface{}, got %T", value)
		}

		if len(matrix) != 2 {
			t.Errorf("Expected 2 rows, got %d", len(matrix))
		}
	})
}
