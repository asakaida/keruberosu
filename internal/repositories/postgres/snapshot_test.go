package postgres

import (
	"testing"
)

func TestSnapshotToken_String(t *testing.T) {
	tests := []struct {
		name     string
		token    SnapshotToken
		expected string
	}{
		{
			name:     "simple token without xip",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: nil},
			expected: "100:200:",
		},
		{
			name:     "token with empty xip",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: []int64{}},
			expected: "100:200:",
		},
		{
			name:     "token with single xip",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: []int64{150}},
			expected: "100:200:150",
		},
		{
			name:     "token with multiple xip",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: []int64{120, 130, 140}},
			expected: "100:200:120,130,140",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseSnapshotToken(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *SnapshotToken
		expectError bool
	}{
		{
			name:        "empty token",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid format - single part",
			input:       "100",
			expectError: true,
		},
		{
			name:  "simple token without xip",
			input: "100:200:",
			expected: &SnapshotToken{
				Xmin: 100,
				Xmax: 200,
				Xip:  nil,
			},
		},
		{
			name:  "token with single xip",
			input: "100:200:150",
			expected: &SnapshotToken{
				Xmin: 100,
				Xmax: 200,
				Xip:  []int64{150},
			},
		},
		{
			name:  "token with multiple xip",
			input: "100:200:120,130,140",
			expected: &SnapshotToken{
				Xmin: 100,
				Xmax: 200,
				Xip:  []int64{120, 130, 140},
			},
		},
		{
			name:        "invalid xmin",
			input:       "abc:200:",
			expectError: true,
		},
		{
			name:        "invalid xmax",
			input:       "100:abc:",
			expectError: true,
		},
		{
			name:        "invalid xip",
			input:       "100:200:abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSnapshotToken(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Xmin != tt.expected.Xmin {
				t.Errorf("Xmin: expected %d, got %d", tt.expected.Xmin, result.Xmin)
			}
			if result.Xmax != tt.expected.Xmax {
				t.Errorf("Xmax: expected %d, got %d", tt.expected.Xmax, result.Xmax)
			}
			if len(result.Xip) != len(tt.expected.Xip) {
				t.Errorf("Xip length: expected %d, got %d", len(tt.expected.Xip), len(result.Xip))
			}
			for i := range result.Xip {
				if result.Xip[i] != tt.expected.Xip[i] {
					t.Errorf("Xip[%d]: expected %d, got %d", i, tt.expected.Xip[i], result.Xip[i])
				}
			}
		})
	}
}

func TestSnapshotToken_IsTransactionVisible(t *testing.T) {
	tests := []struct {
		name     string
		token    SnapshotToken
		xid      int64
		expected bool
	}{
		{
			name:     "transaction before xmin is visible",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: nil},
			xid:      50,
			expected: true,
		},
		{
			name:     "transaction at xmax is not visible",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: nil},
			xid:      200,
			expected: false,
		},
		{
			name:     "transaction after xmax is not visible",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: nil},
			xid:      250,
			expected: false,
		},
		{
			name:     "transaction between xmin and xmax without xip is visible",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: nil},
			xid:      150,
			expected: true,
		},
		{
			name:     "transaction in xip is not visible",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: []int64{150}},
			xid:      150,
			expected: false,
		},
		{
			name:     "transaction between xmin and xmax not in xip is visible",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: []int64{150}},
			xid:      160,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsTransactionVisible(tt.xid)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSnapshotToken_BuildMVCCCondition(t *testing.T) {
	tests := []struct {
		name     string
		token    SnapshotToken
		expected string
	}{
		{
			name:     "simple condition without xip",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: nil},
			expected: "xmin::text::bigint < 200 AND (xmax = 0 OR xmax::text::bigint >= 200)",
		},
		{
			name:     "condition with xip",
			token:    SnapshotToken{Xmin: 100, Xmax: 200, Xip: []int64{120, 130}},
			expected: "xmin::text::bigint < 200 AND (xmax = 0 OR xmax::text::bigint >= 200) AND xmin::text::bigint NOT IN (120,130)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.BuildMVCCCondition()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSnapshotToken_RoundTrip(t *testing.T) {
	tokens := []SnapshotToken{
		{Xmin: 100, Xmax: 200, Xip: nil},
		{Xmin: 100, Xmax: 200, Xip: []int64{}},
		{Xmin: 100, Xmax: 200, Xip: []int64{150}},
		{Xmin: 100, Xmax: 200, Xip: []int64{120, 130, 140}},
		{Xmin: 0, Xmax: 1, Xip: nil},
		{Xmin: 9223372036854775807, Xmax: 9223372036854775807, Xip: nil},
	}

	for _, original := range tokens {
		str := original.String()
		parsed, err := ParseSnapshotToken(str)
		if err != nil {
			t.Errorf("failed to parse token %s: %v", str, err)
			continue
		}

		if parsed.Xmin != original.Xmin || parsed.Xmax != original.Xmax {
			t.Errorf("Xmin/Xmax mismatch: original=%v, parsed=%v", original, parsed)
		}

		// Handle nil vs empty slice comparison
		if len(original.Xip) == 0 && len(parsed.Xip) == 0 {
			continue // Both empty, that's fine
		}

		if len(original.Xip) != len(parsed.Xip) {
			t.Errorf("Xip length mismatch: original=%v, parsed=%v", original.Xip, parsed.Xip)
			continue
		}

		for i := range original.Xip {
			if original.Xip[i] != parsed.Xip[i] {
				t.Errorf("Xip[%d] mismatch: original=%d, parsed=%d", i, original.Xip[i], parsed.Xip[i])
			}
		}
	}
}
