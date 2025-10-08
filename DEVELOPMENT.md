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

- [ ] internal/services/authorization/checker.go
  - [ ] Checker 構造体
  - [ ] NewChecker
  - [ ] Check（メインエントリーポイント）
  - [ ] 深さ制限チェック
  - [ ] contextualTuples 統合
  - [ ] ユニットテスト

#### 5.4 Expand 実装

- [ ] internal/services/authorization/expander.go
  - [ ] Expander 構造体
  - [ ] ExpandNode 構造体
  - [ ] Expand（権限ツリー構築）
  - [ ] expandRule（ルールごとのノード構築）
  - [ ] ユニットテスト

#### 5.5 Lookup 実装

- [ ] internal/services/authorization/lookup.go
  - [ ] Lookup 構造体
  - [ ] LookupEntity（許可されたエンティティ検索）
  - [ ] LookupSubject（許可されたサブジェクト検索）
  - [ ] ページネーション対応
  - [ ] ユニットテスト

---

### 6. ビジネスロジック層（Service）

#### 6.1 Schema Service

- [ ] internal/services/schema_service.go
  - [ ] SchemaService 構造体
  - [ ] WriteSchema（DSL パース → 保存）
  - [ ] ReadSchema（取得 → DSL 生成）
  - [ ] ValidateSchema
  - [ ] ユニットテスト

---

### 7. gRPC ハンドラー層

#### 7.1 Schema Handler

- [ ] internal/handlers/schema_handler.go
  - [ ] SchemaHandler 構造体
  - [ ] WriteSchema
  - [ ] ReadSchema
  - [ ] エラー変換（domain → gRPC）
  - [ ] ユニットテスト

#### 7.2 Data Handler

- [ ] internal/handlers/data_handler.go
  - [ ] DataHandler 構造体
  - [ ] WriteRelation
  - [ ] DeleteRelation
  - [ ] ReadRelations
  - [ ] WriteAttribute
  - [ ] ReadAttributes
  - [ ] バリデーション
  - [ ] ユニットテスト

#### 7.3 Authorization Handler

- [ ] internal/handlers/authorization_handler.go
  - [ ] AuthorizationHandler 構造体
  - [ ] Check
  - [ ] Expand
  - [ ] LookupEntity
  - [ ] LookupSubject
  - [ ] SubjectPermission
  - [ ] メタデータ処理（snap_token, depth）
  - [ ] ユニットテスト

---

### 8. メインエントリーポイント

#### 8.1 マイグレーションコマンド

- [ ] cmd/migrate/main.go
  - [ ] 設定読み込み
  - [ ] DB 接続初期化
  - [ ] マイグレーションライブラリ（golang-migrate/migrate）統合
  - [ ] up コマンド実装
  - [ ] down コマンド実装
  - [ ] goto コマンド実装
  - [ ] エラーハンドリング

#### 8.2 gRPC サーバー

- [ ] cmd/server/main.go
  - [ ] 設定読み込み
  - [ ] DB 接続初期化
  - [ ] Repository 初期化
  - [ ] Service 初期化
  - [ ] Handler 初期化
  - [ ] gRPC サーバー起動
  - [ ] シグナルハンドリング（グレースフルシャットダウン）

---

### 9. テスト

#### 9.1 ユニットテスト

- [ ] 全パッケージのユニットテスト作成
- [ ] カバレッジ計測
- [ ] 目標: 80%以上

#### 9.2 統合テスト

- [ ] PostgreSQL 込みの統合テスト
- [ ] test コンテナ使用
- [ ] マイグレーション自動適用

#### 9.3 E2E テスト

- [ ] gRPC 経由の完全シナリオテスト
- [ ] Google Docs 風の ReBAC 例
- [ ] ABAC 例（全演算子）
- [ ] Permify 互換性検証

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

全体進捗: 45% (基盤構築 + ドメインエンティティ + Repository + DSL パーサー + CEL エンジン + Evaluator 完了)

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
- [x] DSL パーサー実装（Lexer, Parser, Validator）
- [x] CEL エンジン実装（ABAC ルール評価）
- [x] Evaluator 実装（ルール評価エンジン）

#### 進行中タスク

- [ ] Check 実装

#### 次のマイルストーン

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
