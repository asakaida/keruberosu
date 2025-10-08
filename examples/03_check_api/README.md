# Example 3: Check API

このサンプルでは、Check API を使用して権限チェックを実行する方法を示します。

## Check API とは

Check API は、特定のユーザーが特定のリソースに対して特定の権限を持っているかどうかを判定します。

### リクエスト

```go
resp, err := client.Check(ctx, &pb.CheckRequest{
    Entity: &pb.Entity{
        Type: "document",
        Id:   "doc1",
    },
    Permission: "edit",
    Subject: &pb.Subject{
        Type: "user",
        Id:   "alice",
    },
})
```

### レスポンス

```go
if resp.Can == pb.CheckResult_CHECK_RESULT_ALLOWED {
    fmt.Println("✅ 許可")
} else {
    fmt.Println("❌ 拒否")
}
```

## 実行方法

```bash
# サーバーが起動していることを確認
cd examples/03_check_api
go run main.go
```

## 前提条件

- サーバーが起動していること
- スキーマとデータが書き込まれていること（Example 1, 2 を先に実行）

## 期待される出力

```
===== Check API テスト =====

テストケース 1: alice が doc1 を編集できるか
✅ alice は doc1 を edit できます（理由: owner）

テストケース 2: bob が doc1 を編集できるか
✅ bob は doc1 を edit できます（理由: editor）

テストケース 3: charlie が doc1 を編集できるか
❌ charlie は doc1 を edit できません

テストケース 4: charlie が doc1 を閲覧できるか
✅ charlie は doc1 を view できます（理由: viewer）

テストケース 5: 誰でも public な doc1 を閲覧できるか
✅ dave は doc1 を view できます（理由: ABAC rule - public == true）
```

## 関連ドキュメント

- [PRD.md](../../PRD.md) - Check API の詳細仕様
- [DESIGN.md](../../DESIGN.md) - 権限評価のアーキテクチャ
