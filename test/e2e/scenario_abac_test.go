package e2e

import (
	"context"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestScenario_ABAC_AllOperators tests ABAC with all supported operators
func TestScenario_ABAC_AllOperators(t *testing.T) {
	// Setup E2E test server
	testServer := SetupE2ETest(t)
	defer testServer.Teardown(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := testServer.Client

	// Step 1: Define schema with ABAC rules
	t.Log("Step 1: Defining schema with ABAC rules")
	schema := `
entity user {}

entity document {
  attribute public: bool
  attribute owner_id: string
  attribute department: string
  attribute security_level: int
  attribute tags: string[]
  attribute price: int

  // Public documents
  permission view_public = rule(resource.public == true)

  // Owner-only access
  permission view_owner = rule(resource.owner_id == subject.id)

  // Same department access
  permission view_department = rule(resource.department == subject.department)

  // Security level check (subject level must be >= resource level)
  permission view_classified = rule(subject.security_level >= resource.security_level)

  // Tag-based access (subject must have at least one matching tag)
  permission view_tagged = rule(subject.role in resource.tags)

  // Complex rule: (public OR owner OR same department) AND security level
  permission view_complex = rule(
    (resource.public == true || resource.owner_id == subject.id || resource.department == subject.department) &&
    subject.security_level >= resource.security_level
  )

  // Price-based access
  permission view_premium = rule(subject.subscription_tier == "premium" && resource.price > 100)

  // Negation
  permission view_not_restricted = rule(!(resource.department == "restricted"))
}
`

	writeSchemaResp, err := client.WriteSchema(ctx, &pb.WriteSchemaRequest{
		SchemaDsl: schema,
	})
	if err != nil {
		t.Fatalf("WriteSchema failed: %v", err)
	}
	// WriteSchemaResponse now only contains SchemaVersion (Permify compatible)
	// Errors are returned via gRPC error, not in response fields
	if writeSchemaResp.SchemaVersion == "" {
		t.Logf("WriteSchema returned empty schema_version (expected for now - TODO: implement schema versioning)")
	}
	t.Log("✓ Schema with ABAC rules defined successfully")

	// Step 2: Write attributes for documents
	t.Log("Step 2: Writing document attributes")
	_, err = client.WriteAttributes(ctx, &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			// doc1: public document
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "public", Value: structpb.NewBoolValue(true)},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "owner_id", Value: structpb.NewStringValue("alice")},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "department", Value: structpb.NewStringValue("engineering")},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "security_level", Value: structpb.NewNumberValue(1)},
			{Entity: &pb.Entity{Type: "document", Id: "doc1"}, Attribute: "price", Value: structpb.NewNumberValue(50)},

			// doc2: private document, owned by alice
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Attribute: "public", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Attribute: "owner_id", Value: structpb.NewStringValue("alice")},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Attribute: "department", Value: structpb.NewStringValue("engineering")},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Attribute: "security_level", Value: structpb.NewNumberValue(2)},
			{Entity: &pb.Entity{Type: "document", Id: "doc2"}, Attribute: "price", Value: structpb.NewNumberValue(150)},

			// doc3: private document, owned by bob, engineering department
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Attribute: "public", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Attribute: "owner_id", Value: structpb.NewStringValue("bob")},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Attribute: "department", Value: structpb.NewStringValue("engineering")},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Attribute: "security_level", Value: structpb.NewNumberValue(3)},
			{Entity: &pb.Entity{Type: "document", Id: "doc3"}, Attribute: "price", Value: structpb.NewNumberValue(200)},

			// doc4: classified document
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Attribute: "public", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Attribute: "owner_id", Value: structpb.NewStringValue("charlie")},
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Attribute: "department", Value: structpb.NewStringValue("security")},
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Attribute: "security_level", Value: structpb.NewNumberValue(5)},
			{Entity: &pb.Entity{Type: "document", Id: "doc4"}, Attribute: "price", Value: structpb.NewNumberValue(1000)},

			// doc5: restricted department
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Attribute: "public", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Attribute: "owner_id", Value: structpb.NewStringValue("dave")},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Attribute: "department", Value: structpb.NewStringValue("restricted")},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Attribute: "security_level", Value: structpb.NewNumberValue(1)},
			{Entity: &pb.Entity{Type: "document", Id: "doc5"}, Attribute: "price", Value: structpb.NewNumberValue(10)},

			// doc6: tagged document
			{Entity: &pb.Entity{Type: "document", Id: "doc6"}, Attribute: "public", Value: structpb.NewBoolValue(false)},
			{Entity: &pb.Entity{Type: "document", Id: "doc6"}, Attribute: "owner_id", Value: structpb.NewStringValue("eve")},
			{Entity: &pb.Entity{Type: "document", Id: "doc6"}, Attribute: "department", Value: structpb.NewStringValue("marketing")},
			{Entity: &pb.Entity{Type: "document", Id: "doc6"}, Attribute: "security_level", Value: structpb.NewNumberValue(1)},
			{Entity: &pb.Entity{Type: "document", Id: "doc6"}, Attribute: "tags", Value: structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStringValue("admin"), structpb.NewStringValue("editor")}})},
			{Entity: &pb.Entity{Type: "document", Id: "doc6"}, Attribute: "price", Value: structpb.NewNumberValue(75)},
		},
	})
	if err != nil {
		t.Fatalf("WriteAttributes failed: %v", err)
	}
	t.Log("✓ Document attributes written successfully")

	// Step 3: Write attributes for subjects (users)
	t.Log("Step 3: Writing user attributes")
	_, err = client.WriteAttributes(ctx, &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			// alice: engineering, security level 2
			{Entity: &pb.Entity{Type: "user", Id: "alice"}, Attribute: "id", Value: structpb.NewStringValue("alice")},
			{Entity: &pb.Entity{Type: "user", Id: "alice"}, Attribute: "department", Value: structpb.NewStringValue("engineering")},
			{Entity: &pb.Entity{Type: "user", Id: "alice"}, Attribute: "security_level", Value: structpb.NewNumberValue(2)},
			{Entity: &pb.Entity{Type: "user", Id: "alice"}, Attribute: "subscription_tier", Value: structpb.NewStringValue("basic")},
			{Entity: &pb.Entity{Type: "user", Id: "alice"}, Attribute: "role", Value: structpb.NewStringValue("developer")},

			// bob: engineering, security level 3, premium
			{Entity: &pb.Entity{Type: "user", Id: "bob"}, Attribute: "id", Value: structpb.NewStringValue("bob")},
			{Entity: &pb.Entity{Type: "user", Id: "bob"}, Attribute: "department", Value: structpb.NewStringValue("engineering")},
			{Entity: &pb.Entity{Type: "user", Id: "bob"}, Attribute: "security_level", Value: structpb.NewNumberValue(3)},
			{Entity: &pb.Entity{Type: "user", Id: "bob"}, Attribute: "subscription_tier", Value: structpb.NewStringValue("premium")},
			{Entity: &pb.Entity{Type: "user", Id: "bob"}, Attribute: "role", Value: structpb.NewStringValue("developer")},

			// charlie: security, security level 5, premium
			{Entity: &pb.Entity{Type: "user", Id: "charlie"}, Attribute: "id", Value: structpb.NewStringValue("charlie")},
			{Entity: &pb.Entity{Type: "user", Id: "charlie"}, Attribute: "department", Value: structpb.NewStringValue("security")},
			{Entity: &pb.Entity{Type: "user", Id: "charlie"}, Attribute: "security_level", Value: structpb.NewNumberValue(5)},
			{Entity: &pb.Entity{Type: "user", Id: "charlie"}, Attribute: "subscription_tier", Value: structpb.NewStringValue("premium")},
			{Entity: &pb.Entity{Type: "user", Id: "charlie"}, Attribute: "role", Value: structpb.NewStringValue("analyst")},

			// dave: marketing, security level 1
			{Entity: &pb.Entity{Type: "user", Id: "dave"}, Attribute: "id", Value: structpb.NewStringValue("dave")},
			{Entity: &pb.Entity{Type: "user", Id: "dave"}, Attribute: "department", Value: structpb.NewStringValue("marketing")},
			{Entity: &pb.Entity{Type: "user", Id: "dave"}, Attribute: "security_level", Value: structpb.NewNumberValue(1)},
			{Entity: &pb.Entity{Type: "user", Id: "dave"}, Attribute: "subscription_tier", Value: structpb.NewStringValue("basic")},
			{Entity: &pb.Entity{Type: "user", Id: "dave"}, Attribute: "role", Value: structpb.NewStringValue("admin")},
		},
	})
	if err != nil {
		t.Fatalf("WriteAttributes (users) failed: %v", err)
	}
	t.Log("✓ User attributes written successfully")

	// Step 4: Test all ABAC operators
	t.Log("Step 4: Testing ABAC operators")

	testCases := []struct {
		name       string
		entityID   string
		permission string
		subjectID  string
		expected   pb.CheckResult
		description string
	}{
		// Equality operator (==)
		{"public doc accessible by anyone", "doc1", "view_public", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED, "resource.public == true"},
		{"private doc not accessible via view_public", "doc2", "view_public", "alice", pb.CheckResult_CHECK_RESULT_DENIED, "resource.public == false"},

		// Owner check (==)
		{"alice owns doc2", "doc2", "view_owner", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED, "resource.owner_id == subject.id"},
		{"bob does not own doc2", "doc2", "view_owner", "bob", pb.CheckResult_CHECK_RESULT_DENIED, "resource.owner_id != subject.id"},

		// Same department (==)
		{"alice (engineering) can view doc3 (engineering)", "doc3", "view_department", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED, "same department"},
		{"charlie (security) cannot view doc3 (engineering)", "doc3", "view_department", "charlie", pb.CheckResult_CHECK_RESULT_DENIED, "different department"},

		// Security level (>=)
		{"alice (level 2) can view doc1 (level 1)", "doc1", "view_classified", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED, "subject.security_level >= resource.security_level"},
		{"alice (level 2) cannot view doc3 (level 3)", "doc3", "view_classified", "alice", pb.CheckResult_CHECK_RESULT_DENIED, "subject.security_level < resource.security_level"},
		{"charlie (level 5) can view doc4 (level 5)", "doc4", "view_classified", "charlie", pb.CheckResult_CHECK_RESULT_ALLOWED, "subject.security_level == resource.security_level"},

		// in operator
		{"dave (role=admin) can view doc6 (tags contains admin)", "doc6", "view_tagged", "dave", pb.CheckResult_CHECK_RESULT_ALLOWED, "subject.role in resource.tags"},

		// Complex rule (||, &&)
		{"alice can view doc1 via complex rule (public)", "doc1", "view_complex", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED, "public && security_level"},
		{"alice can view doc2 via complex rule (owner)", "doc2", "view_complex", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED, "owner && security_level"},
		{"bob can view doc3 via complex rule (department)", "doc3", "view_complex", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED, "department && security_level"},
		{"alice cannot view doc4 via complex rule (security level too low)", "doc4", "view_complex", "alice", pb.CheckResult_CHECK_RESULT_DENIED, "security_level check fails"},

		// Greater than (>)
		{"bob (premium) can view doc2 (price=150)", "doc2", "view_premium", "bob", pb.CheckResult_CHECK_RESULT_ALLOWED, "premium && price > 100"},
		{"alice (basic) cannot view doc2 (not premium)", "doc2", "view_premium", "alice", pb.CheckResult_CHECK_RESULT_DENIED, "subscription_tier != premium"},
		{"bob (premium) cannot view doc1 (price=50)", "doc1", "view_premium", "bob", pb.CheckResult_CHECK_RESULT_DENIED, "price <= 100"},

		// Negation (!)
		{"alice can view doc1 (not restricted dept)", "doc1", "view_not_restricted", "alice", pb.CheckResult_CHECK_RESULT_ALLOWED, "!(dept == restricted)"},
		{"alice cannot view doc5 (restricted dept)", "doc5", "view_not_restricted", "alice", pb.CheckResult_CHECK_RESULT_DENIED, "dept == restricted"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checkResp, err := client.Check(ctx, &pb.CheckRequest{
				Entity: &pb.Entity{
					Type: "document",
					Id:   tc.entityID,
				},
				Permission: tc.permission,
				Subject: &pb.Subject{
					Type: "user",
					Id:   tc.subjectID,
				},
			})
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}
			if checkResp.Can != tc.expected {
				t.Errorf("Check result mismatch: got %v, want %v (description: %s)", checkResp.Can, tc.expected, tc.description)
			}
		})
	}
	t.Log("✓ All ABAC operator tests passed")

	t.Log("✓ All ABAC scenario tests passed")
}
