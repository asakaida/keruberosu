package handlers

import (
	"testing"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// === Helper Function Tests ===

func TestProtoToRelationTuple(t *testing.T) {
	tests := []struct {
		name      string
		proto     *pb.Tuple
		wantError bool
	}{
		{
			name: "valid tuple",
			proto: &pb.Tuple{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Subject{Type: "user", Id: "alice"},
			},
			wantError: false,
		},
		{
			name: "missing entity",
			proto: &pb.Tuple{
				Relation: "owner",
				Subject:  &pb.Subject{Type: "user", Id: "alice"},
			},
			wantError: true,
		},
		{
			name: "missing relation",
			proto: &pb.Tuple{
				Entity:  &pb.Entity{Type: "document", Id: "1"},
				Subject: &pb.Subject{Type: "user", Id: "alice"},
			},
			wantError: true,
		},
		{
			name: "missing subject",
			proto: &pb.Tuple{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := protoToRelationTuple(tt.proto)
			if (err != nil) != tt.wantError {
				t.Errorf("protoToRelationTuple() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// === protoValueToInterface Tests ===

func TestProtoValueToInterface(t *testing.T) {
	tests := []struct {
		name      string
		value     *structpb.Value
		want      interface{}
		wantError bool
	}{
		{
			name:      "nil value",
			value:     nil,
			wantError: true,
		},
		{
			name:  "null value",
			value: structpb.NewNullValue(),
			want:  nil,
		},
		{
			name:  "number value",
			value: structpb.NewNumberValue(42.5),
			want:  42.5,
		},
		{
			name:  "string value",
			value: structpb.NewStringValue("hello"),
			want:  "hello",
		},
		{
			name:  "bool value true",
			value: structpb.NewBoolValue(true),
			want:  true,
		},
		{
			name:  "bool value false",
			value: structpb.NewBoolValue(false),
			want:  false,
		},
		{
			name: "struct value",
			value: structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": structpb.NewStringValue("Alice"),
					"age":  structpb.NewNumberValue(30),
				},
			}),
			want: map[string]interface{}{
				"name": "Alice",
				"age":  30.0,
			},
		},
		{
			name: "list value",
			value: structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStringValue("a"),
					structpb.NewStringValue("b"),
					structpb.NewNumberValue(123),
				},
			}),
			want: []interface{}{"a", "b", 123.0},
		},
		{
			name: "nested list value",
			value: structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStringValue("nested"),
						},
					}),
				},
			}),
			want: []interface{}{[]interface{}{"nested"}},
		},
		{
			name: "empty list value",
			value: structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{},
			}),
			want: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := protoValueToInterface(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("protoValueToInterface() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError {
				return
			}

			// Special handling for different types
			switch want := tt.want.(type) {
			case map[string]interface{}:
				gotMap, ok := got.(map[string]interface{})
				if !ok {
					t.Errorf("protoValueToInterface() got type %T, want map[string]interface{}", got)
					return
				}
				if len(gotMap) != len(want) {
					t.Errorf("protoValueToInterface() got %d fields, want %d", len(gotMap), len(want))
				}
				for k, v := range want {
					if gotMap[k] != v {
						t.Errorf("protoValueToInterface() field %s = %v, want %v", k, gotMap[k], v)
					}
				}
			case []interface{}:
				gotSlice, ok := got.([]interface{})
				if !ok {
					t.Errorf("protoValueToInterface() got type %T, want []interface{}", got)
					return
				}
				if len(gotSlice) != len(want) {
					t.Errorf("protoValueToInterface() got %d elements, want %d", len(gotSlice), len(want))
					return
				}
				// Deep comparison for nested slices
				for i := range want {
					if !compareInterfaces(gotSlice[i], want[i]) {
						t.Errorf("protoValueToInterface() element[%d] = %v, want %v", i, gotSlice[i], want[i])
					}
				}
			default:
				if got != tt.want {
					t.Errorf("protoValueToInterface() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Helper function for deep comparison
func compareInterfaces(a, b interface{}) bool {
	switch aVal := a.(type) {
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(aVal) != len(bVal) {
			return false
		}
		for i := range aVal {
			if !compareInterfaces(aVal[i], bVal[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func TestProtoToAttribute(t *testing.T) {
	tests := []struct {
		name      string
		proto     *pb.Attribute
		wantError bool
	}{
		{
			name: "valid attribute",
			proto: &pb.Attribute{
				Entity:    &pb.Entity{Type: "document", Id: "1"},
				Attribute: "public",
				Value:     structpb.NewBoolValue(true),
			},
			wantError: false,
		},
		{
			name: "missing entity",
			proto: &pb.Attribute{
				Attribute: "public",
				Value:     structpb.NewBoolValue(true),
			},
			wantError: true,
		},
		{
			name: "empty attribute",
			proto: &pb.Attribute{
				Entity:    &pb.Entity{Type: "document", Id: "1"},
				Attribute: "",
				Value:     structpb.NewBoolValue(true),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr, err := protoToAttribute(tt.proto)
			if (err != nil) != tt.wantError {
				t.Errorf("protoToAttribute() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError && attr == nil {
				t.Errorf("protoToAttribute() returned nil attribute")
			}
		})
	}
}
