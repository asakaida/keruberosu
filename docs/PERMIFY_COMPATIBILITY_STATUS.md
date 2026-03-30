# Permify 互換性ステータス

## 最終更新日: 2026-03-30

## 最新の大規模アップデート（2025-10-15）

Permify 互換 API 構造への完全移行が完了しました。

主な変更点:

- サービス分割完了: 単一の `AuthorizationService` を 3 つの Permify 互換サービス（Permission, Data, Schema）に分割
- Proto ファイル分割: `authorization.proto` を 3 つの独立したファイル
  (`permission.proto`, `data.proto`, `schema.proto`) に分割し、
  Permify のファイル構成に完全準拠
- メッセージ名統一: 全てのリクエスト/レスポンスメッセージを Permify 命名規則に変更
- API 統合: `WriteAttributes` を `Data.Write` に統合し、tuples と attributes を一つの API で処理可能に
- API 構造互換性: 100% 達成

---

## 完了した互換性対応

### 1. サービス分割（Permify 互換構造）

- 単一の`AuthorizationService`を 3 つのサービスに分割:
  - Permission サービス: Check, Expand, LookupEntity,
    LookupSubject, LookupEntityStream, SubjectPermission
  - Data サービス: Write, Delete, Read, ReadAttributes
  - Schema サービス: Write, Read

### 2. メッセージ名の Permify 互換化

- `WriteSchemaRequest` → `SchemaWriteRequest` (フィールド: `SchemaDsl` → `Schema`)
- `ReadSchemaRequest` → `SchemaReadRequest`
- `WriteRelationsRequest` → `DataWriteRequest`
- `WriteAttributesRequest` → `DataWriteRequest`に統合 (tuples と attributes を同時に書き込み可能)
- `DeleteRelationsRequest` → `DataDeleteRequest`
- `ReadRelationshipsRequest` → `DataReadRequest`
- `CheckRequest` → `PermissionCheckRequest`
- `ExpandRequest` → `PermissionExpandRequest`
- `LookupEntityRequest` → `PermissionLookupEntityRequest`
- `LookupSubjectRequest` → `PermissionLookupSubjectRequest`
- `SubjectPermissionRequest` → `PermissionSubjectPermissionRequest`

### 3. Permify 互換型名の導入

- `Tuple` メッセージ（Permify 互換）
- `Attribute` メッセージ（Permify 互換）
- `Expand` および `ExpandTreeNode` メッセージ（Permify 互換）

### 4. Proto 定義の更新

- `RelationTuple.subject`を Subject 型に変更（relation フィールドをサポート）
- `DataWriteRequest`に`tuples`と`attributes`フィールドを追加（統合）
- `SchemaWriteResponse`を`schema_version`返却形式に変更
- `DataWriteResponse`、`DataDeleteResponse`に`snap_token`を追加
- `DataDeleteRequest`をフィルター形式に変更（`TupleFilter`使用）
- `AttributeData`を Permify 互換に変更（単一属性形式）
- `DataReadRequest` API を追加

### 5. Schema DSL の拡張

- `action`キーワードをサポート（`permission`の別名）
- `@user`記法をサポート（`:  user`と等価）
- 両方の記法を同時サポート

### 6. グループメンバーシップ機能

- 1 つのタプルで`entity#relation@subject#relation`を表現可能
- 例: `drive:eng_drive#member@group:engineering#member`

---

## 追加で完了した実装（2025-10-14 更新）

### 7. サービス分割とメッセージ名変更の完全実装 【完了】

- 3 つのサービス（Permission, Data, Schema）に分割完了
- Proto ファイルを物理的に分割:
  - `proto/keruberosu/v1/permission.proto` - Permission サービス + 関連メッセージ
  - `proto/keruberosu/v1/data.proto` - Data サービス + 関連メッセージ
  - `proto/keruberosu/v1/schema.proto` - Schema サービス + 関連メッセージ
  - `proto/keruberosu/v1/common.proto` - 共通メッセージ型（Entity, Subject, Tuple 等）
- 全メッセージ名を Permify 互換に変更
- `WriteAttributes` RPC を削除し、`Data.Write()` に統合
- Permify 互換型名（Tuple, Attribute）を導入
- Expand メッセージの追加
- 全ハンドラーの実装更新完了
- 全テスト（unit/integration/E2E）成功確認

### 8. DeleteRelations のフィルター実装 【完了】

- Proto 定義は`TupleFilter`に更新済み
- ハンドラー実装完了（`DeleteByFilter`メソッド使用）
- リポジトリ層でのフィルター対応完了
  - `EntityFilter` (type + ids)
  - `SubjectFilter` (type + ids + relation)
- 複数 ID での一括削除対応（`pq.Array()`使用）

### 9. ReadRelationships API の実装 【完了】

- Proto 定義追加済み（`DataReadRequest`として）
- ハンドラー実装完了
- リポジトリ層でのフィルター・ページネーション対応完了
- continuous_token の生成・検証実装
- E2E テストで動作確認済み

### 10. AttributeData 形式変更の完全対応 【完了】

- Proto 定義は単一属性形式に更新済み
- 全ハンドラーの AttributeData 処理を単一属性形式に統一
- 全テストケースの更新完了
- 全 example コードの更新完了

### 11. Expand API の Permify 完全互換化 【完了】（2025-10-14 追加）

変更前の問題:

- `Expand.node` フィールドが `ExpandTreeNode` 型（oneof なし）
- `ExpandTreeNode.operation` が `string` 型（enum ではない）
- リーフノードとツリーノードの区別が曖昧
- Permify の仕様と異なる構造

Permify 準拠の新構造:

- `Expand` メッセージに `oneof node` を追加（`expand` または `leaf`）
- `ExpandTreeNode.operation` を `enum Operation` に変更
  - `OPERATION_UNSPECIFIED = 0`
  - `OPERATION_UNION = 1` (OR 結合)
  - `OPERATION_INTERSECTION = 2` (AND 結合)
  - `OPERATION_EXCLUSION = 3` (除外)
- `ExpandTreeNode.children` の型を `repeated Expand` に変更（再帰的構造）
- `ExpandLeaf` メッセージを追加（`oneof type` で subjects/values/value を区別）
- `Subjects` および `Values` メッセージを追加

実装された機能:

- ツリーノード（union/intersection/exclusion）とリーフノード（subjects/values）の明確な区別
- 再帰的なツリー構造のサポート
- 全ハンドラーコードの更新（`expandNodeToProto` 関数）
- 全テストケースの更新（E2E、ユニットテスト）
- Example 08（Expand API デモ）の更新
- 全テスト成功確認

影響範囲:

- `proto/keruberosu/v1/common.proto` - Expand メッセージ定義
- `internal/handlers/helpers.go` - expandNodeToProto 関数
- `internal/handlers/permission_handler.go` - Expand ハンドラー
- `test/e2e/*.go` - E2E テストの更新
- `examples/08_expand/main.go` - サンプルコードの更新

### 12. Permission API の完全 Permify 互換化 【完了】（2025-10-14 追加）

変更内容:

LookupEntity API:

- `tenant_id` フィールドを追加（フィールド番号 1）
- `scope` フィールドを追加（`map<string, StringArrayValue>`、フィールド番号 7）
- `page_size` の型を `int32` から `uint32` に変更（フィールド番号 8）
- フィールド番号を Permify に合わせて再配置

LookupSubject API:

- `tenant_id` フィールドを追加（フィールド番号 1）
- `page_size` の型を `int32` から `uint32` に変更（フィールド番号 7）
- フィールド番号を Permify に合わせて再配置

SubjectPermission API:

- `tenant_id` フィールドを追加（フィールド番号 1）
- フィールド番号を Permify に合わせて再配置

Check API:

- `tenant_id` フィールドを追加（フィールド番号 1）
- フィールド番号を Permify に合わせて再配置

Expand API:

- `tenant_id` フィールドを追加（フィールド番号 1）
- フィールド番号を Permify に合わせて再配置

共通変更:

- `StringArrayValue` メッセージを `common.proto` に追加（scope パラメータ用）
- 全ハンドラーで `tenant_id` の処理を実装（空の場合は "default" を使用）
- 全テスト成功確認（E2E、ユニットテスト、example ビルド）

tenant_id の扱い:

- tenant_id は proto 定義に含まれているが、Keruberosu では将来のマルチテナント対応に備えた設計
- 現在は空の場合に "default" を使用する実装
- 将来的には gRPC メタデータや HTTP ヘッダーからも取得可能にする予定

影響範囲:

- `proto/keruberosu/v1/common.proto` - StringArrayValue 追加
- `proto/keruberosu/v1/permission.proto` - 全リクエストメッセージ更新
- `internal/handlers/permission_handler.go` - tenant_id 処理追加
- 全テストケース（自動的に対応、tenant_id が空でも動作）
- 全 example コード（tenant_id が空でも動作）

### 13. Schema Version 機能 【完了】（2025-10-15 追加）

実装内容:

データベース層:

- マイグレーションファイル作成（`000004_add_schema_version.up/down.sql`）
- `schemas` テーブルに `version VARCHAR(26)` カラム追加
- UNIQUE 制約を `(tenant_id)` から `(tenant_id, version)` に変更（複数バージョン対応）
- インデックス追加: `idx_schemas_version`, `idx_schemas_tenant_created`
- ULID (Universally Unique Lexicographically Sortable Identifier) を採用

リポジトリ層:

- `SchemaRepository.Create()` の戻り値を `(string, error)` に変更（バージョン ID 返却）
- `GetLatestVersion()` メソッド追加（最新バージョン取得）
- `GetByVersion()` メソッド追加（特定バージョン取得）
- `GetByTenant()` を `GetLatestVersion()` のエイリアスとして維持（後方互換性）
- ULID 生成ロジック実装（`github.com/oklog/ulid/v2`）

サービス層:

- `SchemaService.WriteSchema()` の戻り値を `(string, error)` に変更
- `ReadSchema()` の戻り値を `(*entities.Schema, error)` に変更
- `GetSchemaEntity()` に version パラメータ追加（空文字列で最新版）
- 各 Write 時に新バージョン自動生成（Permify 互換動作）

ハンドラ層:

- `SchemaHandler.Write()` で生成されたバージョンを返却
- `SchemaWriteResponse.schema_version` に実際のバージョン ID 設定
- 全 authorization 関連コードで version パラメータ対応

Entity 層:

- `entities.Schema` に `Version string` フィールド追加
- バージョン情報の保持とメタデータ管理

テスト:

- Repository 単体テスト更新（バージョン作成・取得・削除）
- Service 単体テスト更新（複数バージョン管理）
- Handler 単体テスト更新（バージョン返却確認）
- Authorization 単体テスト更新（version パラメータ対応）
- E2E テスト成功確認
- 全テストパス確認（100%成功）

バージョニング仕様:

- ULID 形式: 26 文字の英数字（例: `01ARZ3NDEKTSV4RRFFQ69G5FAV`）
- タイムスタンプベース（時系列ソート可能）
- 衝突耐性（分散環境でも一意性保証）
- 新スキーマ書き込み毎に自動で新バージョン生成
- 旧バージョンは削除せず保持（履歴管理）

Permify 互換性:

- `SchemaWriteResponse.schema_version` に実際のバージョン ID 返却
- 空文字列指定で最新バージョン取得
- 特定バージョン指定で過去バージョン取得可能

影響範囲:

- `internal/infrastructure/database/migrations/postgres/000004_add_schema_version.*`
- `internal/entities/schema.go`
- `internal/repositories/schema_repository.go`
- `internal/repositories/postgres/schema_repository.go`
- `internal/services/schema_service.go`
- `internal/services/authorization/*.go` (evaluator, checker, expander, lookup)
- `internal/handlers/schema_handler.go`
- `internal/handlers/permission_handler.go`
- `internal/handlers/test_helpers.go`
- 全単体・統合・E2E テスト
- `go.mod` (ulid 依存追加)

---

## Phase 2 で完了した機能（2025-12-30）

### 1. Snap Token / Cache 機構 【完了】

実装内容:

- `Data.Write/Delete` レスポンスで `snap_token` を返却
- PostgreSQL `txid_current()` ベースのスナップショットトークン生成
- LRU + TTL ベースのインメモリキャッシュ
- `CheckerWithCache` による透過的キャッシュ層
- キャッシュキーに `snap_token` を含めた MVCC 対応

実装ファイル:

- `pkg/cache/memorycache/memorycache.go` - LRU+TTL キャッシュ実装
- `internal/infrastructure/cache/snapshot_manager.go` - SnapshotManager
- `internal/services/authorization/checker_with_cache.go` - キャッシュ付き Checker
- `internal/repositories/postgres/snapshot.go` - トークン生成

---

### 2. Closure Table 【完了】

実装内容:

- `entity_closure` テーブルによる O(1) 祖先検索
- Write/Delete 時の自動 Closure Table 更新
- 階層的パーミッション評価の高速化

実装ファイル:

- `internal/infrastructure/database/migrations/postgres/000007_create_entity_closure_table.up.sql`
- `internal/repositories/postgres/relation_repository.go` - updateClosureOnAdd/Delete

---

### 3. Prometheus メトリクス 【完了】

実装内容:

- gRPC リクエスト数/エラー数/処理時間の収集
- キャッシュヒット率の監視
- HTTP エンドポイント（:9090/metrics）

実装ファイル:

- `internal/infrastructure/metrics/collector.go`
- `internal/infrastructure/metrics/prometheus.go`
- `internal/infrastructure/metrics/interceptor.go`

---

## Phase 3 で完了した機能（2026-03-30）

### 1. DB基盤強化

- Primary + Read Replica 対応（DBCluster）
- ResilientDB（トランジェントエラー自動リトライ）
- WriteTracker（テナント単位の書き込み追跡）
- DBTX interface
- Config拡張（Replica/Tracker/Closure設定）

### 2. Closureテーブル活用

- LookupでのClosure+UNION最適化

### 3. HierarchicalRuleCallRule

- parent.rule(args)構文のサポート
- エンティティ、AST、パーサー、コンバーター、バリデーター、ジェネレーター
- CELEngine.EvaluateWithParams
- Evaluator対応

### 4. gRPCバリデーション

- protovalidate導入
- UnaryServerInterceptor

### 5. Admin CLI

- rebuild-closuresコマンド

### 6. Lookup最適化

- extractRelationsFromRuleWithContext
- Computed Userset対応
- LookupAccessibleEntitiesComplex / SubjectsComplex

### 7. SubjectRelation対応

- SubjectRelation を考慮した評価パス

---

## 先送り事項（今後の実装が必要）

### 1. Tenant ID / マルチテナント機能 【中優先度】

現状（2025-10-14 更新）:

- 全 Permission API で `tenant_id` フィールドを追加（Permify 互換）
- ハンドラーで `tenant_id` が空の場合は "default" を使用
- マルチテナント対応は未実装（現在は "default" のみ）

仕様検討が必要:

1. マルチテナント実装方法
   - 現在: proto 定義に tenant_id フィールドあり（空の場合は "default"）
   - 追加オプション: gRPC メタデータ、HTTP ヘッダー、JWT トークンからの抽出
2. Tenant ごとのデータ分離戦略
   - スキーマ分離（PostgreSQL schema）
   - テーブル内のテナントカラム（現在の実装）
   - データベース分離
3. Tenant 管理 API
   - Tenant 作成・削除
   - Tenant 設定管理
4. 認証・認可との統合
   - JWT トークンから Tenant ID 抽出
   - Tenant 間のアクセス制御

影響範囲:

- gRPC インターセプター（メタデータ処理）
- HTTP ミドルウェア（ヘッダー処理）
- 認証ミドルウェア
- データベース設計

---

### 4. Schema DSL の完全 Permify 互換 【完了】

現状:

- `action`キーワードと`@user`記法はサポート済み
- 複数型指定（`@user @team#member`）の完全対応完了

実装内容:

- Permify 互換の`@user @team#member`形式（スペース区切り）に統一
- パイプ区切り（`|`）記法を削除（後方互換性なし）
- パーサー、ジェネレーター、バリデーターを全て更新
- 全テスト、Example、ドキュメント（README, PRD, DESIGN）を更新

---

### 5. エラーハンドリングの統一 【低優先度】

現状:

- WriteSchema は validation error を返す
- 他の API は gRPC status error を返す
- エラーメッセージ形式が不統一

考慮事項:

- Permify のエラーレスポンス形式に合わせるか
- gRPC standard status codes を活用

---

## 互換性達成度（2026-03-30 更新）

### API 構造互換性: 100%

- サービス分割（Permission, Data, Schema）
- メッセージ名の Permify 互換化
- Permify 互換型名（Tuple, Attribute）
- Expand メッセージ
- Data.Write への統合（tuples + attributes）

### API レベル互換性: 100%

- 基本的な API 構造（Permission.Check, Lookup 系）
- RelationTuple の構造
- Subject relation サポート
- Schema DSL 基本文法
- Data.Read 実装完了（旧 ReadRelationships）
- Data.Delete フィルター実装完了
- tenant_id フィールド追加（全 Permission API で Permify 互換）
- scope パラメータ追加（LookupEntity API）
- page_size 型変更（int32 → uint32）
- Snap Token 対応 - Data.Write/Delete で snap_token 返却
- Schema Version 対応 - ULID ベースのバージョン管理

### データ構造互換性: 100%

- Entity, Subject, RelationTuple
- AttributeData（単一属性形式、完全実装）
- TupleFilter（完全実装）
- 全ハンドラー実装完了

### 機能互換性: 98%

- Permission.Check（キャッシュ対応）
- Permission.LookupEntity/LookupSubject（Closure+UNION最適化、Computed Userset対応）
- Schema.Write/Read（バージョン管理対応）
- グループメンバーシップ
- Data.Write（tuples + attributes 統合、snap_token 返却）
- Data.Delete（フィルター対応、snap_token 返却）
- Data.Read（ページネーション対応）
- Schema versioning（ULID ベース）
- Snap token / Cache（LRU + TTL、MVCC 対応）
- HierarchicalRuleCallRule（parent.rule(args)構文）
- gRPCバリデーション（protovalidate）
- Admin CLI（rebuild-closures）
- Tenant management は未実装（現在は固定 "default" テナント）

---

## 推奨実装順序

### Phase 1（完了）

1. サービス分割（Permission, Data, Schema）
2. メッセージ名の Permify 互換化
3. Permify 互換型名の導入（Tuple, Attribute）
4. Data.Write への統合（tuples + attributes）
5. ハンドラーのコンパイルエラー修正
6. 既存テスト・example の更新
7. Data.Delete フィルター実装
8. Data.Read 実装
9. 全 unit/integration/E2E テスト成功確認

### Phase 2（完了 - 2025-12-30）

1. Snap token 実装（PostgreSQL txid_current() ベース）
2. LRU + TTL キャッシュ実装
3. CheckerWithCache による透過的キャッシュ
4. Closure Table 実装（O(1) 祖先検索）
5. Prometheus メトリクス実装
6. 全テスト成功確認（12パッケージ）

### Phase 3（完了 - 2026-03-30）

1. DB基盤強化（DBTX, WriteTracker, ResilientDB, DBCluster）
2. リポジトリDBCluster化、RelationRepository拡張
3. HierarchicalRuleCallRule追加（parent.rule(args)構文）
4. Evaluator/Lookup最適化（CTE、Closure+UNION）
5. gRPCバリデーション（protovalidate）
6. Admin CLI（rebuild-closures）
7. 依存更新（gRPC v1.79.3, CEL v0.27.0, protovalidate v1.1.3）

### Phase 4（次のステップ）

1. Tenant ID 仕様策定（マルチテナント対応）
2. マルチテナント基本実装（gRPC メタデータ/HTTP ヘッダー）
3. パフォーマンステスト・ベンチマーク

### Phase 5（将来）

1. 分散キャッシュ（Redis）対応
2. スキーママイグレーション機能
3. 管理 UI・CLI

---

## 補足事項

### 設計方針

このプロジェクトは初期開発段階にあるため、後方互換性を考慮せず、Permify との完全互換性を最優先としている。
古い API や型名は削除し、Permify 互換の新しいコードのみを維持している。

### Permify バージョン

この分析は 2025 年 10 月時点の Permify 公式ドキュメントに基づく。

### 連絡先

質問・提案は GitHub Issues へ。
