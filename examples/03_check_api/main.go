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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// まず、スキーマとデータを準備（Example 1, 2 と同じ）
	setupSchemaAndData(ctx, client)

	// Check API のテスト
	fmt.Println("\n===== Check API テスト =====\n")

	testCases := []struct {
		name       string
		entityType string
		entityID   string
		permission string
		subjectID  string
		expected   pb.CheckResult
		reason     string
	}{
		{"alice が doc1 を編集できるか", "document", "doc1", "edit", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED, "owner"},
		{"bob が doc1 を編集できるか", "document", "doc1", "edit", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED, "editor"},
		{"charlie が doc1 を編集できるか", "document", "doc1", "edit", "charlie", pb.CheckResult_CHECK_RESULT_DENIED, ""},
		{"charlie が doc1 を閲覧できるか", "document", "doc1", "view", "charlie", pb.CheckResult_CHECK_RESULT_ALLOWED, "viewer"},
		{"誰でも public な doc1 を閲覧できるか", "document", "doc1", "view", "dave", pb.CheckResult_CHECK_RESULT_ALLOWED, "ABAC rule - public == true"},
	}

	for i, tc := range testCases {
		fmt.Printf("テストケース %d: %s\n", i+1, tc.name)

		resp, err := client.Check(ctx, &pb.CheckRequest{
			Entity: &pb.Entity{
				Type: tc.entityType,
				Id:   tc.entityID,
			},
			Permission: tc.permission,
			Subject: &pb.Subject{
				Type: "user",
				Id:   tc.subjectID,
			},
		})
		if err != nil {
			log.Fatalf("Check 失敗: %v", err)
		}

		if resp.Can == tc.expected {
			if resp.Can == pb.CheckResult_CHECK_RESULT_ALLOWED {
				if tc.reason != "" {
					fmt.Printf("✅ %s は %s を %s できます（理由: %s）\n\n", tc.subjectID, tc.entityID, tc.permission, tc.reason)
				} else {
					fmt.Printf("✅ %s は %s を %s できます\n\n", tc.subjectID, tc.entityID, tc.permission)
				}
			} else {
				fmt.Printf("❌ %s は %s を %s できません\n\n", tc.subjectID, tc.entityID, tc.permission)
			}
		} else {
			log.Fatalf("❌ テスト失敗: 期待値=%v, 実際=%v", tc.expected, resp.Can)
		}
	}

	fmt.Println("🎉 全てのテストケースが成功しました!")
}

func setupSchemaAndData(ctx context.Context, client pb.AuthorizationServiceClient) {
	// スキーマを書き込み
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

	_, err := client.WriteSchema(ctx, &pb.WriteSchemaRequest{
		SchemaDsl: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込み失敗: %v", err)
	}

	// 関係性を書き込み
	_, err = client.WriteRelations(ctx, &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Entity{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "editor", Subject: &pb.Entity{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "viewer", Subject: &pb.Entity{Type: "user", Id: "charlie"}},
		},
	})
	if err != nil {
		log.Fatalf("関係性書き込み失敗: %v", err)
	}

	// 属性を書き込み
	_, err = client.WriteAttributes(ctx, &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			{
				Entity: &pb.Entity{Type: "document", Id: "doc1"},
				Data: map[string]*structpb.Value{
					"public":   structpb.NewBoolValue(true),
					"owner_id": structpb.NewStringValue("alice"),
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("属性書き込み失敗: %v", err)
	}

	fmt.Println("✅ スキーマとデータを準備しました")
}
