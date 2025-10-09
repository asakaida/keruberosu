# Keruberosu API 使用例

このディレクトリには、Keruberosu の主要な機能を示すサンプルコードが含まれています。

## 前提条件

全てのサンプルを実行する前に、以下を確認してください：

1. **Keruberosu サーバーが起動していること**

   ```bash
   # プロジェクトのルートディレクトリで
   go run cmd/server/main.go
   ```

2. **PostgreSQL データベースが起動していること**

   ```bash
   docker-compose up -d
   ```

3. **マイグレーションが適用されていること**
   ```bash
   go run cmd/migrate/main.go up
   ```

## サンプル一覧

### 1. スキーマ定義 (`01_schema_definition/`)

最も基本的な例。スキーマ DSL を使用してエンティティ、関係性、権限を定義します。

**内容:**

- エンティティの定義（user, document）
- 関係性の定義（owner, editor, viewer）
- 権限ルールの定義（edit, view）

```bash
cd 01_schema_definition
go run main.go
```

### 2. データ書き込み (`02_data_write/`)

関係性（Relations）と属性（Attributes）をデータベースに書き込む方法を示します。

**内容:**

- WriteRelations API の使用
- WriteAttributes API の使用
- 複数タプルの一括書き込み

```bash
cd 02_data_write
go run main.go
```

### 3. Check API (`03_check_api/`)

権限チェックの基本的な使い方を示します。

**内容:**

- Check API の使用方法
- 様々な権限チェックのパターン
- ReBAC と ABAC の組み合わせ

```bash
cd 03_check_api
go run main.go
```

### 4. ReBAC - Google Docs 風の権限管理 (`04_rebac_google_docs/`)

階層的な権限管理を実装する方法を示します。

**内容:**

- 階層的な関係性（parent relation）
- 権限の継承（parent.view）
- フォルダとドキュメントの権限モデル

```bash
cd 04_rebac_google_docs
go run main.go
```

**特徴:**

- フォルダの editor は、そのフォルダ内の全ドキュメントを閲覧可能
- Google Docs のような直感的な権限管理

### 5. ABAC - 属性ベースアクセス制御 (`05_abac_attributes/`)

属性に基づく動的な権限管理を実装する方法を示します。

**内容:**

- CEL（Common Expression Language）の使用
- 属性に基づく条件式（security_level, department）
- 比較演算子、論理演算子の使用

```bash
cd 05_abac_attributes
go run main.go
```

**特徴:**

- セキュリティレベルに基づくアクセス制御
- 部署やサブスクリプション tier に基づくアクセス制御
- 柔軟なルール定義

### 6. ReBAC - GitHub 風の組織管理（3 階層ネスト） (`06_rebac_github_organization/`)

複雑な多段階ネストを使った現実的な権限管理を実装する方法を示します。

**内容:**

- 3 階層のネスト構造（Organization → Repository → Issue）
- 複雑な権限継承パターン
- 役割ベースの権限管理（admin, maintainer, contributor, member）

```bash
cd 06_rebac_github_organization
go run main.go
```

**特徴:**

- GitHub/GitLab のような組織・リポジトリ・Issue 管理
- 組織管理者が全リソースを管理できる設計
- リポジトリごとの権限分離
- 多段階の権限継承（`issue.view` → `repo.read` → `org.view`）

**Example 4 との違い:**

| 項目       | Example 4（Google Docs）        | Example 6（GitHub Organization）                           |
| ---------- | ------------------------------- | ---------------------------------------------------------- |
| 階層数     | 2 階層（folder → document）     | 3 階層（org → repo → issue）                               |
| 権限継承   | シンプル（parent.view）         | 複雑（複数の継承パス）                                     |
| 役割の種類 | 3 種類（owner, editor, viewer） | 5 種類（admin, maintainer, contributor, member, assignee） |

## 推奨学習順序

1. **基礎編:**

   1. スキーマ定義（Example 1）
   2. データ書き込み（Example 2）
   3. Check API（Example 3）

2. **応用編（ReBAC）:**

   4. ReBAC - Google Docs 風（Example 4）- 2 階層の階層的権限管理
   5. ReBAC - GitHub 風（Example 6）- 3 階層の複雑なネスト構造

3. **応用編（ABAC）:**

   5. ABAC - 属性ベースアクセス制御（Example 5）

## よくある質問

### Q: サンプルを実行するとエラーが出ます

**A:** 以下を確認してください：

- サーバーが `localhost:50051` で起動しているか
- データベースが起動しているか
- マイグレーションが適用されているか

### Q: スキーマを更新したい

**A:** Example 1 を再実行すれば、スキーマが上書きされます。

### Q: データをクリアしたい

**A:** PostgreSQL のデータベースをリセットしてください：

```bash
docker-compose down -v
docker-compose up -d
go run cmd/migrate/main.go up
```

## 次のステップ

- [PRD.md](../PRD.md): API の完全な仕様を確認
- [DESIGN.md](../DESIGN.md): アーキテクチャの詳細を理解
- [test/e2e/](../test/e2e/): より複雑なシナリオのテストコードを参照

## トラブルシューティング

### 接続エラー

```
接続失敗: connection refused
```

→ サーバーが起動していない可能性があります。`go run cmd/server/main.go` を実行してください。

### スキーマエラー

```
スキーマ書き込みエラー: validation failed
```

→ スキーマ DSL の構文エラーです。エラーメッセージを確認して修正してください。

### データベースエラー

```
failed to connect to database
```

→ PostgreSQL が起動していない、または `.env.dev` の設定が間違っています。

## サポート

問題が解決しない場合は、以下を確認してください：

- [DEVELOPMENT.md](../DEVELOPMENT.md): 開発環境のセットアップ手順
- GitHub Issues: バグ報告や質問
