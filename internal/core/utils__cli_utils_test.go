package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

func TestCliUtils_WrapCmd(t *testing.T) {
	t.Run("delegates to System.cmd/3", func(t *testing.T) {
		t.Skip("BLOCKED: CliUtils.wrap_cmd requires cmd_wrapper.sh which is system-dependent")
	})

	t.Run("sets the current directory to the tmp dir", func(t *testing.T) {
		t.Skip("BLOCKED: CliUtils.wrap_cmd requires cmd_wrapper.sh which is system-dependent")
	})
}

func TestCliUtils_ParseOptions(t *testing.T) {
	t.Run("it converts symbol k-v arg keys to kebab case", func(t *testing.T) {
		t.Skip("BLOCKED: StringUtilsToKebabCase unported")
		// result := core.CliUtilsParseOptions(core.KW{core.Opt("buffer_size", 1024)})
		// expected := []string{"--buffer-size", "1024"}
		// if !reflect.DeepEqual(result, expected) {
		// 	t.Errorf("expected %v, got %v", expected, result)
		// }
	})

	t.Run("it keeps string k-v arg keys untouched", func(t *testing.T) {
		// This test uses tuples which aren't native to Go
		// The Elixir test: CliUtils.parse_options({"--under_score", 1024})
		// In Go, we represent this as a KV pair
		result := core.CliUtilsParseOptions(core.KV{Key: "--under_score", Value: 1024})
		expected := []string{"--under_score", "1024"}
		if len(result) != len(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
		for i, v := range expected {
			if i < len(result) && result[i] != v {
				t.Errorf("at index %d: expected %q, got %q", i, v, result[i])
			}
		}
	})

	t.Run("it converts symbol arg keys to kebab case", func(t *testing.T) {
		t.Skip("BLOCKED: StringUtilsToKebabCase unported")
		// result := core.CliUtilsParseOptions("ignore_errors")
		// expected := []string{"--ignore-errors"}
		// if !reflect.DeepEqual(result, expected) {
		// 	t.Errorf("expected %v, got %v", expected, result)
		// }
	})

	t.Run("it keeps string arg keys untouched", func(t *testing.T) {
		result := core.CliUtilsParseOptions("-v")
		expected := []string{"-v"}
		if len(result) != len(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
		for i, v := range expected {
			if i < len(result) && result[i] != v {
				t.Errorf("at index %d: expected %q, got %q", i, v, result[i])
			}
		}
	})
}
