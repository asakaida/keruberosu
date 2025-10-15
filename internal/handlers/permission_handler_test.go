package handlers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// === Authorization Tests ===

func TestPermissionHandler_Check_Allowed(t *testing.T) {
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			if req.TenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", req.TenantID)
			}
			if req.EntityType != "document" {
				t.Errorf("expected entity type 'document', got %s", req.EntityType)
			}
			return &authorization.CheckResponse{Allowed: true}, nil
		},
	}

	handler := NewPermissionHandler(
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		&mockSchemaService{},
	)

	req := &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	resp, err := handler.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected ALLOWED, got %v", resp.Can)
	}
}

func TestPermissionHandler_Check_Denied(t *testing.T) {
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			return &authorization.CheckResponse{Allowed: false}, nil
		},
	}

	handler := NewPermissionHandler(
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		&mockSchemaService{},
	)

	req := &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	resp, err := handler.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Errorf("expected DENIED, got %v", resp.Can)
	}
}

func TestPermissionHandler_Check_MissingEntity(t *testing.T) {
	handler := NewPermissionHandler(
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaService{},
	)

	req := &pb.PermissionCheckRequest{
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	_, err := handler.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing entity")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestPermissionHandler_Expand_Success(t *testing.T) {
	mockExpander := &mockExpander{
		expandFunc: func(ctx context.Context, req *authorization.ExpandRequest) (*authorization.ExpandResponse, error) {
			if req.TenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", req.TenantID)
			}
			return &authorization.ExpandResponse{
				Tree: &authorization.ExpandNode{
					Type: "union",
					Children: []*authorization.ExpandNode{
						{Type: "leaf", Subject: "user:alice"},
					},
				},
			}, nil
		},
	}

	handler := NewPermissionHandler(
		&mockChecker{},
		mockExpander,
		&mockLookup{},
		&mockSchemaService{},
	)

	req := &pb.PermissionExpandRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
	}

	resp, err := handler.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Tree == nil {
		t.Fatal("expected tree to be set")
	}

	if resp.Tree.GetExpand() == nil {
		t.Fatal("expected expand node to be set")
	}

	if resp.Tree.GetExpand().Operation != pb.ExpandTreeNode_OPERATION_UNION {
		t.Errorf("expected operation UNION, got %v", resp.Tree.GetExpand().Operation)
	}
}

func TestPermissionHandler_LookupEntity_Success(t *testing.T) {
	mockLookup := &mockLookup{
		lookupEntityFunc: func(ctx context.Context, req *authorization.LookupEntityRequest) (*authorization.LookupEntityResponse, error) {
			if req.TenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", req.TenantID)
			}
			return &authorization.LookupEntityResponse{
				EntityIDs:     []string{"doc1", "doc2"},
				NextPageToken: "",
			}, nil
		},
	}

	handler := NewPermissionHandler(
		&mockChecker{},
		&mockExpander{},
		mockLookup,
		&mockSchemaService{},
	)

	req := &pb.PermissionLookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	resp, err := handler.LookupEntity(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.EntityIds) != 2 {
		t.Errorf("expected 2 entity IDs, got %d", len(resp.EntityIds))
	}
}

func TestPermissionHandler_LookupSubject_Success(t *testing.T) {
	mockLookup := &mockLookup{
		lookupSubjectFunc: func(ctx context.Context, req *authorization.LookupSubjectRequest) (*authorization.LookupSubjectResponse, error) {
			if req.TenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", req.TenantID)
			}
			return &authorization.LookupSubjectResponse{
				SubjectIDs:    []string{"alice", "bob"},
				NextPageToken: "",
			}, nil
		},
	}

	handler := NewPermissionHandler(
		&mockChecker{},
		&mockExpander{},
		mockLookup,
		&mockSchemaService{},
	)

	req := &pb.PermissionLookupSubjectRequest{
		Entity:           &pb.Entity{Type: "document", Id: "1"},
		Permission:       "view",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	}

	resp, err := handler.LookupSubject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.SubjectIds) != 2 {
		t.Errorf("expected 2 subject IDs, got %d", len(resp.SubjectIds))
	}
}

func TestPermissionHandler_SubjectPermission_Success(t *testing.T) {
	checkCount := 0
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			checkCount++
			// Allow "view", deny "edit"
			return &authorization.CheckResponse{Allowed: req.Permission == "view"}, nil
		},
	}

	mockSchemaService := &mockSchemaService{
		getSchemaEntityFunc: func(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID: tenantID,
				Entities: []*entities.Entity{
					{
						Name: "document",
						Permissions: []*entities.Permission{
							{Name: "view"},
							{Name: "edit"},
						},
					},
				},
			}, nil
		},
	}

	handler := NewPermissionHandler(
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		mockSchemaService,
	)

	req := &pb.PermissionSubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	}

	resp, err := handler.SubjectPermission(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}

	if resp.Results["view"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected view to be ALLOWED")
	}

	if resp.Results["edit"] != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Errorf("expected edit to be DENIED")
	}

	if checkCount != 2 {
		t.Errorf("expected 2 checks, got %d", checkCount)
	}
}

func TestPermissionHandler_SubjectPermission_SchemaNotFound(t *testing.T) {
	mockSchemaService := &mockSchemaService{
		getSchemaEntityFunc: func(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
			return nil, fmt.Errorf("schema not found for tenant %s: %w", tenantID, repositories.ErrNotFound)
		},
	}

	handler := NewPermissionHandler(
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		mockSchemaService,
	)

	req := &pb.PermissionSubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	}

	_, err := handler.SubjectPermission(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for schema not found")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound error, got %v", st.Code())
	}
}

func TestPermissionHandler_SubjectPermission_EntityNotFound(t *testing.T) {
	mockSchemaService := &mockSchemaService{
		getSchemaEntityFunc: func(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID: tenantID,
				Entities: []*entities.Entity{
					{Name: "other_entity"},
				},
			}, nil
		},
	}

	handler := NewPermissionHandler(
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		mockSchemaService,
	)

	req := &pb.PermissionSubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	}

	_, err := handler.SubjectPermission(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for entity not found")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound error, got %v", st.Code())
	}
}

func TestPermissionHandler_Check_WithContextualTuples(t *testing.T) {
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			if len(req.ContextualTuples) != 1 {
				t.Errorf("expected 1 contextual tuple, got %d", len(req.ContextualTuples))
			}
			return &authorization.CheckResponse{Allowed: true}, nil
		},
	}

	handler := NewPermissionHandler(
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		&mockSchemaService{},
	)

	req := &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
		Context: &pb.Context{
			Tuples: []*pb.Tuple{
				{
					Entity:   &pb.Entity{Type: "document", Id: "1"},
					Relation: "owner",
					Subject:  &pb.Subject{Type: "user", Id: "alice"},
				},
			},
		},
	}

	resp, err := handler.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected ALLOWED, got %v", resp.Can)
	}
}

func TestPermissionHandler_Check_CheckerError(t *testing.T) {
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			return nil, errors.New("internal error")
		},
	}

	handler := NewPermissionHandler(
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		&mockSchemaService{},
	)

	req := &pb.PermissionCheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	_, err := handler.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for checker error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Internal {
		t.Errorf("expected Internal error, got %v", st.Code())
	}
}
