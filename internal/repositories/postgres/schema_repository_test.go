package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/asakaida/keruberosu/internal/repositories"
)

func TestSchemaRepository_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: スキーマ作成成功", func(t *testing.T) {
		tenantID := "tenant1"
		schemaDSL := "entity user {}"

		version, err := repo.Create(ctx, tenantID, schemaDSL)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if version == "" {
			t.Fatal("Expected non-empty version, got empty string")
		}
		// ULID should be 26 characters
		if len(version) != 26 {
			t.Errorf("Expected version length 26, got %d", len(version))
		}
	})

	t.Run("正常系: 同じテナントIDで複数バージョン作成可能", func(t *testing.T) {
		tenantID := "tenant2"
		schemaDSL1 := "entity user {}"
		schemaDSL2 := "entity user {} entity document {}"

		// 1回目
		version1, err := repo.Create(ctx, tenantID, schemaDSL1)
		if err != nil {
			t.Fatalf("Expected no error on first create, got: %v", err)
		}

		// 2回目も成功（新しいバージョンとして作成される）
		version2, err := repo.Create(ctx, tenantID, schemaDSL2)
		if err != nil {
			t.Fatalf("Expected no error on second create, got: %v", err)
		}

		// 異なるバージョンIDが生成されること
		if version1 == version2 {
			t.Error("Expected different versions for different creates")
		}
	})
}

func TestSchemaRepository_GetLatestVersion(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: 最新バージョン取得成功", func(t *testing.T) {
		tenantID := "tenant3"
		schemaDSL1 := "entity document {}"
		schemaDSL2 := "entity document {} entity user {}"

		// 2つのバージョンを作成
		version1, err := repo.Create(ctx, tenantID, schemaDSL1)
		if err != nil {
			t.Fatalf("Failed to create schema v1: %v", err)
		}

		version2, err := repo.Create(ctx, tenantID, schemaDSL2)
		if err != nil {
			t.Fatalf("Failed to create schema v2: %v", err)
		}

		// 最新バージョンを取得（v2が返されるはず）
		schema, err := repo.GetLatestVersion(ctx, tenantID)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if schema.TenantID != tenantID {
			t.Errorf("Expected tenant_id %s, got %s", tenantID, schema.TenantID)
		}
		if schema.Version != version2 {
			t.Errorf("Expected version %s, got %s", version2, schema.Version)
		}
		if schema.DSL != schemaDSL2 {
			t.Errorf("Expected DSL %s, got %s", schemaDSL2, schema.DSL)
		}
		if schema.CreatedAt.IsZero() {
			t.Error("Expected non-zero created_at")
		}
		if schema.UpdatedAt.IsZero() {
			t.Error("Expected non-zero updated_at")
		}

		// 古いバージョンも存在することを確認
		oldSchema, err := repo.GetByVersion(ctx, tenantID, version1)
		if err != nil {
			t.Fatalf("Expected to get old version, got error: %v", err)
		}
		if oldSchema.DSL != schemaDSL1 {
			t.Errorf("Expected old DSL %s, got %s", schemaDSL1, oldSchema.DSL)
		}
	})

	t.Run("異常系: 存在しないテナントID (ErrNotFoundを返す)", func(t *testing.T) {
		tenantID := "nonexistent"

		schema, err := repo.GetLatestVersion(ctx, tenantID)
		if err == nil {
			t.Fatal("Expected error for nonexistent tenant, got nil")
		}
		if !errors.Is(err, repositories.ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got: %v", err)
		}
		if schema != nil {
			t.Errorf("Expected nil schema when error occurs, got: %+v", schema)
		}
	})
}

func TestSchemaRepository_GetByVersion(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: 特定バージョン取得成功", func(t *testing.T) {
		tenantID := "tenant3b"
		schemaDSL := "entity document {}"

		// スキーマを作成
		version, err := repo.Create(ctx, tenantID, schemaDSL)
		if err != nil {
			t.Fatalf("Failed to create schema: %v", err)
		}

		// 特定バージョンを取得
		schema, err := repo.GetByVersion(ctx, tenantID, version)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if schema.TenantID != tenantID {
			t.Errorf("Expected tenant_id %s, got %s", tenantID, schema.TenantID)
		}
		if schema.Version != version {
			t.Errorf("Expected version %s, got %s", version, schema.Version)
		}
		if schema.DSL != schemaDSL {
			t.Errorf("Expected DSL %s, got %s", schemaDSL, schema.DSL)
		}
	})

	t.Run("異常系: 存在しないバージョン (ErrNotFoundを返す)", func(t *testing.T) {
		tenantID := "tenant3c"
		invalidVersion := "01ARZ3NDEKTSV4RRFFQ69G5FAV"

		// テナントは作成するが、存在しないバージョンIDで検索
		_, err := repo.Create(ctx, tenantID, "entity user {}")
		if err != nil {
			t.Fatalf("Failed to create schema: %v", err)
		}

		schema, err := repo.GetByVersion(ctx, tenantID, invalidVersion)
		if err == nil {
			t.Fatal("Expected error for nonexistent version, got nil")
		}
		if !errors.Is(err, repositories.ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got: %v", err)
		}
		if schema != nil {
			t.Errorf("Expected nil schema when error occurs, got: %+v", schema)
		}
	})
}

func TestSchemaRepository_GetByTenant(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: GetByTenantはGetLatestVersionと同じ（後方互換性）", func(t *testing.T) {
		tenantID := "tenant3d"
		schemaDSL := "entity document {}"

		// スキーマを作成
		_, err := repo.Create(ctx, tenantID, schemaDSL)
		if err != nil {
			t.Fatalf("Failed to create schema: %v", err)
		}

		// GetByTenantで取得
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
	})
}

func TestSchemaRepository_Delete(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewPostgresSchemaRepository(db)
	ctx := context.Background()

	t.Run("正常系: テナントの全バージョン削除成功", func(t *testing.T) {
		tenantID := "tenant5"
		schemaDSL1 := "entity user {}"
		schemaDSL2 := "entity user {} entity document {}"

		// 2つのバージョンを作成
		version1, err := repo.Create(ctx, tenantID, schemaDSL1)
		if err != nil {
			t.Fatalf("Failed to create schema v1: %v", err)
		}

		_, err = repo.Create(ctx, tenantID, schemaDSL2)
		if err != nil {
			t.Fatalf("Failed to create schema v2: %v", err)
		}

		// テナントのスキーマを削除（全バージョンが削除される）
		err = repo.Delete(ctx, tenantID)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 最新バージョンが削除されたことを確認
		_, err = repo.GetLatestVersion(ctx, tenantID)
		if err == nil {
			t.Fatal("Expected error for deleted schema, got nil")
		}

		// 古いバージョンも削除されたことを確認
		_, err = repo.GetByVersion(ctx, tenantID, version1)
		if err == nil {
			t.Fatal("Expected error for deleted old version, got nil")
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
