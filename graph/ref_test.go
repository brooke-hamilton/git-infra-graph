package graph

import (
	"testing"
)

func TestValidateGraphName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		// Valid names
		{name: "simple name", input: "my-infra", wantErr: ""},
		{name: "alphanumeric", input: "abc123", wantErr: ""},
		{name: "with hyphen", input: "my-graph", wantErr: ""},
		{name: "with underscore", input: "my_graph", wantErr: ""},
		{name: "single char", input: "a", wantErr: ""},
		{name: "with dot middle", input: "a.b", wantErr: ""},
		{name: "at sign in middle", input: "a@b", wantErr: ""},

		// Invalid: empty
		{name: "empty string", input: "", wantErr: "graph name must not be empty"},

		// Invalid: single @
		{name: "single @", input: "@", wantErr: "graph name must not be '@'"},

		// Invalid: leading dot
		{name: "leading dot", input: ".hidden", wantErr: "graph name must not start with '.'"},

		// Invalid: trailing dot
		{name: "trailing dot", input: "name.", wantErr: "graph name must not end with '.'"},

		// Invalid: .lock suffix
		{name: ".lock suffix", input: "name.lock", wantErr: "graph name must not end with '.lock'"},

		// Invalid: double dot
		{name: "double dot", input: "a..b", wantErr: "graph name must not contain '..'"},

		// Invalid: @{ sequence
		{name: "@{ sequence", input: "a@{b", wantErr: "graph name must not contain '@{'"},

		// Invalid: control characters
		{name: "null byte", input: "a\x00b", wantErr: "graph name must not contain control character 0x00"},
		{name: "tab", input: "a\tb", wantErr: "graph name must not contain control character 0x09"},
		{name: "newline", input: "a\nb", wantErr: "graph name must not contain control character 0x0a"},
		{name: "DEL", input: "a\x7fb", wantErr: "graph name must not contain control character 0x7f"},

		// Invalid: forbidden characters
		{name: "space", input: "a b", wantErr: "graph name must not contain ' '"},
		{name: "tilde", input: "a~b", wantErr: "graph name must not contain '~'"},
		{name: "caret", input: "a^b", wantErr: "graph name must not contain '^'"},
		{name: "colon", input: "a:b", wantErr: "graph name must not contain ':'"},
		{name: "question mark", input: "a?b", wantErr: "graph name must not contain '?'"},
		{name: "asterisk", input: "a*b", wantErr: "graph name must not contain '*'"},
		{name: "open bracket", input: "a[b", wantErr: "graph name must not contain '['"},
		{name: "backslash", input: "a\\b", wantErr: "graph name must not contain '\\'"},

		// Invalid: slash
		{name: "slash", input: "a/b", wantErr: "graph name must not contain '/'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGraphName(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateGraphName(%q) = %v, want nil", tt.input, err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateGraphName(%q) = nil, want error containing %q", tt.input, tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("ValidateGraphName(%q) = %q, want %q", tt.input, err.Error(), tt.wantErr)
				}
			}
		})
	}
}
