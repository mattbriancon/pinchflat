package web_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

// Port of test/pinchflat_web/controllers/page_controller_test.exs.

func TestPageController_Home(t *testing.T) {
	t.Run("GET / when testing onboarding", func(t *testing.T) {
		t.Run("sets the onboarding setting to true when onboarding", func(t *testing.T) {
			c := webtest.New(t)
			c.Get("/")
			onboarding, _ := c.App.GetSettingBang(c.Ctx, "onboarding")
			if !onboarding.(bool) {
				t.Errorf("expected onboarding to be true")
			}
		})

		t.Run("displays the onboarding page when onboarding is forced", func(t *testing.T) {
			c := webtest.New(t)
			c.App.SetSetting(c.Ctx, "onboarding", false)

			res := c.Get("/?onboarding=1")
			html := res.HTML(t, http.StatusOK)
			if !strings.Contains(html, "Welcome to Pinchflat") {
				t.Errorf("expected onboarding page to contain 'Welcome to Pinchflat'")
			}
		})

		t.Run("sets the onboarding setting to false if you pass the corrent query param", func(t *testing.T) {
			c := webtest.New(t)
			c.Get("/")
			onboarding1, _ := c.App.GetSettingBang(c.Ctx, "onboarding")
			if !onboarding1.(bool) {
				t.Errorf("expected onboarding to be true after first GET")
			}

			c.Get("/?onboarding=0")
			onboarding2, _ := c.App.GetSettingBang(c.Ctx, "onboarding")
			if onboarding2.(bool) {
				t.Errorf("expected onboarding to be false after second GET with onboarding=0")
			}
		})

		t.Run("displays the home page when not onboarding", func(t *testing.T) {
			c := webtest.New(t)
			c.App.SetSetting(c.Ctx, "onboarding", false)

			res := c.Get("/")
			html := res.HTML(t, http.StatusOK)
			if !strings.Contains(html, "Main navigation") {
				t.Errorf("expected home page to contain 'Main navigation'")
			}
		})
	})
}
