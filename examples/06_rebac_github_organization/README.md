# Example 06: ReBAC - GitHub 風の組織・リポジトリ・Issue 管理

## 概要

このサンプルは、3 階層のネストした関係性を使った ReBAC（Relationship-Based Access Control）の実例を示します。

GitHub や GitLab のような組織・リポジトリ・Issue 管理システムを模した、現実的なユースケースです。

## 階層構造

```text
Organization (組織)
  └─ Repository (リポジトリ)
      └─ Issue (課題)
```

## 登場人物

| ユーザー | 役割                               | 権限の範囲                                    |
| -------- | ---------------------------------- | --------------------------------------------- |
| Alice    | 組織管理者 (org admin)             | 全リポジトリ・全 Issue を管理可能             |
| Bob      | リポジトリ管理者 (repo maintainer) | backend-api リポジトリと配下の全 Issue を管理 |
| Charlie  | Issue 担当者 (issue assignee)      | Issue #123 のみ編集可能                       |
| Diana    | 組織メンバー (org member)          | 全リポジトリ・全 Issue を閲覧可能             |
| Eve      | コントリビューター (contributor)   | backend-api リポジトリへの書き込み可能        |

## スキーマ定義

### Organization（組織）

```text
entity organization {
  relation admin @user       # 組織管理者
  relation member @user      # 組織メンバー

  permission manage = admin             # 組織を管理できる
  permission view = admin or member     # 組織を閲覧できる
}
```

### Repository（リポジトリ）

```text
entity repository {
  relation org @organization      # 所属する組織
  relation maintainer @user       # リポジトリ管理者
  relation contributor @user      # コントリビューター

  permission delete = org.admin                                # 組織管理者のみ削除可能
  permission manage = org.admin or maintainer                  # 管理権限
  permission write = org.admin or maintainer or contributor    # 書き込み権限
  permission read = org.admin or maintainer or contributor or org.view  # 閲覧権限
}
```

### Issue（課題）

```text
entity issue {
  relation repo @repository    # 所属するリポジトリ
  relation assignee @user      # 担当者

  permission close = repo.manage       # リポジトリ管理者がクローズ可能
  permission edit = repo.manage or assignee   # 管理者または担当者が編集可能
  permission view = repo.read          # リポジトリの閲覧権限で見える
}
```

## 権限継承の仕組み

### パターン 1: 組織管理者の全権限

```text
Alice (org admin)
  → org.admin
    → repo.delete  （リポジトリ削除）
    → repo.manage  （リポジトリ管理）
      → issue.close  （Issue クローズ）
```

### パターン 2: メンバーの閲覧権限

```text
Diana (org member)
  → org.view
    → repo.read
      → issue.view
```

### パターン 3: リポジトリ管理者の権限

```text
Bob (repo maintainer)
  → repo.manage
    → issue.close
    → issue.edit
```

### パターン 4: Issue 担当者の限定権限

```text
Charlie (issue assignee)
  → issue.edit (Issue #123 のみ)
```

## 実行方法

### 1. サーバーの起動

```bash
# ターミナル1でサーバーを起動
go run cmd/server/main.go
```

### 2. サンプルの実行

```bash
# ターミナル2でサンプルを実行
go run examples/06_rebac_github_organization/main.go
```

## 実行結果（例）

```text
===== ReBAC: GitHub風の組織・リポジトリ・Issue管理（3階層ネスト） =====

📋 スキーマを定義中...
✅ スキーマ定義完了

📁 組織構造:
  Acme Corp (組織)
    ├─ Alice: admin (組織管理者)
    └─ Diana: member (組織メンバー)

  backend-api (リポジトリ)
    ├─ 所属: Acme Corp
    ├─ Bob: maintainer (リポジトリ管理者)
    └─ Eve: contributor (コントリビューター)

  Issue #123 (課題)
    ├─ 所属: backend-api
    └─ Charlie: assignee (担当者)

🔐 権限チェック開始

【Alice（組織管理者）の権限】
   ✅ Alice: acme を manage できます - 組織管理権限
   ✅ Alice: backend-api を delete できます - リポジトリ削除権限（org.admin経由）
   ✅ Alice: 123 を close できます - Issue クローズ権限（repo.manage → org.admin経由）

【Bob（backend-api リポジトリ管理者）の権限】
   ✅ Bob: backend-api を manage できます - リポジトリ管理権限
   ❌ Bob: backend-api を delete できません - リポジトリ削除不可（org.admin のみ）
   ✅ Bob: 123 を close できます - Issue クローズ権限（repo.manage経由）

【Charlie（Issue #123 担当者）の権限】
   ✅ Charlie: 123 を edit できます - 担当Issueの編集権限
   ❌ Charlie: 123 を close できません - Issueクローズ不可（repo.manage が必要）

【Diana（組織メンバー）の権限】
   ✅ Diana: acme を view できます - 組織閲覧権限
   ✅ Diana: backend-api を read できます - リポジトリ閲覧権限（org.view経由）
   ✅ Diana: 123 を view できます - Issue 閲覧権限（repo.read → org.view経由）
   ❌ Diana: 123 を edit できません - Issue 編集不可

🎉 3階層ネストのReBAC シナリオ完了!
```

## この例で学べること

### 1. 3 階層のネスト構造

- Organization → Repository → Issue
- 実際のサービスでよく使われるパターン

### 2. 複雑な権限継承

- `issue.view` → `repo.read` → `org.view` のような多段階継承
- 組織管理者が全てのリソースを管理できる設計

### 3. 役割ベースの権限管理

- admin（全権限）
- maintainer（管理権限）
- contributor（書き込み権限）
- member（閲覧権限）
- assignee（限定的な編集権限）

### 4. 階層間の境界

- Bob は backend-api の Issue は管理できるが、frontend-app の Issue は閲覧できない
- リソースの所属に基づいた権限の分離

## 他の例との比較

| 例                             | 階層数 | 特徴                                               |
| ------------------------------ | ------ | -------------------------------------------------- |
| 04_rebac_google_docs           | 2 階層 | folder → document の基本的なネスト                 |
| `06_rebac_github_organization` | 3 階層 | `organization → repository → issue` の複雑なネスト |

## 応用例

このパターンは以下のようなシステムで利用できます：

- プロジェクト管理ツール: Workspace → Project → Task
- クラウドストレージ: Organization → Bucket → File
- e コマース: Company → Store → Product
- 教育機関: University → Department → Course
- SaaS 製品: Account → Workspace → Resource

## 参考資料

- [Permify Documentation](https://docs.permify.co/)
- [GitHub のアクセス権限](https://docs.github.com/ja/organizations/managing-user-access-to-your-organizations-repositories)
- [Zanzibar: Google's Consistent, Global Authorization System](https://research.google/pubs/pub48190/)
