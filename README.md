# Keruberosu

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

サービス構成:

- AuthorizationService: スキーマ管理、データ管理、認可チェック
- AuditService: 監査ログ管理

詳細は [DESIGN.md](DESIGN.md) を参照してください。

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

### 2. 依存関係のインストール

```bash
go mod download
```

### 3. 環境変数の設定

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

### 4. Protocol Buffers のコード生成

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

### 5. データベースの起動

```bash
docker-compose up -d
```

### 6. マイグレーションの実行

デフォルト（dev 環境）:

```bash
go run cmd/migrate/main.go up
```

環境を指定する場合:

```bash
go run cmd/migrate/main.go up --env dev
go run cmd/migrate/main.go up --env test
```

### 7. サーバーの起動

デフォルト（dev 環境）:

```bash
go run cmd/server/main.go
```

環境を指定する場合:

```bash
go run cmd/server/main.go --env dev
go run cmd/server/main.go --env prod
```

サーバーは `localhost:50051` で起動します。

## 開発環境セットアップ

### Protocol Buffers ツールのインストール

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## ビルドと実行

### ビルド

```bash
go build -o bin/keruberosu cmd/server/main.go
go build -o bin/migrate cmd/migrate/main.go
```

### マイグレーション

```bash
# マイグレーションを適用
go run cmd/migrate/main.go up

# マイグレーションをロールバック
go run cmd/migrate/main.go down

# 特定のバージョンまでマイグレーション
go run cmd/migrate/main.go goto <version>
```

### テストの実行

テストを実行するには、まずテスト用データベースを起動します。

#### 1. テスト環境の準備

```bash
# .env.test ファイルを作成
cp .env.test.example .env.test

# テスト用データベースを起動（healthcheck が成功するまで待機）
docker-compose up -d --wait postgres-test
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
docker-compose down postgres-test

# データも削除する場合
docker-compose down postgres-test -v
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

詳細な使用例は [PRD.md](PRD.md) の「API 利用ガイド」セクションを参照してください。

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
- [DEVELOPMENT.md](DEVELOPMENT.md): 開発進捗管理、タスクリスト

## 開発状況

現在のフェーズ: Phase 1 - 基盤構築

進捗詳細は [DEVELOPMENT.md](DEVELOPMENT.md) を参照してください。

## ライセンス

TBD

## 貢献

TBD

## 参考資料

- [Permify Documentation](https://docs.permify.co/)
- [CEL Language Definition](https://github.com/google/cel-spec)
- [Protocol Buffers Guide](https://protobuf.dev/)
