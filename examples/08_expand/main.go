package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// printExpandTree は、Expand APIで取得した権限ツリーを見やすく表示する関数
func printExpandTree(expand *pb.Expand, indent int) {
	if expand == nil {
		return
	}

	prefix := strings.Repeat("  ", indent)

	// ツリーノードの場合
	if treeNode := expand.GetExpand(); treeNode != nil {
		switch treeNode.Operation {
		case pb.ExpandTreeNode_OPERATION_UNION:
			fmt.Printf("%s🔀 結合（OR）\n", prefix)
		case pb.ExpandTreeNode_OPERATION_INTERSECTION:
			fmt.Printf("%s🔀 交差（AND）\n", prefix)
		case pb.ExpandTreeNode_OPERATION_EXCLUSION:
			fmt.Printf("%s🔀 除外（EXCLUDE）\n", prefix)
		default:
			fmt.Printf("%s🔀 不明な操作: %v\n", prefix, treeNode.Operation)
		}

		for _, child := range treeNode.Children {
			printExpandTree(child, indent+1)
		}
		return
	}

	// リーフノードの場合
	if leafNode := expand.GetLeaf(); leafNode != nil {
		if subjects := leafNode.GetSubjects(); subjects != nil {
			fmt.Printf("%s🍃 直接的な関係:\n", prefix)
			for _, subject := range subjects.Subjects {
				if subject.Relation != "" {
					fmt.Printf("%s   - %s:%s#%s\n", prefix, subject.Type, subject.Id, subject.Relation)
				} else {
					fmt.Printf("%s   - %s:%s\n", prefix, subject.Type, subject.Id)
				}
			}
		} else if values := leafNode.GetValues(); values != nil {
			fmt.Printf("%s🍃 値: %v\n", prefix, values.Values)
		} else if value := leafNode.GetValue(); value != nil {
			fmt.Printf("%s🍃 値: %v\n", prefix, value)
		}
	}
}

func main() {
	fmt.Println("==========================================================")
	fmt.Println("Keruberosu Example 08: Expand API - Permission Tree Visualization")
	fmt.Println("==========================================================")
	fmt.Println()
	fmt.Println("この例では、Expand APIを使用して権限決定ツリーを可視化します。")
	fmt.Println("GitHub風のorganization → repository → issue階層を使って、")
	fmt.Println("複雑な権限継承と結合を実演します。")
	fmt.Println()

	// gRPCサーバーに接続
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("gRPCサーバーへの接続に失敗: %v", err)
	}
	defer conn.Close()

	schemaClient := pb.NewSchemaClient(conn)
	dataClient := pb.NewDataClient(conn)
	permissionClient := pb.NewPermissionClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ステップ1: スキーマ定義
	fmt.Println("ステップ1: スキーマ定義")
	fmt.Println("---")
	fmt.Println("organization, repository, issue の3階層構造を定義します。")
	fmt.Println()

	schema := `
entity user {}

entity organization {
  relation admin: user
  relation member: user

  permission delete = admin
  permission manage = admin
  permission view = admin or member
}

entity repository {
  relation owner: user
  relation maintainer: user
  relation contributor: user
  relation parent: organization

  attribute private: bool

  // private repositories: only direct roles can view
  // public repositories: direct roles OR org members can view
  permission delete = owner
  permission push = owner or maintainer
  permission view = owner or maintainer or contributor or (parent.view and rule(!resource.private))
}

entity issue {
  relation assignee: user
  relation reporter: user
  relation parent: repository

  attribute confidential: bool

  // confidential issues: only assignee or reporter
  // non-confidential: anyone who can view the repository
  permission edit = assignee or reporter
  permission view = (assignee or reporter) or (parent.view and rule(!resource.confidential))
}
`

	_, err = schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込みに失敗: %v", err)
	}
	fmt.Println("✓ スキーマ定義完了")
	fmt.Println()

	// ステップ2: 関係とアトリビュートの登録
	fmt.Println("ステップ2: 関係とアトリビュートの登録")
	fmt.Println("---")
	fmt.Println("以下の構造を作成:")
	fmt.Println("- Organization: acme-corp (admin: alice, member: bob, charlie)")
	fmt.Println("- Repository: backend-api (private, owner: bob, maintainer: charlie)")
	fmt.Println("- Repository: frontend (public, owner: alice, contributor: dave)")
	fmt.Println("- Issue: backend-api/issue-1 (confidential, assignee: bob)")
	fmt.Println("- Issue: frontend/issue-2 (non-confidential, reporter: dave)")
	fmt.Println()

	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// Organization: acme-corp
			{Entity: &pb.Entity{Type: "organization", Id: "acme-corp"}, Relation: "admin", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "organization", Id: "acme-corp"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "organization", Id: "acme-corp"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "charlie"}},

			// Repository: backend-api (private)
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "parent", Subject: &pb.Subject{Type: "organization", Id: "acme-corp"}},
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Relation: "maintainer", Subject: &pb.Subject{Type: "user", Id: "charlie"}},

			// Repository: frontend (public)
			{Entity: &pb.Entity{Type: "repository", Id: "frontend"}, Relation: "parent", Subject: &pb.Subject{Type: "organization", Id: "acme-corp"}},
			{Entity: &pb.Entity{Type: "repository", Id: "frontend"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "repository", Id: "frontend"}, Relation: "contributor", Subject: &pb.Subject{Type: "user", Id: "dave"}},

			// Issue: backend-api/issue-1 (confidential)
			{Entity: &pb.Entity{Type: "issue", Id: "issue-1"}, Relation: "parent", Subject: &pb.Subject{Type: "repository", Id: "backend-api"}},
			{Entity: &pb.Entity{Type: "issue", Id: "issue-1"}, Relation: "assignee", Subject: &pb.Subject{Type: "user", Id: "bob"}},

			// Issue: frontend/issue-2 (non-confidential)
			{Entity: &pb.Entity{Type: "issue", Id: "issue-2"}, Relation: "parent", Subject: &pb.Subject{Type: "repository", Id: "frontend"}},
			{Entity: &pb.Entity{Type: "issue", Id: "issue-2"}, Relation: "reporter", Subject: &pb.Subject{Type: "user", Id: "dave"}},
		},
		Attributes: []*pb.Attribute{
			// backend-api is private
			{Entity: &pb.Entity{Type: "repository", Id: "backend-api"}, Attribute: "private", Value: structpb.NewBoolValue(true)},

			// frontend is public
			{Entity: &pb.Entity{Type: "repository", Id: "frontend"}, Attribute: "private", Value: structpb.NewBoolValue(false)},

			// issue-1 is confidential
			{Entity: &pb.Entity{Type: "issue", Id: "issue-1"}, Attribute: "confidential", Value: structpb.NewBoolValue(true)},

			// issue-2 is not confidential
			{Entity: &pb.Entity{Type: "issue", Id: "issue-2"}, Attribute: "confidential", Value: structpb.NewBoolValue(false)},
		},
	})
	if err != nil {
		log.Fatalf("データ書き込みに失敗: %v", err)
	}
	fmt.Println("✓ データ登録完了")
	fmt.Println()

	// ステップ3: Expand APIで権限ツリーを取得
	fmt.Println("==========================================================")
	fmt.Println("ステップ3: Expand APIで権限決定ツリーを可視化")
	fmt.Println("==========================================================")
	fmt.Println()

	testCases := []struct {
		name       string
		entityType string
		entityID   string
		permission string
		explanation string
	}{
		{
			name:       "パブリックリポジトリの閲覧権限",
			entityType: "repository",
			entityID:   "frontend",
			permission: "view",
			explanation: "パブリックリポジトリなので、直接的な役割（owner, maintainer, contributor）に加えて、\n" +
				"    organizationのメンバーでも閲覧可能。複雑な結合（OR）ツリーが表示されます。",
		},
		{
			name:       "プライベートリポジトリの閲覧権限",
			entityType: "repository",
			entityID:   "backend-api",
			permission: "view",
			explanation: "プライベートリポジトリなので、直接的な役割（owner, maintainer, contributor）のみ。\n" +
				"    parent.view条件はルールで除外されます。",
		},
		{
			name:       "非機密Issueの閲覧権限",
			entityType: "issue",
			entityID:   "issue-2",
			permission: "view",
			explanation: "非機密Issueなので、(assignee or reporter) OR (parent.view)の結合ツリー。\n" +
				"    parent.viewは再帰的にrepositoryの権限ツリーに展開されます。",
		},
		{
			name:       "機密Issueの閲覧権限",
			entityType: "issue",
			entityID:   "issue-1",
			permission: "view",
			explanation: "機密Issueなので、assigneeとreporterのみ。parent.view条件はルールで除外されます。",
		},
		{
			name:       "リポジトリへのプッシュ権限",
			entityType: "repository",
			entityID:   "backend-api",
			permission: "push",
			explanation: "pushはowner or maintainerのシンプルな結合ツリー。",
		},
	}

	for i, tc := range testCases {
		fmt.Printf("%d. %s\n", i+1, tc.name)
		fmt.Printf("   対象: %s:%s#%s\n", tc.entityType, tc.entityID, tc.permission)
		fmt.Printf("   説明: %s\n", tc.explanation)
		fmt.Println()

		expandResp, err := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
			Entity: &pb.Entity{
				Type: tc.entityType,
				Id:   tc.entityID,
			},
			Permission: tc.permission,
		})
		if err != nil {
			log.Printf("   ❌ Expand失敗: %v\n", err)
			fmt.Println()
			continue
		}

		if expandResp.Tree == nil {
			fmt.Println("   ⚠️  権限ツリーが空です")
			fmt.Println()
			continue
		}

		fmt.Println("   📊 権限決定ツリー:")
		if expandResp.Tree != nil {
			printExpandTree(expandResp.Tree, 2)
		}
		fmt.Println()
		fmt.Println("   ---")
		fmt.Println()
	}

	// ステップ4: 実践的な使用例
	fmt.Println("==========================================================")
	fmt.Println("ステップ4: Expand APIの実践的な使用例")
	fmt.Println("==========================================================")
	fmt.Println()

	fmt.Println("使用例1: デバッグ - なぜaliceはbackend-apiを閲覧できないのか？")
	fmt.Println("---")

	// aliceがbackend-apiを閲覧できるか確認
	checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "repository", Id: "backend-api"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		log.Printf("Check失敗: %v\n", err)
	} else {
		if checkResp.Can == pb.CheckResult_CHECK_RESULT_DENIED {
			fmt.Println("❌ alice は backend-api を閲覧できません")
			fmt.Println()
			fmt.Println("理由を調べるために、Expand APIで権限ツリーを確認:")

			expandResp, err := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
				Entity:     &pb.Entity{Type: "repository", Id: "backend-api"},
				Permission: "view",
			})
			if err != nil {
				log.Printf("Expand失敗: %v\n", err)
			} else {
				if expandResp.Tree != nil {
					printExpandTree(expandResp.Tree, 1)
				}
				fmt.Println()
				fmt.Println("📝 分析結果:")
				fmt.Println("   - backend-apiはprivate=trueなので、parent.view条件が除外される")
				fmt.Println("   - aliceはowner, maintainer, contributorいずれでもない")
				fmt.Println("   - aliceはorgのadminだが、privateリポジトリには自動アクセスできない")
				fmt.Println()
			}
		}
	}

	fmt.Println("使用例2: アクセス監査 - frontendリポジトリに誰がアクセスできるか？")
	fmt.Println("---")

	expandResp, err := permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Entity:     &pb.Entity{Type: "repository", Id: "frontend"},
		Permission: "view",
	})
	if err != nil {
		log.Printf("Expand失敗: %v\n", err)
	} else {
		fmt.Println("📊 frontendの閲覧権限ツリー:")
		if expandResp.Tree != nil {
			printExpandTree(expandResp.Tree, 1)
		}
		fmt.Println()
		fmt.Println("📝 分析結果:")
		fmt.Println("   - 直接的な役割: alice (owner), dave (contributor)")
		fmt.Println("   - parent.view経由: bob, charlie (orgメンバー)")
		fmt.Println("   - 合計4人がアクセス可能")
		fmt.Println()
	}

	fmt.Println("使用例3: 権限設計の検証 - issueの閲覧権限が正しく設計されているか？")
	fmt.Println("---")

	fmt.Println("機密Issue (issue-1):")
	expandResp, err = permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Entity:     &pb.Entity{Type: "issue", Id: "issue-1"},
		Permission: "view",
	})
	if err != nil {
		log.Printf("Expand失敗: %v\n", err)
	} else {
		if expandResp.Tree != nil {
			printExpandTree(expandResp.Tree, 1)
		}
		fmt.Println("   ✓ 機密Issueはassigneeとreporterのみがアクセス可能（parent.view除外）")
		fmt.Println()
	}

	fmt.Println("非機密Issue (issue-2):")
	expandResp, err = permissionClient.Expand(ctx, &pb.PermissionExpandRequest{
		Entity:     &pb.Entity{Type: "issue", Id: "issue-2"},
		Permission: "view",
	})
	if err != nil {
		log.Printf("Expand失敗: %v\n", err)
	} else {
		if expandResp.Tree != nil {
			printExpandTree(expandResp.Tree, 1)
		}
		fmt.Println("   ✓ 非機密Issueはrepo閲覧権限を継承（parent.view含む）")
		fmt.Println()
	}

	fmt.Println("==========================================================")
	fmt.Println("まとめ")
	fmt.Println("==========================================================")
	fmt.Println()
	fmt.Println("Expand APIは以下のような場合に有用です:")
	fmt.Println()
	fmt.Println("1. 🐛 デバッグ:")
	fmt.Println("   - なぜアクセスが拒否されたのか？")
	fmt.Println("   - どの条件が満たされていないのか？")
	fmt.Println()
	fmt.Println("2. 📊 監査:")
	fmt.Println("   - 特定リソースに誰がアクセスできるか？")
	fmt.Println("   - 権限がどのように継承されているか？")
	fmt.Println()
	fmt.Println("3. ✅ 検証:")
	fmt.Println("   - 権限設計が意図通りか？")
	fmt.Println("   - ルール条件が正しく動作しているか？")
	fmt.Println()
	fmt.Println("4. 📚 ドキュメント:")
	fmt.Println("   - 複雑な権限ロジックの可視化")
	fmt.Println("   - チームメンバーへの説明資料")
	fmt.Println()
}
