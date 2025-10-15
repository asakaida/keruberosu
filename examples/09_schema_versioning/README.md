# スキーマバージョン管理

このサンプルでは、Keruberos のスキーマバージョン管理機能を実演します。

## 概要

スキーマバージョン管理により、以下のことが可能になります：

- **複数バージョンの保存**: スキーマの変更履歴を自動的に保存
- **バージョン一覧表示**: 全バージョンとタイムスタンプを確認
- **特定バージョンの読み取り**: 過去のスキーマ定義を参照
- **バージョン指定での権限チェック**: 過去のスキーマ定義を使用して権限チェックを実行

## 機能デモ

このサンプルでは以下を実演します：

### 1. スキーマの段階的な進化

```
v1: 基本的なdocument（owner, viewer のみ）
↓
v2: editor ロールとedit パーミッションを追加
↓
v3: admin ロールとdelete パーミッションを追加
```

### 2. バージョン一覧表示

```bash
最新バージョン (HEAD): 01HGW5...
全バージョン一覧:
  1. Version: 01HGW5..., CreatedAt: 2024-01-01T10:03:00Z
  2. Version: 01HGW4..., CreatedAt: 2024-01-01T10:02:00Z
  3. Version: 01HGW3..., CreatedAt: 2024-01-01T10:01:00Z
```

### 3. 特定バージョンでの権限チェック

異なるスキーマバージョンを使用して同じ権限チェックを実行し、スキーマの進化による動作の違いを確認します。

## 実行方法

1. サーバーを起動（別ターミナル）:

```bash
go run cmd/server/main.go
```

2. このサンプルを実行:

```bash
go run examples/09_schema_versioning/main.go
```

## 期待される出力

```
===== スキーマバージョン管理のデモ =====

【ステップ1】初期スキーマを書き込み
✅ スキーマv1が書き込まれました (version: 01HGW3...)

【ステップ2】スキーマを更新してeditorロールを追加
✅ スキーマv2が書き込まれました (version: 01HGW4...)

【ステップ3】スキーマを更新してadminロールを追加
✅ スキーマv3が書き込まれました (version: 01HGW5...)

【ステップ4】全バージョンを一覧表示
最新バージョン (HEAD): 01HGW5...

全バージョン一覧:
  1. Version: 01HGW5..., CreatedAt: 2024-01-01T10:03:00Z
  2. Version: 01HGW4..., CreatedAt: 2024-01-01T10:02:00Z
  3. Version: 01HGW3..., CreatedAt: 2024-01-01T10:01:00Z

【ステップ5】各バージョンのスキーマを読み取り
スキーマv1 (01HGW3...):
entity user {}
entity document {
  relation owner @user
  relation viewer @user
  permission view = owner or viewer
}
...

【ステップ7】異なるスキーマバージョンでパーミッションチェック

--- v1スキーマ (01HGW3...) を使用 ---
daveがdoc1をviewできるかチェック...
✅ 許可されました (daveはviewer)

charlieがdoc1をviewできるかチェック...
❌ 拒否されました (v1にはeditorロールが存在しない)

charlieがdoc1をeditできるかチェック...
❌ エラー: permission edit not found in entity document

--- v2スキーマ (01HGW4...) を使用 ---
charlieがdoc1をeditできるかチェック...
✅ 許可されました (charlieはeditor)

bobがdoc1をdeleteできるかチェック...
❌ エラー: permission delete not found in entity document

--- v3スキーマ (01HGW5...) を使用（最新版） ---
bobがdoc1をdeleteできるかチェック...
✅ 許可されました (bobはadmin)

🎉 スキーマバージョン管理のデモが完了しました!
```

## ユースケース

### 1. 本番環境での段階的ロールアウト

- 新しいスキーマをテスト環境で検証
- 特定のユーザーグループに対して旧バージョンを使用しながら新バージョンをテスト
- 問題があれば即座に前のバージョンにロールバック

### 2. A/B テスト

- 異なる権限モデルを同時にテスト
- メタデータでバージョンを指定して並行実行

### 3. 監査とコンプライアンス

- 過去の時点でのアクセス権限を正確に再現
- 「その時点で誰が何にアクセスできたか」を検証

### 4. 後方互換性の維持

- 古いクライアントが特定のスキーマバージョンを指定して動作継続
- 段階的な移行期間を設定

## Permify 互換性

このバージョン管理機能は[Permify](https://permify.co)と完全に互換性があります：

- **ULID 形式のバージョン ID**: タイムスタンプベースの一意識別子
- **List API**: `HEAD`フィールドと`schemas`配列を含むレスポンス形式
- **メタデータでの指定**: `PermissionCheckMetadata.schema_version`で全 API 対応
- **デフォルト動作**: バージョン未指定時は最新版を自動使用

## 関連 API

- **Schema.Write**: 新しいスキーマバージョンを作成
- **Schema.Read**: スキーマを読み取り（バージョン指定可能）
- **Schema.List**: 全バージョンを一覧表示
- **Permission.Check**: 権限チェック（バージョン指定可能）
- **Permission.Expand**: 権限ツリー展開（バージョン指定可能）
- **Permission.LookupEntity/LookupSubject**: ルックアップ（バージョン指定可能）
