package fsutil_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

func TestXMLSafe(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  string
	}{
		{"escapes invalid characters", "hello' & <world>", "hello&#39; &amp; &lt;world&gt;"},
		{"converts a number to a string", 42, "42"},
		{"converts nil to an empty string", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fsutil.XMLSafe(tc.input); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
