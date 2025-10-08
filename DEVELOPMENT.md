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

- [ ] proto/keruberosu/v1/common.proto
  - [ ] Entity メッセージ
  - [ ] Subject メッセージ
  - [ ] SubjectReference メッセージ
  - [ ] RelationTuple メッセージ
  - [ ] PermissionCheckMetadata メッセージ
  - [ ] Context メッセージ
  - [ ] AttributeData メッセージ
  - [ ] CheckResult enum
- [ ] proto/keruberosu/v1/authorization.proto
  - [ ] AuthorizationService 定義
  - [ ] スキーマ管理 API（WriteSchema/ReadSchema）
  - [ ] データ書き込み API（WriteRelations/DeleteRelations/WriteAttributes）
  - [ ] 認可チェック API（Check/Expand/LookupEntity/LookupSubject/SubjectPermission）
  - [ ] 各 API のリクエスト/レスポンスメッセージ
- [ ] proto/keruberosu/v1/audit.proto
  - [ ] AuditService 定義
  - [ ] WriteAuditLog API
  - [ ] ReadAuditLogs API
  - [ ] AuditLog メッセージ
- [ ] protoc コード生成スクリプト作成
  - [ ] buf.yaml 作成（推奨）または Makefile

---

### 2. データアクセス層（Repository）

#### 2.1 Repository インターフェース定義

- [ ] internal/repositories/schema_repository.go

  - [ ] SchemaRepository インターフェース
  - [ ] Create メソッド定義
  - [ ] GetByTenant メソッド定義
  - [ ] Update メソッド定義
  - [ ] Delete メソッド定義

- [ ] internal/repositories/relation_repository.go

  - [ ] RelationFilter 構造体定義
  - [ ] RelationRepository インターフェース
  - [ ] Write メソッド定義
  - [ ] Delete メソッド定義
  - [ ] Read メソッド定義
  - [ ] CheckExists メソッド定義
  - [ ] BatchWrite メソッド定義
  - [ ] BatchDelete メソッド定義

- [ ] internal/repositories/attribute_repository.go
  - [ ] AttributeRepository インターフェース
  - [ ] Write メソッド定義
  - [ ] Read メソッド定義
  - [ ] Delete メソッド定義
  - [ ] GetValue メソッド定義

#### 2.2 PostgreSQL 実装

- [ ] internal/repositories/postgres/schema_repository.go

  - [ ] PostgresSchemaRepository 構造体
  - [ ] NewPostgresSchemaRepository
  - [ ] Create 実装
  - [ ] GetByTenant 実装
  - [ ] Update 実装
  - [ ] Delete 実装
  - [ ] ユニットテスト

- [ ] internal/repositories/postgres/relation_repository.go

  - [ ] PostgresRelationRepository 構造体
  - [ ] NewPostgresRelationRepository
  - [ ] Write 実装
  - [ ] Delete 実装
  - [ ] Read 実装
  - [ ] CheckExists 実装
  - [ ] BatchWrite 実装
  - [ ] BatchDelete 実装
  - [ ] ユニットテスト

- [ ] internal/repositories/postgres/attribute_repository.go
  - [ ] PostgresAttributeRepository 構造体
  - [ ] NewPostgresAttributeRepository
  - [ ] Write 実装
  - [ ] Read 実装
  - [ ] Delete 実装
  - [ ] GetValue 実装
  - [ ] ユニットテスト

---

### 3. ドメインエンティティ

#### 3.1 基本エンティティ

- [ ] internal/entities/schema.go

  - [ ] Schema 構造体
  - [ ] Entity 構造体
  - [ ] Permission 構造体
  - [ ] Relation 構造体
  - [ ] ヘルパーメソッド（GetPermission, GetEntity 等）

- [ ] internal/entities/relation.go

  - [ ] RelationTuple 構造体
  - [ ] 検証ロジック

- [ ] internal/entities/attribute.go
  - [ ] Attribute 構造体
  - [ ] 型変換ロジック

#### 3.2 権限ルールエンティティ

- [ ] internal/entities/permission.go
  - [ ] PermissionRule インターフェース
  - [ ] RelationRule 構造体
  - [ ] LogicalRule 構造体（OR/AND/NOT）
  - [ ] HierarchicalRule 構造体（parent.permission）
  - [ ] ABACRule 構造体（CEL 式）

---

### 4. DSL パーサー実装

#### 4.1 字句解析（Lexer）

- [ ] internal/services/parser/lexer.go
  - [ ] Token 型定義
  - [ ] Lexer 構造体
  - [ ] NextToken 実装
  - [ ] キーワード認識（entity, relation, permission, rule）
  - [ ] 演算子認識（or, and, not, =）
  - [ ] ユニットテスト

#### 4.2 構文解析（Parser）

- [ ] internal/services/parser/ast.go

  - [ ] AST 構造体定義（SchemaAST, EntityAST 等）
  - [ ] PermissionRuleAST インターフェース実装

- [ ] internal/services/parser/parser.go
  - [ ] Parser 構造体
  - [ ] Parse（メインエントリーポイント）
  - [ ] parseEntity
  - [ ] parseRelation
  - [ ] parseAttribute
  - [ ] parsePermission
  - [ ] parsePermissionRule（再帰的）
  - [ ] エラーハンドリング
  - [ ] ユニットテスト

#### 4.3 検証（Validator）

- [ ] internal/services/parser/validator.go
  - [ ] スキーマ検証
  - [ ] 関係性の循環参照チェック
  - [ ] 未定義の関係参照チェック
  - [ ] 型整合性チェック
  - [ ] ユニットテスト

---

### 5. 認可エンジン

#### 5.1 CEL エンジン

- [ ] internal/services/authorization/cel.go
  - [ ] CELEngine 構造体
  - [ ] NewCELEngine（環境初期化）
  - [ ] Evaluate（式評価）
  - [ ] エラーハンドリング
  - [ ] ユニットテスト（全演算子）
    - [ ] 比較演算子: ==, !=, >, >=, <, <=
    - [ ] in 演算子
    - [ ] 論理演算子: &&, ||, !

#### 5.2 ルール評価（Evaluator）

- [ ] internal/services/authorization/evaluator.go
  - [ ] Evaluator 構造体
  - [ ] NewEvaluator
  - [ ] EvaluateRule（ルール評価ディスパッチャ）
  - [ ] evaluateRelation（関係性チェック）
  - [ ] evaluateLogical（OR/AND/NOT）
  - [ ] evaluateHierarchical（parent.permission）
  - [ ] evaluateABAC（CEL 呼び出し）
  - [ ] ユニットテスト

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

全体進捗: 5% (設計フェーズ完了)

#### 完了タスク

- [x] PRD.md 作成
- [x] DESIGN.md 作成
- [x] DEVELOPMENT.md 作成

#### 進行中タスク

- [ ] プロジェクト初期化

#### 次のマイルストーン

Milestone 1: 基盤構築完了（Week 1）

- プロジェクト構造作成
- docker-compose.yml 作成
- マイグレーション作成
- Protocol Buffers 定義

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
