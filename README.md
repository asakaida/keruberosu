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

### 3. Protocol Buffers のコード生成

```bash
protoc --go_out=. --go-grpc_out=. proto/keruberosu/v1/*.proto
```

### 4. データベースの起動

```bash
docker-compose up -d
```

### 5. マイグレーションの実行

```bash
go run cmd/migrate/main.go up
```

### 6. サーバーの起動

```bash
go run cmd/server/main.go
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

```bash
go test ./...
```

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
