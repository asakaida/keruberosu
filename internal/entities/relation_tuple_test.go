package entities

import "testing"

func TestRelationTuple_String(t *testing.T) {
	tests := []struct {
		name string
		rt   RelationTuple
		want string
	}{
		{
			name: "without subject relation",
			rt: RelationTuple{
				EntityType:  "document",
				EntityID:    "1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			want: "document:1#owner@user:alice",
		},
		{
			name: "with subject relation",
			rt: RelationTuple{
				EntityType:      "document",
				EntityID:        "1",
				Relation:        "viewer",
				SubjectType:     "organization",
				SubjectID:       "org1",
				SubjectRelation: "member",
			},
			want: "document:1#viewer@organization:org1#member",
		},
		{
			name: "complex IDs",
			rt: RelationTuple{
				EntityType:  "folder",
				EntityID:    "abc-123-xyz",
				Relation:    "editor",
				SubjectType: "user",
				SubjectID:   "bob@example.com",
			},
			want: "folder:abc-123-xyz#editor@user:bob@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rt.String(); got != tt.want {
				t.Errorf("RelationTuple.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelationTuple_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rt      RelationTuple
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid relation tuple",
			rt: RelationTuple{
				EntityType:  "document",
				EntityID:    "1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			wantErr: false,
		},
		{
			name: "valid with subject relation",
			rt: RelationTuple{
				EntityType:      "document",
				EntityID:        "1",
				Relation:        "viewer",
				SubjectType:     "organization",
				SubjectID:       "org1",
				SubjectRelation: "member",
			},
			wantErr: false,
		},
		{
			name: "missing entity type",
			rt: RelationTuple{
				EntityType:  "",
				EntityID:    "1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			wantErr: true,
			errMsg:  "entity type is required",
		},
		{
			name: "missing entity ID",
			rt: RelationTuple{
				EntityType:  "document",
				EntityID:    "",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			wantErr: true,
			errMsg:  "entity ID is required",
		},
		{
			name: "missing relation",
			rt: RelationTuple{
				EntityType:  "document",
				EntityID:    "1",
				Relation:    "",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			wantErr: true,
			errMsg:  "relation is required",
		},
		{
			name: "missing subject type",
			rt: RelationTuple{
				EntityType:  "document",
				EntityID:    "1",
				Relation:    "owner",
				SubjectType: "",
				SubjectID:   "alice",
			},
			wantErr: true,
			errMsg:  "subject type is required",
		},
		{
			name: "missing subject ID",
			rt: RelationTuple{
				EntityType:  "document",
				EntityID:    "1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "",
			},
			wantErr: true,
			errMsg:  "subject ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rt.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RelationTuple.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("RelationTuple.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
