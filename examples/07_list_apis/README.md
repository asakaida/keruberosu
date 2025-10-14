# Example 07: List APIs (LookupEntity, LookupSubject, SubjectPermission)

この例では、Keruberos の一覧取得系 API（LookupEntity、LookupSubject、SubjectPermission）の実践的な使用方法を示します。

## 概要

企業のドキュメント管理システムを想定したシナリオで、以下の 3 つの API の使い分けを学習できます：

- **LookupEntity**: ユーザーがアクセスできるリソース一覧を取得
- **LookupSubject**: リソースにアクセスできるユーザー一覧を取得
- **SubjectPermission**: 特定のユーザー・リソースの組み合わせで持つ全権限を取得

## シナリオ

### エンティティ構造

- **5 人のユーザー**:

  - alice (営業部, セキュリティレベル 2)
  - bob (エンジニアリング部, セキュリティレベル 3)
  - charlie (営業部, セキュリティレベル 4)
  - dave (人事部, セキュリティレベル 2)
  - eve (エンジニアリング部, セキュリティレベル 5)

- **3 つの部署**:

  - sales (営業部): alice, charlie
  - engineering (エンジニアリング部): bob, eve
  - hr (人事部): dave

- **5 つのドキュメント**:
  - doc1: 営業部のパブリックドキュメント（非機密）
  - doc2: alice が所有する営業部ドキュメント（機密レベル 3）
  - doc3: エンジニアリング部のパブリックドキュメント（非機密）
  - doc4: bob が所有するエンジニアリング部ドキュメント（機密レベル 4）
  - doc5: 人事部の機密ドキュメント（機密レベル 5）

### 権限ルール

```
permission view = owner or editor or viewer or
  (department.member and rule(!resource.confidential or subject.security_level >= 3))
```

この複雑なルールは、以下を実現します：

1. 直接的な権限（owner, editor, viewer）を持つユーザーは常にアクセス可能
2. 同じ部署のメンバーは以下の条件でアクセス可能：
   - ドキュメントが非機密（confidential=false）の場合、または
   - ユーザーのセキュリティレベルが十分高い場合

## 実行方法

### 前提条件

Keruberos サーバーが起動していることを確認してください：

```bash
# ターミナル1: サーバー起動
cd /path/to/keruberosu
go run cmd/server/main.go
```

### 例の実行

```bash
# ターミナル2: 例の実行
cd examples/07_list_apis
go run main.go
```

## 出力例

### LookupEntity（ユーザーごとのアクセス可能ドキュメント）

```
alice がアクセスできるドキュメント:
  - doc1 (営業部パブリック)
  - doc2 (aliceの所有)
```

### LookupSubject（ドキュメントごとのアクセス可能ユーザー）

```
doc1 にアクセスできるユーザー:
  - alice (営業部メンバー)
  - charlie (営業部メンバー)
```

### SubjectPermission（特定の組み合わせの権限）

```
alice が doc2 に対して持つ権限:
  ✓ view: 可能 (owner)
  ✓ edit: 可能 (owner)
  ✓ delete: 可能 (owner)
```

## ユースケース

### 1. ダッシュボード機能

ユーザーがアクセスできるドキュメント一覧を表示：

```go
lookupResp, _ := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
    EntityType: "document",
    Permission: "view",
    Subject:    &pb.Subject{Type: "user", Id: "alice"},
})
// dashboard.ShowDocuments(lookupResp.EntityIds)
```

### 2. アクセス監査

特定ドキュメントにアクセスできるユーザーを監査：

```go
lookupResp, _ := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
    Entity:           &pb.Entity{Type: "document", Id: "doc5"},
    Permission:       "view",
    SubjectReference: &pb.SubjectReference{Type: "user"},
})
// audit.LogAccess(lookupResp.SubjectIds)
```

### 3. UI 制御

ユーザーが特定ドキュメントで実行できる操作を判定：

```go
subjPermResp, _ := permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
    Entity:  &pb.Entity{Type: "document", Id: "doc1"},
    Subject: &pb.Subject{Type: "user", Id: "alice"},
})

// UI制御
showEditButton := subjPermResp.Results["edit"] == pb.CheckResult_CHECK_RESULT_ALLOWED
showDeleteButton := subjPermResp.Results["delete"] == pb.CheckResult_CHECK_RESULT_ALLOWED
```

## 学習ポイント

1. **LookupEntity vs Check**:

   - Check: 1 つのリソースへのアクセス判定（速い）
   - LookupEntity: 全リソースから絞り込み（一括取得）

2. **LookupSubject の活用**:

   - アクセス監査レポート
   - 共有設定 UI（「誰がアクセスできるか」表示）

3. **SubjectPermission の効率性**:

   - 複数の権限を 1 回の API で取得
   - UI 制御で複数ボタンの表示判定を最適化

4. **ReBAC + ABAC の組み合わせ**:
   - 部署メンバーシップ（ReBAC）
   - セキュリティレベル・機密性（ABAC）
   - 柔軟で強力な権限モデル

## API パラメータ（Permify 互換）

### 共通パラメータ

すべての Permission API で以下のフィールドが使用可能です：

- **tenant_id** (string, optional): テナント識別子。空の場合は "default" を使用。将来のマルチテナント対応に備えた設計。

### LookupEntity 固有パラメータ

- **scope** (map<string, StringArrayValue>, optional): エンティティタイプごとの ID リスト。検索対象を特定のエンティティに限定する場合に使用。
- **page_size** (uint32, optional): 1 ページあたりの結果数（1-100 推奨）
- **continuous_token** (string, optional): ページネーション用トークン

例：

```go
lookupResp, _ := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
    TenantId:   "default",  // オプション（空でも可）
    EntityType: "document",
    Permission: "view",
    Subject:    &pb.Subject{Type: "user", Id: "alice"},
    Scope: map[string]*pb.StringArrayValue{
        "document": {Data: []string{"doc1", "doc2", "doc3"}},
    },
    PageSize: 10,
})
```

### LookupSubject 固有パラメータ

- **page_size** (uint32, optional): 1 ページあたりの結果数
- **continuous_token** (string, optional): ページネーション用トークン

### SubjectPermission

- 追加パラメータなし（tenant_id のみ）

## 関連ドキュメント

- [Example 08: Expand API](../08_expand/README.md) - 権限ツリーの可視化
- [Keruberosu API Reference](../../docs/API.md)
- [Permify Compatibility](../../PERMIFY_COMPATIBILITY_STATUS.md)
