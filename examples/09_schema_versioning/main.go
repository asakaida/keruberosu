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

	fmt.Println("===== スキーマバージョン管理のデモ =====")

	// === ステップ1: 初期スキーマを書き込み（v1） ===
	fmt.Println("【ステップ1】初期スキーマを書き込み")
	schemaV1 := `
entity user {}

entity document {
  relation owner @user
  relation viewer @user

  permission view = owner or viewer
}
`

	v1Resp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schemaV1,
	})
	if err != nil {
		log.Fatalf("スキーマv1書き込み失敗: %v", err)
	}
	v1Version := v1Resp.SchemaVersion
	fmt.Printf("✅ スキーマv1が書き込まれました (version: %s)\n\n", v1Version)

	// === ステップ2: スキーマを更新（v2：editorロールを追加） ===
	time.Sleep(100 * time.Millisecond) // ULIDの一意性を保証
	fmt.Println("【ステップ2】スキーマを更新してeditorロールを追加")
	schemaV2 := `
entity user {}

entity document {
  relation owner @user
  relation editor @user
  relation viewer @user

  permission edit = owner or editor
  permission view = owner or editor or viewer
}
`

	v2Resp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schemaV2,
	})
	if err != nil {
		log.Fatalf("スキーマv2書き込み失敗: %v", err)
	}
	v2Version := v2Resp.SchemaVersion
	fmt.Printf("✅ スキーマv2が書き込まれました (version: %s)\n\n", v2Version)

	// === ステップ3: さらにスキーマを更新（v3：管理者権限を追加） ===
	time.Sleep(100 * time.Millisecond)
	fmt.Println("【ステップ3】スキーマを更新してadminロールを追加")
	schemaV3 := `
entity user {}

entity document {
  relation owner @user
  relation admin @user
  relation editor @user
  relation viewer @user

  permission delete = owner or admin
  permission edit = owner or admin or editor
  permission view = owner or admin or editor or viewer
}
`

	v3Resp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schemaV3,
	})
	if err != nil {
		log.Fatalf("スキーマv3書き込み失敗: %v", err)
	}
	v3Version := v3Resp.SchemaVersion
	fmt.Printf("✅ スキーマv3が書き込まれました (version: %s)\n\n", v3Version)

	// === ステップ4: バージョン一覧を取得 ===
	fmt.Println("【ステップ4】全バージョンを一覧表示")
	listResp, err := schemaClient.List(ctx, &pb.SchemaListRequest{
		PageSize: 10,
	})
	if err != nil {
		log.Fatalf("バージョン一覧取得失敗: %v", err)
	}

	fmt.Printf("最新バージョン (HEAD): %s\n", listResp.Head)
	fmt.Println("\n全バージョン一覧:")
	for i, schema := range listResp.Schemas {
		fmt.Printf("  %d. Version: %s, CreatedAt: %s\n", i+1, schema.Version, schema.CreatedAt)
	}
	fmt.Println()

	// === ステップ5: 特定バージョンのスキーマを読み取り ===
	fmt.Println("【ステップ5】各バージョンのスキーマを読み取り")

	// v1を読み取り
	readV1Resp, err := schemaClient.Read(ctx, &pb.SchemaReadRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: v1Version,
		},
	})
	if err != nil {
		log.Fatalf("スキーマv1読み取り失敗: %v", err)
	}
	fmt.Printf("スキーマv1 (%s):\n%s\n", v1Version, readV1Resp.Schema)

	// 最新版（v3）を読み取り
	readLatestResp, err := schemaClient.Read(ctx, &pb.SchemaReadRequest{})
	if err != nil {
		log.Fatalf("最新スキーマ読み取り失敗: %v", err)
	}
	fmt.Printf("最新スキーマ:\n%s\n", readLatestResp.Schema)

	// === ステップ6: テストデータを準備 ===
	fmt.Println("【ステップ6】テストデータを準備")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "admin", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "dave"}},
		},
	})
	if err != nil {
		log.Fatalf("データ書き込み失敗: %v", err)
	}
	fmt.Println("✅ テストデータを書き込みました")

	// === ステップ7: バージョンを指定してパーミッションチェック ===
	fmt.Println("【ステップ7】異なるスキーマバージョンでパーミッションチェック")

	// v1を使用（editorロールとeditパーミッションは存在しない）
	fmt.Printf("\n--- v1スキーマ (%s) を使用 ---\n", v1Version)
	checkWithVersion(ctx, permissionClient, v1Version, "dave", "doc1", "view", "daveはviewerなのでv1でviewできる")
	checkWithVersion(ctx, permissionClient, v1Version, "charlie", "doc1", "view", "v1にはeditorロールが存在しないのでcharlieはviewできない")
	checkWithVersion(ctx, permissionClient, v1Version, "charlie", "doc1", "edit", "v1にはeditパーミッションが存在しないのでエラーになるはず")

	// v2を使用（editパーミッションはあるがdeleteはない）
	fmt.Printf("\n--- v2スキーマ (%s) を使用 ---\n", v2Version)
	checkWithVersion(ctx, permissionClient, v2Version, "charlie", "doc1", "edit", "v2にはeditパーミッションがある（charlieはeditor）")
	checkWithVersion(ctx, permissionClient, v2Version, "bob", "doc1", "delete", "v2にはdeleteパーミッションが存在しないのでエラーになるはず")

	// v3を使用（最新版、全パーミッションあり）
	fmt.Printf("\n--- v3スキーマ (%s) を使用（最新版） ---\n", v3Version)
	checkWithVersion(ctx, permissionClient, v3Version, "bob", "doc1", "delete", "v3にはdeleteパーミッションがある（bobはadmin）")
	checkWithVersion(ctx, permissionClient, v3Version, "charlie", "doc1", "delete", "charlieはeditorなのでdeleteできない")

	// バージョン指定なし（最新版を使用）
	fmt.Println("\n--- バージョン指定なし（最新版を自動使用） ---")
	checkWithVersion(ctx, permissionClient, "", "bob", "doc1", "delete", "最新版（v3）が使用される")

	fmt.Println("\n🎉 スキーマバージョン管理のデモが完了しました!")
}

func checkWithVersion(ctx context.Context, client pb.PermissionClient, version, user, doc, perm, description string) {
	fmt.Printf("\n%sが%sを%sできるかチェック... (%s)\n", user, doc, perm, description)

	metadata := &pb.PermissionCheckMetadata{}
	if version != "" {
		metadata.SchemaVersion = version
	}

	resp, err := client.Check(ctx, &pb.PermissionCheckRequest{
		Metadata: metadata,
		Entity: &pb.Entity{
			Type: "document",
			Id:   doc,
		},
		Permission: perm,
		Subject: &pb.Subject{
			Type: "user",
			Id:   user,
		},
	})

	if err != nil {
		fmt.Printf("❌ エラー: %v\n", err)
		return
	}

	if resp.Can == pb.CheckResult_CHECK_RESULT_ALLOWED {
		fmt.Printf("✅ 許可されました\n")
	} else {
		fmt.Printf("❌ 拒否されました\n")
	}
}
