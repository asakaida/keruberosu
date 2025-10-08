# Example 1: スキーマ定義

このサンプルでは、Keruberosu のスキーマを定義する方法を示します。

## スキーマ DSL

Keruberosu は Permify 互換のスキーマ DSL をサポートしています。

### 基本的なエンティティ定義

```text
entity user {}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user

  permission edit = owner or editor
  permission view = owner or editor or viewer
}
```

### 構成要素

1. **entity**: リソースやユーザーなどのエンティティタイプを定義
2. **relation**: エンティティ間の関係を定義（例: owner, editor）
3. **permission**: 権限ルールを定義（relations を組み合わせて表現）

## 実行方法

```bash
# サーバーが起動していることを確認
cd examples/01_schema_definition
go run main.go
```

## 期待される出力

```
✅ スキーマが正常に書き込まれました
```

## 関連ドキュメント

- [PRD.md](../../PRD.md) - スキーマ DSL の詳細仕様
- [DESIGN.md](../../DESIGN.md) - アーキテクチャ設計
