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

	permissionClient := pb.NewPermissionClient(conn)
	dataClient := pb.NewDataClient(conn)
	schemaClient := pb.NewSchemaClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("=== 企業文書管理システムのアクセス権限一覧 ===")
	fmt.Println()

	// Step 1: スキーマを定義
	fmt.Println("Step 1: スキーマ定義")
	schema := `
entity user {}

entity department {
  relation member @user

  permission view = member
}

entity document {
  relation owner @user
  relation editor @user
  relation viewer @user
  relation department @department

  attribute confidential: bool
  attribute security_level: int

  // 所有者は全権限
  permission delete = owner
  permission edit = owner or editor

  // 閲覧権限: 直接の閲覧者、編集者、所有者、または部署メンバー
  // ただし機密文書の場合はセキュリティレベルが3以上必要
  permission view = owner or editor or viewer or
    (department.member and rule(!resource.confidential or subject.security_level >= 3))
}
`

	_, err = schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込み失敗: %v", err)
	}
	fmt.Println("✓ スキーマ定義完了")
	fmt.Println()

	// Step 2: 組織構造とデータを準備
	fmt.Println("Step 2: 組織構造とデータ準備")

	// 部署メンバーシップ
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// 営業部
			{Entity: &pb.Entity{Type: "department", Id: "sales"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "department", Id: "sales"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "bob"}},

			// 開発部
			{Entity: &pb.Entity{Type: "department", Id: "engineering"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
			{Entity: &pb.Entity{Type: "department", Id: "engineering"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "dave"}},

			// 人事部
			{Entity: &pb.Entity{Type: "department", Id: "hr"}, Relation: "member", Subject: &pb.Subject{Type: "user", Id: "eve"}},
		},
	})
	if err != nil {
		log.Fatalf("部署データ書き込み失敗: %v", err)
	}

	// 文書の権限設定
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// doc1: 営業資料（非機密、営業部所属）
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Relation: "department", Subject: &pb.Subject{Type: "department", Id: "sales"}},

			// doc2: 製品仕様書（非機密、開発部所属、営業部のBobも閲覧可能）
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Relation: "department", Subject: &pb.Subject{Type: "department", Id: "engineering"}},

			// doc3: 機密給与データ（機密、人事部所属）
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "eve"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Relation: "department", Subject: &pb.Subject{Type: "department", Id: "hr"}},

			// doc4: 技術文書（非機密、開発部所属）
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "dave"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Relation: "editor", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Relation: "department", Subject: &pb.Subject{Type: "department", Id: "engineering"}},

			// doc5: 社内規定（非機密、全部署からアクセス可能だが明示的に設定）
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Relation: "owner", Subject: &pb.Subject{Type: "user", Id: "eve"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "alice"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "bob"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "charlie"}},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Relation: "viewer", Subject: &pb.Subject{Type: "user", Id: "dave"}},
		},
	})
	if err != nil {
		log.Fatalf("文書データ書き込み失敗: %v", err)
	}

	// 文書の属性（機密性、セキュリティレベル）
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Attributes: []*pb.Attribute{
			// doc1: 営業資料（非機密）
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "confidential", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "security_level", Value: structpb.NewNumberValue(1)},

			// doc2: 製品仕様書（非機密）
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Attribute: "confidential", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Attribute: "security_level", Value: structpb.NewNumberValue(2)},

			// doc3: 機密給与データ（機密！）
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Attribute: "confidential", Value: structpb.NewBoolValue(true)},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Attribute: "security_level", Value: structpb.NewNumberValue(5)},

			// doc4: 技術文書（非機密）
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Attribute: "confidential", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Attribute: "security_level", Value: structpb.NewNumberValue(2)},

			// doc5: 社内規定（非機密）
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Attribute: "confidential", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Attribute: "security_level", Value: structpb.NewNumberValue(1)},
		},
	})
	if err != nil {
		log.Fatalf("属性データ書き込み失敗: %v", err)
	}

	// ユーザーのセキュリティレベル
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Attributes: []*pb.Attribute{
			{Entity: &pb.Entity{Type: "user", Id: "alice"}, Attribute: "security_level", Value: structpb.NewNumberValue(2)},
			{Entity: &pb.Entity{Type: "user", Id: "bob"}, Attribute: "security_level", Value: structpb.NewNumberValue(2)},
			{Entity: &pb.Entity{Type: "user", Id: "charlie"}, Attribute: "security_level", Value: structpb.NewNumberValue(3)},
			{Entity: &pb.Entity{Type: "user", Id: "dave"}, Attribute: "security_level", Value: structpb.NewNumberValue(3)},
			{Entity: &pb.Entity{Type: "user", Id: "eve"}, Attribute: "security_level", Value: structpb.NewNumberValue(5)}, // 人事部長は最高レベル
		},
	})
	if err != nil {
		log.Fatalf("ユーザー属性書き込み失敗: %v", err)
	}

	fmt.Println("✓ データ準備完了")
	fmt.Println("  - 5ユーザー（alice, bob, charlie, dave, eve）")
	fmt.Println("  - 3部署（営業、開発、人事）")
	fmt.Println("  - 5文書（営業資料、仕様書、給与データ、技術文書、社内規定）")
	fmt.Println()

	// Step 3: LookupEntity - 特定ユーザーがアクセスできる全ドキュメント
	fmt.Println("=== Step 3: LookupEntity - ユーザーごとのアクセス可能ドキュメント ===")

	users := []string{"alice", "bob", "charlie", "dave", "eve"}
	for _, user := range users {
		fmt.Printf("\n[%s がアクセスできるドキュメント]\n", user)

		// view 権限でLookupEntity
		lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
			EntityType: "document",
			Permission: "view",
			Subject: &pb.Subject{
				Type: "user",
				Id:   user,
			},
		})
		if err != nil {
			log.Fatalf("LookupEntity失敗 (%s): %v", user, err)
		}

		if len(lookupResp.EntityIds) == 0 {
			fmt.Printf("  → アクセス可能な文書なし\n")
		} else {
			fmt.Printf("  → %d件の文書にアクセス可能:\n", len(lookupResp.EntityIds))
			for _, docId := range lookupResp.EntityIds {
				fmt.Printf("     - %s\n", docId)
			}
		}
	}

	// Step 4: LookupSubject - 特定ドキュメントにアクセスできる全ユーザー
	fmt.Println("\n\n=== Step 4: LookupSubject - ドキュメントごとのアクセス可能ユーザー ===")

	documents := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}
	for _, doc := range documents {
		fmt.Printf("\n[%s にアクセスできるユーザー]\n", doc)

		// view 権限でLookupSubject
		lookupResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   doc,
			},
			Permission: "view",
			SubjectReference: &pb.SubjectReference{
				Type: "user",
			},
		})
		if err != nil {
			log.Fatalf("LookupSubject失敗 (%s): %v", doc, err)
		}

		if len(lookupResp.SubjectIds) == 0 {
			fmt.Printf("  → アクセス可能なユーザーなし\n")
		} else {
			fmt.Printf("  → %d人のユーザーがアクセス可能:\n", len(lookupResp.SubjectIds))
			for _, userId := range lookupResp.SubjectIds {
				fmt.Printf("     - %s\n", userId)
			}
		}
	}

	// Step 5: SubjectPermission - 特定ユーザーが特定ドキュメントに持つ全権限
	fmt.Println("\n\n=== Step 5: SubjectPermission - ユーザーが持つ権限の詳細 ===")

	testCases := []struct {
		user string
		doc  string
	}{
		{"alice", "doc1"}, // 所有者
		{"bob", "doc2"},   // 閲覧者
		{"charlie", "doc4"}, // 編集者
		{"dave", "doc4"},  // 所有者
		{"eve", "doc3"},   // 所有者（機密文書）
	}

	for _, tc := range testCases {
		fmt.Printf("\n[%s が %s に持つ権限]\n", tc.user, tc.doc)

		subjPermResp, err := permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
			Entity: &pb.Entity{
				Type: "document",
				Id:   tc.doc,
			},
			Subject: &pb.Subject{
				Type: "user",
				Id:   tc.user,
			},
		})
		if err != nil {
			log.Fatalf("SubjectPermission失敗 (%s, %s): %v", tc.user, tc.doc, err)
		}

		// 権限をチェック
		permissions := []string{"view", "edit", "delete"}
		hasAnyPermission := false
		for _, perm := range permissions {
			if result, ok := subjPermResp.Results[perm]; ok {
				if result == pb.CheckResult_CHECK_RESULT_ALLOWED {
					fmt.Printf("  ✓ %s: 可能\n", perm)
					hasAnyPermission = true
				} else {
					fmt.Printf("  ✗ %s: 不可\n", perm)
				}
			}
		}

		if !hasAnyPermission {
			fmt.Printf("  → 権限なし\n")
		}
	}

	// Step 6: 実用的なユースケース例
	fmt.Println("\n\n=== Step 6: 実用的なユースケース例 ===")

	fmt.Println("\n【ユースケース1】管理画面: Charlie のダッシュボード")
	fmt.Println("  - 「あなたが編集できる文書」を表示する場合:")

	editableResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "edit",
		Subject: &pb.Subject{
			Type: "user",
			Id:   "charlie",
		},
	})
	if err != nil {
		log.Fatalf("LookupEntity (edit) 失敗: %v", err)
	}

	fmt.Printf("  → Charlie が編集可能: %v\n", editableResp.EntityIds)

	fmt.Println("\n【ユースケース2】監査ログ: doc2 のアクセス権限レポート")
	fmt.Println("  - 「この文書に誰がアクセスできるか」を確認:")

	accessResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc2",
		},
		Permission: "view",
		SubjectReference: &pb.SubjectReference{
			Type: "user",
		},
	})
	if err != nil {
		log.Fatalf("LookupSubject失敗: %v", err)
	}

	fmt.Printf("  → doc2 にアクセス可能なユーザー: %v\n", accessResp.SubjectIds)

	fmt.Println("\n【ユースケース3】権限確認UI: Eve の doc3 での権限")
	fmt.Println("  - 「編集ボタン」「削除ボタン」の表示判定:")

	evePermResp, err := permissionClient.SubjectPermission(ctx, &pb.PermissionSubjectPermissionRequest{
		Entity: &pb.Entity{
			Type: "document",
			Id:   "doc3",
		},
		Subject: &pb.Subject{
			Type: "user",
			Id:   "eve",
		},
	})
	if err != nil {
		log.Fatalf("SubjectPermission失敗: %v", err)
	}

	fmt.Println("  → UI表示判定:")
	if evePermResp.Results["view"] == pb.CheckResult_CHECK_RESULT_ALLOWED {
		fmt.Println("     ✓ 文書を表示")
	}
	if evePermResp.Results["edit"] == pb.CheckResult_CHECK_RESULT_ALLOWED {
		fmt.Println("     ✓ 「編集」ボタンを表示")
	}
	if evePermResp.Results["delete"] == pb.CheckResult_CHECK_RESULT_ALLOWED {
		fmt.Println("     ✓ 「削除」ボタンを表示")
	}

	fmt.Println("\n🎉 一覧系APIのデモ完了!")
	fmt.Println("\n💡 これらのAPIの使い分け:")
	fmt.Println("  - LookupEntity: 「このユーザーがアクセスできる全リソース」を取得")
	fmt.Println("  - LookupSubject: 「このリソースにアクセスできる全ユーザー」を取得")
	fmt.Println("  - SubjectPermission: 「このユーザーがこのリソースに持つ全権限」を取得")
	fmt.Println("\n📊 典型的な利用シーン:")
	fmt.Println("  - ダッシュボードの「あなたのファイル」一覧 → LookupEntity")
	fmt.Println("  - 共有設定画面の「共有相手一覧」 → LookupSubject")
	fmt.Println("  - ファイル詳細画面のボタン表示制御 → SubjectPermission")
}
