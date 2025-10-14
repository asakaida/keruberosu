# Example 4: ReBAC - Google Docs 風の権限管理

このサンプルでは、Google Docs のような階層的な権限管理を実装します。

## シナリオ

- フォルダには owner と editor がいる
- ドキュメントは フォルダに所属し、フォルダの権限を継承する
- ドキュメント自体にも直接の owner, editor, viewer を設定できる

## スキーマ

```text
entity user {}

entity folder {
  relation owner: user
  relation editor: user

  permission edit = owner or editor
  permission view = owner or editor
}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user
  relation parent: folder

  permission delete = owner
  permission edit = owner or editor
  permission view = owner or editor or viewer or parent.view
}
```

### 階層的権限の継承

`permission view = ... or parent.view`

この定義により、ドキュメントの親フォルダに対する view 権限を持つユーザーは、自動的にそのドキュメントの view 権限も持ちます。

## 実行方法

```bash
cd examples/04_rebac_google_docs
go run main.go
```

## 期待される出力

```
===== ReBAC: Google Docs風の権限管理 =====

alice は folder1 の owner です
bob は folder1 の editor です

✅ alice (owner) は folder1 を edit できます
✅ bob (editor) は folder1 を edit できます
❌ charlie は folder1 を edit できません

doc1 は folder1 に所属しています
doc1 の owner は alice です

✅ alice (owner) は doc1 を delete できます
✅ alice (owner) は doc1 を edit できます
✅ alice (owner) は doc1 を view できます

✅ bob (folder editor) は doc1 を view できます（parent.view 経由）
❌ bob (folder editor) は doc1 を edit できません
❌ bob (folder editor) は doc1 を delete できません
```

## 関連ドキュメント

- [PRD.md](../../PRD.md) - 階層的権限の詳細
- [DESIGN.md](../../DESIGN.md) - HierarchicalRule の実装
