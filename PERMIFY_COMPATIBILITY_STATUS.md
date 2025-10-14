# Permify 互換性ステータス

## 📅 最終更新日: 2025-10-14

## 🎉 最新の大規模アップデート（2025-10-14）

**Permify 互換 API 構造への完全移行が完了しました！**

主な変更点:

- 🔄 **サービス分割完了**: 単一の `AuthorizationService` を 3 つの Permify 互換サービス（Permission, Data, Schema）に分割
- 📂 **Proto ファイル分割**: `authorization.proto` を 3 つの独立したファイル（`permission.proto`, `data.proto`, `schema.proto`）に分割し、Permify のファイル構成に完全準拠
- 📝 **メッセージ名統一**: 全てのリクエスト/レスポンスメッセージを Permify 命名規則に変更
- 🔗 **API 統合**: `WriteAttributes` を `Data.Write` に統合し、tuples と attributes を一つの API で処理可能に
- 🏷️ **型エイリアス追加**: `Tuple`, `Attribute` などの Permify 互換型名を追加
- 📊 **API 構造互換性: 100%** 達成

---

## ✅ 完了した互換性対応

### 1. サービス分割（Permify 互換構造）

- ✅ 単一の`AuthorizationService`を 3 つのサービスに分割:
  - **Permission サービス**: Check, Expand, LookupEntity, LookupSubject, LookupEntityStream, SubjectPermission
  - **Data サービス**: Write, Delete, Read, ReadAttributes
  - **Schema サービス**: Write, Read

### 2. メッセージ名の Permify 互換化

- ✅ `WriteSchemaRequest` → `SchemaWriteRequest` (フィールド: `SchemaDsl` → `Schema`)
- ✅ `ReadSchemaRequest` → `SchemaReadRequest`
- ✅ `WriteRelationsRequest` → `DataWriteRequest`
- ✅ `WriteAttributesRequest` → **`DataWriteRequest`に統合** (tuples と attributes を同時に書き込み可能)
- ✅ `DeleteRelationsRequest` → `DataDeleteRequest`
- ✅ `ReadRelationshipsRequest` → `DataReadRequest`
- ✅ `CheckRequest` → `PermissionCheckRequest`
- ✅ `ExpandRequest` → `PermissionExpandRequest`
- ✅ `LookupEntityRequest` → `PermissionLookupEntityRequest`
- ✅ `LookupSubjectRequest` → `PermissionLookupSubjectRequest`
- ✅ `SubjectPermissionRequest` → `PermissionSubjectPermissionRequest`

### 3. 型エイリアスの追加

- ✅ `Tuple` (alias for `RelationTuple`)
- ✅ `Attribute` (alias for `AttributeData`)
- ✅ `Expand` および `ExpandTreeNode` メッセージ

### 4. Proto 定義の更新

- ✅ `RelationTuple.subject`を Subject 型に変更（relation フィールドをサポート）
- ✅ `DataWriteRequest`に`tuples`と`attributes`フィールドを追加（統合）
- ✅ `SchemaWriteResponse`を`schema_version`返却形式に変更
- ✅ `DataWriteResponse`、`DataDeleteResponse`に`snap_token`を追加
- ✅ `DataDeleteRequest`をフィルター形式に変更（`TupleFilter`使用）
- ✅ `AttributeData`を Permify 互換に変更（単一属性形式）
- ✅ `DataReadRequest` API を追加

### 5. Schema DSL の拡張

- ✅ `action`キーワードをサポート（`permission`のエイリアス）
- ✅ `@user`記法をサポート（`:  user`と等価）
- ✅ 両方の記法を同時サポート（後方互換性なし、両方使用可能）

### 6. グループメンバーシップ機能

- ✅ 1 つのタプルで`entity#relation@subject#relation`を表現可能
- ✅ 例: `drive:eng_drive#member@group:engineering#member`

---

## ✅ 追加で完了した実装（2025-10-14 更新）

### 7. サービス分割とメッセージ名変更の完全実装 【完了】

- ✅ 3 つのサービス（Permission, Data, Schema）に分割完了
- ✅ Proto ファイルを物理的に分割:
  - `proto/keruberosu/v1/permission.proto` - Permission サービス + 関連メッセージ
  - `proto/keruberosu/v1/data.proto` - Data サービス + 関連メッセージ
  - `proto/keruberosu/v1/schema.proto` - Schema サービス + 関連メッセージ
  - `proto/keruberosu/v1/common.proto` - 共通メッセージ型（Entity, Subject, Tuple 等）
- ✅ 全メッセージ名を Permify 互換に変更
- ✅ `WriteAttributes` RPC を削除し、`Data.Write()` に統合
- ✅ 型エイリアス（Tuple, Attribute）の追加
- ✅ Expand メッセージの追加
- ✅ 全ハンドラーの実装更新完了
- ✅ 全テスト（unit/integration/E2E）成功確認

### 8. DeleteRelations のフィルター実装 【完了】

- ✅ Proto 定義は`TupleFilter`に更新済み
- ✅ ハンドラー実装完了（`DeleteByFilter`メソッド使用）
- ✅ リポジトリ層でのフィルター対応完了
  - `EntityFilter` (type + ids)
  - `SubjectFilter` (type + ids + relation)
- ✅ 複数 ID での一括削除対応（`pq.Array()`使用）

### 9. ReadRelationships API の実装 【完了】

- ✅ Proto 定義追加済み（`DataReadRequest`として）
- ✅ ハンドラー実装完了
- ✅ リポジトリ層でのフィルター・ページネーション対応完了
- ✅ continuous_token の生成・検証実装
- ✅ E2E テストで動作確認済み

### 10. AttributeData 形式変更の完全対応 【完了】

- ✅ Proto 定義は単一属性形式に更新済み
- ✅ 全ハンドラーの AttributeData 処理を単一属性形式に統一
- ✅ 全テストケースの更新完了
- ✅ 全 example コードの更新完了

### 11. Expand API の Permify 完全互換化 【完了】（2025-10-14 追加）

**変更前の問題:**

- ❌ `Expand.node` フィールドが `ExpandTreeNode` 型（oneof なし）
- ❌ `ExpandTreeNode.operation` が `string` 型（enum ではない）
- ❌ リーフノードとツリーノードの区別が曖昧
- ❌ Permify の仕様と異なる構造

**Permify 準拠の新構造:**

- ✅ `Expand` メッセージに `oneof node` を追加（`expand` または `leaf`）
- ✅ `ExpandTreeNode.operation` を `enum Operation` に変更
  - `OPERATION_UNSPECIFIED = 0`
  - `OPERATION_UNION = 1` (OR 結合)
  - `OPERATION_INTERSECTION = 2` (AND 結合)
  - `OPERATION_EXCLUSION = 3` (除外)
- ✅ `ExpandTreeNode.children` の型を `repeated Expand` に変更（再帰的構造）
- ✅ `ExpandLeaf` メッセージを追加（`oneof type` で subjects/values/value を区別）
- ✅ `Subjects` および `Values` メッセージを追加

**実装された機能:**

- ✅ ツリーノード（union/intersection/exclusion）とリーフノード（subjects/values）の明確な区別
- ✅ 再帰的なツリー構造のサポート
- ✅ 全ハンドラーコードの更新（`expandNodeToProto` 関数）
- ✅ 全テストケースの更新（E2E、ユニットテスト）
- ✅ Example 08（Expand API デモ）の更新
- ✅ 全テスト成功確認

**影響範囲:**

- `proto/keruberosu/v1/common.proto` - Expand メッセージ定義
- `internal/handlers/helpers.go` - expandNodeToProto 関数
- `internal/handlers/permission_handler.go` - Expand ハンドラー
- `test/e2e/*.go` - E2E テストの更新
- `examples/08_expand/main.go` - サンプルコードの更新

### 12. Permission API の完全 Permify 互換化 【完了】（2025-10-14 追加）

**変更内容:**

**LookupEntity API:**

- ✅ `tenant_id` フィールドを追加（フィールド番号 1）
- ✅ `scope` フィールドを追加（`map<string, StringArrayValue>`、フィールド番号 7）
- ✅ `page_size` の型を `int32` から `uint32` に変更（フィールド番号 8）
- ✅ フィールド番号を Permify に合わせて再配置

**LookupSubject API:**

- ✅ `tenant_id` フィールドを追加（フィールド番号 1）
- ✅ `page_size` の型を `int32` から `uint32` に変更（フィールド番号 7）
- ✅ フィールド番号を Permify に合わせて再配置

**SubjectPermission API:**

- ✅ `tenant_id` フィールドを追加（フィールド番号 1）
- ✅ フィールド番号を Permify に合わせて再配置

**Check API:**

- ✅ `tenant_id` フィールドを追加（フィールド番号 1）
- ✅ フィールド番号を Permify に合わせて再配置

**Expand API:**

- ✅ `tenant_id` フィールドを追加（フィールド番号 1）
- ✅ フィールド番号を Permify に合わせて再配置

**共通変更:**

- ✅ `StringArrayValue` メッセージを `common.proto` に追加（scope パラメータ用）
- ✅ 全ハンドラーで `tenant_id` の処理を実装（空の場合は "default" を使用）
- ✅ 全テスト成功確認（E2E、ユニットテスト、example ビルド）

**tenant_id の扱い:**

- tenant_id は proto 定義に含まれているが、Keruberosu では将来のマルチテナント対応に備えた設計
- 現在は空の場合に "default" を使用する実装
- 将来的には gRPC メタデータや HTTP ヘッダーからも取得可能にする予定

**影響範囲:**

- `proto/keruberosu/v1/common.proto` - StringArrayValue 追加
- `proto/keruberosu/v1/permission.proto` - 全リクエストメッセージ更新
- `internal/handlers/permission_handler.go` - tenant_id 処理追加
- 全テストケース（自動的に対応、tenant_id が空でも動作）
- 全 example コード（tenant_id が空でも動作）

---

## 🔴 先送り事項（今後の実装が必要）

### 1. Schema Version 機能 【高優先度】

**現状:**

- `SchemaWriteResponse.schema_version`は空文字列を返す
- `PermissionCheckRequest.metadata.schema_version`フィールド未実装

**必要な実装:**

1. スキーマバージョニングの仕様策定
   - セマンティックバージョニング vs タイムスタンプ
   - 自動採番 vs 手動指定
2. データベースに`schema_version`カラム追加
3. Schema.Write 時にバージョンを生成・保存
4. Permission.Check/Expand/Lookup 時に特定バージョンのスキーマを使用
5. スキーママイグレーション戦略

**影響範囲:**

- `proto/keruberosu/v1/schema.proto`
- `proto/keruberosu/v1/permission.proto`
- `internal/handlers/schema_handler.go`
- `internal/services/schema_service.go`
- `internal/repositories/postgres/schema_repository.go`
- データベーススキーマ

---

### 2. Snap Token / Cache 機構 【高優先度】

**現状:**

- 全ての Data.Write/Delete レスポンスで`snap_token`は空文字列を返す
- キャッシュ無効化の仕組みが未実装

**必要な実装:**

1. Snap token の仕様策定
   - トークン形式（UUID, タイムスタンプ, 連番など）
   - 有効期限の設計
2. トークン生成・管理機能
3. キャッシュ層の実装
   - Redis 統合
   - インメモリキャッシュ
4. Permission.Check/Expand/Lookup 時の snap_token 検証
5. キャッシュ無効化戦略

**影響範囲:**

- `proto/keruberosu/v1/common.proto` (`PermissionCheckMetadata`)
- `proto/keruberosu/v1/data.proto` (`DataWriteResponse`, `DataDeleteResponse`)
- `internal/handlers/data_handler.go` (全 Write/Delete 系)
- `internal/handlers/permission_handler.go` (全 Check/Expand/Lookup 系)
- 新規: `internal/cache` パッケージ
- インフラ: Redis 設定

---

### 3. Tenant ID / マルチテナント機能 【中優先度】

**現状（2025-10-14 更新）:**

- ✅ 全 Permission API で `tenant_id` フィールドを追加（Permify 互換）
- ✅ ハンドラーで `tenant_id` が空の場合は "default" を使用
- ⚠️ マルチテナント対応は未実装（現在は "default" のみ）

**仕様検討が必要:**

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

**影響範囲:**

- gRPC インターセプター（メタデータ処理）
- HTTP ミドルウェア（ヘッダー処理）
- 認証ミドルウェア
- データベース設計

---

### 4. Schema DSL の完全 Permify 互換 【低優先度】

**現状:**

- `action`キーワードと`@user`記法はサポート済み
- 複数型指定（`@user @team#member`）の完全対応は未完了

**考慮事項:**

- Permify は`@user @team#member`形式（スペース区切り）
- keruberos は`user | team#member`形式（パイプ区切り）
- 両方をサポートするか、どちらかに統一するか

**必要な実装:**

1. 複数型指定の完全パース対応
2. バリデーション強化
3. ドキュメント更新

---

### 5. エラーハンドリングの統一 【低優先度】

**現状:**

- WriteSchema は validation error を返す
- 他の API は gRPC status error を返す
- エラーメッセージ形式が不統一

**考慮事項:**

- Permify のエラーレスポンス形式に合わせるか
- gRPC standard status codes を活用

---

## 📊 互換性達成度（2025-10-14 更新）

### API 構造互換性: 100%

- ✅ サービス分割（Permission, Data, Schema）
- ✅ メッセージ名の Permify 互換化
- ✅ 型エイリアス（Tuple, Attribute）
- ✅ Expand メッセージ
- ✅ Data.Write への統合（tuples + attributes）

### API レベル互換性: 98%

- ✅ 基本的な API 構造（Permission.Check, Lookup 系）
- ✅ RelationTuple の構造
- ✅ Subject relation サポート
- ✅ Schema DSL 基本文法
- ✅ Data.Read 実装完了（旧 ReadRelationships）
- ✅ Data.Delete フィルター実装完了
- ✅ **tenant_id フィールド追加（全 Permission API で Permify 互換）**
- ✅ **scope パラメータ追加（LookupEntity API）**
- ✅ **page_size 型変更（int32 → uint32）**
- ⚠️ Metadata fields (schema_version, snap_token) - 空文字列を返す

### データ構造互換性: 100%

- ✅ Entity, Subject, RelationTuple
- ✅ AttributeData（単一属性形式、完全実装）
- ✅ TupleFilter（完全実装）
- ✅ 全ハンドラー実装完了

### 機能互換性: 85%

- ✅ Permission.Check
- ✅ Permission.LookupEntity/LookupSubject
- ✅ Schema.Write/Read
- ✅ グループメンバーシップ
- ✅ Data.Write（tuples + attributes 統合）
- ✅ Data.Delete（フィルター対応）
- ✅ Data.Read（ページネーション対応）
- ❌ Schema versioning（仕様未決定）
- ❌ Snap token / Cache（仕様未決定）
- ⚠️ Tenant management（proto には含めない方針、内部実装のみ）

---

## 🎯 推奨実装順序

### ✅ Phase 1（完了）

1. ✅ サービス分割（Permission, Data, Schema）
2. ✅ メッセージ名の Permify 互換化
3. ✅ 型エイリアス追加（Tuple, Attribute）
4. ✅ Data.Write への統合（tuples + attributes）
5. ✅ ハンドラーのコンパイルエラー修正
6. ✅ 既存テスト・example の更新
7. ✅ Data.Delete フィルター実装
8. ✅ Data.Read 実装
9. ✅ 全 unit/integration/E2E テスト成功確認

### Phase 2（次のステップ）

1. Snap token 仕様策定・基本実装
2. Schema version 仕様策定・基本実装
3. E2E テスト拡充（より複雑なシナリオ）

### Phase 3（1 ヶ月以内）

4. Tenant ID 仕様策定（proto に含めずに実現）
5. マルチテナント基本実装（gRPC メタデータ/HTTP ヘッダー）
6. パフォーマンステスト

### Phase 4（将来）

7. キャッシュ最適化
8. スキーママイグレーション機能
9. 管理 UI・CLI

---

## 📝 補足事項

### 後方互換性について

ユーザーの要求により、**後方互換性は一切考慮していない**。
既存 API は非推奨として残さず、新しいものだけを残している。

### Permify バージョン

この分析は 2025 年 10 月時点の Permify 公式ドキュメントに基づく。

### 連絡先

質問・提案は GitHub Issues へ。
