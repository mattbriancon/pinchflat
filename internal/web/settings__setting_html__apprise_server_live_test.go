package web_test

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/web"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestAppriseServerLive_InitialRendering(t *testing.T) {
	t.Run("renders the input", func(t *testing.T) {
		html, err := templ.ToGoHTML(context.Background(), web.SettingHTMLAppriseServerLiveFragment("", "hero-paper-airplane", "Send Test"))
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(html), `type="text"`) || !strings.Contains(string(html), `name="setting[apprise_server]"`) {
			t.Error(`expected input type="text" name="setting[apprise_server]"`)
		}
	})

	t.Run("sets the initial value from the session", func(t *testing.T) {
		html, err := templ.ToGoHTML(context.Background(), web.SettingHTMLAppriseServerLiveFragment("cool-value", "hero-paper-airplane", "Send Test"))
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(html), `value="cool-value"`) {
			t.Error(`expected value="cool-value"`)
		}
	})

	t.Run("shows a relevant button icon", func(t *testing.T) {
		html, err := templ.ToGoHTML(context.Background(), web.SettingHTMLAppriseServerLiveFragment("", "hero-paper-airplane", "Send Test"))
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(html), "hero-paper-airplane") {
			t.Error("expected 'hero-paper-airplane' in HTML")
		}
		if strings.Contains(string(html), "hero-check") {
			t.Error("should not have 'hero-check' in initial render")
		}
	})
}

func TestAppriseServerLive_PressingTheButton(t *testing.T) {
	t.Run("sends a test message to the specified server", func(t *testing.T) {
		c := webtest.New(t)
		c.AppriseMock.Run.Stub(func(endpoints []string, opts core.KW) error { return nil })

		var gotServers []string
		var gotOpts core.KW
		c.AppriseMock.Run.Expect(func(servers []string, opts core.KW) error {
			gotServers = servers
			gotOpts = opts
			return nil
		})

		c.Post("/_live/settings/apprise_test", "setting", core.Attrs{"apprise_server": "cool-value"})

		if len(gotServers) != 1 || gotServers[0] != "cool-value" {
			t.Errorf(`expected servers=["cool-value"], got %v`, gotServers)
		}
		want := core.KW{core.Opt("title", "Pinchflat Test"), core.Opt("body", "This is a test message from Pinchflat")}
		if len(gotOpts) != len(want) {
			t.Errorf("expected opts %v, got %v", want, gotOpts)
		} else {
			for i := range want {
				if gotOpts[i] != want[i] {
					t.Errorf("expected opts %v, got %v", want, gotOpts)
					break
				}
			}
		}
	})

	t.Run("sets the button icon to a checkmark", func(t *testing.T) {
		c := webtest.New(t)
		c.AppriseMock.Run.Stub(func(servers []string, opts core.KW) error { return nil })

		res := c.Post("/_live/settings/apprise_test", "setting", core.Attrs{"apprise_server": "cool-value"})

		html := res.HTML(t, 200)
		if strings.Contains(html, "hero-paper-airplane") {
			t.Error("should not have 'hero-paper-airplane' after click")
		}
		if !strings.Contains(html, "hero-check") {
			t.Error("expected 'hero-check' after button click")
		}
	})
}
