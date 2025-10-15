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

	fmt.Println("===== ReBAC: GitHub風の組織・リポジトリ・Issue管理（3階層ネスト） =====")

	// Step 1: スキーマを定義
	// Organization → Repository → Issue の3階層構造 + Team (グループメンバーシップ)
	schema := `
entity user {}

entity team {
  relation member @user

  permission view = member
}

entity organization {
  relation admin @user
  relation member @user

  permission manage = admin
  permission view = admin or member
}

entity repository {
  relation org @organization
  relation maintainer @user
  relation contributor @user @team#member

  permission delete = org.admin
  permission manage = org.admin or maintainer
  permission write = org.admin or maintainer or contributor
  permission read = org.admin or maintainer or contributor or org.view
}

entity issue {
  relation repo @repository
  relation assignee @user

  permission close = repo.manage
  permission edit = repo.manage or assignee
  permission view = repo.read
}
`

	fmt.Println("📋 スキーマを定義中...")
	schemaResp, err := schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込み失敗: %v", err)
	}
	fmt.Printf("✅ スキーマ定義完了 (version: %s)\n", schemaResp.SchemaVersion)

	// Step 2: データ構造の説明
	fmt.Println("📁 組織構造:")
	fmt.Println("  Acme Corp (組織)")
	fmt.Println("    ├─ Alice: admin (組織管理者)")
	fmt.Println("    └─ Diana: member (組織メンバー)")
	fmt.Println()
	fmt.Println("  backend-team (チーム) ✨ グループメンバーシップ")
	fmt.Println("    ├─ Frank: member (チームメンバー)")
	fmt.Println("    └─ Grace: member (チームメンバー)")
	fmt.Println()
	fmt.Println("  backend-api (リポジトリ)")
	fmt.Println("    ├─ 所属: Acme Corp")
	fmt.Println("    ├─ Bob: maintainer (リポジトリ管理者)")
	fmt.Println("    ├─ Eve: contributor (コントリビューター)")
	fmt.Println("    └─ backend-team#member: contributor ✨ 1つのタプルでチーム全員に権限付与")
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
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// Acme Corp 組織
			{Entity: &pb.Entity{Type: "organization", Id: "acme"}, Relation: "admin", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "organization", Id: "acme"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "diana"}},

			// backend-team チームメンバー
			{Entity: &pb.Entity{Type: "team", Id: "backend-team"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "frank"}},
			{Entity: &pb.Entity{Type: "team", Id: "backend-team"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "grace"}},

			// backend-api リポジトリ
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "org", Subject: &pb.Subject{Type: "organization", Id: "acme"}},
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "maintainer", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "contributor", Subject: &pb.Subject{Type: "user", Id: "eve"}},
			// ✨ Permify互換: 1つのタプルでチーム全員にcontributor権限を付与
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "contributor", Subject: &pb.Subject{Type: "team", Id: "backend-team", Relation: "member"}},

			// frontend-app リポジトリ
			{Entity: &pb.Entity{Type: "repository", Id: "frontend-app"}, Relation: "org", Subject: &pb.Subject{Type: "organization", Id: "acme"}},

			// Issue #123（backend-api に所属）
			{Entity: &pb.Entity{Type: "issue", Id: "123"}, Relation: "repo", Subject: &pb.Subject{Type: "repository", Id: "backend-api"}},
			{Entity: &pb.Entity{Type: "issue", Id: "123"}, Relation: "assignee", Subject: &pb.Subject{Type: "user", Id: "charlie"}},

			// Issue #456（frontend-app に所属）
			{Entity: &pb.Entity{Type: "issue", Id: "456"}, Relation: "repo", Subject: &pb.Subject{Type: "repository", Id: "frontend-app"}},
		},
	})
	if err != nil {
		log.Fatalf("関係性書き込み失敗: %v", err)
	}
	fmt.Println("✅ データ書き込み完了")

	// Step 4: 階層的権限のテスト
	fmt.Println("🔐 権限チェック開始")

	// 4-1: Alice（組織管理者）の権限
	fmt.Println("【Alice（組織管理者）の権限】")
	checkPermission(ctx, permissionClient, "Alice", "organization", "acme", "manage", "alice", true, "組織管理権限")
	checkPermission(ctx, permissionClient, "Alice", "repository", "backend-api", "delete", "alice", true, "リポジトリ削除権限（org.admin経由）")
	checkPermission(ctx, permissionClient, "Alice", "repository", "backend-api", "write", "alice", true, "リポジトリ書き込み権限（org.admin経由）")
	checkPermission(ctx, permissionClient, "Alice", "issue", "123", "close", "alice", true, "Issue クローズ権限（repo.manage → org.admin経由）")
	checkPermission(ctx, permissionClient, "Alice", "issue", "123", "view", "alice", true, "Issue 閲覧権限（repo.read → org.view経由）")
	fmt.Println()

	// 4-2: Bob（リポジトリ管理者）の権限
	fmt.Println("【Bob（backend-api リポジトリ管理者）の権限】")
	checkPermission(ctx, permissionClient, "Bob", "repository", "backend-api", "manage", "bob", true, "リポジトリ管理権限")
	checkPermission(ctx, permissionClient, "Bob", "repository", "backend-api", "delete", "bob", false, "リポジトリ削除不可（org.admin のみ）")
	checkPermission(ctx, permissionClient, "Bob", "issue", "123", "close", "bob", true, "Issue クローズ権限（repo.manage経由）")
	checkPermission(ctx, permissionClient, "Bob", "issue", "123", "edit", "bob", true, "Issue 編集権限（repo.manage経由）")
	checkPermission(ctx, permissionClient, "Bob", "issue", "456", "view", "bob", false, "他リポジトリのIssue閲覧不可")
	fmt.Println()

	// 4-3: Charlie（Issue担当者）の権限
	fmt.Println("【Charlie（Issue #123 担当者）の権限】")
	checkPermission(ctx, permissionClient, "Charlie", "issue", "123", "edit", "charlie", true, "担当Issueの編集権限")
	checkPermission(ctx, permissionClient, "Charlie", "issue", "123", "close", "charlie", false, "Issueクローズ不可（repo.manage が必要）")
	checkPermission(ctx, permissionClient, "Charlie", "issue", "456", "edit", "charlie", false, "他のIssue編集不可")
	fmt.Println()

	// 4-4: Diana（組織メンバー）の権限
	fmt.Println("【Diana（組織メンバー）の権限】")
	checkPermission(ctx, permissionClient, "Diana", "organization", "acme", "view", "diana", true, "組織閲覧権限")
	checkPermission(ctx, permissionClient, "Diana", "organization", "acme", "manage", "diana", false, "組織管理不可")
	checkPermission(ctx, permissionClient, "Diana", "repository", "backend-api", "read", "diana", true, "リポジトリ閲覧権限（org.view経由）")
	checkPermission(ctx, permissionClient, "Diana", "repository", "backend-api", "write", "diana", false, "リポジトリ書き込み不可")
	checkPermission(ctx, permissionClient, "Diana", "issue", "123", "view", "diana", true, "Issue 閲覧権限（repo.read → org.view経由）")
	checkPermission(ctx, permissionClient, "Diana", "issue", "123", "edit", "diana", false, "Issue 編集不可")
	fmt.Println()

	// 4-5: Eve（コントリビューター）の権限
	fmt.Println("【Eve（backend-api コントリビューター）の権限】")
	checkPermission(ctx, permissionClient, "Eve", "repository", "backend-api", "write", "eve", true, "リポジトリ書き込み権限")
	checkPermission(ctx, permissionClient, "Eve", "repository", "backend-api", "manage", "eve", false, "リポジトリ管理不可")
	checkPermission(ctx, permissionClient, "Eve", "issue", "123", "view", "eve", true, "Issue 閲覧権限（repo.read経由）")
	checkPermission(ctx, permissionClient, "Eve", "issue", "123", "edit", "eve", false, "Issue 編集不可（担当者でない）")
	fmt.Println()

	// 4-6: Frank（backend-team メンバー）の権限 ✨ グループメンバーシップ経由
	fmt.Println("【Frank（backend-team メンバー）の権限】✨ 1つのタプルによるチーム権限継承")
	checkPermission(ctx, permissionClient, "Frank", "repository", "backend-api", "write", "frank", true, "リポジトリ書き込み権限（team#member経由）")
	checkPermission(ctx, permissionClient, "Frank", "repository", "backend-api", "manage", "frank", false, "リポジトリ管理不可")
	checkPermission(ctx, permissionClient, "Frank", "issue", "123", "view", "frank", true, "Issue 閲覧権限（repo.read → team#member経由）")
	checkPermission(ctx, permissionClient, "Frank", "issue", "123", "edit", "frank", false, "Issue 編集不可（担当者でない）")
	checkPermission(ctx, permissionClient, "Frank", "repository", "frontend-app", "write", "frank", false, "他リポジトリ書き込み不可")
	fmt.Println()

	// 4-7: Grace（backend-team メンバー）の権限 ✨ グループメンバーシップ経由
	fmt.Println("【Grace（backend-team メンバー）の権限】✨ 1つのタプルによるチーム権限継承")
	checkPermission(ctx, permissionClient, "Grace", "repository", "backend-api", "write", "grace", true, "リポジトリ書き込み権限（team#member経由）")
	checkPermission(ctx, permissionClient, "Grace", "repository", "backend-api", "manage", "grace", false, "リポジトリ管理不可")
	checkPermission(ctx, permissionClient, "Grace", "issue", "123", "view", "grace", true, "Issue 閲覧権限（repo.read → team#member経由）")
	checkPermission(ctx, permissionClient, "Grace", "issue", "123", "edit", "grace", false, "Issue 編集不可（担当者でない）")
	fmt.Println()

	// Step 5: LookupEntity でIssue検索
	fmt.Println("🔍 LookupEntity: Bob が閲覧できる Issue を検索")
	lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
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
	fmt.Println("🎉 3階層ネスト + グループメンバーシップのReBAC シナリオ完了!")
	fmt.Println()
	fmt.Println("階層構造:")
	fmt.Println("  Organization (組織)")
	fmt.Println("    └─ Repository (リポジトリ)")
	fmt.Println("        └─ Issue (課題)")
	fmt.Println()
	fmt.Println("✨ グループメンバーシップ (Permify互換):")
	fmt.Println("  Team (チーム)")
	fmt.Println("    └─ 1つのタプルでチーム全員に権限付与")
	fmt.Println("    └─ repository:backend-api#contributor@team:backend-team#member")
	fmt.Println()
	fmt.Println("権限継承の例:")
	fmt.Println("  - issue.view → repo.read → org.view")
	fmt.Println("  - issue.close → repo.manage → org.admin")
	fmt.Println("  - repo.delete → org.admin")
	fmt.Println("  - repo.write → contributor → team#member (グループ経由) ✨")
}

func checkPermission(ctx context.Context, client pb.PermissionClient, user, entityType, entityID, permission, subjectID string, expected bool, description string) {
	resp, err := client.Check(ctx, &pb.PermissionCheckRequest{
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
