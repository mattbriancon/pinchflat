package fsutil_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

func TestToKebabCase(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"converts a space-delimited string", "hello world", "hello-world"},
		{"converts an underscore-delimited string", "hello_world", "hello-world"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fsutil.ToKebabCase(tc.input); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRandomString(t *testing.T) {
	t.Run("generates a random string", func(t *testing.T) {
		str1 := fsutil.RandomString(32)
		str2 := fsutil.RandomString(32)

		if !isValidHex(str1) {
			t.Errorf("RandomString(32) returned non-hex string: %q", str1)
		}
		if str1 == str2 {
			t.Error("RandomString should return different values on successive calls")
		}
	})

	t.Run("can generate a string of a given length", func(t *testing.T) {
		for _, length := range []int{32, 64} {
			if got := len(fsutil.RandomString(length)); got != length {
				t.Errorf("RandomString(%d): got length %d", length, got)
			}
		}
	})
}

func TestDoubleBrace(t *testing.T) {
	if got := fsutil.DoubleBrace("hello"); got != "{{ hello }}" {
		t.Errorf("got %q, want %q", got, "{{ hello }}")
	}
}

func isValidHex(s string) bool {
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
