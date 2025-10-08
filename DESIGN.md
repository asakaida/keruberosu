# Keruberosu - 設計書 (DESIGN.md)

## 概要

Keruberosu は、Permify ライクな ReBAC/ABAC 認可マイクロサービスです。
本ドキュメントでは、Phase 1 の実装設計を定義します。

### Phase 1 スコープ定義

Phase 1 スコープ: キャッシュ機構を除く完全な実装

#### Phase 1 に含まれる機能（完全実装）

1. 完全な ReBAC 実装

   - 関係性の定義と管理（リレーション）
   - 階層的な関係（parent.edit, org.member）
   - OR/AND/NOT 論理演算
   - 全ての Check API（権限チェック）
   - Expand API（権限ツリー展開）
   - LookupEntity API（アクセス可能エンティティ検索）
   - LookupSubject API（アクセス可能サブジェクト検索）
   - SubjectPermission API（サブジェクトの権限一覧）

2. 完全な ABAC 実装（全演算子サポート）

   - CEL（Common Expression Language）統合
   - 全ての比較演算子: `==`, `!=`, `>`, `>=`, `<`, `<=`
   - コレクション演算子: `in`
   - 論理演算子: `&&`, `||`, `!`
   - 属性（Attribute）の保存と取得
   - コンテキスト（Context）を使った動的評価
   - ルール評価エンジン

3. スキーマ管理（完全実装）

   - DSL パーサー（Lexer → Parser → AST → Validator）
   - スキーマの CRUD 操作
   - スキーマ検証ロジック
   - DSL ↔ Protocol Buffers 変換

4. データ管理 API（完全実装）

   - RelationTuple CRUD（Write, Read, Delete）
   - Attribute CRUD（Write, Read, Delete）
   - Batch 操作（WriteRelations, DeleteRelations）

5. Permify 互換 gRPC API（完全実装）
   - 全ての Protocol Buffers メッセージ定義
   - 全ての gRPC サービス実装
   - エラーハンドリング
   - メタデータ（snap_token, depth）

#### Phase 1 から除外される機能

キャッシュ機構のみ:

- L1 キャッシュ（自前実装の LRU）
- L2 キャッシュ（Redis）
- キャッシュ無効化機構
- キャッシュウォームアップ

その他は全て実装します。

### アーキテクチャ方針：単一サービスアプローチ

Keruberosu は **単一の gRPC サービス（AuthorizationService）** として実装されます。

**理由**：

1. **業界標準に準拠**: Google Zanzibar、Permify、Auth0 FGA、Ory Keto など、全ての主要な認可システムが単一サービスアプローチを採用
2. **ドメインの不可分性**: 認可は Schema（定義）、Relations（データ）、Authorization（判定）が密接に連携する 1 つのドメイン
3. **クライアント利便性**: 1 つのサービスに接続するだけで全操作が可能
4. **運用の単純化**: デプロイ、モニタリング、トラブルシューティングが容易
5. **Permify 互換性**: Permify の API 設計を完全に踏襲

**実装方針**：

- `internal/handlers/authorization_handler.go` が全ての API を提供
  - Schema 管理: WriteSchema, ReadSchema
  - Data 管理: WriteRelations, DeleteRelations, WriteAttributes
  - Authorization: Check, Expand, LookupEntity, LookupSubject, SubjectPermission
- 内部的には責務ごとにサービス層を分離（SchemaService, Checker, Expander, Lookup）
- gRPC レベルでは単一の AuthorizationService として公開

---

## アーキテクチャ設計

### 1. プロジェクト構造

```text
keruberosu/
├── cmd/
│   ├── server/
│   │   └── main.go                    # gRPC サーバーエントリーポイント
│   └── migrate/
│       └── main.go                    # DB マイグレーションコマンド
├── internal/
│   ├── entities/                      # ドメインエンティティ
│   │   ├── schema.go                 # Schema（スキーマ全体）
│   │   ├── entity.go                 # Entity（エンティティ定義）
│   │   ├── relation.go               # Relation（リレーション定義）
│   │   ├── attribute_schema.go       # AttributeSchema（属性型定義）
│   │   ├── permission.go             # Permission（権限定義）
│   │   ├── rule.go                   # PermissionRule + 各ルール実装
│   │   ├── relation_tuple.go         # RelationTuple（実際のリレーションデータ）
│   │   └── attribute.go              # Attribute（実際の属性データ）
│   ├── handlers/                      # gRPC ハンドラー
│   │   └── authorization_handler.go  # 統合 Authorization Handler（全API提供）
│   ├── services/                      # ビジネスロジック
│   │   ├── parser/                   # DSL パーサー
│   │   │   ├── lexer.go             # 字句解析
│   │   │   ├── parser.go            # 構文解析
│   │   │   ├── ast.go               # AST 定義
│   │   │   ├── validator.go         # スキーマ検証
│   │   │   ├── converter.go         # AST ↔ entities.Schema 変換
│   │   │   └── generator.go         # DSL 文字列生成（AST → DSL）
│   │   ├── schema_service.go        # スキーマ管理サービス
│   │   └── authorization/            # 認可処理
│   │       ├── evaluator.go         # ルール評価（ReBAC + ABAC）
│   │       ├── checker.go           # Check 処理
│   │       ├── expander.go          # Expand 処理
│   │       ├── lookup.go            # Lookup 処理
│   │       └── cel.go               # CEL 評価エンジン
│   ├── repositories/                  # データアクセス
│   │   ├── schema_repository.go     # スキーマリポジトリ インターフェース定義
│   │   ├── relation_repository.go   # リレーションリポジトリ インターフェース定義
│   │   ├── attribute_repository.go  # アトリビュートリポジトリ インターフェース定義
│   │   └── postgres/                 # PostgreSQL 実装
│   │       ├── schema_repository.go     # スキーマリポジトリ実装
│   │       ├── relation_repository.go   # リレーションリポジトリ実装
│   │       └── attribute_repository.go  # アトリビュートリポジトリ実装
│   └── infrastructure/                # インフラ層
│       ├── database/
│       │   ├── postgres.go          # PostgreSQL 接続
│       │   └── migrations/
│       │       └── postgres/         # PostgreSQL 用マイグレーション
│       │           ├── 000001_create_schemas_table.up.sql
│       │           ├── 000001_create_schemas_table.down.sql
│       │           ├── 000002_create_relations_table.up.sql
│       │           ├── 000002_create_relations_table.down.sql
│       │           ├── 000003_create_attributes_table.up.sql
│       │           └── 000003_create_attributes_table.down.sql
│       └── config/
│           └── config.go            # 設定管理
├── proto/
│   └── keruberosu/
│       └── v1/
│           ├── common.proto          # 共通メッセージ定義
│           ├── authorization.proto   # AuthorizationService 定義
│           └── audit.proto           # AuditService 定義
├── scripts/
│   └── generate-proto.sh             # Protocol Buffers コード生成スクリプト
├── docker-compose.yml                 # PostgreSQL 環境（dev/test）
├── go.mod
├── go.sum
├── PRD.md                            # 要求仕様書
├── DESIGN.md                         # 本ドキュメント
└── DEVELOPMENT.md                    # 開発進捗管理
```

### 設計の考え方

#### entities 層の設計

entities 層は 1 ファイル 1 構造体の原則に従い、責務を明確に分離しています。

**スキーマ定義系**（DSL から生成される内部表現）：

- `schema.go`: Schema - スキーマ全体を表現
- `entity.go`: Entity - エンティティ定義（例: "document", "user"）
- `relation.go`: Relation - リレーション定義（例: "owner: user"）
- `attribute_schema.go`: AttributeSchema - 属性型定義（例: "public: boolean"）
- `permission.go`: Permission - 権限定義（例: "edit = owner or editor"）
- `rule.go`: PermissionRule インターフェース + 各ルール実装（RelationRule, LogicalRule, HierarchicalRule, ABACRule）

**データ系**（実際に保存されるデータ）：

- `relation_tuple.go`: RelationTuple - 関係データ（例: document:1#owner@user:alice）
- `attribute.go`: Attribute - 属性データ（例: document:1.public = true）

この設計により、スキーマの「定義」と実際の「データ」が明確に分離され、可読性と保守性が向上します。

#### services 層の設計

services 層は機能ごとに分割し、ABAC/ReBAC の区別を実装詳細として隠蔽します。

**Service 層の責務**:

- Repository 層から取得した生データを処理
- ビジネスロジック、パース処理、検証を担当
- 上位層（ハンドラー）に必要な形式でデータを提供

**ファイル構成**:

- `parser/`: DSL の字句解析・構文解析・検証を担当
- `schema_service.go`: スキーマ管理サービス
  - DSL パース（Lexer → Parser → AST → Validator）
  - Entities の生成（`GetSchemaEntity`メソッドでパース済み Schema を返す）
  - スキーマの保存・取得・検証
  - Repository 層から取得した DSL 文字列を解析し、内部表現に変換
- `authorization/`: 全ての認可処理を統合
  - `evaluator.go`: ルール評価のコア（ReBAC の関係性チェックと ABAC の CEL 評価を統合）
  - `checker.go`: Check API の実装
  - `expander.go`: Expand API の実装
  - `lookup.go`: LookupEntity/LookupSubject API の実装（ABAC ルールにも対応）
  - `cel.go`: CEL エンジンのラッパー

**重要な設計パターン**:

- `SchemaServiceInterface`: authorization パッケージで定義（循環依存回避）
  - Evaluator、Checker、Expander、Lookup はこのインターフェースを使用
  - `GetSchemaEntity(ctx, tenantID)`メソッドでパース済み Schema を取得

この設計により、Lookup などの機能が ReBAC/ABAC 両方で使えることが明確になります。

#### repositories 層の設計

DB の差し替えを想定し、インターフェースと実装を分離します。

**Repository 層の責務**:

- **DB への入出力のみを担当**（Create/Read/Update/Delete）
- パース処理やビジネスロジックは含まない
- 生データ（DSL 文字列、タプル、属性値）のみを扱う
- エラーハンドリング：センチネルエラー（`repositories.ErrNotFound`）を使用

**ファイル構成**:

- `errors.go`: センチネルエラー定義（`ErrNotFound`など）
- `schema_repository.go`: SchemaRepository インターフェース定義
- `relation_repository.go`: RelationRepository インターフェース定義
- `attribute_repository.go`: AttributeRepository インターフェース定義
- `postgres/`: PostgreSQL 実装
  - `schema_repository.go`: PostgreSQL 用の SchemaRepository 実装
  - `relation_repository.go`: PostgreSQL 用の RelationRepository 実装
  - `attribute_repository.go`: PostgreSQL 用の AttributeRepository 実装

将来的に MySQL や他の DB に切り替える場合は、`mysql/` ディレクトリを追加し、各リポジトリの MySQL 実装を配置するだけで済みます。

#### infrastructure 層の設計

外部システムとの接続を管理します。

- `database/`: PostgreSQL 接続とマイグレーション管理
  - `postgres.go`: DB 接続プール、ヘルスチェック、マイグレーション実行
  - `migrations/postgres/`: PostgreSQL 用マイグレーション SQL ファイル
- `config/`: 環境変数や設定ファイルの読み込み
  - `config.go`: 設定構造体と環境変数読み込み

---

## データベース設計

### 2. PostgreSQL スキーマ（最終版）

#### 2.1 schemas テーブル

```sql
CREATE TABLE schemas (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    schema_dsl TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

CREATE INDEX idx_schemas_tenant ON schemas(tenant_id);
```

#### 2.2 relations テーブル

```sql
CREATE TABLE relations (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    relation VARCHAR(255) NOT NULL,
    subject_type VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    subject_relation VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, entity_type, entity_id, relation, subject_type, subject_id, COALESCE(subject_relation, ''))
);

CREATE INDEX idx_relations_entity ON relations(tenant_id, entity_type, entity_id);
CREATE INDEX idx_relations_subject ON relations(tenant_id, subject_type, subject_id);
CREATE INDEX idx_relations_lookup ON relations(tenant_id, entity_type, relation, subject_type, subject_id);
```

設計ポイント:

- VARCHAR(255)を使用（正規化なし、将来的に最適化可能）
- Permify 互換: `entity_type/entity_id`（object_type ではない）
- subject_relation は NULLABLE（user:alice のような単純なサブジェクトの場合は NULL）

#### 2.3 attributes テーブル

```sql
CREATE TABLE attributes (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    attribute VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, entity_type, entity_id, attribute)
);

CREATE INDEX idx_attributes_entity ON attributes(tenant_id, entity_type, entity_id);
```

設計ポイント:

- value は TEXT 型（JSON 文字列として保存）
- 将来的に JSONB 型への移行を検討可能
- Phase 1 では文字列として扱い、CEL 評価時にパース

---

## コア実装設計

### 3. DSL パーサー設計

#### 3.1 パース処理フロー

**WriteSchema（DSL → DB 保存）**:

```text
DSL文字列
   ↓
Lexer（字句解析） → Tokens
   ↓
Parser（構文解析） → AST（抽象構文木）
   ↓
Validator（検証） → Validated AST
   ↓
Converter（AST → entities.Schema） → Schema Entity（内部表現）
   ↓
DB保存（schema_dsl として保存）
```

**ReadSchema（DB 取得 → DSL 生成）**:

```text
DB取得
   ↓
DSL文字列をパース → AST
   ↓
Generator（AST → DSL文字列） → DSL文字列
   ↓
クライアントに返却
```

**注**: DB には DSL 文字列（schema_dsl）として保存されるため、ReadSchema では保存された DSL 文字列をそのまま返すことも可能ですが、将来的にスキーマを正規化して保存する場合に備え、AST 経由での生成機能を実装します。

#### 3.2 AST 構造

```go
// internal/services/parser/ast.go

type SchemaAST struct {
    Entities []*EntityAST
}

type EntityAST struct {
    Name       string
    Relations  []*RelationAST
    Attributes []*AttributeAST
    Permissions []*PermissionAST
}

type RelationAST struct {
    Name       string
    TargetType string
}

type AttributeAST struct {
    Name string
    Type string // "string", "int", "bool", "string[]"
}

type PermissionAST struct {
    Name string
    Rule PermissionRuleAST
}

type PermissionRuleAST interface {
    isPermissionRule()
}

// 関係性ベースの権限
type RelationPermissionAST struct {
    Relation string
}

// 論理演算
type LogicalPermissionAST struct {
    Operator string // "or", "and", "not"
    Left     PermissionRuleAST
    Right    PermissionRuleAST // notの場合はnil
}

// 階層的アクセス
type HierarchicalPermissionAST struct {
    Relation   string
    Permission string
}

// ABACルール
type RulePermissionAST struct {
    Expression string // CEL式
}
```

#### 3.3 Lexer 実装（概要）

```go
// internal/services/parser/lexer.go

type TokenType int

const (
    TOKEN_ENTITY TokenType = iota
    TOKEN_RELATION
    TOKEN_ATTRIBUTE
    TOKEN_PERMISSION
    TOKEN_RULE
    TOKEN_IDENTIFIER
    TOKEN_COLON
    TOKEN_EQUALS
    TOKEN_OR
    TOKEN_AND
    TOKEN_NOT
    // ...
)

type Token struct {
    Type    TokenType
    Value   string
    Line    int
    Column  int
}

type Lexer struct {
    input   string
    pos     int
    line    int
    column  int
}

func NewLexer(input string) *Lexer
func (l *Lexer) NextToken() (*Token, error)
```

#### 3.4 Parser 実装（概要）

```go
// internal/services/parser/parser.go

type Parser struct {
    lexer   *Lexer
    current *Token
    peek    *Token
}

func NewParser(lexer *Lexer) *Parser
func (p *Parser) Parse() (*SchemaAST, error)
func (p *Parser) parseEntity() (*EntityAST, error)
func (p *Parser) parsePermission() (*PermissionAST, error)
func (p *Parser) parsePermissionRule() (PermissionRuleAST, error)
```

#### 3.5 Converter 実装（概要）

```go
// internal/services/parser/converter.go

// AST → entities.Schema 変換
func ASTToSchema(tenantID string, ast *SchemaAST) (*entities.Schema, error)

// entities.Schema → AST 変換
func SchemaToAST(schema *entities.Schema) (*SchemaAST, error)

// 内部ヘルパー関数
func convertEntity(ast *EntityAST) (*entities.Entity, error)
func convertPermissionRule(ast PermissionRuleAST) (entities.PermissionRule, error)
func convertEntityToAST(entity *entities.Entity) (*EntityAST, error)
func convertPermissionRuleToAST(rule entities.PermissionRule) (PermissionRuleAST, error)
```

設計ポイント:

- AST と entities.Schema の双方向変換を提供
- WriteSchema 時: AST → entities.Schema（スキーマ検証で使用）
- ReadSchema 時（将来拡張用）: entities.Schema → AST → DSL 文字列

#### 3.6 Generator 実装（概要）

```go
// internal/services/parser/generator.go

type Generator struct {
    indent string
}

func NewGenerator() *Generator

// AST から DSL 文字列を生成
func (g *Generator) Generate(schema *SchemaAST) string

// 内部ヘルパー関数
func (g *Generator) generateEntity(entity *EntityAST) string
func (g *Generator) generateRelation(relation *RelationAST) string
func (g *Generator) generateAttribute(attr *AttributeAST) string
func (g *Generator) generatePermission(perm *PermissionAST) string
func (g *Generator) generatePermissionRule(rule PermissionRuleAST) string
```

設計ポイント:

- AST から正しくフォーマットされた DSL 文字列を生成
- 演算子の優先順位を考慮した括弧の追加
- インデント整形（デフォルト 2 スペース）
- ReadSchema での DSL 文字列生成に使用

---

### 4. Repository 設計

#### 4.1 スキーマリポジトリ

**責務**: DB への入出力のみ。DSL パース処理は Service 層で実施。

**エラーハンドリング**:

- スキーマが存在しない場合: `repositories.ErrNotFound`を wrap して返す
- Service 層で`errors.Is(err, repositories.ErrNotFound)`でチェック

インターフェース定義:

```go
// internal/repositories/schema_repository.go

type SchemaRepository interface {
    // Create creates a new schema for the tenant
    Create(ctx context.Context, tenantID string, schemaDSL string) error

    // GetByTenant retrieves schema by tenant ID
    // Returns ErrNotFound if schema does not exist
    // Note: Entities field will be empty (populated by service layer)
    GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error)

    // Update updates an existing schema
    Update(ctx context.Context, tenantID string, schemaDSL string) error

    // Delete deletes a schema
    Delete(ctx context.Context, tenantID string) error
}
```

PostgreSQL 実装:

```go
// internal/repositories/postgres/schema_repository.go

type PostgresSchemaRepository struct {
    db *sql.DB
}

func NewPostgresSchemaRepository(db *sql.DB) *PostgresSchemaRepository {
    return &PostgresSchemaRepository{db: db}
}

func (r *PostgresSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) error {
    query := `INSERT INTO schemas (tenant_id, schema_dsl) VALUES ($1, $2)`
    _, err := r.db.ExecContext(ctx, query, tenantID, schemaDSL)
    return err
}

func (r *PostgresSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
    query := `
        SELECT schema_dsl, created_at, updated_at
        FROM schemas
        WHERE tenant_id = $1
    `
    var schemaDSL string
    var createdAt, updatedAt time.Time

    err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&schemaDSL, &createdAt, &updatedAt)
    if err == sql.ErrNoRows {
        // Return ErrNotFound wrapped with context
        return nil, fmt.Errorf("schema not found for tenant %s: %w", tenantID, repositories.ErrNotFound)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get schema: %w", err)
    }

    schema := &entities.Schema{
        TenantID:  tenantID,
        DSL:       schemaDSL,
        CreatedAt: createdAt,
        UpdatedAt: updatedAt,
        // Note: Entities will be populated by the parser in the service layer
    }

    return schema, nil
}

// その他のメソッド実装...
```

**センチネルエラー定義**:

```go
// internal/repositories/errors.go

package repositories

import "errors"

// ErrNotFound is returned when a requested resource is not found
var ErrNotFound = errors.New("not found")
```

#### 4.2 リレーションリポジトリ

インターフェース定義:

```go
// internal/repositories/relation_repository.go

type RelationFilter struct {
    EntityType      string
    EntityID        string
    Relation        string
    SubjectType     string
    SubjectID       string
    SubjectRelation string
}

type RelationRepository interface {
    Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error
    Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error
    Read(ctx context.Context, tenantID string, filter *RelationFilter) ([]*entities.RelationTuple, error)
    CheckExists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error)
    BatchWrite(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
    BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
}
```

PostgreSQL 実装:

```go
// internal/repositories/postgres/relation_repository.go

type PostgresRelationRepository struct {
    db *sql.DB
}

func NewPostgresRelationRepository(db *sql.DB) *PostgresRelationRepository {
    return &PostgresRelationRepository{db: db}
}

func (r *PostgresRelationRepository) Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
    query := `
        INSERT INTO relations (tenant_id, entity_type, entity_id, relation, subject_type, subject_id, subject_relation)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT DO NOTHING
    `
    _, err := r.db.ExecContext(ctx, query,
        tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
        tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
    )
    return err
}

// その他のメソッド実装...
```

#### 4.3 アトリビュートリポジトリ

インターフェース定義:

```go
// internal/repositories/attribute_repository.go

type AttributeRepository interface {
    Write(ctx context.Context, tenantID string, attr *entities.Attribute) error
    Read(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error)
    Delete(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) error
    GetValue(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) (interface{}, error)
}
```

PostgreSQL 実装:

```go
// internal/repositories/postgres/attribute_repository.go

type PostgresAttributeRepository struct {
    db *sql.DB
}

func NewPostgresAttributeRepository(db *sql.DB) *PostgresAttributeRepository {
    return &PostgresAttributeRepository{db: db}
}

func (r *PostgresAttributeRepository) Write(ctx context.Context, tenantID string, attr *entities.Attribute) error {
    query := `
        INSERT INTO attributes (tenant_id, entity_type, entity_id, attribute, value)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (tenant_id, entity_type, entity_id, attribute) DO UPDATE SET value = EXCLUDED.value
    `
    _, err := r.db.ExecContext(ctx, query,
        tenantID, attr.EntityType, attr.EntityID, attr.Name, attr.Value,
    )
    return err
}

// その他のメソッド実装...
```

---

### 5. 認可エンジン設計

#### 5.1 ルール評価（Evaluator）

**SchemaServiceInterface**: 循環依存を回避するため、authorization パッケージで定義

```go
// internal/services/authorization/evaluator.go

// SchemaServiceInterface defines the interface for schema operations
// This interface is defined here to avoid circular dependency
type SchemaServiceInterface interface {
    GetSchemaEntity(ctx context.Context, tenantID string) (*entities.Schema, error)
}

type Evaluator struct {
    schemaService SchemaServiceInterface
    relationRepo  repositories.RelationRepository
    attributeRepo repositories.AttributeRepository
    celEngine     *CELEngine
}

func NewEvaluator(
    schemaService SchemaServiceInterface,
    relationRepo repositories.RelationRepository,
    attributeRepo repositories.AttributeRepository,
    celEngine *CELEngine,
) *Evaluator {
    return &Evaluator{
        schemaService: schemaService,
        relationRepo:  relationRepo,
        attributeRepo: attributeRepo,
        celEngine:     celEngine,
    }
}

// ルール評価のコア処理
func (e *Evaluator) EvaluateRule(
    ctx context.Context,
    tenantID string,
    schema *entities.Schema,
    entity *pb.Entity,
    rule entities.PermissionRule,
    subject *pb.Subject,
    contextualTuples []*pb.RelationTuple,
    contextualAttrs map[string]interface{},
    depth int,
) (bool, error) {
    switch r := rule.(type) {
    case *entities.RelationRule:
        return e.evaluateRelation(ctx, tenantID, entity, r, subject, contextualTuples)
    case *entities.LogicalRule:
        return e.evaluateLogical(ctx, tenantID, schema, entity, r, subject, contextualTuples, contextualAttrs, depth)
    case *entities.HierarchicalRule:
        return e.evaluateHierarchical(ctx, tenantID, schema, entity, r, subject, contextualTuples, contextualAttrs, depth)
    case *entities.ABACRule:
        return e.evaluateABAC(ctx, tenantID, entity, r, subject, contextualAttrs)
    default:
        return false, fmt.Errorf("unknown rule type")
    }
}

func (e *Evaluator) evaluateRelation(
    ctx context.Context,
    tenantID string,
    entity *pb.Entity,
    rule *entities.RelationRule,
    subject *pb.Subject,
    contextualTuples []*pb.RelationTuple,
) (bool, error) {
    // ReBAC: 関係性の存在チェック
    tuple := &entities.RelationTuple{
        EntityType:  entity.Type,
        EntityID:    entity.Id,
        Relation:    rule.Relation,
        SubjectType: subject.Type,
        SubjectID:   subject.Id,
        SubjectRelation: subject.Relation,
    }

    // Contextual tuples をチェック
    for _, ct := range contextualTuples {
        if tupleMatches(ct, tuple) {
            return true, nil
        }
    }

    // DB をチェック
    return e.relationRepo.CheckExists(ctx, tenantID, tuple)
}

func (e *Evaluator) evaluateABAC(
    ctx context.Context,
    tenantID string,
    entity *pb.Entity,
    rule *entities.ABACRule,
    subject *pb.Subject,
    contextualAttrs map[string]interface{},
) (bool, error) {
    // ABAC: CEL 式の評価

    // サブジェクトの属性を取得
    subjectAttrs, err := e.attributeRepo.Read(ctx, tenantID, subject.Type, subject.Id)
    if err != nil {
        return false, err
    }

    // リソースの属性を取得
    resourceAttrs, err := e.attributeRepo.Read(ctx, tenantID, entity.Type, entity.Id)
    if err != nil {
        return false, err
    }

    // Contextual attributes をマージ
    // ...

    // CEL 評価
    return e.celEngine.Evaluate(rule.Expression, subjectAttrs, resourceAttrs, contextualAttrs)
}
```

#### 5.2 CEL エンジン

```go
// internal/services/authorization/cel.go

import (
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/checker/decls"
)

type CELEngine struct {
    env *cel.Env
}

func NewCELEngine() (*CELEngine, error) {
    env, err := cel.NewEnv(
        cel.Declarations(
            decls.NewVar("subject", decls.NewMapType(decls.String, decls.Any)),
            decls.NewVar("resource", decls.NewMapType(decls.String, decls.Any)),
            decls.NewVar("context", decls.NewMapType(decls.String, decls.Any)),
        ),
    )
    if err != nil {
        return nil, err
    }
    return &CELEngine{env: env}, nil
}

func (e *CELEngine) Evaluate(
    expression string,
    subjectAttrs map[string]interface{},
    resourceAttrs map[string]interface{},
    contextAttrs map[string]interface{},
) (bool, error) {
    ast, issues := e.env.Compile(expression)
    if issues != nil && issues.Err() != nil {
        return false, issues.Err()
    }

    prg, err := e.env.Program(ast)
    if err != nil {
        return false, err
    }

    out, _, err := prg.Eval(map[string]interface{}{
        "subject":  subjectAttrs,
        "resource": resourceAttrs,
        "context":  contextAttrs,
    })
    if err != nil {
        return false, err
    }

    result, ok := out.Value().(bool)
    if !ok {
        return false, fmt.Errorf("expression did not evaluate to boolean")
    }

    return result, nil
}
```

サポートする演算子（Phase 1 で完全サポート）:

- 比較: `==`, `!=`, `>`, `>=`, `<`, `<=`
- コレクション: `in`
- 論理: `&&`, `||`, `!`

例:

```cel
resource.owner == subject.id
subject.age >= 18
subject.role in ["admin", "editor"]
resource.public == true || resource.owner == subject.id
```

#### 5.3 Check 実装

```go
// internal/services/authorization/checker.go

type Checker struct {
    schemaService SchemaServiceInterface
    evaluator     *Evaluator
}

func NewChecker(schemaService SchemaServiceInterface, evaluator *Evaluator) *Checker {
    return &Checker{
        schemaService: schemaService,
        evaluator:     evaluator,
    }
}

func (c *Checker) Check(
    ctx context.Context,
    tenantID string,
    schema *entities.Schema,
    entity *pb.Entity,
    permission string,
    subject *pb.Subject,
    contextualTuples []*pb.RelationTuple,
    contextualAttrs map[string]interface{},
    depth int,
) (bool, error) {
    // 深さ制限チェック
    if depth > 10 {
        return false, fmt.Errorf("max depth exceeded")
    }

    // スキーマから権限定義を取得
    permDef := schema.GetPermission(entity.Type, permission)
    if permDef == nil {
        return false, fmt.Errorf("permission not found")
    }

    // Evaluator を使ってルールを評価
    return c.evaluator.EvaluateRule(
        ctx, tenantID, schema, entity, permDef.Rule, subject,
        contextualTuples, contextualAttrs, depth,
    )
}
```

#### 5.4 Expand 実装

```go
// internal/services/authorization/expander.go

type Expander struct {
    schemaService SchemaServiceInterface
    relationRepo  repositories.RelationRepository
}

func NewExpander(schemaService SchemaServiceInterface, relationRepo repositories.RelationRepository) *Expander {
    return &Expander{
        schemaService: schemaService,
        relationRepo:  relationRepo,
    }
}

type ExpandNode struct {
    Type     string // "union", "intersection", "leaf"
    Children []*ExpandNode
    Subject  *pb.Subject // leafの場合
}

func (e *Expander) Expand(
    ctx context.Context,
    tenantID string,
    schema *entities.Schema,
    entity *pb.Entity,
    permission string,
) (*ExpandNode, error) {
    permDef := schema.GetPermission(entity.Type, permission)
    if permDef == nil {
        return nil, fmt.Errorf("permission not found")
    }

    return e.expandRule(ctx, tenantID, schema, entity, permDef.Rule)
}

func (e *Expander) expandRule(
    ctx context.Context,
    tenantID string,
    schema *entities.Schema,
    entity *pb.Entity,
    rule entities.PermissionRule,
) (*ExpandNode, error) {
    // ルールの種類に応じてツリーを構築
    // ...
}
```

#### 5.5 Lookup 実装

```go
// internal/services/authorization/lookup.go

type Lookup struct {
    schemaService SchemaServiceInterface
    checker       *Checker
    relationRepo  repositories.RelationRepository
}

func NewLookup(
    schemaService SchemaServiceInterface,
    checker *Checker,
    relationRepo repositories.RelationRepository,
) *Lookup {
    return &Lookup{
        schemaService: schemaService,
        checker:       checker,
        relationRepo:  relationRepo,
    }
}

// LookupEntity: ABAC/ReBAC 両方に対応
func (l *Lookup) LookupEntity(
    ctx context.Context,
    tenantID string,
    schema *entities.Schema,
    entityType string,
    permission string,
    subject *pb.Subject,
    contextualTuples []*pb.RelationTuple,
    contextualAttrs map[string]interface{},
) ([]string, error) {
    // 1. 対象エンティティタイプの候補を列挙
    // 2. 各エンティティに対して Check を実行
    // 3. 許可されたエンティティの ID を返す
    // 注: Phase 1 では愚直な実装、最適化は後で
}

// LookupSubject: ABAC/ReBAC 両方に対応
func (l *Lookup) LookupSubject(
    ctx context.Context,
    tenantID string,
    schema *entities.Schema,
    entity *pb.Entity,
    permission string,
    subjectType string,
    contextualTuples []*pb.RelationTuple,
    contextualAttrs map[string]interface{},
) ([]string, error) {
    // 1. 対象サブジェクトタイプの候補を列挙
    // 2. 各サブジェクトに対して Check を実行
    // 3. 許可されたサブジェクトの ID を返す
}
```

この設計により、Lookup が ABAC ルールを含む全てのルールタイプに対応できることが明確になります。

---

### 6. gRPC ハンドラー設計

#### 設計原則：単一サービスアプローチ

Google Zanzibar、Permify、Auth0 FGA などの業界標準に従い、認可サービスは**単一の gRPC サービス**として実装します。

**理由**：

- 認可は分離できない 1 つのドメイン（Schema、Relations、Authorization は密接に連携）
- クライアントが 1 つのサービスに接続するだけで全操作が可能
- Permify 互換性の維持
- デプロイ・運用の単純化

#### 6.1 統合 Authorization Handler

```go
// internal/handlers/authorization_handler.go

import (
    "context"
    "errors"  // For ErrNotFound handling
    // ...
)

type AuthorizationHandler struct {
    // Schema management
    schemaService services.SchemaServiceInterface

    // Data management (Relations & Attributes)
    relationRepo  repositories.RelationRepository
    attributeRepo repositories.AttributeRepository

    // Authorization operations
    checker  CheckerInterface
    expander ExpanderInterface
    lookup   LookupInterface

    pb.UnimplementedAuthorizationServiceServer
}

// === Schema Management ===

func (h *AuthorizationHandler) WriteSchema(ctx context.Context, req *pb.WriteSchemaRequest) (*pb.WriteSchemaResponse, error) {
    // 1. テナント ID を取得（Phase 1: 固定 "default"）
    // 2. SchemaService.WriteSchema 呼び出し
    // 3. エラー変換（domain → gRPC）
    // 4. レスポンスを返す
}

func (h *AuthorizationHandler) ReadSchema(ctx context.Context, req *pb.ReadSchemaRequest) (*pb.ReadSchemaResponse, error) {
    // 1. SchemaService.ReadSchema でDSL取得
    // 2. SchemaService.GetSchemaEntity でメタデータ取得
    // 3. レスポンスを返す
}

// === Data Management ===

func (h *AuthorizationHandler) WriteRelations(ctx context.Context, req *pb.WriteRelationsRequest) (*pb.WriteRelationsResponse, error) {
    // 1. proto RelationTuple → entities.RelationTuple 変換
    // 2. RelationRepository.BatchWrite 呼び出し
    // 3. レスポンスを返す
}

func (h *AuthorizationHandler) DeleteRelations(ctx context.Context, req *pb.DeleteRelationsRequest) (*pb.DeleteRelationsResponse, error) {
    // 1. proto → entities 変換
    // 2. RelationRepository.BatchDelete 呼び出し
    // 3. レスポンスを返す
}

func (h *AuthorizationHandler) WriteAttributes(ctx context.Context, req *pb.WriteAttributesRequest) (*pb.WriteAttributesResponse, error) {
    // 1. proto AttributeData → entities.Attribute 変換（展開）
    // 2. AttributeRepository.Write 呼び出し
    // 3. レスポンスを返す
}

// === Authorization Operations ===

func (h *AuthorizationHandler) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
    // 1. リクエスト検証
    // 2. proto → authorization.CheckRequest 変換
    // 3. Checker.Check 呼び出し
    // 4. 結果を ALLOWED/DENIED に変換
}

func (h *AuthorizationHandler) Expand(ctx context.Context, req *pb.ExpandRequest) (*pb.ExpandResponse, error) {
    // 1. proto → authorization.ExpandRequest 変換
    // 2. Expander.Expand 呼び出し
    // 3. ExpandNode を proto に変換
}

func (h *AuthorizationHandler) LookupEntity(ctx context.Context, req *pb.LookupEntityRequest) (*pb.LookupEntityResponse, error) {
    // 1. proto → authorization.LookupEntityRequest 変換
    // 2. Lookup.LookupEntity 呼び出し
    // 3. ページネーション対応レスポンス
}

func (h *AuthorizationHandler) LookupSubject(ctx context.Context, req *pb.LookupSubjectRequest) (*pb.LookupSubjectResponse, error) {
    // 1. proto → authorization.LookupSubjectRequest 変換
    // 2. Lookup.LookupSubject 呼び出し
    // 3. ページネーション対応レスポンス
}

func (h *AuthorizationHandler) SubjectPermission(ctx context.Context, req *pb.SubjectPermissionRequest) (*pb.SubjectPermissionResponse, error) {
    // 1. schemaService.GetSchemaEntity() でパース済みスキーマを取得
    // 2. errors.Is(err, repositories.ErrNotFound) でエラーハンドリング
    // 3. 対象エンティティの全パーミッションを取得
    // 4. 各パーミッションに対して Checker.Check 実行
    // 5. 結果を map[permission]CheckResult で返却
}

func (h *AuthorizationHandler) LookupEntityStream(req *pb.LookupEntityRequest, stream pb.AuthorizationService_LookupEntityStreamServer) error {
    // Phase 1: Unimplemented
    return status.Error(codes.Unimplemented, "LookupEntityStream not implemented in Phase 1")
}
```

#### 6.2 ヘルパー関数

```go
// protoToRelationTuple: proto RelationTuple → entities.RelationTuple 変換
func protoToRelationTuple(proto *pb.RelationTuple) (*entities.RelationTuple, error)

// protoToAttributes: proto AttributeData → []entities.Attribute 変換（展開）
func protoToAttributes(proto *pb.AttributeData) ([]*entities.Attribute, error)

// protoContextToTuples: proto Context → []entities.RelationTuple 変換
func protoContextToTuples(ctx *pb.Context) ([]*entities.RelationTuple, error)

// expandNodeToProto: authorization.ExpandNode → proto ExpandNode 変換
func expandNodeToProto(node *authorization.ExpandNode) *pb.ExpandNode
```

---

## インフラストラクチャ設計

### 7. 設定管理

```go
// internal/infrastructure/config/config.go

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
}

type ServerConfig struct {
    Port int
}

type DatabaseConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    Database string
    SSLMode  string
}

func Load() (*Config, error) {
    // 環境変数から設定を読み込む
    // .env ファイルのサポート（godotenv）
    return &Config{
        Server: ServerConfig{
            Port: getEnvAsInt("SERVER_PORT", 50051),
        },
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     getEnvAsInt("DB_PORT", 5432),
            User:     getEnv("DB_USER", "keruberosu"),
            Password: getEnv("DB_PASSWORD", ""),
            Database: getEnv("DB_NAME", "keruberosu_dev"),
            SSLMode:  getEnv("DB_SSLMODE", "disable"),
        },
    }, nil
}
```

### 8. PostgreSQL 接続

```go
// internal/infrastructure/database/postgres.go

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
)

type Postgres struct {
    DB *sql.DB
}

func NewPostgres(cfg *config.DatabaseConfig) (*Postgres, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
    )

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // 接続プール設定
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)

    // 疎通確認
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &Postgres{DB: db}, nil
}

func (p *Postgres) Close() error {
    return p.DB.Close()
}

func (p *Postgres) HealthCheck() error {
    return p.DB.Ping()
}

func (p *Postgres) RunMigrations() error {
    // マイグレーションファイルは internal/infrastructure/database/migrations/postgres/ に配置
    migrationsPath := "internal/infrastructure/database/migrations/postgres"

    driver, err := postgres.WithInstance(p.DB, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("failed to create migration driver: %w", err)
    }

    m, err := migrate.NewWithDatabaseInstance(
        "file://"+migrationsPath,
        "postgres",
        driver,
    )
    if err != nil {
        return fmt.Errorf("failed to create migration instance: %w", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    return nil
}
```

---

## 依存ライブラリ

### 必須ライブラリ

```go
// go.mod

module github.com/asakaida/keruberosu

go 1.21

require (
    github.com/google/cel-go v0.18.2              // ABAC 評価エンジン
    google.golang.org/grpc v1.59.0                // gRPC フレームワーク
    google.golang.org/protobuf v1.31.0            // Protocol Buffers
    github.com/lib/pq v1.10.9                     // PostgreSQL ドライバー
    github.com/golang-migrate/migrate/v4 v4.16.2  // マイグレーションツール
    github.com/joho/godotenv v1.5.1               // 環境変数管理
    go.uber.org/zap v1.26.0                       // 構造化ログ
)
```

---

## 環境構成

### Docker Compose 設計

2 つの環境を用意:

- dev: 開発用（ポート 5432）
- test: テスト用（ポート 5433）

```yaml
# docker-compose.yml

version: "3.8"

services:
  postgres-dev:
    image: postgres:18.0-alpine3.22
    container_name: keruberosu-postgres-dev
    environment:
      POSTGRES_USER: keruberosu
      POSTGRES_PASSWORD: keruberosu_dev
      POSTGRES_DB: keruberosu_dev
    ports:
      - "5432:5432"
    volumes:
      - postgres-dev-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U keruberosu"]
      interval: 5s
      timeout: 5s
      retries: 5

  postgres-test:
    image: postgres:18.0-alpine3.22
    container_name: keruberosu-postgres-test
    environment:
      POSTGRES_USER: keruberosu
      POSTGRES_PASSWORD: keruberosu_test
      POSTGRES_DB: keruberosu_test
    ports:
      - "5433:5432"
    volumes:
      - postgres-test-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U keruberosu"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres-dev-data:
  postgres-test-data:
```

---

## マイグレーション戦略

### golang-migrate 使用

```bash
# マイグレーション作成
migrate create -ext sql -dir migrations -seq create_schemas_table

# マイグレーション適用（dev）
migrate -path migrations -database "postgres://keruberosu:keruberosu_dev@localhost:5432/keruberosu_dev?sslmode=disable" up

# マイグレーション適用（test）
migrate -path migrations -database "postgres://keruberosu:keruberosu_test@localhost:5433/keruberosu_test?sslmode=disable" up
```

---

## 実装の優先順位

### Phase 1 実装ステップ

1. 基盤構築（Week 1）

   - プロジェクト構造作成
   - docker-compose.yml 作成
   - マイグレーションファイル作成
   - Protocol Buffers 定義

2. データ層実装（Week 1-2）

   - PostgreSQL 接続
   - Repository インターフェース定義
   - PostgreSQL 実装（Schema, Relation, Attribute）
   - 基本的な CRUD 操作

3. DSL パーサー実装（Week 2-3）

   - Lexer 実装
   - Parser 実装
   - AST 定義
   - Validator 実装

4. 認可エンジン実装（Week 3-6）

   - CEL エンジン実装
   - Evaluator 実装（ReBAC + ABAC）
   - Checker 実装
   - Expander 実装
   - Lookup 実装

5. gRPC ハンドラー実装（Week 6-7）

   - Authorization Handler
   - Data Handler
   - Schema Handler
   - エラーハンドリング

6. 統合テスト（Week 7-8）
   - E2E テスト
   - パフォーマンステスト
   - Permify 互換性テスト

---

## テスト戦略

### テストレベル

1. ユニットテスト

   - 各パッケージごとにテスト
   - カバレッジ 80%以上目標

2. 統合テスト

   - PostgreSQL 込みのテスト
   - test コンテナ使用

3. E2E テスト
   - gRPC 経由の完全なシナリオテスト
   - Permify 互換性検証

---

## エラーハンドリング

### gRPC エラーコード

```go
import "google.golang.org/grpc/codes"

// 使用するエラーコード
codes.InvalidArgument   // リクエストパラメータ不正
codes.NotFound          // リソースが存在しない
codes.AlreadyExists     // リソースが既に存在
codes.PermissionDenied  // 権限不足
codes.Internal          // 内部エラー
codes.Unavailable       // サービス利用不可
```

---

## パフォーマンス考慮事項

### Phase 1 での方針

- 最適化は後回し: まず動作することを優先
- VARCHAR 設計: 正規化なし、後から最適化可能
- キャッシュなし: Phase 2 で実装
- インデックス: 基本的なインデックスのみ設定

### 将来的な最適化候補（Phase 2 以降）

- L1/L2 キャッシュ実装
- クエリ最適化
- JSONB 型への移行
- 接続プーリング改善
- 並列処理最適化

---

## セキュリティ考慮事項

1. テナント分離: 全てのクエリで tenantID をチェック
2. SQL インジェクション対策: プリペアドステートメント使用
3. 入力検証: 全ての gRPC リクエストで検証
4. 深さ制限: 再帰的な Check 処理で無限ループ防止（depth パラメータ）

---

## まとめ

本設計書は、Keruberosu Phase 1 の完全な実装ガイドです。

重要な方針:

- ABAC は全ての演算子をサポート（最小限ではない）
- 「シンプル」はキャッシュ機構がないだけで、機能は完全実装
- Phase 2 でキャッシュを追加するが、Phase 1 でも完全に動作する
- Permify 互換性を完全に維持
- Repository はインターフェースと実装を分離し、DB 差し替えに対応
- services 層は機能ごとに整理し、ABAC/ReBAC の区別を隠蔽

次のステップ: DEVELOPMENT.md で具体的な実装タスクを管理します。
