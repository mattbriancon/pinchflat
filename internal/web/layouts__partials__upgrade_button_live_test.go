package web_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestUpgradeButtonLive(t *testing.T) {
	proEnabled := func(t *testing.T, c *webtest.Client) bool {
		t.Helper()
		v, err := c.App.SettingsGet(c.Ctx, "pro_enabled")
		if err != nil {
			t.Fatal(err)
		}
		b, _ := v.(bool)
		return b
	}

	t.Run("enables pro when the text matches, ignoring case and whitespace", func(t *testing.T) {
		c := webtest.New(t)
		res := c.Post("/_live/upgrade", "", core.Attrs{"unlock-pro-textbox": "  Got It "})
		if res.Status != 204 {
			t.Fatalf("status = %d", res.Status)
		}
		if !proEnabled(t, c) {
			t.Error("pro_enabled should be true")
		}
	})

	t.Run("does nothing for other text", func(t *testing.T) {
		c := webtest.New(t)
		c.Post("/_live/upgrade", "", core.Attrs{"unlock-pro-textbox": "got i"})
		if proEnabled(t, c) {
			t.Error("pro_enabled should still be false")
		}
	})
}
