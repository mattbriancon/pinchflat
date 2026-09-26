package core_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestCliUtils_WrapCmd(t *testing.T) {
	t.Run("delegates to System.cmd/3", func(t *testing.T) {
		ta := coretest.NewApp(t)
		output, status, err := ta.App.CliUtilsWrapCmd(ta.Ctx, "echo", []string{"output"}, nil, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if status != 0 {
			t.Errorf("expected status 0, got %d", status)
		}
		if !strings.Contains(output, "output") {
			t.Errorf("expected output to contain 'output', got: %q", output)
		}
	})

	t.Run("sets the current directory to the tmp dir", func(t *testing.T) {
		ta := coretest.NewApp(t)
		output, status, err := ta.App.CliUtilsWrapCmd(ta.Ctx, "pwd", []string{}, nil, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if status != 0 {
			t.Errorf("expected status 0, got %d", status)
		}
		// The output should end with the tmpfiles directory
		if !strings.Contains(output, "tmpfiles") {
			t.Errorf("expected output to contain 'tmpfiles', got: %q", output)
		}
	})
}

func TestCliUtils_ParseOptions(t *testing.T) {
	t.Run("it converts symbol k-v arg keys to kebab case", func(t *testing.T) {
		result := core.CliUtilsParseOptions(core.KW{core.Opt("buffer_size", 1024)})
		expected := []string{"--buffer-size", "1024"}
		if len(result) != len(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
		for i, v := range expected {
			if i < len(result) && result[i] != v {
				t.Errorf("at index %d: expected %q, got %q", i, v, result[i])
			}
		}
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
		result := core.CliUtilsParseOptions(core.Flag("ignore_errors"))
		expected := []string{"--ignore-errors"}
		if len(result) != len(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
		for i, v := range expected {
			if i < len(result) && result[i] != v {
				t.Errorf("at index %d: expected %q, got %q", i, v, result[i])
			}
		}
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
