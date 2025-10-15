package e2e

import (
	"context"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestGroupMembershipWithSubjectRelation tests the core feature of Permify compatibility:
// The ability to specify subject relations in a single tuple.
// Example: drive:eng_drive#member@group:engineering#member
// NOTE: このテストはサーバーを起動してから個別に実行してください
func TestGroupMembershipWithSubjectRelation(t *testing.T) {
	t.Skip("このテストは個別に実行してください（go test -run TestGroupMembership）")
	// Connect to gRPC server
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	schemaClient := pb.NewSchemaClient(conn)
	dataClient := pb.NewDataClient(conn)
	permissionClient := pb.NewPermissionClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Define schema with groups and drives
	schema := `
entity user {}

entity group {
  relation member @user

  permission view = member
}

entity drive {
  relation member @user | group#member

  permission view = member
}
`

	t.Log("Step 1: Writing schema with group membership support")
	_, err = schemaClient.Write(ctx, &pb.SchemaWriteRequest{
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}
	t.Log("✓ Schema written successfully")

	// Step 2: Create group memberships
	t.Log("Step 2: Creating group memberships")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			// Engineering group members
			{
				Entity:   &pb.Entity{Type: "group", Id: "engineering"},
				Relation: "member",
				Subject:  &pb.Subject{Type: "user", Id: "alice"},
			},
			{
				Entity:   &pb.Entity{Type: "group", Id: "engineering"},
				Relation: "member",
				Subject:  &pb.Subject{Type: "user", Id: "bob"},
			},
			// Marketing group members
			{
				Entity:   &pb.Entity{Type: "group", Id: "marketing"},
				Relation: "member",
				Subject:  &pb.Subject{Type: "user", Id: "charlie"},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteRelations (groups) failed: %v", err)
	}
	t.Log("✓ Group memberships created")

	// Step 3: Assign group as drive member with subject relation
	// This is the KEY FEATURE: drive:eng_drive#member@group:engineering#member
	t.Log("Step 3: Assigning group to drive with subject relation")
	_, err = dataClient.Write(ctx, &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{
				Entity:   &pb.Entity{Type: "drive", Id: "eng_drive"},
				Relation: "member",
				Subject: &pb.Subject{
					Type:     "group",
					Id:       "engineering",
					Relation: "member", // ✅ This is the Permify-compatible feature!
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteRelations (drive-group) failed: %v", err)
	}
	t.Log("✓ Group assigned to drive with subject relation")

	// Step 4: Verify that alice (engineering member) can view eng_drive
	t.Log("Step 4: Verifying alice can view eng_drive")
	checkResp, err := permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "drive", Id: "eng_drive"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("Check failed for alice: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Fatalf("Expected alice to have view permission on eng_drive")
	}
	t.Log("✓ Alice can view eng_drive (via group:engineering#member)")

	// Step 5: Verify that bob (engineering member) can view eng_drive
	t.Log("Step 5: Verifying bob can view eng_drive")
	checkResp, err = permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "drive", Id: "eng_drive"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "bob"},
	})
	if err != nil {
		t.Fatalf("Check failed for bob: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Fatalf("Expected bob to have view permission on eng_drive")
	}
	t.Log("✓ Bob can view eng_drive (via group:engineering#member)")

	// Step 6: Verify that charlie (marketing member) CANNOT view eng_drive
	t.Log("Step 6: Verifying charlie cannot view eng_drive")
	checkResp, err = permissionClient.Check(ctx, &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "drive", Id: "eng_drive"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "charlie"},
	})
	if err != nil {
		t.Fatalf("Check failed for charlie: %v", err)
	}
	if checkResp.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Fatalf("Expected charlie to NOT have view permission on eng_drive")
	}
	t.Log("✓ Charlie cannot view eng_drive (not in engineering group)")

	// Step 7: LookupEntity - find all drives alice can view
	t.Log("Step 7: Looking up all drives alice can view")
	lookupResp, err := permissionClient.LookupEntity(ctx, &pb.PermissionLookupEntityRequest{
		EntityType: "drive",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	})
	if err != nil {
		t.Fatalf("LookupEntity failed: %v", err)
	}
	if len(lookupResp.EntityIds) != 1 || lookupResp.EntityIds[0] != "eng_drive" {
		t.Fatalf("Expected alice to see [eng_drive], got: %v", lookupResp.EntityIds)
	}
	t.Log("✓ LookupEntity correctly returns eng_drive for alice")

	// Step 8: LookupSubject - find all users who can view eng_drive
	t.Log("Step 8: Looking up all users who can view eng_drive")
	lookupSubjectResp, err := permissionClient.LookupSubject(ctx, &pb.PermissionLookupSubjectRequest{
		Entity:           &pb.Entity{Type: "drive", Id: "eng_drive"},
		Permission:       "view",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	})
	if err != nil {
		t.Fatalf("LookupSubject failed: %v", err)
	}

	// Should return alice and bob
	expectedUsers := map[string]bool{"alice": true, "bob": true}
	if len(lookupSubjectResp.SubjectIds) != 2 {
		t.Fatalf("Expected 2 subjects, got %d: %v", len(lookupSubjectResp.SubjectIds), lookupSubjectResp.SubjectIds)
	}
	for _, subjectID := range lookupSubjectResp.SubjectIds {
		if !expectedUsers[subjectID] {
			t.Fatalf("Unexpected subject: %s", subjectID)
		}
	}
	t.Log("✓ LookupSubject correctly returns alice and bob")

	t.Log("\n🎉 Group membership with subject relation test passed!")
	t.Log("✅ Permify-compatible single-tuple group assignment works correctly")
}
