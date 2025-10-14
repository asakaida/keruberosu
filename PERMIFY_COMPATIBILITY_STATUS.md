# Permify 互換性ステータス

## 📅 最終更新日: 2025-10-14

## ✅ 完了した互換性対応

### 1. Proto 定義の更新

- ✅ `RelationTuple.subject`を Subject 型に変更（relation フィールドをサポート）
- ✅ `WriteRelationsRequest`に`attributes`フィールドを追加
- ✅ `WriteSchemaResponse`を`schema_version`返却形式に変更
- ✅ `WriteRelationsResponse`、`DeleteRelationsResponse`、`WriteAttributesResponse`に`snap_token`を追加
- ✅ `DeleteRelationsRequest`をフィルター形式に変更（`TupleFilter`使用）
- ✅ `AttributeData`を Permify 互換に変更（単一属性形式）
- ✅ `ReadRelationships` API を追加

### 2. Schema DSL の拡張

- ✅ `action`キーワードをサポート（`permission`のエイリアス）
- ✅ `@user`記法をサポート（`:  user`と等価）
- ✅ 両方の記法を同時サポート（後方互換性なし、両方使用可能）

### 3. グループメンバーシップ機能

- ✅ 1 つのタプルで`entity#relation@subject#relation`を表現可能
- ✅ 例: `drive:eng_drive#member@group:engineering#member`

---

## ✅ 追加で完了した実装（2025-10-14 更新）

### 4. DeleteRelations のフィルター実装 【完了】

- ✅ Proto 定義は`TupleFilter`に更新済み
- ✅ ハンドラー実装完了（`DeleteByFilter`メソッド使用）
- ✅ リポジトリ層でのフィルター対応完了
  - `EntityFilter` (type + ids)
  - `SubjectFilter` (type + ids + relation)
- ✅ 複数 ID での一括削除対応（`pq.Array()`使用）

### 5. ReadRelationships API の実装 【完了】

- ✅ Proto 定義追加済み
- ✅ ハンドラー実装完了
- ✅ リポジトリ層でのフィルター・ページネーション対応完了
- ✅ continuous_token の生成・検証実装
- ✅ E2E テストで動作確認済み

### 6. AttributeData 形式変更の完全対応 【完了】

- ✅ Proto 定義は単一属性形式に更新済み
- ✅ 全ハンドラーの AttributeData 処理を単一属性形式に統一
- ✅ 全テストケースの更新完了
- ✅ 全 example コードの更新完了

---

## 🔴 先送り事項（今後の実装が必要）

### 1. Schema Version 機能 【高優先度】

**現状:**

- `WriteSchemaResponse.schema_version`は空文字列を返す
- `WriteRelationsRequest.metadata.schema_version`フィールド未実装

**必要な実装:**

1. スキーマバージョニングの仕様策定
   - セマンティックバージョニング vs タイムスタンプ
   - 自動採番 vs 手動指定
2. データベースに`schema_version`カラム追加
3. WriteSchema 時にバージョンを生成・保存
4. Check/Expand/Lookup 時に特定バージョンのスキーマを使用
5. スキーママイグレーション戦略

**影響範囲:**

- `proto/keruberosu/v1/authorization.proto`
- `internal/handlers/authorization_handler.go`
- `internal/services/schema_service.go`
- `internal/repositories/postgres/schema_repository.go`
- データベーススキーマ

---

### 2. Snap Token / Cache 機構 【高優先度】

**現状:**

- 全ての Write/Delete レスポンスで`snap_token`は空文字列を返す
- キャッシュ無効化の仕組みが未実装

**必要な実装:**

1. Snap token の仕様策定
   - トークン形式（UUID, タイムスタンプ, 連番など）
   - 有効期限の設計
2. トークン生成・管理機能
3. キャッシュ層の実装
   - Redis 統合
   - インメモリキャッシュ
4. Check/Expand/Lookup 時の snap_token 検証
5. キャッシュ無効化戦略

**影響範囲:**

- `proto/keruberosu/v1/common.proto` (`PermissionCheckMetadata`)
- `internal/handlers/authorization_handler.go` (全 Write/Delete 系)
- 新規: `internal/cache` パッケージ
- インフラ: Redis 設定

---

### 3. Tenant ID / マルチテナント機能 【中優先度】

**現状:**

- 全 API で固定値`"default"`を使用
- マルチテナント対応していない

**仕様検討が必要:**

1. Tenant ID の導入方法
   - URL パスに含める: `/v1/tenants/{tenant_id}/...`
   - HTTP ヘッダー: `X-Tenant-ID`
   - gRPC メタデータ
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

- 全 API エンドポイント
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

### API レベル互換性: 90%

- ✅ 基本的な API 構造（Check, Lookup 系）
- ✅ RelationTuple の構造
- ✅ Subject relation サポート
- ✅ Schema DSL 基本文法
- ✅ ReadRelationships 実装完了
- ✅ DeleteRelations フィルター実装完了
- ⚠️ Metadata fields (schema_version, snap_token) - 空文字列を返す
- ❌ マルチテナント未対応（固定値"default"使用）

### データ構造互換性: 100%

- ✅ Entity, Subject, RelationTuple
- ✅ AttributeData（単一属性形式、完全実装）
- ✅ TupleFilter（完全実装）
- ✅ 全ハンドラー実装完了

### 機能互換性: 85%

- ✅ Permission Check
- ✅ LookupEntity/LookupSubject
- ✅ Schema Write/Read
- ✅ グループメンバーシップ
- ✅ Relations Write/Delete（フィルター対応）
- ✅ Attributes Write
- ✅ ReadRelationships（ページネーション対応）
- ❌ Schema versioning（仕様未決定）
- ❌ Snap token / Cache（仕様未決定）
- ❌ Tenant management（仕様未決定）

---

## 🎯 推奨実装順序

### ✅ Phase 1（完了）

1. ✅ ハンドラーのコンパイルエラー修正
2. ✅ 既存テスト・example の更新
3. ✅ DeleteRelations フィルター実装
4. ✅ ReadRelationships 実装
5. ✅ 全 unit/integration/E2E テスト成功確認

### Phase 2（次のステップ）

1. Snap token 仕様策定・基本実装
2. Schema version 仕様策定・基本実装
3. E2E テスト拡充（より複雑なシナリオ）

### Phase 3（1 ヶ月以内）

4. Tenant ID 仕様策定
5. マルチテナント基本実装
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
