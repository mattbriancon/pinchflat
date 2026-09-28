package app_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
)

func TestOutputPathParser_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		variables  map[string]string
		fetcher    func(string, map[string]string) string
		want       string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "returns the rendered string when the string is valid",
			input:     "{{ foo }}",
			variables: map[string]string{"foo": "bar"},
			fetcher:   defaultFetcher,
			want:      "bar",
		},
		{
			name:      "works with filepath-like strings",
			input:     "{{ foo }}/{{ bar }}",
			variables: map[string]string{"foo": "bar", "bar": "baz"},
			fetcher:   defaultFetcher,
			want:      "bar/baz",
		},
		{
			name:      "works when mixing text and variables",
			input:     "{{ foo }} text {{ bar }}",
			variables: map[string]string{"foo": "bar", "bar": "baz"},
			fetcher:   defaultFetcher,
			want:      "bar text baz",
		},
		{
			name:      "removes the placeholder but doesn't blow up when the variable isn't provided",
			input:     "{{ foo }}",
			variables: map[string]string{},
			fetcher:   defaultFetcher,
			want:      "",
		},
		{
			name:      "accepts any number of spaces between open and closing tags (no spaces)",
			input:     "{{foo}}",
			variables: map[string]string{"foo": "bar"},
			fetcher:   defaultFetcher,
			want:      "bar",
		},
		{
			name:      "accepts any number of spaces between open and closing tags (left space)",
			input:     "{{ foo}}",
			variables: map[string]string{"foo": "bar"},
			fetcher:   defaultFetcher,
			want:      "bar",
		},
		{
			name:      "accepts any number of spaces between open and closing tags (right space)",
			input:     "{{foo }}",
			variables: map[string]string{"foo": "bar"},
			fetcher:   defaultFetcher,
			want:      "bar",
		},
		{
			name:      "accepts any number of spaces between open and closing tags (many spaces)",
			input:     "{{   foo   }}",
			variables: map[string]string{"foo": "bar"},
			fetcher:   defaultFetcher,
			want:      "bar",
		},
		{
			name:      "doesn't interpret single braces as variables",
			input:     "{foo}",
			variables: map[string]string{},
			fetcher:   defaultFetcher,
			want:      "{foo}",
		},
		{
			name:       "returns an error when the string is invalid",
			input:      "{{ 1-1 }",
			variables:  map[string]string{},
			fetcher:    defaultFetcher,
			want:       "",
			wantErr:    true,
			wantErrMsg: "expected end of string",
		},
		{
			name:      "supports a custom fetcher function",
			input:     "{{ foo }}",
			variables: map[string]string{},
			fetcher: func(identifier string, variables map[string]string) string {
				return "quux"
			},
			want: "quux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := app.OutputPathParserParse(tt.input, tt.variables, tt.fetcher)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if err.Error() != tt.wantErrMsg {
					t.Errorf("expected '%s', got '%s'", tt.wantErrMsg, err.Error())
				}
				if result != "" {
					t.Errorf("expected empty result on error, got '%s'", result)
				}
			} else {
				if err != nil {
					t.Errorf("got unexpected error: %v", err)
				}
				if result != tt.want {
					t.Errorf("expected '%s', got '%s'", tt.want, result)
				}
			}
		})
	}
}

// defaultFetcher looks identifiers up in variables, rendering missing ones as "".
func defaultFetcher(identifier string, variables map[string]string) string {
	return variables[identifier]
}
