package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

func TestOutputPathParser_Parse(t *testing.T) {
	t.Run("it returns the rendered string when the string is valid", func(t *testing.T) {
		result, err := core.OutputPathParserParse("{{ foo }}", map[string]string{"foo": "bar"}, core.OutputPathParserDefaultFetcher)
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		if result != "bar" {
			t.Errorf("expected 'bar', got '%s'", result)
		}
	})

	t.Run("it works with filepath-like strings", func(t *testing.T) {
		result, err := core.OutputPathParserParse("{{ foo }}/{{ bar }}", map[string]string{"foo": "bar", "bar": "baz"}, core.OutputPathParserDefaultFetcher)
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		if result != "bar/baz" {
			t.Errorf("expected 'bar/baz', got '%s'", result)
		}
	})

	t.Run("it works when mixing text and variables", func(t *testing.T) {
		result, err := core.OutputPathParserParse("{{ foo }} text {{ bar }}", map[string]string{"foo": "bar", "bar": "baz"}, core.OutputPathParserDefaultFetcher)
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		if result != "bar text baz" {
			t.Errorf("expected 'bar text baz', got '%s'", result)
		}
	})

	t.Run("it removes the placeholder but doesn't blow up when the variable isn't provided", func(t *testing.T) {
		result, err := core.OutputPathParserParse("{{ foo }}", map[string]string{}, core.OutputPathParserDefaultFetcher)
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		if result != "" {
			t.Errorf("expected '', got '%s'", result)
		}
	})

	t.Run("it accepts any number of spaces between open and closing tags", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"{{foo}}", "bar"},
			{"{{ foo}}", "bar"},
			{"{{foo }}", "bar"},
			{"{{   foo   }}", "bar"},
		}
		variables := map[string]string{"foo": "bar"}

		for _, tc := range testCases {
			result, err := core.OutputPathParserParse(tc.input, variables, core.OutputPathParserDefaultFetcher)
			if err != nil {
				t.Errorf("got error for '%s': %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("for '%s': expected '%s', got '%s'", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("it doesn't interpret single braces as variables", func(t *testing.T) {
		result, err := core.OutputPathParserParse("{foo}", map[string]string{}, core.OutputPathParserDefaultFetcher)
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		if result != "{foo}" {
			t.Errorf("expected '{foo}', got '%s'", result)
		}
	})

	t.Run("it returns an error when the string is invalid", func(t *testing.T) {
		result, err := core.OutputPathParserParse("{{ 1-1 }", map[string]string{}, core.OutputPathParserDefaultFetcher)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if err.Error() != "expected end of string" {
			t.Errorf("expected 'expected end of string', got '%s'", err.Error())
		}
		if result != "" {
			t.Errorf("expected empty result on error, got '%s'", result)
		}
	})

	t.Run("it supports a custom fetcher function", func(t *testing.T) {
		customFetcher := func(identifier string, variables map[string]string) string {
			return "quux"
		}

		result, err := core.OutputPathParserParse("{{ foo }}", map[string]string{}, customFetcher)
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		if result != "quux" {
			t.Errorf("expected 'quux', got '%s'", result)
		}
	})
}
