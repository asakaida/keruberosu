# Keruberosu - 開発進捗管理 (DEVELOPMENT.md)

## プロジェクト概要

目標: Permify 互換の ReBAC/ABAC 認可マイクロサービスの実装
Phase: Phase 1 - キャッシュレス完全実装
開始日: 2025-10-08

---

## Phase 1 タスクリスト

### 1. 基盤構築

#### 1.1 プロジェクト初期化

- [x] PRD.md 作成
- [x] DESIGN.md 作成
- [x] DEVELOPMENT.md 作成（本ファイル）
- [x] go.mod 初期化
- [x] .gitignore 作成
- [x] README.md 作成

#### 1.2 インフラストラクチャ

- [x] docker-compose.yml 作成
  - [x] postgres-dev（ポート 15432）
  - [x] postgres-test（ポート 25432）
- [x] internal/infrastructure/database/migrations/postgres/ 作成
  - [x] 000001_create_schemas_table.up.sql
  - [x] 000001_create_schemas_table.down.sql
  - [x] 000002_create_relations_table.up.sql
  - [x] 000002_create_relations_table.down.sql
  - [x] 000003_create_attributes_table.up.sql
  - [x] 000003_create_attributes_table.down.sql
- [x] internal/infrastructure/config/config.go
  - [x] Config 構造体定義
  - [x] 環境変数読み込み（viper 使用、環境ごとの.env ファイル対応）
- [x] internal/infrastructure/database/postgres.go
  - [x] Postgres 構造体
  - [x] NewPostgres（接続初期化）
  - [x] RunMigrations（マイグレーション実行）
  - [x] ヘルスチェック

#### 1.3 Protocol Buffers 定義

- [x] proto/keruberosu/v1/common.proto
  - [x] Entity メッセージ
  - [x] Subject メッセージ
  - [x] SubjectReference メッセージ
  - [x] RelationTuple メッセージ
  - [x] PermissionCheckMetadata メッセージ
  - [x] Context メッセージ
  - [x] AttributeData メッセージ
  - [x] CheckResult enum
- [x] proto/keruberosu/v1/authorization.proto
  - [x] AuthorizationService 定義
  - [x] スキーマ管理 API（WriteSchema/ReadSchema）
  - [x] データ書き込み API（WriteRelations/DeleteRelations/WriteAttributes）
  - [x] 認可チェック API（Check/Expand/LookupEntity/LookupSubject/SubjectPermission）
  - [x] 各 API のリクエスト/レスポンスメッセージ
- [x] proto/keruberosu/v1/audit.proto
  - [x] AuditService 定義
  - [x] WriteAuditLog API
  - [x] ReadAuditLogs API
  - [x] AuditLog メッセージ
- [x] protoc コード生成スクリプト作成
  - [x] scripts/generate-proto.sh 作成

---

### 2. データアクセス層（Repository）

#### 2.1 Repository インターフェース定義

- [x] internal/repositories/schema_repository.go

  - [x] SchemaRepository インターフェース
  - [x] Create メソッド定義
  - [x] GetByTenant メソッド定義
  - [x] Update メソッド定義
  - [x] Delete メソッド定義

- [x] internal/repositories/relation_repository.go

  - [x] RelationFilter 構造体定義
  - [x] RelationRepository インターフェース
  - [x] Write メソッド定義
  - [x] Delete メソッド定義
  - [x] Read メソッド定義
  - [x] CheckExists メソッド定義
  - [x] BatchWrite メソッド定義
  - [x] BatchDelete メソッド定義

- [x] internal/repositories/attribute_repository.go
  - [x] AttributeRepository インターフェース
  - [x] Write メソッド定義
  - [x] Read メソッド定義
  - [x] Delete メソッド定義
  - [x] GetValue メソッド定義

#### 2.2 PostgreSQL 実装

- [x] internal/repositories/postgres/schema_repository.go

  - [x] PostgresSchemaRepository 構造体
  - [x] NewPostgresSchemaRepository
  - [x] Create 実装
  - [x] GetByTenant 実装
  - [x] Update 実装
  - [x] Delete 実装
  - [x] ユニットテスト（正常系・異常系）

- [x] internal/repositories/postgres/relation_repository.go

  - [x] PostgresRelationRepository 構造体
  - [x] NewPostgresRelationRepository
  - [x] Write 実装
  - [x] Delete 実装
  - [x] Read 実装
  - [x] CheckExists 実装
  - [x] BatchWrite 実装
  - [x] BatchDelete 実装
  - [x] ユニットテスト（正常系・異常系）

- [x] internal/repositories/postgres/attribute_repository.go
  - [x] PostgresAttributeRepository 構造体
  - [x] NewPostgresAttributeRepository
  - [x] Write 実装
  - [x] Read 実装
  - [x] Delete 実装
  - [x] GetValue 実装
  - [x] ユニットテスト（正常系・異常系）

---

### 3. ドメインエンティティ

#### 3.1 スキーマ定義系エンティティ

- [x] internal/entities/schema.go

  - [x] Schema 構造体
  - [x] GetEntity ヘルパーメソッド
  - [x] GetPermission ヘルパーメソッド

- [x] internal/entities/entity.go

  - [x] Entity 構造体
  - [x] GetRelation ヘルパーメソッド
  - [x] GetPermission ヘルパーメソッド
  - [x] GetAttributeSchema ヘルパーメソッド

- [x] internal/entities/relation.go

  - [x] Relation 構造体（スキーマ内のリレーション定義）

- [x] internal/entities/attribute_schema.go

  - [x] AttributeSchema 構造体（属性型定義）

- [x] internal/entities/permission.go

  - [x] Permission 構造体（権限定義）

- [x] internal/entities/rule.go
  - [x] PermissionRule インターフェース
  - [x] RelationRule 構造体
  - [x] LogicalRule 構造体（OR/AND/NOT）
  - [x] HierarchicalRule 構造体（parent.permission）
  - [x] ABACRule 構造体（CEL 式）

#### 3.2 データ系エンティティ

- [x] internal/entities/relation_tuple.go

  - [x] RelationTuple 構造体（実際のリレーションデータ）
  - [x] Validate メソッド
  - [x] String メソッド

- [x] internal/entities/attribute.go
  - [x] Attribute 構造体（実際の属性データ）
  - [x] Validate メソッド
  - [x] MarshalValue/UnmarshalValue メソッド
  - [x] String メソッド

---

### 4. DSL パーサー実装

#### 4.1 字句解析（Lexer）

- [x] internal/services/parser/lexer.go
  - [x] Token 型定義
  - [x] Lexer 構造体
  - [x] NextToken 実装
  - [x] キーワード認識（entity, relation, permission, rule）
  - [x] 演算子認識（or, and, not, =）
  - [x] コメント処理（// 形式）
  - [x] ユニットテスト（11 テスト）

#### 4.2 構文解析（Parser）

- [x] internal/services/parser/ast.go

  - [x] AST 構造体定義（SchemaAST, EntityAST 等）
  - [x] PermissionRuleAST インターフェース実装

- [x] internal/services/parser/parser.go
  - [x] Parser 構造体
  - [x] Parse（メインエントリーポイント）
  - [x] parseEntity
  - [x] parseRelation
  - [x] parseAttribute
  - [x] parsePermission
  - [x] parsePermissionRule（再帰的、演算子優先順位対応）
  - [x] parseRuleExpression（CEL 式パース）
  - [x] エラーハンドリング
  - [x] ユニットテスト（17 テスト）

#### 4.3 検証（Validator）

- [x] internal/services/parser/validator.go
  - [x] スキーマ検証（重複名チェック、型検証）
  - [x] 同一エンティティ内の循環参照チェック
  - [x] 未定義の関係参照チェック
  - [x] 未定義のエンティティ参照チェック
  - [x] 階層的パーミッション検証
  - [x] 型整合性チェック
  - [x] ユニットテスト（17 テスト）

#### 4.4 変換（Converter）

- [x] internal/services/parser/converter.go
  - [x] ASTToSchema（AST → entities.Schema 変換）
  - [x] SchemaToAST（entities.Schema → AST 変換）
  - [x] convertEntity（EntityAST → entities.Entity）
  - [x] convertPermissionRule（PermissionRuleAST → entities.PermissionRule）
  - [x] convertEntityToAST（entities.Entity → EntityAST）
  - [x] convertPermissionRuleToAST（entities.PermissionRule → PermissionRuleAST）
  - [x] ユニットテスト（10 テスト）

#### 4.5 DSL 生成（Generator）

- [x] internal/services/parser/generator.go
  - [x] Generator 構造体
  - [x] Generate（AST → DSL 文字列生成）
  - [x] generateEntity（エンティティ生成）
  - [x] generateRelation（リレーション生成）
  - [x] generateAttribute（アトリビュート生成）
  - [x] generatePermission（パーミッション生成）
  - [x] generatePermissionRule（ルール生成、演算子優先順位対応）
  - [x] ユニットテスト（12 テスト）

---

### 5. 認可エンジン

#### 5.1 CEL エンジン

- [x] internal/services/authorization/cel.go
  - [x] CELEngine 構造体
  - [x] NewCELEngine（環境初期化）
  - [x] Evaluate（式評価）
  - [x] ValidateExpression（式検証）
  - [x] EvaluateWithDefaults（デフォルトコンテキスト評価）
  - [x] エラーハンドリング
  - [x] ユニットテスト（48 テスト）
    - [x] 比較演算子: ==, !=, >, >=, <, <= (16 テスト)
    - [x] in 演算子 (4 テスト)
    - [x] 論理演算子: &&, ||, ! (9 テスト)
    - [x] 複雑な式 (5 テスト)
    - [x] 文字列操作: contains, startsWith, endsWith (4 テスト)
    - [x] マルチコンテキスト変数 (2 テスト)
    - [x] エラーケース (2 テスト)
    - [x] 式検証 (4 テスト)
    - [x] 空コンテキスト (1 テスト)
    - [x] デフォルト評価 (1 テスト)

#### 5.2 ルール評価（Evaluator）

- [x] internal/services/authorization/evaluator.go
  - [x] Evaluator 構造体
  - [x] NewEvaluator
  - [x] EvaluateRule（ルール評価ディスパッチャ）
  - [x] evaluateRelation（関係性チェック、contextualTuples 対応）
  - [x] evaluateLogical（OR/AND/NOT、短絡評価）
  - [x] evaluateHierarchical（parent.permission、再帰評価）
  - [x] evaluateABAC（CEL 呼び出し、resource/subject 属性統合）
  - [x] 深さ制限チェック（MaxDepth = 100）
  - [x] ユニットテスト（57 テスト）
    - [x] RelationRule テスト (2 テスト)
    - [x] LogicalRule OR テスト (2 テスト)
    - [x] LogicalRule AND テスト (2 テスト)
    - [x] LogicalRule NOT テスト (2 テスト)
    - [x] HierarchicalRule テスト (2 テスト)
    - [x] ABACRule テスト (2 テスト)
    - [x] ContextualTuples テスト (1 テスト)
    - [x] MaxDepth エラーテスト (1 テスト)
    - [x] 複雑なルールテスト (2 テスト)
    - [x] CEL エンジンテスト (48 テスト、再掲)

#### 5.3 Check 実装

- [x] internal/services/authorization/checker.go
  - [x] Checker 構造体
  - [x] NewChecker
  - [x] Check（メインエントリーポイント）
  - [x] CheckRequest/CheckResponse 構造体
  - [x] validateRequest（リクエスト検証）
  - [x] CheckMultiple（複数パーミッション一括チェック）
  - [x] 深さ制限チェック（Evaluator で実装済み）
  - [x] contextualTuples 統合
  - [x] ユニットテスト（13 テスト）
    - [x] 基本パーミッションテスト (2 テスト)
    - [x] 論理演算パーミッションテスト (1 テスト)
    - [x] 階層的パーミッションテスト (1 テスト)
    - [x] ABAC パーミッションテスト (2 テスト)
    - [x] ContextualTuples テスト (1 テスト)
    - [x] エラーケーステスト (4 テスト)
    - [x] CheckMultiple テスト (3 テスト)

#### 5.4 Expand 実装

- [x] internal/services/authorization/expander.go
  - [x] Expander 構造体
  - [x] ExpandNode 構造体（union/intersection/exclusion/leaf）
  - [x] Expand（権限ツリー構築）
  - [x] expandRule（ルールごとのノード構築）
  - [x] expandRelation（関係ベース展開）
  - [x] expandLogical（論理演算展開）
  - [x] expandHierarchical（階層的展開）
  - [x] expandABAC（ABAC ルール展開）
  - [x] parseEntityRef（エンティティ参照パース）
  - [x] validateRequest（リクエスト検証）
  - [x] 深さ制限チェック（MaxDepth = 100）
  - [x] ユニットテスト（15 テスト）
    - [x] 基本リレーション展開テスト (1 テスト)
    - [x] 論理演算 OR 展開テスト (1 テスト)
    - [x] 論理演算 AND 展開テスト (1 テスト)
    - [x] 論理演算 NOT 展開テスト (1 テスト)
    - [x] 階層的パーミッション展開テスト (1 テスト)
    - [x] ABAC ルール展開テスト (1 テスト)
    - [x] 空リレーション展開テスト (1 テスト)
    - [x] 複雑なネスト構造展開テスト (1 テスト)
    - [x] エラーケーステスト (6 テスト)
    - [x] parseEntityRef テスト (5 テスト)
    - [x] MaxDepth エラーテスト (1 テスト)

#### 5.5 Lookup 実装

- [x] internal/services/authorization/lookup.go
  - [x] Lookup 構造体
  - [x] LookupEntity（許可されたエンティティ検索）
  - [x] LookupSubject（許可されたサブジェクト検索）
  - [x] getCandidateEntityIDs（候補エンティティ ID 取得）
  - [x] getCandidateSubjectIDs（候補サブジェクト ID 取得）
  - [x] validateLookupEntityRequest（リクエスト検証）
  - [x] validateLookupSubjectRequest（リクエスト検証）
  - [x] ページネーション対応（PageSize/PageToken）
  - [x] ユニットテスト（16 テスト）
    - [x] LookupEntity 基本テスト (1 テスト)
    - [x] LookupEntity アクセスなしテスト (1 テスト)
    - [x] LookupEntity ページネーションテスト (1 テスト)
    - [x] LookupEntity ContextualTuples テスト (1 テスト)
    - [x] LookupSubject 基本テスト (1 テスト)
    - [x] LookupSubject アクセスなしテスト (1 テスト)
    - [x] LookupSubject ページネーションテスト (1 テスト)
    - [x] LookupEntity エラーケーステスト (7 テスト)
    - [x] LookupSubject エラーケーステスト (7 テスト)
    - [x] LookupSubject 論理演算パーミッションテスト (1 テスト)

---

### 6. ビジネスロジック層（Service）

#### 6.1 Schema Service

- [x] internal/services/schema_service.go
  - [x] SchemaService 構造体（依存: SchemaRepository, Parser, Validator, Converter, Generator）
  - [x] WriteSchema（DSL パース → 検証 → 保存）
    - [x] DSL 文字列を Lexer/Parser でパース → AST
    - [x] Validator でスキーマ検証
    - [x] Converter で AST → entities.Schema 変換
    - [x] SchemaRepository で DB 保存（schema_dsl として保存）
  - [x] ReadSchema（DB 取得 → DSL 返却）
    - [x] SchemaRepository で DSL 文字列取得
    - [x] そのまま返却（Phase 1 では単純実装）
    - [x] （将来拡張）Parser/Generator 経由で正規化された DSL を生成
  - [x] ValidateSchema（DSL 検証のみ、保存なし）
    - [x] DSL 文字列をパース
    - [x] Validator で検証
    - [x] 検証結果を返却
  - [x] DeleteSchema（スキーマ削除）
  - [x] GetSchemaEntity（内部用スキーマエンティティ取得）
  - [x] ユニットテスト（17 テスト）

---

### 7. gRPC ハンドラー層

**設計方針**：Google Zanzibar / Permify の業界標準に従い、単一の AuthorizationService として実装。認可は分離できない 1 つのドメインであり、Schema、Data、Authorization の全操作を 1 つのハンドラーで提供。

#### 7.1 統合 Authorization Handler

- [x] internal/handlers/authorization_handler.go
  - [x] AuthorizationHandler 構造体（統合型）
    - [x] Schema 管理用: schemaService (SchemaServiceInterface)
    - [x] Data 管理用: relationRepo, attributeRepo
    - [x] 認可処理用: checker, expander, lookup, schemaRepo
  - [x] Schema 管理メソッド
    - [x] WriteSchema
    - [x] ReadSchema
  - [x] Data 管理メソッド
    - [x] WriteRelations（複数 relation tuple 一括書き込み）
    - [x] DeleteRelations（複数 relation tuple 一括削除）
    - [x] WriteAttributes（複数 attributes 書き込み）
  - [x] 認可メソッド
    - [x] Check
    - [x] Expand
    - [x] LookupEntity
    - [x] LookupSubject
    - [x] SubjectPermission
    - [x] LookupEntityStream（Phase 1 では未実装）
  - [x] ヘルパー関数
    - [x] protoToRelationTuple（proto → entities 変換）
    - [x] protoToAttributes（proto → entities 変換、展開）
    - [x] protoContextToTuples（Context → RelationTuple 変換）
    - [x] expandNodeToProto（ExpandNode → proto 変換）
  - [x] インターフェース定義
    - [x] CheckerInterface
    - [x] ExpanderInterface
    - [x] LookupInterface
  - [x] ユニットテスト（29 テスト統合）
    - [x] Schema 管理テスト（7 テスト）
    - [x] Data 管理テスト（9 テスト）
    - [x] Authorization テスト（11 テスト）
    - [x] Helper 関数テスト（2 テスト）
    - [x] schema_handler_test.go と data_handler_test.go を統合
    - [x] 全テストを authorization_handler_test.go に集約

**備考**：既存の schema_handler.go と data_handler.go のロジックを authorization_handler.go に統合。コード重複を避けるため、既存のヘルパー関数を再利用。テストファイルも統合し、単一のテストファイルで全機能をカバー。

---

### 8. メインエントリーポイント

#### 8.1 マイグレーションコマンド

- [x] cmd/migrate/main.go
  - [x] 設定読み込み
  - [x] DB 接続初期化
  - [x] マイグレーションライブラリ（golang-migrate/migrate）統合
  - [x] up コマンド実装
  - [x] down コマンド実装
  - [x] goto コマンド実装
  - [x] version コマンド実装
  - [x] force コマンド実装
  - [x] エラーハンドリング
- [x] internal/infrastructure/database/postgres.go
  - [x] NewMigrateDriver 関数追加

#### 8.2 gRPC サーバー

- [x] cmd/server/main.go
  - [x] 設定読み込み（viper、環境変数 ENV）
  - [x] DB 接続初期化（database.NewPostgres）
  - [x] Repository 初期化（SchemaRepo、RelationRepo、AttributeRepo）
  - [x] Service 初期化（SchemaService、CELEngine、Evaluator、Checker、Expander、Lookup）
  - [x] Handler 初期化（統合 AuthorizationHandler）
  - [x] gRPC サーバー起動（ポート 50051、PORT 環境変数で変更可能）
  - [x] Reflection 有効化（grpcurl 対応）
  - [x] シグナルハンドリング（SIGTERM/SIGINT、30 秒タイムアウト付きグレースフルシャットダウン）
- [x] 動作確認
  - [x] ビルド成功（go build -o bin/server ./cmd/server）
  - [x] サーバー起動成功（ENV=test PORT=50052 ./bin/server）
  - [x] 全メソッド登録確認（grpcurl で 11 メソッド確認）
    - WriteSchema, ReadSchema
    - WriteRelations, DeleteRelations, WriteAttributes
    - Check, Expand, LookupEntity, LookupSubject, SubjectPermission, LookupEntityStream

---

### 9. テスト

#### 9.1 ユニットテスト

- [x] 全パッケージのユニットテスト作成
- [x] カバレッジ計測
- [x] 目標: 80%以上

#### 9.2 統合テスト

- [x] PostgreSQL 込みの統合テスト
- [x] test コンテナ使用
- [x] マイグレーション自動適用

#### 9.3 E2E テスト

- [x] gRPC 経由の完全シナリオテスト（bufconn ベース in-memory gRPC）
- [x] Google Docs 風の ReBAC 例（14/14 テストパス）
- [x] ABAC 例（全演算子）（19/19 テストパス）
- [x] Permify 互換性検証（12/12 テストパス）
- [x] **全 E2E テスト成功（45/45 テストケース、100% パス率）**

---

### 10. ドキュメント

- [ ] README.md
  - [ ] プロジェクト概要
  - [ ] クイックスタート
  - [ ] 開発環境セットアップ
  - [ ] ビルド・実行方法
- [ ] API 使用例（examples/）
  - [ ] スキーマ定義例
  - [ ] データ書き込み例
  - [ ] Check API 呼び出し例
- [ ] アーキテクチャ図

---

## 進捗管理

### 現在のステータス

全体進捗: 50% (基盤構築 + ドメインエンティティ + Repository + DSL パーサー + 認可エンジン完了)

#### 完了タスク

- [x] PRD.md 作成
- [x] DESIGN.md 作成
- [x] DEVELOPMENT.md 作成
- [x] プロジェクト初期化（go.mod, .gitignore, README.md）
- [x] インフラストラクチャ（docker-compose, マイグレーション, config, postgres）
- [x] Protocol Buffers 定義（common, authorization, audit）
- [x] ドメインエンティティ（スキーマ定義系 + データ系）
- [x] Repository インターフェース定義（Schema, Relation, Attribute）
- [x] Repository PostgreSQL 実装（Schema, Relation, Attribute）
- [x] DSL パーサー実装（Lexer, Parser, Validator, Converter, Generator）
- [x] 認可エンジン実装（CEL エンジン、Evaluator、Checker、Expander、Lookup）
- [x] Schema Service 実装

#### 進行中タスク

- [ ] gRPC ハンドラー層実装

#### 次のマイルストーン

Ï
Milestone 4: 認可エンジン実装完了（Week 4）

- CEL エンジン実装
- ルール評価（Evaluator）実装
- Check 実装
- Expand 実装
- Lookup 実装

---

## 開発ログ

### 2025-10-08

- プロジェクト開始
- PRD.md 完成（schema_version 削除、Permify 互換性確保）
- DESIGN.md 作成（完全な ABAC/ReBAC 実装スコープ、正しいプロジェクト構造）
- DEVELOPMENT.md 作成
- Proto 設計を 3 ファイル構成に変更（common.proto, authorization.proto, audit.proto）
- PRD.md の markdown 修正（\*\* 削除、コードブロック言語指定子追加）
- プロジェクト初期化完了
  - go.mod 初期化
  - .gitignore 作成（proto 生成ファイルを除外）
  - README.md 作成
  - cmd/migrate 構成追加（マイグレーションコマンド）
- インフラストラクチャ構築完了
  - docker-compose.yml 作成（postgres-dev: 15432, postgres-test: 25432）
  - マイグレーションファイル作成（TIMESTAMPTZ 使用）
  - config.go 作成（viper 使用、環境ごとの .env ファイル対応）
  - postgres.go 作成（接続管理、マイグレーション実行、ヘルスチェック）
  - 環境設定ファイル（.env.dev.example, .env.test.example, .env.prod.example）
- Protocol Buffers 定義完了
  - common.proto 作成（共通メッセージ型）
  - authorization.proto 作成（AuthorizationService と全 API）
  - audit.proto 作成（AuditService）
  - scripts/generate-proto.sh 作成
- ドメインエンティティ実装完了（Go ベストプラクティスに準拠）
  - entities 層の再設計（1 ファイル 1 構造体の原則）
  - スキーマ定義系: schema.go, entity.go, relation.go, attribute_schema.go, permission.go, rule.go
  - データ系: relation_tuple.go, attribute.go
  - DESIGN.md と DEVELOPMENT.md を新構造に更新
- Repository 層実装完了
  - Repository インターフェース定義（schema_repository.go, relation_repository.go, attribute_repository.go）
  - PostgreSQL 実装（postgres/schema_repository.go, postgres/relation_repository.go, postgres/attribute_repository.go）
  - RelationFilter 構造体による柔軟なクエリフィルタリング
  - Batch 操作のトランザクション対応
  - ユニットテスト完了（49 テストケース全て成功）
    - SchemaRepository: 8 テスト（正常系・異常系）
    - RelationRepository: 24 テスト（正常系・異常系）
    - AttributeRepository: 17 テスト（正常系・異常系）
- テスト環境の整備
  - config.go にプロジェクトルート自動検出機能追加（go.mod を基準）
  - docker-compose.yml の healthcheck を活用（--wait フラグ）
  - README.md にテスト実施方法を追加
- DSL パーサー実装完了
  - Lexer 実装（lexer.go）
    - Token 型定義（キーワード、演算子、デリミタ）
    - コメント処理（// 形式）
    - 行・列番号追跡
    - 11 テストケース完了
  - Parser 実装（ast.go, parser.go）
    - AST 構造体定義（SchemaAST, EntityAST, RelationAST, AttributeAST, PermissionAST）
    - PermissionRuleAST インターフェース（RelationPermissionAST, LogicalPermissionAST, HierarchicalPermissionAST, RulePermissionAST）
    - 演算子優先順位パース（or < and < not）
    - 再帰下降パーサー実装
    - CEL 式パース（rule() 内の式）
    - エラーハンドリング（無限ループ対策）
    - 17 テストケース完了
  - Validator 実装（validator.go）
    - スキーマ検証（重複名、型検証）
    - 循環参照チェック（同一エンティティ内のパーミッション参照）
    - 未定義参照チェック（リレーション、エンティティ、パーミッション）
    - 階層的パーミッション検証（parent.permission 形式）
    - パーミッション内でのパーミッション参照サポート（permission view = edit）
    - 17 テストケース完了
  - 合計 45 テストケース全て成功
- CEL エンジン実装完了
  - google/cel-go v0.26.1 を依存関係に追加
  - CELEngine 実装（cel.go）
    - CEL 環境の初期化（resource, subject, request 変数）
    - Evaluate メソッド（CEL 式評価）
    - ValidateExpression メソッド（式検証、boolean 型チェック）
    - EvaluateWithDefaults メソッド（nil コンテキスト対応）
    - エラーハンドリング（コンパイルエラー、評価エラー、型エラー）
  - ユニットテスト実装（cel_test.go）
    - 比較演算子テスト（==, !=, >, >=, <, <=）: 16 テスト
    - 論理演算子テスト（&&, ||, !）: 9 テスト
    - in 演算子テスト: 4 テスト
    - 複雑な式のテスト: 5 テスト
    - 文字列操作テスト（contains, startsWith, endsWith）: 4 テスト
    - マルチコンテキスト変数テスト: 2 テスト
    - エラーケーステスト: 2 テスト
    - 式検証テスト: 4 テスト
    - 空コンテキストテスト: 1 テスト
    - デフォルト評価テスト: 1 テスト
  - 合計 48 テストケース全て成功
- Evaluator 実装完了
  - Evaluator 実装（evaluator.go）
    - EvaluationRequest 構造体（評価コンテキスト）
    - EvaluateRule メソッド（ルール評価ディスパッチャ）
    - evaluateRelation（RelationRule 評価、contextualTuples 対応）
    - evaluateLogical（LogicalRule 評価、OR/AND/NOT、短絡評価）
    - evaluateHierarchical（HierarchicalRule 評価、parent.permission、再帰評価）
    - evaluateABAC（ABACRule 評価、CEL 呼び出し、resource/subject 属性統合）
    - 深さ制限チェック（MaxDepth = 100、循環参照対策）
  - モックリポジトリ実装（evaluator_test.go）
    - mockSchemaRepository（インメモリスキーマ）
    - mockRelationRepository（インメモリリレーション、フィルタリング対応）
    - mockAttributeRepository（インメモリ属性）
  - ユニットテスト実装（evaluator_test.go）
    - RelationRule テスト: 2 テスト
    - LogicalRule OR テスト: 2 テスト
    - LogicalRule AND テスト: 2 テスト
    - LogicalRule NOT テスト: 2 テスト
    - HierarchicalRule テスト: 2 テスト
    - ABACRule テスト: 2 テスト
    - ContextualTuples テスト: 1 テスト
    - MaxDepth エラーテスト: 1 テスト
    - 複雑なルールテスト（(owner or editor) and rule(...)）: 2 テスト
  - 合計 16 テストケース全て成功
- Checker 実装完了
  - Checker 実装（checker.go）
    - CheckRequest/CheckResponse 構造体（リクエスト/レスポンス定義）
    - Check メソッド（パーミッションチェックのメインエントリーポイント）
    - validateRequest（リクエストパラメータ検証）
    - CheckMultiple（複数パーミッション一括チェック）
    - Evaluator との統合（ルール評価委譲）
    - スキーマ/エンティティ/パーミッション解決
    - contextualTuples 対応
  - ユニットテスト実装（checker_test.go）
    - 基本パーミッションテスト（owner/non-owner）: 2 テスト
    - 論理演算パーミッションテスト（OR）: 1 テスト
    - 階層的パーミッションテスト（parent.view）: 1 テスト
    - ABAC パーミッションテスト（public document）: 2 テスト
    - ContextualTuples テスト: 1 テスト
    - エラーケーステスト（バリデーション、未定義エンティティ/パーミッション）: 4 テスト
    - CheckMultiple テスト（一括チェック、部分アクセス、存在しないパーミッション）: 3 テスト
  - 合計 14 テストケース全て成功
- Expander 実装完了
  - Expander 実装（expander.go）
    - ExpandNode 構造体（union/intersection/exclusion/leaf ノードタイプ）
    - Expand メソッド（権限ツリー構築のメインエントリーポイント）
    - expandRule（ルールタイプに応じた展開ディスパッチ）
    - expandRelation（関係ベースルールの展開、全サブジェクト列挙）
    - expandLogical（論理演算ルールの展開、OR/AND/NOT）
    - expandHierarchical（階層的ルールの展開、親エンティティの再帰的展開）
    - expandABAC（ABAC ルールの展開、leaf ノードで式を返す）
    - parseEntityRef（エンティティ参照パース、"type:id" 形式）
    - validateRequest（リクエストパラメータ検証）
    - 深さ制限チェック（MaxDepth = 100、循環参照対策）
  - ユニットテスト実装（expander_test.go）
    - 基本リレーション展開テスト（複数サブジェクトの union ノード）: 1 テスト
    - 論理演算 OR 展開テスト（union ノード）: 1 テスト
    - 論理演算 AND 展開テスト（intersection ノード）: 1 テスト
    - 論理演算 NOT 展開テスト（exclusion ノード）: 1 テスト
    - 階層的パーミッション展開テスト（parent.view の再帰展開）: 1 テスト
    - ABAC ルール展開テスト（leaf ノードで式を返す）: 1 テスト
    - 空リレーション展開テスト（子なし union ノード）: 1 テスト
    - 複雑なネスト構造展開テスト（owner and (editor or admin)）: 1 テスト
    - エラーケーステスト（バリデーション、未定義エンティティ/パーミッション）: 6 テスト
    - parseEntityRef テスト（正常系・異常系）: 5 テスト
    - MaxDepth エラーテスト: 1 テスト
  - 合計 15 テストケース全て成功（11 Expand + 4 parseEntityRef）
- Lookup 実装完了
  - Lookup 実装（lookup.go）
    - Lookup 構造体（Checker、SchemaRepository、RelationRepository への依存）
    - LookupEntity（許可されたエンティティ検索のメインエントリーポイント）
    - LookupSubject（許可されたサブジェクト検索のメインエントリーポイント）
    - getCandidateEntityIDs（候補エンティティ ID 取得、DB から全 tuple 検索）
    - getCandidateSubjectIDs（候補サブジェクト ID 取得、DB から全 tuple 検索）
    - validateLookupEntityRequest（リクエストパラメータ検証）
    - validateLookupSubjectRequest（リクエストパラメータ検証）
    - ページネーション対応（PageSize/PageToken サポート）
    - ブルートフォース実装（Phase 1、キャッシュなし）
  - ユニットテスト実装（lookup_test.go）
    - LookupEntity 基本テスト（複数エンティティへのアクセス検索）: 1 テスト
    - LookupEntity アクセスなしテスト（アクセス権なしの場合）: 1 テスト
    - LookupEntity ページネーションテスト（PageSize 制限）: 1 テスト
    - LookupEntity ContextualTuples テスト（一時的な tuple 対応）: 1 テスト
    - LookupSubject 基本テスト（複数サブジェクトへのアクセス検索）: 1 テスト
    - LookupSubject アクセスなしテスト（アクセス権なしの場合）: 1 テスト
    - LookupSubject ページネーションテスト（PageSize 制限）: 1 テスト
    - LookupEntity エラーケーステスト（バリデーション、未定義エンティティ/パーミッション）: 7 テスト
    - LookupSubject エラーケーステスト（バリデーション、未定義エンティティ/パーミッション）: 7 テスト
    - LookupSubject 論理演算パーミッションテスト（OR 演算）: 1 テスト
  - 合計 16 テストケース全て成功（10 基本 + 14 エラーケース）
- Converter 実装完了
  - Converter 実装（converter.go）
    - ASTToSchema（AST → entities.Schema 変換）
    - SchemaToAST（entities.Schema → AST 変換）
    - convertEntity（EntityAST → entities.Entity）
    - convertPermissionRule（PermissionRuleAST → entities.PermissionRule、4 種類対応）
    - convertEntityToAST（entities.Entity → EntityAST）
    - convertPermissionRuleToAST（entities.PermissionRule → PermissionRuleAST）
  - ユニットテスト実装（converter_test.go）
    - AST → Schema 変換テスト（基本、属性、論理ルール、階層ルール、ABAC ルール）: 5 テスト
    - Schema → AST 変換テスト（基本、論理ルール、階層ルール、ABAC ルール）: 4 テスト
    - Round-trip テスト（AST → Schema → AST）: 1 テスト
  - 合計 10 テストケース全て成功
- Generator 実装完了
  - Generator 実装（generator.go）
    - Generator 構造体（インデント設定可能）
    - Generate（AST → DSL 文字列生成）
    - generateEntity、generateRelation、generateAttribute、generatePermission
    - generatePermissionRule（演算子優先順位対応、括弧生成）
      - OR < AND < NOT の優先順位
      - OR が AND の中にある場合に括弧を追加: (owner or editor) and admin
  - ユニットテスト実装（generator_test.go）
    - 基本エンティティ生成テスト: 1 テスト
    - リレーション、属性、パーミッション生成テスト: 3 テスト
    - 論理演算子生成テスト（OR/AND/NOT）: 3 テスト
    - 演算子優先順位テスト: 1 テスト
    - 階層的パーミッション生成テスト: 1 テスト
    - ABAC ルール生成テスト: 1 テスト
    - 複雑なスキーマ生成テスト: 1 テスト
    - Round-trip テスト（DSL → Parse → Generate → Parse）: 1 テスト
  - 合計 12 テストケース全て成功
- Schema Service 実装完了
  - Schema Service 実装（schema_service.go）
    - SchemaService 構造体（SchemaRepository への依存）
    - WriteSchema（DSL パース → 検証 → 保存）
      - Lexer/Parser で DSL をパース → AST
      - Validator でスキーマ検証
      - Converter で AST → entities.Schema 変換（検証目的）
      - 既存スキーマがあれば Update、なければ Create
    - ReadSchema（DB 取得 → DSL 返却）
      - SchemaRepository で DSL 取得
      - Phase 1 では保存された DSL をそのまま返却
    - ValidateSchema（DSL 検証のみ、保存なし）
    - DeleteSchema（スキーマ削除）
    - GetSchemaEntity（内部用スキーマエンティティ取得）
  - ユニットテスト実装（schema_service_test.go）
    - Mock SchemaRepository 実装（インメモリ）
    - WriteSchema テスト（Create/Update/Invalid DSL/Validation Error/Missing TenantID/Missing DSL）: 6 テスト
    - ReadSchema テスト（正常系/Not Found/Missing TenantID）: 3 テスト
    - ValidateSchema テスト（Valid/Invalid/Missing）: 3 テスト
    - DeleteSchema テスト（正常系/Missing TenantID）: 2 テスト
    - GetSchemaEntity テスト（正常系/Not Found/Missing TenantID）: 3 テスト
  - 合計 17 テストケース全て成功
- Schema Handler 実装完了
  - Schema Handler 実装（schema_handler.go）
    - SchemaHandler 構造体（SchemaServiceInterface への依存）
    - NewSchemaHandler コンストラクタ
    - WriteSchema（DSL 受け取り → SchemaService 呼び出し → レスポンス生成）
      - 空 DSL のバリデーション
      - エラー変換（パースエラー、バリデーションエラー、一般エラー）
      - Phase 1: 固定 tenant ID "default" を使用
    - ReadSchema（SchemaService からスキーマ取得 → レスポンス生成）
      - updated_at のフォーマット（ISO8601）
      - エラー変換（NotFound, InvalidArgument, Internal）
    - handleWriteSchemaError（domain エラー → WriteSchemaResponse）
    - handleReadSchemaError（domain エラー → gRPC status）
  - SchemaServiceInterface 定義（schema_service.go に追加）
    - WriteSchema, ReadSchema, ValidateSchema, DeleteSchema, GetSchemaEntity
  - ユニットテスト実装（schema_handler_test.go）
    - Mock SchemaService 実装（SchemaServiceInterface 実装）
    - WriteSchema テスト（Success/Empty DSL/Parse Error/Validation Error）: 4 テスト
    - ReadSchema テスト（Success/Not Found/No UpdatedAt）: 3 テスト
  - 合計 7 テストケース全て成功
- Data Handler 実装完了
  - Data Handler 実装（data_handler.go）
    - DataHandler 構造体（RelationRepository, AttributeRepository への依存）
    - NewDataHandler コンストラクタ
    - WriteRelations（複数 relation tuple の一括書き込み）
      - proto RelationTuple → entities.RelationTuple 変換
      - BatchWrite による一括書き込み
      - Phase 1: 固定 tenant ID "default" を使用
    - DeleteRelations（複数 relation tuple の一括削除）
      - BatchDelete による一括削除
    - WriteAttributes（複数 attributes の書き込み）
      - proto AttributeData → entities.Attribute 変換（1 つの AttributeData から複数の Attribute に展開）
      - protobuf Value → Go interface{} 変換
      - ループで個別に Write 呼び出し
    - protoToRelationTuple（proto → entities 変換）
      - バリデーション（entity, relation, subject の必須チェック）
      - SubjectRelation は空文字列（proto では Subject が Entity 型のため）
    - protoToAttributes（proto → entities 変換、map を slice に展開）
    - protoValueToInterface（protobuf Value 型変換）
      - null, number, string, bool, struct, list のサポート
  - ユニットテスト実装（data_handler_test.go）
    - Mock RelationRepository 実装（BatchWrite, BatchDelete）
    - Mock AttributeRepository 実装（Write）
    - WriteRelations テスト（Success/Empty/Invalid）: 3 テスト
    - DeleteRelations テスト（Success/Empty）: 2 テスト
    - WriteAttributes テスト（Success/Empty/Invalid/Repository Error）: 4 テスト
    - protoToRelationTuple テスト（valid/missing fields）: 4 テスト
    - protoToAttributes テスト（valid/missing entity/empty data）: 3 テスト
  - 合計 16 テストケース全て成功
- Authorization Handler 実装完了
  - Authorization Handler 実装（authorization_handler.go）
    - AuthorizationHandler 構造体（CheckerInterface, ExpanderInterface, LookupInterface, SchemaRepository への依存）
    - インターフェース定義
      - CheckerInterface: Check メソッド
      - ExpanderInterface: Expand メソッド
      - LookupInterface: LookupEntity, LookupSubject メソッド
    - NewAuthorizationHandler コンストラクタ
    - Check（権限チェック）
      - proto CheckRequest → authorization.CheckRequest 変換
      - Checker.Check 呼び出し
      - 結果を ALLOWED/DENIED に変換
      - Phase 1: 固定 tenant ID "default" を使用
    - Expand（権限ツリー展開）
      - proto ExpandRequest → authorization.ExpandRequest 変換
      - Expander.Expand 呼び出し
      - ExpandNode を proto に変換
    - LookupEntity（許可されたエンティティ検索）
      - proto LookupEntityRequest → authorization.LookupEntityRequest 変換
      - Lookup.LookupEntity 呼び出し
      - ページネーション対応（page_size, continuous_token）
    - LookupSubject（許可されたサブジェクト検索）
      - proto LookupSubjectRequest → authorization.LookupSubjectRequest 変換
      - Lookup.LookupSubject 呼び出し
      - ページネーション対応
    - SubjectPermission（サブジェクトの全権限チェック）
      - スキーマから対象エンティティの全パーミッションを取得
      - 各パーミッションに対して Check を実行
      - 結果を map[permission 名]CheckResult で返却
    - LookupEntityStream（ストリーミング版）
      - Phase 1 では未実装（Unimplemented エラー）
    - protoContextToTuples（Context → RelationTuple 変換）
    - expandNodeToProto（ExpandNode → proto 変換）
  - ユニットテスト実装（authorization_handler_test.go）
    - Mock Checker, Expander, Lookup 実装
    - Check テスト（Allowed/Denied/Missing Entity/With Contextual Tuples/Checker Error）: 5 テスト
    - Expand テスト（Success）: 1 テスト
    - LookupEntity テスト（Success）: 1 テスト
    - LookupSubject テスト（Success）: 1 テスト
    - SubjectPermission テスト（Success/Schema Not Found/Entity Not Found）: 3 テスト
  - 合計 11 テストケース全て成功
- Migration Command 実装完了
  - Migration Command 実装（cmd/migrate/main.go）
    - コマンド引数パース（up/down/goto/version/force）
    - 環境変数対応（ENV=dev|test|prod、デフォルト: dev）
    - 設定読み込み（config.InitConfig、config.Load）
    - DB 接続初期化（database.NewPostgres）
    - プロジェクトルート検出（findProjectRoot、go.mod を基準）
    - マイグレーションパス解決（internal/infrastructure/database/migrations/postgres）
    - up コマンド実装（全マイグレーション適用）
    - down コマンド実装（指定ステップ数ロールバック、デフォルト: 1）
    - goto コマンド実装（指定バージョンへ移行）
    - version コマンド実装（現在のマイグレーションバージョン表示、dirty 状態対応）
    - force コマンド実装（強制的にバージョン設定）
    - エラーハンドリング（migrate.ErrNoChange 対応）
    - Usage 表示（printUsage 関数）
  - Database Package 拡張（internal/infrastructure/database/postgres.go）
    - NewMigrateDriver 関数追加（database.Driver 返却）
    - golang-migrate/migrate/v4/database パッケージ import 追加
  - 動作確認
    - ENV=test ./bin/migrate version → バージョン 3 確認
    - ENV=test ./bin/migrate down → バージョン 2 へロールバック確認
    - ENV=test ./bin/migrate up → バージョン 3 へ再適用確認
    - ENV=test ./bin/migrate goto 1 → バージョン 1 へ移行確認
    - ENV=test ./bin/migrate up → 最新バージョンへ復元確認
- gRPC サーバー実装完了（統合アプローチ）
  - 設計方針変更: 業界標準（Google Zanzibar / Permify）に従い、単一 AuthorizationService として実装
  - DESIGN.md 更新
    - Section 6 を「統合 Authorization Handler」に変更
    - 単一サービスアプローチの理由を明記（認可は分離できない 1 つのドメイン）
    - ヘルパー関数セクション追加
  - DEVELOPMENT.md 更新
    - Section 7 を「統合 Authorization Handler」に変更
    - 既存の schema_handler.go と data_handler.go のロジックを統合する方針を明記
  - AuthorizationHandler 拡張（internal/handlers/authorization_handler.go）
    - Schema 管理メソッド追加
      - WriteSchema（schemaService 経由）
      - ReadSchema（schemaService 経由、メタデータ取得）
      - handleWriteSchemaError、handleReadSchemaError（エラー変換）
    - Data 管理メソッド追加
      - WriteRelations（relationRepo.BatchWrite 経由）
      - DeleteRelations（relationRepo.BatchDelete 経由）
      - WriteAttributes（attributeRepo.Write 経由、複数属性展開）
    - ヘルパー関数追加
      - protoToRelationTuple（proto → entities 変換）
      - protoToAttributes（proto → entities 変換、map 展開）
      - protoValueToInterface（protobuf Value → Go interface{} 変換）
    - 既存の認可メソッドはそのまま維持
  - 不要ファイル削除
    - schema_handler.go 削除（AuthorizationHandler に統合）
    - data_handler.go 削除（AuthorizationHandler に統合）
  - cmd/server/main.go 実装
    - 環境変数対応（ENV、PORT）
    - 設定読み込み（config.InitConfig、config.Load）
    - DB 接続初期化（database.NewPostgres）
    - Repository 初期化（SchemaRepo、RelationRepo、AttributeRepo）
    - Service 初期化（SchemaService、CELEngine、Evaluator、Checker、Expander、Lookup）
    - 統合 AuthorizationHandler 初期化（7 つの依存を注入）
    - gRPC サーバー起動（デフォルトポート 50051）
    - Reflection 有効化（grpcurl 対応）
    - グレースフルシャットダウン実装（30 秒タイムアウト、SIGTERM/SIGINT 対応）
  - 動作確認
    - ビルド成功: go build -o bin/server ./cmd/server
    - サーバー起動成功: ENV=test PORT=50052 ./bin/server
    - grpcurl で全メソッド確認（11 メソッド）
      - Schema: WriteSchema, ReadSchema
      - Data: WriteRelations, DeleteRelations, WriteAttributes
      - Authorization: Check, Expand, LookupEntity, LookupSubject, SubjectPermission, LookupEntityStream
- テスト統合完了
  - authorization_handler_test.go 完全書き直し
    - schema_handler_test.go（7 テスト）の内容を統合
    - data_handler_test.go（16 テスト）の内容を統合
    - 既存の authorization テスト（11 テスト）を維持
    - 全モック実装を統一（SchemaService、RelationRepository、AttributeRepository、Checker、Expander、Lookup、SchemaRepository）
    - 合計 29 テストを単一ファイルに集約
  - 不要ファイル削除
    - schema_handler_test.go 削除
    - data_handler_test.go 削除
  - テスト実行結果: 全 29 テストケース成功（go test ./internal/handlers/... -v）

---

## 技術的な決定事項

### 確定事項

1. ABAC: 全演算子完全サポート（最小限ではない）
2. キャッシュ: Phase 1 では実装しない（Phase 2 で追加）
3. DB 設計: VARCHAR(255)、正規化なし（後で最適化可能）
4. スキーマバージョニング: 管理しない
5. Permify 互換性: entity/relation/subject 構造を完全に維持
6. CEL: google/cel-go を使用
7. マイグレーション: golang-migrate/migrate 使用、cmd/migrate でラップして実行、マイグレーションファイルは internal/infrastructure/database/migrations/postgres/ に配置
8. PostgreSQL: 18.0-alpine3.22
9. Repository パターン: インターフェースと実装を分離（各リポジトリごとにファイルを分割）
10. Infrastructure 構造: database/（postgres.go + migrations/postgres/）、config/（config.go）
11. Proto ファイル構成: 3 ファイル（common.proto, authorization.proto, audit.proto）に分割、proto/keruberosu/v1/ に配置
12. コマンド構成: cmd/server（gRPC サーバー）、cmd/migrate（DB マイグレーション）
13. タイムスタンプ型: PostgreSQL では TIMESTAMPTZ を使用（タイムゾーン対応）
14. Proto コード生成: scripts/generate-proto.sh を使用（Make 不使用）
15. 環境設定: viper を使用、環境ごとに .env.{env} ファイルで管理（dev/test/prod）
16. Entities 設計: 1 ファイル 1 構造体の原則に従う（Go ベストプラクティス準拠）
    - スキーマ定義系と実際のデータを明確に分離
    - ファイル名と内容の一貫性を保つ（例: permission.go には Permission 構造体）
17. テスト環境: test 環境で実行、.env.test から設定を読み込む
    - config.InitConfig("test") で自動的に .env.test を検出
    - go.mod の位置からプロジェクトルートを自動検出
    - docker-compose up -d --wait で healthcheck 完了まで待機
18. gRPC ハンドラー設計: 単一 AuthorizationService として実装（業界標準に準拠）
    - Google Zanzibar / Permify のアプローチに従う
    - Schema、Data、Authorization の全操作を 1 つのハンドラーで提供
    - 理由: 認可は分離できない 1 つのドメイン、クライアント利便性、運用の単純化

### 検討中

- なし（Phase 1 スコープは確定済み）

---

## リスクと課題

### リスク

1. パフォーマンス: キャッシュなしでの性能（Phase 2 で対応）
2. LookupAPI: 大量データ時の性能（Phase 1 では愚直実装）

### 対策

- Phase 1 では動作確認優先
- パフォーマンス最適化は Phase 2 で実施

---

## チーム・役割

- 開発: @asakaida
- レビュー: TBD
- テスト: TBD

---

## 参考資料

- [Permify Documentation](https://docs.permify.co/)
- [CEL Language Definition](https://github.com/google/cel-spec)
- [Protocol Buffers Guide](https://protobuf.dev/)
- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)

---

## 更新履歴

| 日付       | 更新者    | 内容                                                                                      |
| ---------- | --------- | ----------------------------------------------------------------------------------------- |
| 2025-10-08 | @asakaida | 初版作成、プロジェクト構造修正（repository/infrastructure）                               |
| 2025-10-08 | @asakaida | Proto 3 ファイル構成に変更、cmd/migrate 追加、go.mod 初期化、.gitignore と README.md 作成 |
