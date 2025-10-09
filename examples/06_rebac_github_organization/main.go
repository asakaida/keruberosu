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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("===== ReBAC: GitHub風の組織・リポジトリ・Issue管理（3階層ネスト） =====\n")

	// Step 1: スキーマを定義
	// Organization → Repository → Issue の3階層構造
	schema := `
entity user {}

entity organization {
  relation admin: user
  relation member: user

  permission manage = admin
  permission view = admin or member
}

entity repository {
  relation org: organization
  relation maintainer: user
  relation contributor: user

  permission delete = org.admin
  permission manage = org.admin or maintainer
  permission write = org.admin or maintainer or contributor
  permission read = org.admin or maintainer or contributor or org.view
}

entity issue {
  relation repo: repository
  relation assignee: user

  permission close = repo.manage
  permission edit = repo.manage or assignee
  permission view = repo.read
}
`

	fmt.Println("📋 スキーマを定義中...")
	_, err = client.WriteSchema(ctx, &pb.WriteSchemaRequest{
		SchemaDsl: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込み失敗: %v", err)
	}
	fmt.Println("✅ スキーマ定義完了\n")

	// Step 2: データ構造の説明
	fmt.Println("📁 組織構造:")
	fmt.Println("  Acme Corp (組織)")
	fmt.Println("    ├─ Alice: admin (組織管理者)")
	fmt.Println("    └─ Diana: member (組織メンバー)")
	fmt.Println()
	fmt.Println("  backend-api (リポジトリ)")
	fmt.Println("    ├─ 所属: Acme Corp")
	fmt.Println("    ├─ Bob: maintainer (リポジトリ管理者)")
	fmt.Println("    └─ Eve: contributor (コントリビューター)")
	fmt.Println()
	fmt.Println("  frontend-app (リポジトリ)")
	fmt.Println("    └─ 所属: Acme Corp")
	fmt.Println()
	fmt.Println("  Issue #123 (課題)")
	fmt.Println("    ├─ 所属: backend-api")
	fmt.Println("    └─ Charlie: assignee (担当者)")
	fmt.Println()

	// Step 3: 関係性データを書き込み
	fmt.Println("💾 関係性データを書き込み中...")
	_, err = client.WriteRelations(ctx, &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			// Acme Corp 組織
			{Entity: &pb.Entity{Type: "organization", Id: "acme"}, Relation: "admin", Subject: &pb.Entity{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "organization", Id: "acme"}, Relation: "member", Subject: &pb.Entity{Type: "user", Id: "diana"}},

			// backend-api リポジトリ
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "org", Subject: &pb.Entity{Type: "organization", Id: "acme"}},
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "maintainer", Subject: &pb.Entity{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "contributor", Subject: &pb.Entity{Type: "user", Id: "eve"}},

			// frontend-app リポジトリ
			{Entity: &pb.Entity{Type: "repository", Id: "frontend-app"}, Relation: "org", Subject: &pb.Entity{Type: "organization", Id: "acme"}},

			// Issue #123（backend-api に所属）
			{Entity: &pb.Entity{Type: "issue", Id: "123"}, Relation: "repo", Subject: &pb.Entity{Type: "repository", Id: "backend-api"}},
			{Entity: &pb.Entity{Type: "issue", Id: "123"}, Relation: "assignee", Subject: &pb.Entity{Type: "user", Id: "charlie"}},

			// Issue #456（frontend-app に所属）
			{Entity: &pb.Entity{Type: "issue", Id: "456"}, Relation: "repo", Subject: &pb.Entity{Type: "repository", Id: "frontend-app"}},
		},
	})
	if err != nil {
		log.Fatalf("関係性書き込み失敗: %v", err)
	}
	fmt.Println("✅ データ書き込み完了\n")

	// Step 4: 階層的権限のテスト
	fmt.Println("🔐 権限チェック開始\n")

	// 4-1: Alice（組織管理者）の権限
	fmt.Println("【Alice（組織管理者）の権限】")
	checkPermission(ctx, client, "Alice", "organization", "acme", "manage", "alice", true, "組織管理権限")
	checkPermission(ctx, client, "Alice", "repository", "backend-api", "delete", "alice", true, "リポジトリ削除権限（org.admin経由）")
	checkPermission(ctx, client, "Alice", "repository", "backend-api", "write", "alice", true, "リポジトリ書き込み権限（org.admin経由）")
	checkPermission(ctx, client, "Alice", "issue", "123", "close", "alice", true, "Issue クローズ権限（repo.manage → org.admin経由）")
	checkPermission(ctx, client, "Alice", "issue", "123", "view", "alice", true, "Issue 閲覧権限（repo.read → org.view経由）")
	fmt.Println()

	// 4-2: Bob（リポジトリ管理者）の権限
	fmt.Println("【Bob（backend-api リポジトリ管理者）の権限】")
	checkPermission(ctx, client, "Bob", "repository", "backend-api", "manage", "bob", true, "リポジトリ管理権限")
	checkPermission(ctx, client, "Bob", "repository", "backend-api", "delete", "bob", false, "リポジトリ削除不可（org.admin のみ）")
	checkPermission(ctx, client, "Bob", "issue", "123", "close", "bob", true, "Issue クローズ権限（repo.manage経由）")
	checkPermission(ctx, client, "Bob", "issue", "123", "edit", "bob", true, "Issue 編集権限（repo.manage経由）")
	checkPermission(ctx, client, "Bob", "issue", "456", "view", "bob", false, "他リポジトリのIssue閲覧不可")
	fmt.Println()

	// 4-3: Charlie（Issue担当者）の権限
	fmt.Println("【Charlie（Issue #123 担当者）の権限】")
	checkPermission(ctx, client, "Charlie", "issue", "123", "edit", "charlie", true, "担当Issueの編集権限")
	checkPermission(ctx, client, "Charlie", "issue", "123", "close", "charlie", false, "Issueクローズ不可（repo.manage が必要）")
	checkPermission(ctx, client, "Charlie", "issue", "456", "edit", "charlie", false, "他のIssue編集不可")
	fmt.Println()

	// 4-4: Diana（組織メンバー）の権限
	fmt.Println("【Diana（組織メンバー）の権限】")
	checkPermission(ctx, client, "Diana", "organization", "acme", "view", "diana", true, "組織閲覧権限")
	checkPermission(ctx, client, "Diana", "organization", "acme", "manage", "diana", false, "組織管理不可")
	checkPermission(ctx, client, "Diana", "repository", "backend-api", "read", "diana", true, "リポジトリ閲覧権限（org.view経由）")
	checkPermission(ctx, client, "Diana", "repository", "backend-api", "write", "diana", false, "リポジトリ書き込み不可")
	checkPermission(ctx, client, "Diana", "issue", "123", "view", "diana", true, "Issue 閲覧権限（repo.read → org.view経由）")
	checkPermission(ctx, client, "Diana", "issue", "123", "edit", "diana", false, "Issue 編集不可")
	fmt.Println()

	// 4-5: Eve（コントリビューター）の権限
	fmt.Println("【Eve（backend-api コントリビューター）の権限】")
	checkPermission(ctx, client, "Eve", "repository", "backend-api", "write", "eve", true, "リポジトリ書き込み権限")
	checkPermission(ctx, client, "Eve", "repository", "backend-api", "manage", "eve", false, "リポジトリ管理不可")
	checkPermission(ctx, client, "Eve", "issue", "123", "view", "eve", true, "Issue 閲覧権限（repo.read経由）")
	checkPermission(ctx, client, "Eve", "issue", "123", "edit", "eve", false, "Issue 編集不可（担当者でない）")
	fmt.Println()

	// Step 5: LookupEntity でIssue検索
	fmt.Println("🔍 LookupEntity: Bob が閲覧できる Issue を検索")
	lookupResp, err := client.LookupEntity(ctx, &pb.LookupEntityRequest{
		EntityType: "issue",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		log.Fatalf("LookupEntity 失敗: %v", err)
	}
	fmt.Printf("   → 見つかった Issue: %v\n", lookupResp.EntityIds)
	fmt.Println()

	// まとめ
	fmt.Println("🎉 3階層ネストのReBAC シナリオ完了!")
	fmt.Println()
	fmt.Println("階層構造:")
	fmt.Println("  Organization (組織)")
	fmt.Println("    └─ Repository (リポジトリ)")
	fmt.Println("        └─ Issue (課題)")
	fmt.Println()
	fmt.Println("権限継承の例:")
	fmt.Println("  - issue.view → repo.read → org.view")
	fmt.Println("  - issue.close → repo.manage → org.admin")
	fmt.Println("  - repo.delete → org.admin")
}

func checkPermission(ctx context.Context, client pb.AuthorizationServiceClient, user, entityType, entityID, permission, subjectID string, expected bool, description string) {
	resp, err := client.Check(ctx, &pb.CheckRequest{
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
			fmt.Printf("   ✅ %s: %s を %s できます - %s\n", user, entityID, permission, description)
		} else {
			fmt.Printf("   ❌ %s: %s を %s できません - %s\n", user, entityID, permission, description)
		}
	} else {
		log.Fatalf("テスト失敗: %s/%s/%s - 期待=%v, 実際=%v", user, entityID, permission, expected, allowed)
	}
}
