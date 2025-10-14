package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// gRPC サーバーへ接続
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("接続失敗: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuthorizationServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// スキーマDSLの定義
	schema := `
entity user {}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user

  permission edit = owner or editor
  permission view = owner or editor or viewer
}
`

	// スキーマを書き込み
	resp, err := client.WriteSchema(ctx, &pb.WriteSchemaRequest{
		SchemaDsl: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込み失敗: %v", err)
	}

	fmt.Printf("✅ スキーマが正常に書き込まれました (version: %s)\n", resp.SchemaVersion)

	// スキーマを読み込んで確認
	readResp, err := client.ReadSchema(ctx, &pb.ReadSchemaRequest{})
	if err != nil {
		log.Fatalf("スキーマ読み込み失敗: %v", err)
	}

	fmt.Printf("\n📄 登録されたスキーマ:\n%s\n", readResp.SchemaDsl)
	fmt.Printf("🕒 更新日時: %s\n", readResp.UpdatedAt)
}
