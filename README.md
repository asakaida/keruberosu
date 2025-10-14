# Keruberosu

<img src="keruberosu-logo.png" alt="Keruberosu Logo" style="display: block; margin: 0 auto;" />

Permify 互換の ReBAC/ABAC 認可マイクロサービス

## 概要

Keruberosu は、関係性ベース (ReBAC) と属性ベース (ABAC) の両方をサポートする認可マイクロサービスです。Permify の API とスキーマ DSL に互換性を持ち、柔軟で強力な認可判定を実現します。

主な機能:

- ReBAC (Relationship-Based Access Control): Google Docs や GitHub のようなリソース共有システムの認可
- ABAC (Attribute-Based Access Control): 属性に基づく柔軟なルール定義
- Permify 互換 API: Check, Expand, LookupEntity, LookupSubject, SubjectPermission
- gRPC API: 高性能な認可判定
- PostgreSQL バックエンド: 信頼性の高いデータストレージ

## アーキテクチャ

**単一サービスアプローチ**:

Keruberosu は単一の `AuthorizationService` として設計されており、以下の全ての機能を提供します：

- **Schema 管理**: スキーマ定義の作成・更新・取得
- **Data 管理**: 関係性（Relations）と属性（Attributes）の書き込み・削除
- **Authorization**: 権限チェック、ツリー展開、エンティティ検索

この設計は、Google Zanzibar、Permify、Auth0 FGA などの業界標準に従っています。

詳細は [DESIGN.md](DESIGN.md) および [PRD.md](PRD.md) の「アーキテクチャ方針」セクションを参照してください。

## 必要要件

- Go 1.21 以上
- PostgreSQL 18.0 以上
- Protocol Buffers コンパイラ (protoc)
- Docker と Docker Compose (開発環境用)

## クイックスタート

### 1. リポジトリのクローン

```bash
git clone https://github.com/asakaida/keruberosu.git
cd keruberosu
```

### 2. 環境変数の設定

開発環境用の設定ファイルを作成します：

```bash
cp .env.dev.example .env.dev
```

`.env.dev` を編集してデータベースパスワードを設定します：

```bash
DB_PASSWORD=keruberosu_dev_password
```

環境ごとの設定ファイル：

- `.env.dev` - 開発環境（ポート 15432）
- `.env.test` - テスト環境（ポート 25432）
- `.env.prod` - 本番環境

### 3. 必要なツールのインストール

Protocol Buffers のコード生成に必要なツールをインストールします。

#### 3-1. Protocol Buffers コンパイラ（protoc）のインストール

**macOS（Homebrew を使用）:**

```bash
brew install protobuf
```

**Ubuntu / Debian:**

```bash
sudo apt update
sudo apt install -y protobuf-compiler
```

**その他の OS や手動インストール:**

公式ドキュメントを参照してください：https://grpc.io/docs/protoc-installation/

#### 3-2. Go 用の Protocol Buffers プラグインのインストール

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

#### 3-3. インストールの確認

以下のコマンドでバージョンが表示されれば、インストール成功です：

```bash
# protoc のバージョン確認
protoc --version
# 例: libprotoc 3.21.12

# protoc-gen-go のバージョン確認
protoc-gen-go --version
# 例: protoc-gen-go v1.31.0

# protoc-gen-go-grpc のバージョン確認
protoc-gen-go-grpc --version
# 例: protoc-gen-go-grpc 1.3.0
```

> **注意**: `protoc-gen-go` と `protoc-gen-go-grpc` は `$GOPATH/bin` にインストールされます。
> `$GOPATH/bin` が PATH に含まれていることを確認してください。
>
> ```bash
> # PATH の確認（出力に $GOPATH/bin が含まれているか確認）
> echo $PATH
>
> # 含まれていない場合は追加（~/.bashrc, ~/.zshrc などに記載）
> export PATH="$PATH:$(go env GOPATH)/bin"
> ```

#### 代替手段: buf を使う場合

**buf** は Protocol Buffers の最新管理ツールで、`protoc` 本体のインストールが不要になります。

**buf のインストール:**

macOS（Homebrew を使用）:

```bash
brew install bufbuild/buf/buf
```

Linux:

```bash
# BIN=/usr/local/bin にインストール（要sudo）
BIN="/usr/local/bin" && \
curl -sSL \
  "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" \
  -o "${BIN}/buf" && \
chmod +x "${BIN}/buf"
```

その他の OS や手動インストール:

https://buf.build/docs/installation

**インストール確認:**

```bash
buf --version
# 例: 1.57.2
```

> **注意**: buf を使う場合、`protoc`、`protoc-gen-go`、`protoc-gen-go-grpc` のインストールは**不要**です。
> buf がリモートプラグインを使用して自動的に処理します。

### 4. Protocol Buffers のコード生成

**重要**: この手順は、Go のコマンド（`go run` など）を実行する前に必ず完了してください。

#### protoc を使う場合（従来の方法）

シェルスクリプトを使用:

```bash
./scripts/generate-proto.sh
```

または直接 protoc を使用する場合:

```bash
protoc \
  --proto_path=proto \
  --go_out=proto \
  --go_opt=paths=source_relative \
  --go-grpc_out=proto \
  --go-grpc_opt=paths=source_relative \
  proto/keruberosu/v1/*.proto
```

#### buf を使う場合（推奨）

シェルスクリプトを使用:

```bash
./scripts/generate-proto-buf.sh
```

または直接 buf を使用する場合:

```bash
buf generate
```

> **注意**: Go のソースコードは `proto/keruberosu/v1` パッケージをインポートしています。
> このパッケージは proto ファイルから生成されるため、proto 生成を先に実行しないと
> `go run` や `go build` などのコマンドが失敗します。
>
> エラーが発生した場合は、[トラブルシューティング](#トラブルシューティング) セクションを参照してください。

### 5. データベースの起動

```bash
docker compose up -d
```

### 6. マイグレーションの実行

デフォルト（dev 環境、`.env.dev` を使用）:

```bash
go run cmd/migrate/main.go up
```

環境を指定する場合（対応する `.env.{環境名}` ファイルを読み込みます）:

```bash
go run cmd/migrate/main.go up --env dev   # .env.dev を使用
go run cmd/migrate/main.go up --env test  # .env.test を使用
go run cmd/migrate/main.go up --env prod  # .env.prod を使用
```

その他のマイグレーションコマンド:

```bash
# バージョン確認
go run cmd/migrate/main.go version

# ロールバック
go run cmd/migrate/main.go down

# ヘルプ表示
go run cmd/migrate/main.go --help
```

### 7. サーバーの起動

デフォルト（dev 環境、`.env.dev` を使用）:

```bash
go run cmd/server/main.go
```

環境を指定する場合（対応する `.env.{環境名}` ファイルを読み込みます）:

```bash
go run cmd/server/main.go --env dev   # .env.dev を使用
go run cmd/server/main.go --env test  # .env.test を使用
go run cmd/server/main.go --env prod  # .env.prod を使用
```

ポート番号を指定する場合:

```bash
go run cmd/server/main.go --port 8080
go run cmd/server/main.go --env prod --port 9090
```

ヘルプの表示:

```bash
go run cmd/server/main.go --help
```

サーバーはデフォルトで `localhost:50051` で起動します。

## 開発環境セットアップ

開発環境のセットアップ手順については、[クイックスタート](#クイックスタート) セクションを参照してください。

特に、Protocol Buffers 関連のツールのインストール方法は [Step 3: 必要なツールのインストール](#3-必要なツールのインストール) を参照してください。

## トラブルシューティング

### Proto コード生成前に Go コマンドを実行するとエラーが出る

**問題**: `go run`、`go build`、`go mod download` などを実行すると、以下のようなエラーや警告が出る：

```
no required module provides package github.com/asakaida/keruberosu/proto/keruberosu/v1
```

**原因**: Keruberosu のソースコードは `proto/keruberosu/v1` パッケージをインポートしていますが、このパッケージは Protocol Buffers から自動生成されるものです。proto コード生成を実行する前に Go コマンドを実行すると、存在しないパッケージを参照しようとしてエラーになります。

**解決方法**:

1. まず Protocol Buffers のコード生成を実行してください：

   ```bash
   ./scripts/generate-proto.sh
   ```

2. その後、Go コマンドを実行してください：
   ```bash
   go run cmd/server/main.go
   ```

**注意**: `go mod download` は不要です。Go は `go run` や `go build` の実行時に自動的に依存パッケージをダウンロードします。

### 依存関係の手動インストール

通常は不要ですが、依存パッケージを事前にダウンロードしたい場合は、proto コード生成後に実行してください：

```bash
# 1. まず proto 生成
./scripts/generate-proto.sh

# 2. その後、依存関係をダウンロード
go mod download
```

### protoc が見つからない

**問題**: `./scripts/generate-proto.sh` を実行すると、以下のエラーが出る：

```
Error: protoc is not installed. Please install Protocol Buffers compiler.
```

**原因**: Protocol Buffers コンパイラ（protoc）がインストールされていません。

**解決方法**:

クイックスタートの [Step 3-1](#3-1-protocol-buffers-コンパイラprotocのインストール) を参照して、`protoc` をインストールしてください。

インストール後、以下のコマンドでバージョンが表示されることを確認：

```bash
protoc --version
```

### protoc-gen-go または protoc-gen-go-grpc が見つからない

**問題**: `./scripts/generate-proto.sh` を実行すると、以下のエラーが出る：

```
Error: protoc-gen-go is not installed.
Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

または：

```
Error: protoc-gen-go-grpc is not installed.
Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**原因**: Go 用の Protocol Buffers プラグインがインストールされていないか、PATH に含まれていません。

**解決方法**:

1. クイックスタートの [Step 3-2](#3-2-go-用の-protocol-buffers-プラグインのインストール) を参照して、プラグインをインストールしてください。

2. インストール後も同じエラーが出る場合は、`$GOPATH/bin` が PATH に含まれているか確認：

   ```bash
   # PATH の確認
   echo $PATH | grep "$(go env GOPATH)/bin"

   # 何も出力されない場合は、PATH に追加
   export PATH="$PATH:$(go env GOPATH)/bin"

   # シェル設定ファイルに永続化（bash の場合）
   echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
   source ~/.bashrc

   # zsh の場合
   echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
   source ~/.zshrc
   ```

3. 再度バージョン確認：

   ```bash
   protoc-gen-go --version
   protoc-gen-go-grpc --version
   ```

### buf が見つからない

**問題**: `./scripts/generate-proto-buf.sh` または `buf generate` を実行すると、以下のエラーが出る：

```
Error: buf is not installed.
```

**原因**: buf ツールがインストールされていません。

**解決方法**:

クイックスタートの [代替手段: buf を使う場合](#代替手段-bufを使う場合) を参照して、`buf` をインストールしてください。

インストール後、以下のコマンドでバージョンが表示されることを確認：

```bash
buf --version
```

## ビルドと実行

### ビルド

```bash
go build -o bin/keruberosu cmd/server/main.go
go build -o bin/migrate cmd/migrate/main.go
```

### マイグレーション

```bash
# マイグレーションを適用（デフォルト: dev環境）
go run cmd/migrate/main.go up

# 環境を指定してマイグレーション
go run cmd/migrate/main.go up --env test
go run cmd/migrate/main.go up --env prod

# マイグレーションをロールバック
go run cmd/migrate/main.go down

# 複数ステップのロールバック
go run cmd/migrate/main.go down 2

# 特定のバージョンまでマイグレーション
go run cmd/migrate/main.go goto <version>

# 現在のバージョン確認
go run cmd/migrate/main.go version
```

### テストの実行

テストを実行するには、まずテスト用データベースを起動します。

#### 1. テスト環境の準備

```bash
# .env.test ファイルを作成
cp .env.test.example .env.test

# テスト用データベースを起動（healthcheck が成功するまで待機）
docker compose up -d --wait postgres-test
```

#### 2. テストの実行

```bash
# 全てのテストを実行
go test ./...

# 特定のパッケージのテストを実行
go test ./internal/repositories/postgres/...

# 詳細出力でテストを実行
go test -v ./...

# キャッシュを無視してテストを実行
go test -count=1 ./...
```

#### 3. テスト環境のクリーンアップ

```bash
# テスト用データベースを停止
docker compose down postgres-test

# データも削除する場合
docker compose down postgres-test -v
```

**注意**: テストは自動的に `.env.test` を読み込み、test 環境で実行されます。`DB_PASSWORD` などの設定は `.env.test` に記載してください。

## 使用例

### スキーマの定義

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

### クライアントからの認可チェック

```go
import pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"

client := pb.NewAuthorizationServiceClient(conn)

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

if resp.Can == pb.CheckResult_CHECK_RESULT_ALLOWED {
    // 許可
}
```

### グループメンバーシップ（Permify 完全互換）

Keruberosu は、Permify と完全に互換性のある**subject relation**機能をサポートしています。これにより、`drive:eng_drive#member@group:engineering#member`のような、グループ全体を一つのタプルで関係付けることができます。

```go
// グループに所属するユーザーを定義
client.WriteRelations(ctx, &pb.WriteRelationsRequest{
    Tuples: []*pb.RelationTuple{
        {
            Entity:   &pb.Entity{Type: "group", Id: "engineering"},
            Relation: "member",
            Subject:  &pb.Subject{Type: "user", Id: "alice"},
        },
        {
            Entity:   &pb.Entity{Type: "group", Id: "engineering"},
            Relation: "member",
            Subject:  &pb.Subject{Type: "user", Id: "bob"},
        },
    },
})

// グループ全体をドライブのメンバーとして割り当て（1つのタプルで完結）
client.WriteRelations(ctx, &pb.WriteRelationsRequest{
    Tuples: []*pb.RelationTuple{
        {
            Entity:   &pb.Entity{Type: "drive", Id: "eng_drive"},
            Relation: "member",
            Subject: &pb.Subject{
                Type:     "group",
                Id:       "engineering",
                Relation: "member", // ✅ subject relationを指定
            },
        },
    },
})

// これでaliceとbobはeng_driveのメンバーとして自動的に権限を持ちます
```

詳細な使用例は [PRD.md](PRD.md) の「API 利用ガイド」セクションと [examples/](examples/) ディレクトリを参照してください。

## プロジェクト構造

```text
keruberosu/
├── cmd/
│   ├── server/          # メインサーバー
│   └── migrate/         # マイグレーションコマンド
├── internal/
│   ├── entities/        # ドメインエンティティ
│   ├── handlers/        # gRPC ハンドラー
│   ├── repositories/    # データアクセス層
│   ├── services/        # ビジネスロジック
│   └── infrastructure/  # インフラ層（DB、設定）
├── proto/
│   └── keruberosu/v1/   # Protocol Buffers 定義
└── docker-compose.yml   # 開発環境
```

## ドキュメント

- [PRD.md](PRD.md): 要求仕様書、API 利用ガイド
- [DESIGN.md](DESIGN.md): 設計ドキュメント、アーキテクチャ
- [ARCHITECTURE.md](ARCHITECTURE.md): アーキテクチャ図（Mermaid）
- [DEVELOPMENT.md](DEVELOPMENT.md): 開発進捗管理、タスクリスト
- [examples/](examples/): API 使用例・サンプルコード

## 開発状況

現在のフェーズ: Phase 1 - 基盤構築（完了）

### テスト結果

✅ **全 E2E テスト成功: 45/45 テストケース (100% パス率)**

- ReBAC シナリオ（Google Docs 風）: 14/14 ✓
- ABAC シナリオ（全演算子）: 19/19 ✓
- Permify 互換性検証: 12/12 ✓

進捗詳細は [DEVELOPMENT.md](DEVELOPMENT.md) を参照してください。

## ライセンス

TBD

## 貢献

TBD

## 参考資料

- [Permify Documentation](https://docs.permify.co/)
- [CEL Language Definition](https://github.com/google/cel-spec)
- [Protocol Buffers Guide](https://protobuf.dev/)
