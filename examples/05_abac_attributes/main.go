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

	fmt.Println("===== ABAC: 属性ベースアクセス制御 =====\n")

	// Step 1: スキーマを定義
	schema := `
entity user {}

entity document {
  attribute public: bool
  attribute security_level: int
  attribute department: string
  attribute price: int

  permission view_public = rule(resource.public == true)
  permission view_classified = rule(subject.security_level >= resource.security_level)
  permission view_department = rule(resource.department == subject.department)
  permission view_premium = rule(subject.subscription_tier == "premium" && resource.price > 100)
}
`

	_, err = client.WriteSchema(ctx, &pb.WriteSchemaRequest{
		SchemaDsl: schema,
	})
	if err != nil {
		log.Fatalf("スキーマ書き込み失敗: %v", err)
	}

	// Step 2: ドキュメントの属性を書き込み
	_, err = client.WriteAttributes(ctx, &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			// doc1: 公開ドキュメント
			{
				Entity: &pb.Entity{Type: "document", Id: "doc1"},
				Data: map[string]*structpb.Value{
					"public":         structpb.NewBoolValue(true),
					"security_level": structpb.NewNumberValue(1),
					"department":     structpb.NewStringValue("general"),
					"price":          structpb.NewNumberValue(0),
				},
			},
			// doc2: セキュリティレベル2の機密文書
			{
				Entity: &pb.Entity{Type: "document", Id: "doc2"},
				Data: map[string]*structpb.Value{
					"public":         structpb.NewBoolValue(false),
					"security_level": structpb.NewNumberValue(2),
					"department":     structpb.NewStringValue("engineering"),
					"price":          structpb.NewNumberValue(0),
				},
			},
			// doc3: engineering部署限定
			{
				Entity: &pb.Entity{Type: "document", Id: "doc3"},
				Data: map[string]*structpb.Value{
					"public":         structpb.NewBoolValue(false),
					"security_level": structpb.NewNumberValue(1),
					"department":     structpb.NewStringValue("engineering"),
					"price":          structpb.NewNumberValue(50),
				},
			},
			// doc4: プレミアムコンテンツ（高額）
			{
				Entity: &pb.Entity{Type: "document", Id: "doc4"},
				Data: map[string]*structpb.Value{
					"public":         structpb.NewBoolValue(false),
					"security_level": structpb.NewNumberValue(1),
					"department":     structpb.NewStringValue("general"),
					"price":          structpb.NewNumberValue(150),
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("ドキュメント属性書き込み失敗: %v", err)
	}

	// Step 3: ユーザーの属性を書き込み
	_, err = client.WriteAttributes(ctx, &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			// alice: セキュリティレベル3、engineering部署、プレミアムユーザー
			{
				Entity: &pb.Entity{Type: "user", Id: "alice"},
				Data: map[string]*structpb.Value{
					"security_level":     structpb.NewNumberValue(3),
					"department":         structpb.NewStringValue("engineering"),
					"subscription_tier":  structpb.NewStringValue("premium"),
				},
			},
			// bob: セキュリティレベル1、engineering部署、ベーシックユーザー
			{
				Entity: &pb.Entity{Type: "user", Id: "bob"},
				Data: map[string]*structpb.Value{
					"security_level":     structpb.NewNumberValue(1),
					"department":         structpb.NewStringValue("engineering"),
					"subscription_tier":  structpb.NewStringValue("basic"),
				},
			},
			// charlie: セキュリティレベル5、security部署
			{
				Entity: &pb.Entity{Type: "user", Id: "charlie"},
				Data: map[string]*structpb.Value{
					"security_level":     structpb.NewNumberValue(5),
					"department":         structpb.NewStringValue("security"),
					"subscription_tier":  structpb.NewStringValue("basic"),
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("ユーザー属性書き込み失敗: %v", err)
	}

	// Step 4: 権限チェックテスト
	fmt.Println("テストケース 1: 公開ドキュメント")
	checkABAC(ctx, client, "誰でも", "document", "doc1", "view_public", "guest123", true)

	fmt.Println("\nテストケース 2: セキュリティレベル")
	checkABAC(ctx, client, "alice (level=3)", "document", "doc2", "view_classified", "alice", true)
	checkABAC(ctx, client, "bob (level=1)", "document", "doc2", "view_classified", "bob", false)

	fmt.Println("\nテストケース 3: 部署制限")
	checkABAC(ctx, client, "alice (engineering)", "document", "doc3", "view_department", "alice", true)
	checkABAC(ctx, client, "charlie (security)", "document", "doc3", "view_department", "charlie", false)

	fmt.Println("\nテストケース 4: プレミアムコンテンツ")
	checkABAC(ctx, client, "alice (premium, price>100)", "document", "doc4", "view_premium", "alice", true)
	checkABAC(ctx, client, "bob (basic)", "document", "doc4", "view_premium", "bob", false)

	fmt.Println("\n🎉 ABAC シナリオ完了!")
	fmt.Println("属性ベースの柔軟な権限管理が実現できました")
}

func checkABAC(ctx context.Context, client pb.AuthorizationServiceClient, description, entityType, entityID, permission, subjectID string, expected bool) {
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
			fmt.Printf("✅ %s は %s を %s で閲覧できます\n", description, entityID, permission)
		} else {
			fmt.Printf("❌ %s は %s を %s で閲覧できません\n", description, entityID, permission)
		}
	} else {
		log.Fatalf("テスト失敗: %s - 期待=%v, 実際=%v", description, expected, allowed)
	}
}
