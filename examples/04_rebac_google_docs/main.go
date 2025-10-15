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

	permissionClient := pb.NewPermissionClient(conn)
	dataClient := pb.NewDataClient(conn)
	schemaClient := pb.NewSchemaClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("===== ReBAC: Google Docs風の権限管理 =====")

	// Step 1: スキーマを定義
	schema := `
entity user {}

entity folder {
  relation owner @user
  relation editor @user

  permission edit = owner or editor
  permission view = owner or editor
}

entity document {
  relation owner @user
  relation editor @user
  relation viewer @user
  relation parent @folder

  permission delete = owner
  permission edit = owner or editor
  permission view = owner or editor or viewer or parent.view
}
`

	schemaResp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込み失敗: %v", err)
	}
	fmt.Printf("✅ スキーマが書き込まれました (version: %s)\n\n", schemaResp.SchemaVersion)

	// Step 2: データを書き込み
	fmt.Println("alice は folder1 の owner です")
	fmt.Println("bob は folder1 の editor です")

	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// folder1
			{Entity: &pb.Entity{Type: "folder", Id: "folder1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "folder", Id: "folder1"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "bob"}},

			// doc1 は folder1 に所属、alice が所有
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "parent", Subject: &pb.Subject{Type: "folder", Id: "folder1"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
		},
	})
	if err != nil {
		log.Fatalf("関係性書き込み失敗: %v", err)
	}

	// Step 3: フォルダの権限チェック
	checkPermission(ctx, permissionClient, "alice (owner)", "folder", "folder1", "edit", "alice", true)
	checkPermission(ctx, permissionClient, "bob (editor)", "folder", "folder1", "edit", "bob", true)
	checkPermission(ctx, permissionClient, "charlie", "folder", "folder1", "edit", "charlie", false)

	fmt.Println("\ndoc1 は folder1 に所属しています")
	fmt.Println("doc1 の owner は alice です")

	// Step 4: ドキュメントの権限チェック（直接権限）
	checkPermission(ctx, permissionClient, "alice (owner)", "document", "doc1", "delete", "alice", true)
	checkPermission(ctx, permissionClient, "alice (owner)", "document", "doc1", "edit", "alice", true)
	checkPermission(ctx, permissionClient, "alice (owner)", "document", "doc1", "view", "alice", true)

	// Step 5: 階層的権限の継承チェック
	fmt.Println()
	checkPermission(ctx, permissionClient, "bob (folder editor)", "document", "doc1", "view", "bob", true)
	checkPermission(ctx, permissionClient, "bob (folder editor)", "document", "doc1", "edit", "bob", false)
	checkPermission(ctx, permissionClient, "bob (folder editor)", "document", "doc1", "delete", "bob", false)

	fmt.Println("\n🎉 ReBAC シナリオ完了!")
	fmt.Println("bob は folder1 の editor なので、parent.view 経由で doc1 を閲覧できます")
}

func checkPermission(ctx context.Context, permissionClient pb.PermissionClient, description, entityType, entityID, permission, subjectID string, expected bool) {
	resp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: entityType, Id: entityID},
		Permission: permission,
		Subject:    &pb.Subject{Type: "user", Id: subjectID},
	})
	if err != nil {
		log.Fatalf("Check 失敗: %v", err)
	}

	allowed := resp.Can == pb.CheckResult_CHECK_RESULT_ALLOWED
	if allowed == expected {
		if allowed {
			if description == "bob (folder editor)" && permission == "view" {
				fmt.Printf("✅ %s は %s を %s できます（parent.view 経由）\n", description, entityID, permission)
			} else {
				fmt.Printf("✅ %s は %s を %s できます\n", description, entityID, permission)
			}
		} else {
			fmt.Printf("❌ %s は %s を %s できません\n", description, entityID, permission)
		}
	} else {
		log.Fatalf("テスト失敗: %s - 期待=%v, 実際=%v", description, expected, allowed)
	}
}
