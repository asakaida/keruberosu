package entities

import (
	"testing"
	"time"
)

func TestAttribute_String(t *testing.T) {
	tests := []struct {
		name string
		attr Attribute
		want string
	}{
		{
			name: "string value",
			attr: Attribute{
				EntityType: "document",
				EntityID:   "1",
				Name:       "public",
				Value:      true,
			},
			want: "document:1.public = true",
		},
		{
			name: "number value",
			attr: Attribute{
				EntityType: "document",
				EntityID:   "123",
				Name:       "version",
				Value:      42,
			},
			want: "document:123.version = 42",
		},
		{
			name: "slice value",
			attr: Attribute{
				EntityType: "document",
				EntityID:   "xyz",
				Name:       "tags",
				Value:      []string{"foo", "bar"},
			},
			want: "document:xyz.tags = [foo bar]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.attr.String(); got != tt.want {
				t.Errorf("Attribute.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttribute_Validate(t *testing.T) {
	tests := []struct {
		name    string
		attr    Attribute
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid attribute",
			attr: Attribute{
				EntityType: "document",
				EntityID:   "1",
				Name:       "public",
				Value:      true,
			},
			wantErr: false,
		},
		{
			name: "missing entity type",
			attr: Attribute{
				EntityType: "",
				EntityID:   "1",
				Name:       "public",
				Value:      true,
			},
			wantErr: true,
			errMsg:  "entity type is required",
		},
		{
			name: "missing entity ID",
			attr: Attribute{
				EntityType: "document",
				EntityID:   "",
				Name:       "public",
				Value:      true,
			},
			wantErr: true,
			errMsg:  "entity ID is required",
		},
		{
			name: "missing attribute name",
			attr: Attribute{
				EntityType: "document",
				EntityID:   "1",
				Name:       "",
				Value:      true,
			},
			wantErr: true,
			errMsg:  "attribute name is required",
		},
		{
			name: "missing value",
			attr: Attribute{
				EntityType: "document",
				EntityID:   "1",
				Name:       "public",
				Value:      nil,
			},
			wantErr: true,
			errMsg:  "attribute value is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.attr.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Attribute.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Attribute.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestAttribute_MarshalValue(t *testing.T) {
	tests := []struct {
		name    string
		attr    Attribute
		want    string
		wantErr bool
	}{
		{
			name: "bool value",
			attr: Attribute{
				Value: true,
			},
			want:    "true",
			wantErr: false,
		},
		{
			name: "string value",
			attr: Attribute{
				Value: "hello",
			},
			want:    `"hello"`,
			wantErr: false,
		},
		{
			name: "number value",
			attr: Attribute{
				Value: 42,
			},
			want:    "42",
			wantErr: false,
		},
		{
			name: "slice value",
			attr: Attribute{
				Value: []string{"foo", "bar"},
			},
			want:    `["foo","bar"]`,
			wantErr: false,
		},
		{
			name: "map value",
			attr: Attribute{
				Value: map[string]interface{}{
					"key": "value",
				},
			},
			want:    `{"key":"value"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.attr.MarshalValue()
			if (err != nil) != tt.wantErr {
				t.Errorf("Attribute.MarshalValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Attribute.MarshalValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttribute_UnmarshalValue(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    interface{}
		wantErr bool
	}{
		{
			name:    "bool value",
			data:    "true",
			want:    true,
			wantErr: false,
		},
		{
			name:    "string value",
			data:    `"hello"`,
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "number value",
			data:    "42",
			want:    float64(42), // JSON numbers unmarshal to float64
			wantErr: false,
		},
		{
			name:    "slice value",
			data:    `["foo","bar"]`,
			want:    []interface{}{"foo", "bar"},
			wantErr: false,
		},
		{
			name:    "map value",
			data:    `{"key":"value"}`,
			want:    map[string]interface{}{"key": "value"},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			data:    `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := &Attribute{}
			err := attr.UnmarshalValue(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Attribute.UnmarshalValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Compare based on type
			switch want := tt.want.(type) {
			case []interface{}:
				got, ok := attr.Value.([]interface{})
				if !ok {
					t.Errorf("Attribute.UnmarshalValue() got type %T, want []interface{}", attr.Value)
					return
				}
				if len(got) != len(want) {
					t.Errorf("Attribute.UnmarshalValue() got %d elements, want %d", len(got), len(want))
					return
				}
				for i := range want {
					if got[i] != want[i] {
						t.Errorf("Attribute.UnmarshalValue() element[%d] = %v, want %v", i, got[i], want[i])
					}
				}
			case map[string]interface{}:
				got, ok := attr.Value.(map[string]interface{})
				if !ok {
					t.Errorf("Attribute.UnmarshalValue() got type %T, want map[string]interface{}", attr.Value)
					return
				}
				if len(got) != len(want) {
					t.Errorf("Attribute.UnmarshalValue() got %d fields, want %d", len(got), len(want))
					return
				}
				for k, v := range want {
					if got[k] != v {
						t.Errorf("Attribute.UnmarshalValue() field %s = %v, want %v", k, got[k], v)
					}
				}
			default:
				if attr.Value != tt.want {
					t.Errorf("Attribute.UnmarshalValue() = %v, want %v", attr.Value, tt.want)
				}
			}
		})
	}
}

func TestAttribute_MarshalUnmarshal_RoundTrip(t *testing.T) {
	now := time.Now()
	attr := &Attribute{
		EntityType: "document",
		EntityID:   "1",
		Name:       "metadata",
		Value: map[string]interface{}{
			"author":  "Alice",
			"version": float64(2),
			"public":  true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Marshal
	data, err := attr.MarshalValue()
	if err != nil {
		t.Fatalf("MarshalValue() error = %v", err)
	}

	// Unmarshal
	attr2 := &Attribute{}
	if err := attr2.UnmarshalValue(data); err != nil {
		t.Fatalf("UnmarshalValue() error = %v", err)
	}

	// Verify
	got, ok := attr2.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("UnmarshalValue() got type %T, want map[string]interface{}", attr2.Value)
	}

	want := attr.Value.(map[string]interface{})
	if len(got) != len(want) {
		t.Errorf("Round trip got %d fields, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("Round trip field %s = %v, want %v", k, got[k], v)
		}
	}
}
