package web_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestAppriseServerLive_InitialRendering(t *testing.T) {
	t.Run("renders the input", func(t *testing.T) {
		c := webtest.New(t)
		c.AppriseMock.Run.Stub(func(endpoints []string, opts core.KW) error {
			return nil
		})

		res := c.Post("/_live/settings/apprise_test", "setting", core.Attrs{"apprise_server": ""})

		html := res.HTML(t, 200)
		if !strings.Contains(html, `input`) || !strings.Contains(html, `name="setting[apprise_server]"`) {
			t.Error("expected input with apprise_server field in HTML")
		}
	})

	t.Run("sets the initial value from the session", func(t *testing.T) {
		c := webtest.New(t)
		c.AppriseMock.Run.Stub(func(endpoints []string, opts core.KW) error {
			return nil
		})

		res := c.Post("/_live/settings/apprise_test", "setting", core.Attrs{"apprise_server": "cool-value"})

		html := res.HTML(t, 200)
		if !strings.Contains(html, `value="cool-value"`) {
			t.Error("expected value='cool-value' in HTML")
		}
	})

	t.Run("shows a relevant button icon", func(t *testing.T) {
		c := webtest.New(t)
		c.AppriseMock.Run.Stub(func(endpoints []string, opts core.KW) error {
			return nil
		})

		res := c.Post("/_live/settings/apprise_test", "setting", core.Attrs{"apprise_server": ""})

		html := res.HTML(t, 200)
		if !strings.Contains(html, "hero-paper-airplane") {
			t.Error("expected 'hero-paper-airplane' in HTML")
		}
		if strings.Contains(html, "hero-check") {
			t.Error("should not have 'hero-check' in initial render")
		}
	})
}

func TestAppriseServerLive_PressingTheButton(t *testing.T) {
	t.Run("sends a test message to the specified server", func(t *testing.T) {
		c := webtest.New(t)

		// Set up mock to track the call
		callCount := 0
		c.AppriseMock.Run.Expect(func(servers []string, opts core.KW) error {
			callCount++
			if len(servers) != 1 || servers[0] != "cool-value" {
				t.Errorf("expected servers=['cool-value'], got %v", servers)
			}
			return nil
		})

		res := c.Post("/_live/settings/apprise_test", "setting", core.Attrs{"apprise_server": "cool-value"})

		if callCount == 0 {
			t.Error("expected apprise runner to be called")
		}

		// Status should be 200 and result should contain check icon
		html := res.HTML(t, 200)
		if !strings.Contains(html, "hero-check") {
			t.Error("expected 'hero-check' after button click")
		}
	})

	t.Run("sets the button icon to a checkmark", func(t *testing.T) {
		c := webtest.New(t)

		c.AppriseMock.Run.Stub(func(servers []string, opts core.KW) error {
			return nil
		})

		res := c.Post("/_live/settings/apprise_test", "setting", core.Attrs{"apprise_server": "cool-value"})

		html := res.HTML(t, 200)
		if !strings.Contains(html, "hero-check") {
			t.Error("expected 'hero-check' icon in result")
		}
		if strings.Contains(html, "hero-paper-airplane") {
			t.Error("should not have 'hero-paper-airplane' after click")
		}
	})
}
