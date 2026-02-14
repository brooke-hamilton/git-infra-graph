package graph

import (
	"testing"
)

func TestParseNodePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantGraph    string
		wantSegments []string
		wantErr      string
	}{
		// Valid paths
		{
			name:         "single segment (graph name only)",
			input:        "default",
			wantGraph:    "default",
			wantSegments: []string{},
		},
		{
			name:         "two segments",
			input:        "default/network",
			wantGraph:    "default",
			wantSegments: []string{"network"},
		},
		{
			name:         "multi segment",
			input:        "default/network/vpc",
			wantGraph:    "default",
			wantSegments: []string{"network", "vpc"},
		},
		{
			name:         "deep path",
			input:        "default/a/b/c/d/e",
			wantGraph:    "default",
			wantSegments: []string{"a", "b", "c", "d", "e"},
		},
		{
			name:         "leading slash trimmed",
			input:        "/default/network",
			wantGraph:    "default",
			wantSegments: []string{"network"},
		},
		{
			name:         "trailing slash trimmed",
			input:        "default/network/",
			wantGraph:    "default",
			wantSegments: []string{"network"},
		},
		{
			name:         "leading and trailing slashes trimmed",
			input:        "/default/network/vpc/",
			wantGraph:    "default",
			wantSegments: []string{"network", "vpc"},
		},

		// Invalid paths
		{
			name:    "empty string",
			input:   "",
			wantErr: "invalid path: empty",
		},
		{
			name:    "single slash",
			input:   "/",
			wantErr: "invalid path: empty",
		},
		{
			name:    "multiple slashes only",
			input:   "///",
			wantErr: "invalid path: empty",
		},
		{
			name:    "empty segment in middle",
			input:   "default//vpc",
			wantErr: "invalid path: empty segment",
		},
		{
			name:    "empty segment in middle before trailing slash",
			input:   "default//network/",
			wantErr: "invalid path: empty segment",
		},
		{
			name:    "invalid graph name with double dots",
			input:   "a..b/node",
			wantErr: "graph name must not contain '..'",
		},
		{
			name:    "invalid graph name with space",
			input:   "my graph/node",
			wantErr: "graph name must not contain ' '",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			graphName, segments, err := ParseNodePath(tt.input)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseNodePath(%q) returned no error, want error containing %q", tt.input, tt.wantErr)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Errorf("ParseNodePath(%q) error = %q, want error containing %q", tt.input, err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseNodePath(%q) returned unexpected error: %v", tt.input, err)
			}

			if graphName != tt.wantGraph {
				t.Errorf("ParseNodePath(%q) graphName = %q, want %q", tt.input, graphName, tt.wantGraph)
			}

			if len(segments) != len(tt.wantSegments) {
				t.Fatalf("ParseNodePath(%q) segments = %v (len %d), want %v (len %d)",
					tt.input, segments, len(segments), tt.wantSegments, len(tt.wantSegments))
			}

			for i, seg := range segments {
				if seg != tt.wantSegments[i] {
					t.Errorf("ParseNodePath(%q) segments[%d] = %q, want %q",
						tt.input, i, seg, tt.wantSegments[i])
				}
			}
		})
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
