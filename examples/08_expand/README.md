# Example 08: Expand API - Permission Tree Visualization

この例では、Keruberos の**Expand API**を使用して、権限決定ツリーを可視化する実践的な方法を示します。

## 概要

Expand API は、特定の権限がどのように決定されるかを**ツリー構造**で返します。これにより、以下が可能になります：

- 🐛 **デバッグ**: なぜアクセスが拒否されたのか？
- 📊 **監査**: リソースへのアクセス経路を可視化
- ✅ **検証**: 権限ルールが意図通りに動作しているか確認
- 📚 **ドキュメント**: 複雑な権限ロジックをチームに説明

## シナリオ

GitHub 風の**organization → repository → issue**の 3 階層構造を使用します。

### エンティティ構造

#### Organization

- **acme-corp**
  - admin: alice
  - member: bob, charlie

#### Repositories

- **backend-api** (プライベート)

  - parent: acme-corp
  - owner: bob
  - maintainer: charlie

- **frontend** (パブリック)
  - parent: acme-corp
  - owner: alice
  - contributor: dave

#### Issues

- **backend-api/issue-1** (機密)

  - parent: backend-api
  - assignee: bob

- **frontend/issue-2** (非機密)
  - parent: frontend
  - reporter: dave

### 権限ルール

#### Repository

```
permission view = owner or maintainer or contributor or
  (parent.view and rule(!resource.private))
```

- プライベートリポジトリ: 直接的な役割のみ
- パブリックリポジトリ: 直接的な役割 + organization メンバー

#### Issue

```
permission view = (assignee or reporter) or
  (parent.view and rule(!resource.confidential))
```

- 機密 Issue: assignee と reporter のみ
- 非機密 Issue: 上記 + リポジトリ閲覧権限を継承

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
cd examples/08_expand
go run main.go
```

## 出力の見方

Expand API は権限決定ツリーを返します。ツリーには以下のノードがあります：

### ノードタイプ

#### 🔀 結合ノード（Operation）

- **union（OR）**: いずれかの条件が満たされれば OK
- **intersection（AND）**: 全ての条件が満たされる必要あり
- **exclusion（EXCLUDE）**: 特定の条件を除外

#### 🍃 リーフノード（Leaf）

実際の関係やルール評価の結果を表します：

- 直接的な関係: `user:alice`, `user:bob#member`
- サブジェクト参照: `organization#member` など

#### 🔄 Rewrite ノード

ルールベースの権限評価を表します。

## 出力例

### パブリックリポジトリの閲覧権限

```
repository:frontend#view の権限ツリー:

🔀 結合（OR）
  🍃 直接的な関係:
     - user:alice          (owner)
     - user:dave           (contributor)
  🔀 結合（AND）
    🍃 参照先: organization:acme-corp#view
    🔄 ルール評価 (!resource.private)
```

**解釈**:

- alice と dave は直接的な役割で閲覧可能
- 他のユーザーは`parent.view`（organization 閲覧権限）と`!resource.private`の両方を満たせば閲覧可能
- frontend はパブリックなので、bob, charlie も閲覧可能

### プライベートリポジトリの閲覧権限

```
repository:backend-api#view の権限ツリー:

🔀 結合（OR）
  🍃 直接的な関係:
     - user:bob            (owner)
     - user:charlie        (maintainer)
```

**解釈**:

- backend-api はプライベート（private=true）
- ルール`!resource.private`が false になるため、`parent.view`の分岐は除外される
- bob と charlie のみが閲覧可能

### 非機密 Issue の閲覧権限（再帰的展開）

```
issue:issue-2#view の権限ツリー:

🔀 結合（OR）
  🍃 直接的な関係:
     - user:dave           (reporter)
  🔀 結合（AND）
    🔀 結合（OR）           (parent.view = repository:frontend#view)
      🍃 直接的な関係:
         - user:alice      (repo owner)
         - user:dave       (repo contributor)
      🔀 結合（AND）
        🍃 参照先: organization:acme-corp#view
        🔄 ルール評価 (!resource.private)
    🔄 ルール評価 (!resource.confidential)
```

**解釈**:

- dave は reporter として直接閲覧可能
- issue-2 は非機密なので、repository の閲覧権限を継承
- repository のツリーが再帰的に展開される

## 実践的な使用例

### 1. デバッグ: アクセス拒否の理由を調査

```go
// aliceがbackend-apiを閲覧できない理由を調査
checkResp, _ := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
    Entity:     &pb.Entity{Type: "repository", Id: "backend-api"},
    Permission: "view",
    Subject:    &pb.Subject{Type: "user", Id: "alice"},
})

if checkResp.Can == pb.CheckResult_CHECK_RESULT_DENIED {
    // Expand APIでツリーを確認
    expandResp, _ := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
        Entity:     &pb.Entity{Type: "repository", Id: "backend-api"},
        Permission: "view",
    })

    printExpandTree(expandResp.Tree, 0)
    // => backend-apiはprivateなので、parent.view分岐が除外されている
    // => aliceはowner/maintainerでもないため、アクセス不可
}
```

### 2. 監査: リソースへのアクセス経路を可視化

```go
// frontendリポジトリに誰がどう経由してアクセスできるか
expandResp, _ := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
    Entity:     &pb.Entity{Type: "repository", Id: "frontend"},
    Permission: "view",
})

printExpandTree(expandResp.Tree, 0)
// => 直接的な役割: alice, dave
// => parent.view経由: bob, charlie
```

### 3. 検証: 権限ルールの動作確認

```go
// 機密Issueと非機密Issueの違いを確認
expandResp1, _ := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
    Entity:     &pb.Entity{Type: "issue", Id: "issue-1"}, // 機密
    Permission: "view",
})

expandResp2, _ := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
    Entity:     &pb.Entity{Type: "issue", Id: "issue-2"}, // 非機密
    Permission: "view",
})

// issue-1: assigneeとreporterのみ（parent.view除外）
// issue-2: 上記 + parent.viewの再帰展開
```

## ツリーの読み方のコツ

### OR（union）ノード

いずれか 1 つでも満たされればアクセス許可：

```
🔀 結合（OR）
  🍃 user:alice
  🍃 user:bob
```

→ alice または bob なら OK

### AND（intersection）ノード

全ての条件を満たす必要あり：

```
🔀 交差（AND）
  🍃 organization:acme-corp#member
  🔄 ルール評価 (security_level >= 3)
```

→ acme-corp のメンバーかつセキュリティレベル 3 以上が必要

### 再帰的展開

`parent.view`などの参照は、参照先の権限ツリーに再帰的に展開されます：

```
issue#view
  └─ parent.view (repository#view)
       └─ parent.view (organization#view)
```

## 学習ポイント

1. **Expand vs Check**:

   - Check: アクセス可否のみを返す（速い）
   - Expand: 決定理由を返す（詳細だが重い）

2. **デバッグ時の活用**:

   - 本番環境では Check を使用
   - 開発・テスト時に Expand でロジック検証

3. **複雑なルールの可視化**:

   - `parent.view`の再帰的継承
   - ABAC 条件（`rule(...)`）の評価結果
   - OR/AND の組み合わせ

4. **パフォーマンス考慮**:
   - Expand は計算コストが高い
   - 必要な場合のみ使用（デバッグ、監査）
   - 通常の権限チェックには Check を使用

## API パラメータ（Permify 互換）

### Expand API のパラメータ

Expand API では以下のフィールドが使用可能です：

- **tenant_id** (string, optional): テナント識別子。空の場合は "default" を使用。将来のマルチテナント対応に備えた設計。
- **entity** (Entity, required): 権限を展開する対象エンティティ
- **permission** (string, required): 展開する権限名
- **context** (Context, optional): コンテキスト情報（contextual tuples, attributes）
- **arguments** (repeated Value, optional): パラメータ付き権限用の引数

基本的な使用例：

```go
expandResp, _ := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
    TenantId:   "default",  // オプション（空でも可）
    Entity:     &pb.Entity{Type: "repository", Id: "backend-api"},
    Permission: "view",
})
```

### 新しい Expand レスポンス構造（Permify 完全互換）

Expand API のレスポンスは、Permify に完全準拠した構造を返します：

```go
type Expand struct {
    oneof node {
        ExpandTreeNode expand = 1;  // ツリーノード（OR/AND/EXCLUDE）
        ExpandLeaf leaf = 2;         // リーフノード（具体的な関係）
    }
}

type ExpandTreeNode struct {
    enum Operation {
        OPERATION_UNION = 1;         // OR結合
        OPERATION_INTERSECTION = 2;  // AND結合
        OPERATION_EXCLUSION = 3;     // 除外
    }
    Operation operation = 1;
    repeated Expand children = 2;    // 再帰的な子ノード
}

type ExpandLeaf struct {
    oneof type {
        Subjects subjects = 1;       // サブジェクトリスト
        Values values = 2;           // 値マップ
        Any value = 3;               // 単一値
    }
}
```

レスポンスの処理例：

```go
if treeNode := expandResp.Tree.GetExpand(); treeNode != nil {
    // ツリーノードの場合
    switch treeNode.Operation {
    case pb.ExpandTreeNode_OPERATION_UNION:
        fmt.Println("OR結合")
    case pb.ExpandTreeNode_OPERATION_INTERSECTION:
        fmt.Println("AND結合")
    }
    // 子ノードを再帰的に処理
    for _, child := range treeNode.Children {
        processNode(child)
    }
} else if leafNode := expandResp.Tree.GetLeaf(); leafNode != nil {
    // リーフノードの場合
    if subjects := leafNode.GetSubjects(); subjects != nil {
        for _, subject := range subjects.Subjects {
            fmt.Printf("- %s:%s\n", subject.Type, subject.Id)
        }
    }
}
```

## 関連ドキュメント

- [Example 07: List APIs](../07_list_apis/README.md) - LookupEntity/LookupSubject/SubjectPermission
- [Keruberosu API Reference](../../docs/API.md)
- [Permify Compatibility](../../PERMIFY_COMPATIBILITY_STATUS.md)
