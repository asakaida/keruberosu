package postgres

import (
	"context"
	"testing"
)

func TestSchemaRepository_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: スキーマ作成成功", func(t *testing.T) {
		tenantID := "tenant1"
		schemaDSL := "entity user {}"

		err := repo.Create(ctx, tenantID, schemaDSL)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("異常系: 同じテナントIDで重複作成", func(t *testing.T) {
		tenantID := "tenant2"
		schemaDSL := "entity user {}"

		// 1回目は成功
		err := repo.Create(ctx, tenantID, schemaDSL)
		if err != nil {
			t.Fatalf("Expected no error on first create, got: %v", err)
		}

		// 2回目は失敗（UNIQUE制約違反）
		err = repo.Create(ctx, tenantID, schemaDSL)
		if err == nil {
			t.Fatal("Expected error on duplicate create, got nil")
		}
	})
}

func TestSchemaRepository_GetByTenant(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: スキーマ取得成功", func(t *testing.T) {
		tenantID := "tenant3"
		schemaDSL := "entity document {}"

		// スキーマを作成
		err := repo.Create(ctx, tenantID, schemaDSL)
		if err != nil {
			t.Fatalf("Failed to create schema: %v", err)
		}

		// スキーマを取得
		schema, err := repo.GetByTenant(ctx, tenantID)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if schema.TenantID != tenantID {
			t.Errorf("Expected tenant_id %s, got %s", tenantID, schema.TenantID)
		}
		if schema.DSL != schemaDSL {
			t.Errorf("Expected DSL %s, got %s", schemaDSL, schema.DSL)
		}
		if schema.CreatedAt.IsZero() {
			t.Error("Expected non-zero created_at")
		}
		if schema.UpdatedAt.IsZero() {
			t.Error("Expected non-zero updated_at")
		}
	})

	t.Run("異常系: 存在しないテナントID", func(t *testing.T) {
		tenantID := "nonexistent"

		_, err := repo.GetByTenant(ctx, tenantID)
		if err == nil {
			t.Fatal("Expected error for nonexistent tenant, got nil")
		}
	})
}

func TestSchemaRepository_Update(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: スキーマ更新成功", func(t *testing.T) {
		tenantID := "tenant4"
		originalDSL := "entity user {}"
		updatedDSL := "entity user {} entity document {}"

		// スキーマを作成
		err := repo.Create(ctx, tenantID, originalDSL)
		if err != nil {
			t.Fatalf("Failed to create schema: %v", err)
		}

		// スキーマを更新
		err = repo.Update(ctx, tenantID, updatedDSL)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 更新されたスキーマを取得
		schema, err := repo.GetByTenant(ctx, tenantID)
		if err != nil {
			t.Fatalf("Failed to get schema: %v", err)
		}

		if schema.DSL != updatedDSL {
			t.Errorf("Expected DSL %s, got %s", updatedDSL, schema.DSL)
		}
		if schema.UpdatedAt.Equal(schema.CreatedAt) || schema.UpdatedAt.Before(schema.CreatedAt) {
			t.Error("Expected updated_at to be after created_at")
		}
	})

	t.Run("異常系: 存在しないテナントIDで更新", func(t *testing.T) {
		tenantID := "nonexistent"
		schemaDSL := "entity user {}"

		err := repo.Update(ctx, tenantID, schemaDSL)
		if err == nil {
			t.Fatal("Expected error for nonexistent tenant, got nil")
		}
	})
}

func TestSchemaRepository_Delete(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: スキーマ削除成功", func(t *testing.T) {
		tenantID := "tenant5"
		schemaDSL := "entity user {}"

		// スキーマを作成
		err := repo.Create(ctx, tenantID, schemaDSL)
		if err != nil {
			t.Fatalf("Failed to create schema: %v", err)
		}

		// スキーマを削除
		err = repo.Delete(ctx, tenantID)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 削除されたことを確認
		_, err = repo.GetByTenant(ctx, tenantID)
		if err == nil {
			t.Fatal("Expected error for deleted schema, got nil")
		}
	})

	t.Run("異常系: 存在しないテナントIDで削除", func(t *testing.T) {
		tenantID := "nonexistent"

		err := repo.Delete(ctx, tenantID)
		if err == nil {
			t.Fatal("Expected error for nonexistent tenant, got nil")
		}
	})
}
