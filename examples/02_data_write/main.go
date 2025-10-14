package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
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

	// Step 1: スキーマを書き込み
	schema := `
entity user {}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user

  attribute public: bool
  attribute owner_id: string

  permission edit = owner or editor
  permission view = owner or editor or viewer or rule(resource.public == true)
}
`

	schemaResp, err := client.WriteSchema(ctx, &pb.WriteSchemaRequest{
		SchemaDsl: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込み失敗: %v", err)
	}
	fmt.Printf("✅ スキーマが書き込まれました (version: %s)\n", schemaResp.SchemaVersion)

	// Step 2: 関係性（Relations）を書き込み
	relResp, err := client.WriteRelations(ctx, &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			// doc1 は alice が所有
			{
				Entity:   &pb.Entity{Type: "document", Id: "doc1"},
				Relation: "owner",
				Subject:  &pb.Subject{Type: "user", Id: "alice"},
			},
			// doc1 は bob が編集可能
			{
				Entity:   &pb.Entity{Type: "document", Id: "doc1"},
				Relation: "editor",
				Subject:  &pb.Subject{Type: "user", Id: "bob"},
			},
			// doc1 は charlie が閲覧可能
			{
				Entity:   &pb.Entity{Type: "document", Id: "doc1"},
				Relation: "viewer",
				Subject:  &pb.Subject{Type: "user", Id: "charlie"},
			},
		},
	})
	if err != nil {
		log.Fatalf("関係性書き込み失敗: %v", err)
	}
	fmt.Printf("✅ 関係性が書き込まれました (snap_token: %s)\n", relResp.SnapToken)

	// Step 3: 属性（Attributes）を書き込み（Permify互換: 単一属性形式）
	_, err = client.WriteAttributes(ctx, &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			{
				Entity:    &pb.Entity{Type: "document", Id: "doc1"},
				Attribute: "public",
				Value:     structpb.NewBoolValue(true),
			},
			{
				Entity:    &pb.Entity{Type: "document", Id: "doc1"},
				Attribute: "owner_id",
				Value:     structpb.NewStringValue("alice"),
			},
			{
				Entity:    &pb.Entity{Type: "document", Id: "doc2"},
				Attribute: "public",
				Value:     structpb.NewBoolValue(false),
			},
			{
				Entity:    &pb.Entity{Type: "document", Id: "doc2"},
				Attribute: "owner_id",
				Value:     structpb.NewStringValue("bob"),
			},
		},
	})
	if err != nil {
		log.Fatalf("属性書き込み失敗: %v", err)
	}
	fmt.Println("✅ 属性が書き込まれました")

	fmt.Println("\n📊 データ書き込み完了!")
	fmt.Println("次は Example 3 で Check API を試してみましょう")
}
