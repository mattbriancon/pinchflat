package core_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

func TestStringUtils_ToKebabCase(t *testing.T) {
	t.Run("converts a space-delimited string to kebab-case", func(t *testing.T) {
		result := core.StringUtilsToKebabCase("hello world")
		if result != "hello-world" {
			t.Errorf("got %q, want %q", result, "hello-world")
		}
	})

	t.Run("converts an underscore-delimited string to kebab-case", func(t *testing.T) {
		result := core.StringUtilsToKebabCase("hello_world")
		if result != "hello-world" {
			t.Errorf("got %q, want %q", result, "hello-world")
		}
	})
}

func TestStringUtils_RandomString(t *testing.T) {
	t.Run("generates a random string", func(t *testing.T) {
		str1 := core.StringUtilsRandomString(32)
		str2 := core.StringUtilsRandomString(32)

		if !isValidHex(str1) {
			t.Errorf("RandomString(32) returned non-hex string: %q", str1)
		}
		if str1 == str2 {
			t.Error("RandomString should return different values on successive calls")
		}
	})

	t.Run("has a defined default length", func(t *testing.T) {
		result := core.StringUtilsRandomString(32)
		if len(result) != 32 {
			t.Errorf("got length %d, want 32", len(result))
		}
	})

	t.Run("can generate a string of a given length", func(t *testing.T) {
		result := core.StringUtilsRandomString(64)
		if len(result) != 64 {
			t.Errorf("got length %d, want 64", len(result))
		}
	})
}

func TestStringUtils_DoubleBrace(t *testing.T) {
	t.Run("wraps a string in double braces", func(t *testing.T) {
		result := core.StringUtilsDoubleBrace("hello")
		if result != "{{ hello }}" {
			t.Errorf("got %q, want %q", result, "{{ hello }}")
		}
	})
}

// Helper function to check if a string is valid hexadecimal
func isValidHex(s string) bool {
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
