# Keruberosu - 設計書 (DESIGN.md)

## 概要

Keruberosu は、Permify ライクな ReBAC/ABAC 認可マイクロサービスです。
本ドキュメントでは、Phase 1-3 の実装設計を定義します。

### 実装スコープ（Phase 1-3 完了）

全ての機能が実装済みです。

#### Phase 1: 基盤構築（完了）

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

3. スキーマ管理

   - DSL パーサー（Lexer → Parser → AST → Validator）
   - スキーマの CRUD 操作
   - スキーマ検証ロジック
   - DSL ↔ Protocol Buffers 変換
   - ULID ベースのバージョン管理

4. データ管理 API

   - RelationTuple CRUD（Write, Read, Delete）
   - Attribute CRUD（Write, Read, Delete）
   - Batch 操作（WriteRelations, DeleteRelations）

5. Permify 互換 gRPC API
   - 3 サービス構成（Permission, Data, Schema）
   - 全ての Protocol Buffers メッセージ定義
   - エラーハンドリング
   - メタデータ（snap_token, schema_version）

#### Phase 2: パフォーマンス最適化（完了）

1. キャッシュシステム
   - LRU + TTL ベースのインメモリキャッシュ
   - SnapshotToken による MVCC 対応
   - PostgreSQL txid_current() でトークン生成
   - CheckerWithCache による透過的キャッシュ

2. Closure Table
   - O(1) 祖先検索のための最適化テーブル
   - Write/Delete 時の自動更新
   - 階層的パーミッション評価の高速化

3. メトリクスシステム
   - Prometheus 形式でメトリクス公開
   - gRPC リクエスト/エラー/処理時間
   - キャッシュヒット率

#### Phase 3: DB 基盤強化と高度な認可機能（完了）

1. DB 基盤強化

   - DBCluster: Primary + Read Replica 対応（Writer()/ReaderFor(tenantID) による自動ルーティング）
   - ResilientDB: トランジェントエラー（接続断、リソース不足等）の自動リトライ（指数バックオフ+ジッター）
   - WriteTracker: レプリカ整合性のための書き込み追跡（テナント単位で最近の書き込みを追跡し、レプリカラグ中は Primary にルーティング）
   - DBTX interface: `*sql.DB` の抽象化（ResilientDB が実装）
   - Config 拡張: DB_REPLICA_HOST, DB_REPLICA_PORT, WRITE_TRACKER_WINDOW_SECONDS, CLOSURE_EXCLUDED_RELATIONS

2. リポジトリ改修

   - 全リポジトリが DBCluster 化（Writer()/ReaderFor(tenantID) で読み書き分離）
   - RelationRepository に新メソッド追加: Exists, ExistsWithSubjectRelation, FindByEntityWithRelation, LookupAncestorsViaRelation, FindHierarchicalWithSubject, RebuildClosure, GetSortedEntityIDs, GetSortedSubjectIDs, LookupAccessibleEntitiesComplex, LookupAccessibleSubjectsComplex
   - Closure 除外設定（closureExcludedRelations）

3. HierarchicalRuleCallRule

   - 新ルールタイプ: `parent.check_conf(authority)` 構文
   - CELEngine.EvaluateWithParams: `this` + 動的パラメータ
   - パーサー、コンバーター、バリデーター、ジェネレーター対応

4. Evaluator 最適化

   - evaluateRelation: Read() から Exists() + FindByEntityWithRelation() に変更
   - evaluateHierarchical: 同一型階層 CTE 最適化

5. Lookup 再実装

   - extractRelationsFromRuleWithContext: スキーマ参照の再帰パーミッション展開
   - LookupAccessibleEntitiesComplex/SubjectsComplex: Closure+UNION 最適化
   - フォールバック: バッチ式 GetSortedEntityIDs + Check
   - SubjectRelation 対応

6. gRPC バリデーション

   - protovalidate + UnaryServerInterceptor
   - Proto 制約: Entity.type/id min_len=1, Tuple required fields 等
   - ChainUnaryInterceptor でメトリクスとバリデーションを連結

7. Admin CLI

   - `cmd/admin/main.go`: rebuild-closures コマンド（全テナントの Closure Table 再構築）
   - cobra ベースの CLI

8. 依存更新
   - gRPC v1.79.3
   - CEL v0.27.0
   - protovalidate v1.1.3
   - Go 1.25.1

### アーキテクチャ方針：3 サービス構成

Keruberosu は Permify 互換の 3 つの gRPC サービスとして実装されます。

サービス構成:

1. Permission Service: 権限チェック・検索
   - Check, Expand, LookupEntity, LookupSubject, SubjectPermission

2. Data Service: 関係性・属性データ管理
   - Write, Delete, Read, ReadAttributes

3. Schema Service: スキーマ定義管理
   - Write, Read, ListVersions

理由:

1. Permify 完全互換: Permify の API 構造に完全準拠
2. 関心の分離: 権限チェック、データ管理、スキーマ管理を明確に分離
3. 業界標準に準拠: Google Zanzibar、Auth0 FGA などの設計に従う
4. 柔軟なスケーリング: サービスごとに独立したスケーリングが可能

実装方針:

- `internal/handlers/permission_handler.go`: 権限チェック API
- `internal/handlers/data_handler.go`: データ管理 API
- `internal/handlers/schema_handler.go`: スキーマ管理 API
- 内部的には責務ごとにサービス層を分離（SchemaService, Checker, Expander, Lookup）

---

## アーキテクチャ設計

### 1. プロジェクト構造

```text
keruberosu/
├── cmd/
│   ├── server/
│   │   └── main.go                    # gRPC サーバーエントリーポイント
│   ├── migrate/
│   │   └── main.go                    # DB マイグレーションコマンド
│   └── admin/
│       └── main.go                    # Admin CLI（rebuild-closures等）
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
│   ├── handlers/                      # gRPC ハンドラー（3サービス）
│   │   ├── permission_handler.go     # Permission Service（Check, Expand, Lookup）
│   │   ├── data_handler.go           # Data Service（Write, Delete, Read）
│   │   ├── schema_handler.go         # Schema Service（Write, Read）
│   │   └── helpers.go                # 共通ヘルパー関数
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
│   │       ├── checker_with_cache.go # キャッシュ付き Check 処理
│   │       ├── expander.go          # Expand 処理
│   │       ├── lookup.go            # Lookup 処理
│   │       └── cel.go               # CEL 評価エンジン
│   ├── repositories/                  # データアクセス
│   │   ├── schema_repository.go     # スキーマリポジトリ インターフェース定義
│   │   ├── relation_repository.go   # リレーションリポジトリ インターフェース定義
│   │   ├── attribute_repository.go  # アトリビュートリポジトリ インターフェース定義
│   │   ├── errors.go                # センチネルエラー定義
│   │   └── postgres/                 # PostgreSQL 実装
│   │       ├── schema_repository.go     # スキーマリポジトリ実装（DBCluster対応）
│   │       ├── relation_repository.go   # リレーションリポジトリ実装（DBCluster+Closure Table）
│   │       ├── attribute_repository.go  # アトリビュートリポジトリ実装（DBCluster対応）
│   │       └── snapshot.go              # スナップショットトークン管理
│   └── infrastructure/                # インフラ層
│       ├── cache/
│       │   └── snapshot_manager.go   # SnapshotManager（MVCC対応）
│       ├── config/
│       │   └── config.go            # 設定管理（DB/Cache/Replica設定含む）
│       ├── database/
│       │   ├── cluster.go           # DBCluster（Primary + Read Replica）
│       │   ├── dbtx.go              # DBTX interface（DB抽象化）
│       │   ├── resilient_db.go      # ResilientDB（自動リトライ）
│       │   ├── write_tracker.go     # WriteTracker（書き込み追跡）
│       │   ├── postgres.go          # PostgreSQL 接続ヘルパー
│       │   ├── testing.go           # テスト用ヘルパー
│       │   └── migrations/postgres/  # PostgreSQL 用マイグレーション
│       ├── metrics/
│       │   ├── collector.go          # メトリクス収集
│       │   ├── prometheus.go         # Prometheus エクスポーター
│       │   └── interceptor.go        # gRPC インターセプター
│       └── validation/
│           └── interceptor.go        # protovalidate gRPC インターセプター
├── pkg/
│   └── cache/
│       ├── cache.go                  # キャッシュインターフェース
│       └── memorycache/
│           └── memorycache.go        # LRU + TTL インメモリキャッシュ
├── proto/
│   └── keruberosu/
│       └── v1/
│           ├── common.proto          # 共通メッセージ定義（protovalidate制約付き）
│           ├── permission.proto      # Permission Service 定義
│           ├── data.proto            # Data Service 定義
│           └── schema.proto          # Schema Service 定義
├── docs/                              # ドキュメント
│   ├── PRD.md                        # 要求仕様書
│   ├── DESIGN.md                     # 本ドキュメント
│   ├── ARCHITECTURE.md               # アーキテクチャ図
│   └── DEVELOPMENT.md                # 開発進捗管理
├── examples/                          # サンプルコード
├── scripts/
│   └── generate-proto.sh             # Protocol Buffers コード生成スクリプト
├── docker-compose.yml                 # PostgreSQL 環境（dev/test）
├── go.mod
└── go.sum
```

### 設計の考え方

#### entities 層の設計

entities 層は 1 ファイル 1 構造体の原則に従い、責務を明確に分離しています。

スキーマ定義系（DSL から生成される内部表現）:

- `schema.go`: Schema - スキーマ全体を表現
- `entity.go`: Entity - エンティティ定義（例: "document", "user"）
- `relation.go`: Relation - リレーション定義（例: "owner @user" or "contributor @user @team#member"）
- `attribute_schema.go`: AttributeSchema - 属性型定義（例: "public: boolean"）
- `permission.go`: Permission - 権限定義（例: "edit = owner or editor"）
- `rule.go`: PermissionRule インターフェース + 各ルール実装（RelationRule, LogicalRule, HierarchicalRule, ABACRule, RuleCallRule, HierarchicalRuleCallRule）

データ系（実際に保存されるデータ）:

- `relation_tuple.go`: RelationTuple - 関係データ（例: document:1#owner@user:alice、repository:1#contributor@team:backend-team#member）
- `attribute.go`: Attribute - 属性データ（例: document:1.public = true）

Subject Relation 機能: Permify 互換の subject relation 機能により、`team#member` のような記法でグループメンバーシップを 1 つのタプルで表現できます。例：`relation contributor @user @team#member` というスキーマ定義で、`repository:backend-api#contributor@team:backend-team#member` というタプルにより、チームメンバー全員に権限を付与できます。

この設計により、スキーマの「定義」と実際の「データ」が明確に分離され、可読性と保守性が向上します。

#### ルールタイプ一覧

```go
// internal/entities/rule.go

type PermissionRule interface {
    isPermissionRule()
}

// RelationRule: 関係性ベースの権限（例: "owner"）
type RelationRule struct { Relation string }

// LogicalRule: 論理演算（OR/AND/NOT）
type LogicalRule struct { Operator string; Left, Right PermissionRule }

// HierarchicalRule: 階層的権限（例: "parent.edit"）
type HierarchicalRule struct { Relation, Permission string }

// ABACRule: CEL式による属性ベースルール
type ABACRule struct { Expression string }

// RuleCallRule: トップレベルルール呼び出し（例: "is_public(resource)"）
type RuleCallRule struct { RuleName string; Arguments []string }

// HierarchicalRuleCallRule: 階層的ルール呼び出し（例: "parent.check_conf(authority)"）
type HierarchicalRuleCallRule struct { Relation, RuleName string; Arguments []string }
```

#### services 層の設計

services 層は機能ごとに分割し、ABAC/ReBAC の区別を実装詳細として隠蔽します。

Service 層の責務:

- Repository 層から取得した生データを処理
- ビジネスロジック、パース処理、検証を担当
- 上位層（ハンドラー）に必要な形式でデータを提供

ファイル構成:

- `parser/`: DSL の字句解析・構文解析・検証を担当
- `schema_service.go`: スキーマ管理サービス
  - DSL パース（Lexer → Parser → AST → Validator）
  - Entities の生成（`GetSchemaEntity` メソッドでパース済み Schema を返す）
  - スキーマの保存・取得・検証
  - Repository 層から取得した DSL 文字列を解析し、内部表現に変換
- `authorization/`: 全ての認可処理を統合
  - `evaluator.go`: ルール評価のコア（ReBAC の関係性チェックと ABAC の CEL 評価を統合）
  - `checker.go`: Check API の実装
  - `expander.go`: Expand API の実装
  - `lookup.go`: LookupEntity/LookupSubject API の実装（ABAC ルールにも対応）
  - `cel.go`: CEL エンジンのラッパー（EvaluateWithParams 含む）

重要な設計パターン:

- `SchemaServiceInterface`: authorization パッケージで定義（循環依存回避）
  - Evaluator、Checker、Expander、Lookup はこのインターフェースを使用
  - `GetSchemaEntity(ctx, tenantID, version)` メソッドでパース済み Schema を取得

この設計により、Lookup などの機能が ReBAC/ABAC 両方で使えることが明確になります。

#### repositories 層の設計

DB の差し替えを想定し、インターフェースと実装を分離します。

Repository 層の責務:

- DB への入出力のみを担当（Create/Read/Update/Delete）
- パース処理やビジネスロジックは含まない
- 生データ（DSL 文字列、タプル、属性値）のみを扱う
- エラーハンドリング：センチネルエラー（`repositories.ErrNotFound`）を使用

ファイル構成:

- `errors.go`: センチネルエラー定義（`ErrNotFound` など）
- `schema_repository.go`: SchemaRepository インターフェース定義
- `relation_repository.go`: RelationRepository インターフェース定義
- `attribute_repository.go`: AttributeRepository インターフェース定義
- `postgres/`: PostgreSQL 実装（全リポジトリが DBCluster を使用）
  - `schema_repository.go`: PostgreSQL 用の SchemaRepository 実装
  - `relation_repository.go`: PostgreSQL 用の RelationRepository 実装（Closure Table + closureExcludedRelations）
  - `attribute_repository.go`: PostgreSQL 用の AttributeRepository 実装
  - `snapshot.go`: スナップショットトークン管理

将来的に MySQL や他の DB に切り替える場合は、`mysql/` ディレクトリを追加し、各リポジトリの MySQL 実装を配置するだけで済みます。

#### infrastructure 層の設計

外部システムとの接続を管理します。

- `database/`: PostgreSQL 接続と DB 基盤
  - `cluster.go`: DBCluster - Primary と Read Replica の管理、WriteTracker による整合性制御
  - `dbtx.go`: DBTX interface - `*sql.DB` の抽象化（ExecContext, QueryContext, QueryRowContext, BeginTx, PingContext）
  - `resilient_db.go`: ResilientDB - トランジェントエラーの自動リトライ（指数バックオフ+ジッター）
  - `write_tracker.go`: WriteTracker - テナント単位の書き込み追跡（レプリカ整合性保証）
  - `postgres.go`: DB 接続ヘルパー
  - `migrations/postgres/`: PostgreSQL 用マイグレーション SQL ファイル
- `config/`: 環境変数や設定ファイルの読み込み
  - `config.go`: 設定構造体と環境変数読み込み（Server, Database, Cache 設定）
- `metrics/`: メトリクス収集と Prometheus エクスポーター
- `validation/`: protovalidate による gRPC リクエストバリデーション

---

## データベース設計

### 2. PostgreSQL スキーマ（最終版）

#### 2.1 schemas テーブル

```sql
CREATE TABLE schemas (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    version VARCHAR(26) NOT NULL,              -- ULID形式のバージョンID
    schema_dsl TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, version)                 -- テナントとバージョンの組み合わせで一意
);

CREATE INDEX idx_schemas_tenant ON schemas(tenant_id);
CREATE INDEX idx_schemas_version ON schemas(version);
CREATE INDEX idx_schemas_tenant_created ON schemas(tenant_id, created_at DESC);
```

設計ポイント:

- `version` カラム: ULID（26 文字）を使用したバージョン管理
- 各スキーマ書き込みで新しいバージョンを自動生成
- 最新バージョンは `created_at DESC` でソート取得
- Permify 互換: `schema_version` フィールドを返却

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

- VARCHAR(255) を使用（正規化なし、将来的に最適化可能）
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

WriteSchema（DSL → DB 保存）:

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

ReadSchema（DB 取得 → DSL 生成）:

```text
DB取得
   ↓
DSL文字列をパース → AST
   ↓
Generator（AST → DSL文字列） → DSL文字列
   ↓
クライアントに返却
```

注: DB には DSL 文字列（schema_dsl）として保存されるため、ReadSchema では保存された DSL 文字列をそのまま返すことも可能ですが、将来的にスキーマを正規化して保存する場合に備え、AST 経由での生成機能を実装します。

#### 3.2 AST 構造

```go
// internal/services/parser/ast.go

type SchemaAST struct {
    Rules    []*RuleDefinitionAST // トップレベルルール定義
    Entities []*EntityAST
}

type RuleDefinitionAST struct {
    Name       string   // ルール名 (例: "is_public")
    Parameters []string // パラメータリスト (例: ["resource", "subject"])
    Body       string   // CEL式本体
}

type EntityAST struct {
    Name        string
    Relations   []*RelationAST
    Attributes  []*AttributeAST
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

// ルール呼び出し (Permify 互換)
type RuleCallPermissionAST struct {
    RuleName  string   // 呼び出すルール名
    Arguments []string // 引数リスト (例: ["resource", "subject"])
}

// 階層的ルール呼び出し (Phase 3 追加)
type HierarchicalRuleCallPermissionAST struct {
    Relation  string   // 走査するリレーション (例: "parent")
    RuleName  string   // 親エンティティ上で呼び出すルール名
    Arguments []string // ルールに渡す引数名
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
- HierarchicalRuleCallRule / HierarchicalRuleCallPermissionAST の変換に対応

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

責務: DB への入出力のみ。DSL パース処理は Service 層で実施。

エラーハンドリング:

- スキーマが存在しない場合: `repositories.ErrNotFound` を wrap して返す
- Service 層で `errors.Is(err, repositories.ErrNotFound)` でチェック

インターフェース定義:

```go
// internal/repositories/schema_repository.go

type SchemaRepository interface {
    // Create creates a new schema version for the tenant
    // Returns the generated version ID (ULID format)
    Create(ctx context.Context, tenantID string, schemaDSL string) (string, error)

    // GetLatestVersion retrieves the latest schema version by tenant ID
    // Returns ErrNotFound if schema does not exist
    // Note: Entities field will be empty (populated by service layer)
    GetLatestVersion(ctx context.Context, tenantID string) (*entities.Schema, error)

    // GetByVersion retrieves a specific schema version
    // Returns ErrNotFound if schema version does not exist
    GetByVersion(ctx context.Context, tenantID string, version string) (*entities.Schema, error)

    // GetByTenant is deprecated, use GetLatestVersion instead
    // Kept for backward compatibility
    GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error)

    // Delete deletes all schema versions for a tenant
    Delete(ctx context.Context, tenantID string) error
}
```

PostgreSQL 実装:

```go
// internal/repositories/postgres/schema_repository.go

type PostgresSchemaRepository struct {
    cluster *database.DBCluster
}

func NewPostgresSchemaRepository(cluster *database.DBCluster) repositories.SchemaRepository {
    return &PostgresSchemaRepository{cluster: cluster}
}
```

センチネルエラー定義:

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
    EntityIDs       []string // Filter by multiple entity IDs (Permify互換)
    Relation        string
    SubjectType     string
    SubjectID       string
    SubjectIDs      []string // Filter by multiple subject IDs (Permify互換)
    SubjectRelation string
}

type RelationRepository interface {
    // 基本CRUD
    Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error
    Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error
    Read(ctx context.Context, tenantID string, filter *RelationFilter) ([]*entities.RelationTuple, error)
    CheckExists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error)
    BatchWrite(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
    BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
    DeleteByFilter(ctx context.Context, tenantID string, filter *RelationFilter) error
    ReadByFilter(ctx context.Context, tenantID string, filter *RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error)

    // Phase 3: 最適化されたクエリメソッド
    Exists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error)
    ExistsWithSubjectRelation(ctx context.Context, tenantID string,
        entityType, entityID, relation, subjectType, subjectID, subjectRelation string) (bool, error)
    FindByEntityWithRelation(ctx context.Context, tenantID string,
        entityType, entityID, relation string) ([]*entities.RelationTuple, error)
    LookupAncestorsViaRelation(ctx context.Context, tenantID string,
        entityType, entityID string, maxDepth int) ([]*entities.RelationTuple, error)
    FindHierarchicalWithSubject(ctx context.Context, tenantID string,
        entityType, entityID, relation, subjectType, subjectID string,
        maxDepth int) (bool, error)

    // Closure Table 管理
    RebuildClosure(ctx context.Context, tenantID string) error

    // ページネーション付きID取得
    GetSortedEntityIDs(ctx context.Context, tenantID string,
        entityType string, cursor string, limit int) ([]string, error)
    GetSortedSubjectIDs(ctx context.Context, tenantID string,
        subjectType string, cursor string, limit int) ([]string, error)

    // 複合Lookup（Closure+UNION最適化）
    LookupAccessibleEntitiesComplex(ctx context.Context, tenantID string,
        entityType string, relations []string, parentRelations []string,
        subjectType string, subjectID string,
        maxDepth int, cursor string, limit int) ([]string, error)
    LookupAccessibleSubjectsComplex(ctx context.Context, tenantID string,
        entityType string, entityID string, relations []string, parentRelations []string,
        subjectType string,
        maxDepth int, cursor string, limit int) ([]string, error)
}
```

PostgreSQL 実装:

```go
// internal/repositories/postgres/relation_repository.go

type PostgresRelationRepository struct {
    cluster                  *database.DBCluster
    closureExcludedRelations map[string]bool
}

func NewPostgresRelationRepository(
    cluster *database.DBCluster,
    closureExcluded map[string]bool,
) repositories.RelationRepository {
    return &PostgresRelationRepository{
        cluster:                  cluster,
        closureExcludedRelations: closureExcluded,
    }
}
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
    cluster *database.DBCluster
}

func NewPostgresAttributeRepository(cluster *database.DBCluster) repositories.AttributeRepository {
    return &PostgresAttributeRepository{cluster: cluster}
}
```

---

### 5. 認可エンジン設計

#### 5.1 ルール評価（Evaluator）

SchemaServiceInterface: 循環依存を回避するため、authorization パッケージで定義

```go
// internal/services/authorization/evaluator.go

// SchemaServiceInterface defines the interface for schema operations
// This interface is defined here to avoid circular dependency
type SchemaServiceInterface interface {
    GetSchemaEntity(ctx context.Context, tenantID string, version string) (*entities.Schema, error)
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
    req *EvaluationRequest,
    schema *entities.Schema,
    rule entities.PermissionRule,
) (bool, error) {
    switch r := rule.(type) {
    case *entities.RelationRule:
        return e.evaluateRelation(ctx, req, r)
    case *entities.LogicalRule:
        return e.evaluateLogical(ctx, req, schema, r)
    case *entities.HierarchicalRule:
        return e.evaluateHierarchical(ctx, req, schema, r)
    case *entities.ABACRule:
        return e.evaluateABAC(ctx, req, r)
    case *entities.RuleCallRule:
        return e.evaluateRuleCall(ctx, req, schema, r)
    case *entities.HierarchicalRuleCallRule:
        return e.evaluateHierarchicalRuleCall(ctx, req, schema, r)
    default:
        return false, fmt.Errorf("unknown rule type")
    }
}
```

Phase 3 での evaluateRelation 最適化:

```go
// evaluateRelation: Exists() + FindByEntityWithRelation() に最適化
// Read() による全件取得ではなく、存在チェックのみを行う
func (e *Evaluator) evaluateRelation(
    ctx context.Context,
    req *EvaluationRequest,
    rule *entities.RelationRule,
) (bool, error) {
    // 1. Contextual tuples をチェック
    // 2. Exists() で直接一致を確認
    // 3. FindByEntityWithRelation() で間接的な関係を探索
}
```

Phase 3 での evaluateHierarchical 最適化:

```go
// evaluateHierarchical: 同一型階層の場合はCTE最適化
// FindHierarchicalWithSubject() を使用して再帰CTEで一括検索
```

#### 5.2 CEL エンジン

```go
// internal/services/authorization/cel.go

type CELEngine struct {
    env *cel.Env
}

func NewCELEngine() (*CELEngine, error) {
    env, err := cel.NewEnv(
        cel.Variable("resource", cel.MapType(cel.StringType, cel.DynType)),
        cel.Variable("subject", cel.MapType(cel.StringType, cel.DynType)),
        cel.Variable("request", cel.MapType(cel.StringType, cel.DynType)),
    )
    if err != nil {
        return nil, err
    }
    return &CELEngine{env: env}, nil
}

// Evaluate evaluates a CEL expression with the given context
func (e *CELEngine) Evaluate(expression string, context *EvaluationContext) (bool, error)

// EvaluateWithParams evaluates a CEL expression with "this" (parent attributes)
// and dynamically named parameters. Used for HierarchicalRuleCallRule evaluation.
func (e *CELEngine) EvaluateWithParams(
    expression string,
    thisAttrs map[string]interface{},
    params map[string]interface{},
) (bool, error)

// ValidateExpression validates a CEL expression without evaluating it
func (e *CELEngine) ValidateExpression(expression string) error
```

サポートする演算子:

- 比較: `==`, `!=`, `>`, `>=`, `<`, `<=`
- コレクション: `in`
- 論理: `&&`, `||`, `!`
- 文字列: `contains`, `startsWith`, `endsWith`, `matches`
- コレクション関数: `size`, `all`, `exists`, `exists_one`, `filter`, `map`

例:

```cel
resource.owner == subject.id
subject.age >= 18
subject.role in ["admin", "editor"]
resource.public == true || resource.owner == subject.id
```

EvaluateWithParams の例（HierarchicalRuleCallRule 用）:

```cel
// rule check_conf(authority) { this.confidentiality_level <= authority }
// "this" は親エンティティの属性、"authority" は現在のエンティティの属性値
this.confidentiality_level <= authority
```

#### 5.3 Check 実装

```go
// internal/services/authorization/checker.go

type CheckerInterface interface {
    Check(ctx context.Context, req *CheckRequest) (*CheckResponse, error)
}

type Checker struct {
    schemaService SchemaServiceInterface
    evaluator     *Evaluator
}

func NewChecker(schemaService SchemaServiceInterface, evaluator *Evaluator) *Checker
```

#### 5.4 Expand 実装

```go
// internal/services/authorization/expander.go

type Expander struct {
    schemaService SchemaServiceInterface
    relationRepo  repositories.RelationRepository
}

func NewExpander(
    schemaService SchemaServiceInterface,
    relationRepo repositories.RelationRepository,
) *Expander
```

#### 5.5 Lookup 実装

```go
// internal/services/authorization/lookup.go

type Lookup struct {
    checker       CheckerInterface
    schemaService SchemaServiceInterface
    relationRepo  repositories.RelationRepository
}

func NewLookup(
    checker CheckerInterface,
    schemaService SchemaServiceInterface,
    relationRepo repositories.RelationRepository,
) *Lookup

// LookupEntity: ABAC/ReBAC 両方に対応
func (l *Lookup) LookupEntity(
    ctx context.Context,
    req *LookupEntityRequest,
) (*LookupEntityResponse, error)

// LookupSubject: ABAC/ReBAC 両方に対応（SubjectRelation対応）
func (l *Lookup) LookupSubject(
    ctx context.Context,
    req *LookupSubjectRequest,
) (*LookupSubjectResponse, error)
```

Phase 3 での Lookup 最適化:

- extractRelationsFromRuleWithContext: スキーマルールを再帰的に展開し、必要なリレーション名と階層リレーション名を抽出
- LookupAccessibleEntitiesComplex/SubjectsComplex: Closure Table を利用した UNION クエリで一括取得
- フォールバック: 複合クエリが使えない場合（ABAC ルール含む等）、バッチ式 GetSortedEntityIDs + Check で実行
- SubjectRelation 対応: LookupSubjectRequest に SubjectRelation フィールドを追加

この設計により、Lookup が ABAC ルールを含む全てのルールタイプに対応できることが明確になります。

---

### 6. gRPC ハンドラー設計

#### 設計原則：3 サービス構成

Permify 互換の 3 つの gRPC サービスとして実装します。
protovalidate による入力バリデーションを UnaryServerInterceptor で実施します。

#### 6.1 Permission Handler

```go
// internal/handlers/permission_handler.go

type PermissionHandler struct {
    checker       CheckerInterface  // キャッシュ付きChecker
    expander      ExpanderInterface
    lookup        LookupInterface
    schemaService SchemaServiceInterface

    pb.UnimplementedPermissionServer
}

// Check: 権限チェック
func (h *PermissionHandler) Check(ctx context.Context, req *pb.PermissionCheckRequest) (*pb.PermissionCheckResponse, error)

// Expand: 権限ツリー展開
func (h *PermissionHandler) Expand(ctx context.Context, req *pb.PermissionExpandRequest) (*pb.PermissionExpandResponse, error)

// LookupEntity: 許可されたエンティティ検索
func (h *PermissionHandler) LookupEntity(ctx context.Context, req *pb.PermissionLookupEntityRequest) (*pb.PermissionLookupEntityResponse, error)

// LookupSubject: 許可されたサブジェクト検索
func (h *PermissionHandler) LookupSubject(ctx context.Context, req *pb.PermissionLookupSubjectRequest) (*pb.PermissionLookupSubjectResponse, error)

// SubjectPermission: サブジェクトの全権限取得
func (h *PermissionHandler) SubjectPermission(ctx context.Context, req *pb.PermissionSubjectPermissionRequest) (*pb.PermissionSubjectPermissionResponse, error)
```

#### 6.2 Data Handler

```go
// internal/handlers/data_handler.go

type DataHandler struct {
    relationRepo   RelationRepository
    attributeRepo  AttributeRepository
    tokenGenerator SnapTokenGenerator  // 書き込み時のSnapToken生成

    pb.UnimplementedDataServer
}

// Write: 関係性・属性の書き込み
func (h *DataHandler) Write(ctx context.Context, req *pb.DataWriteRequest) (*pb.DataWriteResponse, error)

// Delete: 関係性の削除（フィルター対応）
func (h *DataHandler) Delete(ctx context.Context, req *pb.DataDeleteRequest) (*pb.DataDeleteResponse, error)

// Read: 関係性の読み取り
func (h *DataHandler) Read(ctx context.Context, req *pb.DataReadRequest) (*pb.DataReadResponse, error)

// ReadAttributes: 属性の読み取り
func (h *DataHandler) ReadAttributes(ctx context.Context, req *pb.AttributeReadRequest) (*pb.AttributeReadResponse, error)
```

#### 6.3 Schema Handler

```go
// internal/handlers/schema_handler.go

type SchemaHandler struct {
    schemaService SchemaServiceInterface

    pb.UnimplementedSchemaServer
}

// Write: スキーマ書き込み（バージョン自動生成）
func (h *SchemaHandler) Write(ctx context.Context, req *pb.SchemaWriteRequest) (*pb.SchemaWriteResponse, error)

// Read: スキーマ読み取り
func (h *SchemaHandler) Read(ctx context.Context, req *pb.SchemaReadRequest) (*pb.SchemaReadResponse, error)

// ListVersions: バージョン一覧取得
func (h *SchemaHandler) ListVersions(ctx context.Context, req *pb.SchemaListVersionsRequest) (*pb.SchemaListVersionsResponse, error)
```

#### 6.4 ヘルパー関数

```go
// internal/handlers/helpers.go

// protoToRelationTuple: proto Tuple → entities.RelationTuple 変換
func protoToRelationTuple(proto *pb.Tuple) (*entities.RelationTuple, error)

// protoToAttributes: proto Attribute → entities.Attribute 変換
func protoToAttribute(proto *pb.Attribute) (*entities.Attribute, error)

// protoContextToTuples: proto Context → []entities.RelationTuple 変換
func protoContextToTuples(ctx *pb.Context) ([]*entities.RelationTuple, error)

// expandNodeToProto: authorization.ExpandNode → proto Expand 変換
func expandNodeToProto(node *authorization.ExpandNode) *pb.Expand
```

#### 6.5 gRPC バリデーション

```go
// internal/infrastructure/validation/interceptor.go

// UnaryServerInterceptor returns a gRPC unary server interceptor that validates
// incoming requests using protovalidate annotations.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor
```

Proto 制約の例:

```protobuf
message Entity {
  string type = 1 [(buf.validate.field).string.min_len = 1];
  string id = 2   [(buf.validate.field).string.min_len = 1];
}

message Tuple {
  Entity entity = 1   [(buf.validate.field).required = true];
  string relation = 2 [(buf.validate.field).string.min_len = 1];
  Subject subject = 3 [(buf.validate.field).required = true];
}
```

サーバー起動時のインターセプター連結:

```go
grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        metrics.UnaryServerInterceptor(metricsCollector, prometheusExporter),
        validation.UnaryServerInterceptor(),
    ),
)
```

---

## インフラストラクチャ設計

### 7. 設定管理

```go
// internal/infrastructure/config/config.go

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Cache    CacheConfig
}

type ServerConfig struct {
    Host        string
    Port        int
    MetricsPort int // Port for Prometheus metrics HTTP server
}

type DatabaseConfig struct {
    Host                      string
    Port                      int
    User                      string
    Password                  string
    Database                  string
    SSLMode                   string
    ReplicaHost               string // empty means no replica
    ReplicaPort               int    // 0 means same as primary Port
    WriteTrackerWindowSeconds int    // seconds to route reads to primary after a write
    ClosureExcludedRelations  string // comma-separated relation names to exclude from closure updates
}

type CacheConfig struct {
    Enabled        bool
    NumCounters    int64
    MaxMemoryBytes int64
    BufferItems    int64
    Metrics        bool
    TTLMinutes     int
}
```

環境変数一覧:

| 変数名 | デフォルト値 | 説明 |
| --- | --- | --- |
| SERVER_HOST | 0.0.0.0 | サーバーバインドアドレス |
| SERVER_PORT | 50051 | gRPC ポート |
| METRICS_PORT | 9090 | Prometheus メトリクスポート |
| DB_HOST | localhost | Primary DB ホスト |
| DB_PORT | 15432 | Primary DB ポート |
| DB_USER | keruberosu | DB ユーザー |
| DB_PASSWORD | (必須) | DB パスワード |
| DB_NAME | keruberosu_dev | DB 名 |
| DB_SSLMODE | disable | SSL モード |
| DB_REPLICA_HOST | (空) | Read Replica ホスト |
| DB_REPLICA_PORT | 0 | Read Replica ポート（0 の場合 Primary と同じ） |
| WRITE_TRACKER_WINDOW_SECONDS | 1 | 書き込み後に Primary にルーティングする秒数 |
| CLOSURE_EXCLUDED_RELATIONS | (空) | Closure 更新から除外するリレーション名（カンマ区切り） |
| CACHE_ENABLED | true | キャッシュ有効化 |
| CACHE_TTL_MINUTES | 5 | キャッシュ TTL（分） |

### 8. DB 基盤

#### 8.1 DBCluster

```go
// internal/infrastructure/database/cluster.go

type DBCluster struct {
    primary      *ResilientDB
    replica      *ResilientDB // nil if no replica configured
    writeTracker *WriteTracker
}

func NewDBCluster(cfg *config.DatabaseConfig) (*DBCluster, error)

// Writer returns the primary database for write operations.
func (c *DBCluster) Writer() DBTX

// ReaderFor returns the appropriate database for read operations.
// Returns primary if no replica is configured or if the tenant had a recent write.
func (c *DBCluster) ReaderFor(tenantID string) DBTX

// RecordWrite records a write for the given tenant.
func (c *DBCluster) RecordWrite(tenantID string)

// PrimaryDB returns the underlying *sql.DB of the primary.
func (c *DBCluster) PrimaryDB() *sql.DB

func (c *DBCluster) Start()       // WriteTracker cleanup開始
func (c *DBCluster) Stop()        // WriteTracker cleanup停止
func (c *DBCluster) Close() error // 全接続クローズ
func (c *DBCluster) HealthCheck() error
```

#### 8.2 DBTX interface

```go
// internal/infrastructure/database/dbtx.go

// DBTX is an interface that abstracts *sql.DB so that ResilientDB can wrap it.
type DBTX interface {
    ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
    BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
    PingContext(ctx context.Context) error
}
```

#### 8.3 ResilientDB

```go
// internal/infrastructure/database/resilient_db.go

type RetryConfig struct {
    MaxRetries int           // デフォルト: 3
    BaseDelay  time.Duration // デフォルト: 100ms
    MaxDelay   time.Duration // デフォルト: 2s
}

type ResilientDB struct {
    db     *sql.DB
    config RetryConfig
}

func NewResilientDB(db *sql.DB, cfg RetryConfig) *ResilientDB
```

リトライ対象のトランジェントエラー:

- `driver.ErrBadConn`: 不正な接続
- PostgreSQL エラーコード `08*`: 接続例外
- PostgreSQL エラーコード `53*`: リソース不足
- PostgreSQL エラーコード `57P01`: 管理シャットダウン
- `connection refused`, `connection reset`, `broken pipe`

#### 8.4 WriteTracker

```go
// internal/infrastructure/database/write_tracker.go

type WriteTracker struct {
    writes map[string]time.Time // tenantID → last write time
    window time.Duration        // tracking window (default: 1s)
}

func NewWriteTracker(windowSeconds int) *WriteTracker
func (w *WriteTracker) RecordWrite(tenantID string)
func (w *WriteTracker) HasRecentWrite(tenantID string) bool
func (w *WriteTracker) Start() // background cleanup goroutine
func (w *WriteTracker) Stop()
```

---

## 依存ライブラリ

### 必須ライブラリ

```go
// go.mod

module github.com/asakaida/keruberosu

go 1.25.1

require (
    github.com/golang-migrate/migrate/v4 v4.19.0  // マイグレーションツール
    github.com/google/cel-go v0.27.0               // ABAC 評価エンジン
    github.com/lib/pq v1.10.9                      // PostgreSQL ドライバー
    github.com/oklog/ulid/v2 v2.1.1                // ULID 生成
    github.com/prometheus/client_golang v1.18.0     // Prometheus メトリクス
    github.com/spf13/cobra v1.10.1                  // CLI フレームワーク
    github.com/spf13/viper v1.21.0                  // 設定管理
    google.golang.org/grpc v1.79.3                  // gRPC フレームワーク
    google.golang.org/protobuf v1.36.11             // Protocol Buffers
)

require (
    buf.build/go/protovalidate v1.1.3               // Proto バリデーション
    // その他の間接依存...
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

## Admin CLI

### rebuild-closures コマンド

```bash
# 全テナントの Closure Table を再構築
go run cmd/admin/main.go rebuild-closures --env dev
```

- 全テナントの schemas テーブルからテナント ID を列挙
- 各テナントに対して RebuildClosure() を実行
- 実行前後の closure 件数を表示

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
codes.InvalidArgument   // リクエストパラメータ不正（protovalidateで自動検出）
codes.NotFound          // リソースが存在しない
codes.AlreadyExists     // リソースが既に存在
codes.PermissionDenied  // 権限不足
codes.Internal          // 内部エラー
codes.Unavailable       // サービス利用不可
```

---

## パフォーマンス考慮事項

### 実装済みの最適化

- LRU + TTL キャッシュ: CheckerWithCache による透過的キャッシュ（Phase 2）
- Closure Table: O(1) 祖先検索（Phase 2）
- DBCluster: Primary + Read Replica による読み書き分離（Phase 3）
- ResilientDB: トランジェントエラーの自動リトライ（Phase 3）
- WriteTracker: レプリカ整合性のための書き込み追跡（Phase 3）
- evaluateRelation 最適化: Exists() + FindByEntityWithRelation() による効率的なクエリ（Phase 3）
- evaluateHierarchical 最適化: 同一型階層 CTE による一括検索（Phase 3）
- LookupAccessibleEntitiesComplex/SubjectsComplex: Closure+UNION 最適化クエリ（Phase 3）
- Closure 除外設定: 不要なリレーションの closure 更新をスキップ（Phase 3）

### 将来的な最適化候補

- JSONB 型への移行
- 接続プーリング改善
- 並列処理最適化
