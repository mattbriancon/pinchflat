package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

func TestXmlUtils_Safe(t *testing.T) {
	t.Run("escapes invalid characters", func(t *testing.T) {
		result := core.XmlUtilsSafe("hello' & <world>")
		expected := "hello&#39; &amp; &lt;world&gt;"
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})

	t.Run("converts input to string", func(t *testing.T) {
		result := core.XmlUtilsSafe(42)
		if result != "42" {
			t.Errorf("got %q, want %q", result, "42")
		}

		result = core.XmlUtilsSafe(nil)
		if result != "" {
			t.Errorf("got %q, want empty string", result)
		}
	})
}
