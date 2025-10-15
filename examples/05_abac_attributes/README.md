# Example 5: ABAC - 属性ベースアクセス制御

このサンプルでは、属性ベースアクセス制御 (ABAC) を使用して、動的なルールに基づく権限管理を実装します。

## ABAC とは

ABAC (Attribute-Based Access Control) は、エンティティやユーザーの属性に基づいて権限を判定する方式です。

## シナリオ

- `public ドキュメント`: 誰でも閲覧可能
- `セキュリティレベル`: ユーザーのセキュリティレベルがドキュメントのレベル以上なら閲覧可能
- `部署制限`: 同じ部署のユーザーのみ閲覧可能
- `プレミアムコンテンツ`: プレミアムユーザーのみ閲覧可能

## スキーマ（CEL 式を使用）

```text
rule is_public(resource) {
  resource.public == true
}

rule has_sufficient_clearance(subject, resource) {
  subject.security_level >= resource.security_level
}

rule same_department(subject, resource) {
  resource.department == subject.department
}

rule is_premium_content(subject, resource) {
  subject.subscription_tier == "premium" && resource.price > 100
}

entity document {
  attribute public boolean
  attribute security_level integer
  attribute department string
  attribute price integer

  // 誰でも公開ドキュメントを閲覧可能
  permission view_public = is_public(resource)

  // セキュリティレベルチェック
  permission view_classified = has_sufficient_clearance(subject, resource)

  // 同じ部署のみ
  permission view_department = same_department(subject, resource)

  // プレミアムユーザー向け高額コンテンツ
  permission view_premium = is_premium_content(subject, resource)
}
```

## CEL (Common Expression Language)

Keruberosu は Google の CEL を使用して、柔軟な条件式を記述できます。

### サポートされる演算子

- 比較: `==`, `!=`, `<`, `<=`, `>`, `>=`
- 論理: `&&`, `||`, `!`
- その他: `in` (配列の要素チェック)

## 実行方法

```bash
cd examples/05_abac_attributes
go run main.go
```

## 期待される出力

```
===== ABAC: 属性ベースアクセス制御 =====

テストケース 1: 公開ドキュメント
✅ 誰でも public=true の doc1 を閲覧できます

テストケース 2: セキュリティレベル
✅ alice (level=3) は doc2 (level=2) を閲覧できます
❌ bob (level=1) は doc2 (level=2) を閲覧できません

テストケース 3: 部署制限
✅ alice (engineering) は doc3 (engineering) を閲覧できます
❌ charlie (security) は doc3 (engineering) を閲覧できません

テストケース 4: プレミアムコンテンツ
✅ alice (premium, price>100) は doc4 を閲覧できます
❌ bob (basic) は doc4 を閲覧できません
```

## 関連ドキュメント

- [PRD.md](../../PRD.md) - ABAC の詳細仕様
- [CEL Specification](https://github.com/google/cel-spec)
