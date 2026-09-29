package web_test

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestMaybeBasicAuth(t *testing.T) {
	t.Run("uses basic auth when expose_feed_endpoints is false", func(t *testing.T) {
		c := webtest.New(t)
		c.Server.App.Config.ExposeFeedEndpoints = false
		c.Server.App.Config.BasicAuthUsername = "user"
		c.Server.App.Config.BasicAuthPassword = "pass"

		res := c.Get("/sources/opml", map[string]string{"route_token": ""})

		if res.Status != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, res.Status)
		}
		if auth := res.Header.Get("WWW-Authenticate"); !strings.Contains(auth, "Basic realm=\"Pinchflat\"") {
			t.Errorf("expected WWW-Authenticate header with Basic realm, got %q", auth)
		}
	})

	t.Run("supplying the correct username and password allows access", func(t *testing.T) {
		c := webtest.New(t)
		c.Server.App.Config.ExposeFeedEndpoints = false
		c.Server.App.Config.BasicAuthUsername = "user"
		c.Server.App.Config.BasicAuthPassword = "pass"

		// Build the Authorization header
		auth := base64.StdEncoding.EncodeToString([]byte("user:pass"))
		req := c.Do(newRequestWithAuth("GET", "/sources/opml?route_token=test", auth))

		if req.Status == http.StatusUnauthorized {
			t.Errorf("expected to be authorized, got status %d", req.Status)
		}
	})

	t.Run("does not use basic auth when expose_feed_endpoints is true", func(t *testing.T) {
		c := webtest.New(t)
		c.Server.App.Config.ExposeFeedEndpoints = true
		c.Server.App.Config.BasicAuthUsername = "user"
		c.Server.App.Config.BasicAuthPassword = "pass"

		res := c.Get("/sources/opml", map[string]string{"route_token": ""})

		// The request should not be blocked by basic auth; it will fail with route_token 401
		if res.Status == http.StatusUnauthorized && res.Header.Get("WWW-Authenticate") != "" {
			t.Errorf("should not require basic auth when expose_feed_endpoints is true")
		}
	})

	t.Run("does not use basic auth when username/password aren't set", func(t *testing.T) {
		c := webtest.New(t)
		c.Server.App.Config.ExposeFeedEndpoints = false
		c.Server.App.Config.BasicAuthUsername = ""
		c.Server.App.Config.BasicAuthPassword = ""

		res := c.Get("/sources/opml", map[string]string{"route_token": ""})

		// Should not get a 401 from basic auth (will fail from route_token)
		if res.Status == http.StatusUnauthorized && res.Header.Get("WWW-Authenticate") != "" {
			t.Errorf("should not require basic auth when credentials aren't set")
		}
	})
}

func TestBasicAuth(t *testing.T) {
	t.Run("uses basic auth when both username and password are set", func(t *testing.T) {
		c := webtest.New(t)
		c.Server.App.Config.BasicAuthUsername = "user"
		c.Server.App.Config.BasicAuthPassword = "pass"

		res := c.Get("/sources")

		if res.Status != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, res.Status)
		}
		if auth := res.Header.Get("WWW-Authenticate"); !strings.Contains(auth, "Basic realm=\"Pinchflat\"") {
			t.Errorf("expected WWW-Authenticate header, got %q", auth)
		}
	})

	t.Run("providing the username and password allows access", func(t *testing.T) {
		c := webtest.New(t)
		c.Server.App.Config.BasicAuthUsername = "user"
		c.Server.App.Config.BasicAuthPassword = "pass"

		auth := base64.StdEncoding.EncodeToString([]byte("user:pass"))
		req := c.Do(newRequestWithAuth("GET", "/sources", auth))

		if req.Status == http.StatusUnauthorized {
			t.Errorf("expected authorized access, got status %d", req.Status)
		}
	})

	t.Run("does not use basic auth when either username or password is not set", func(t *testing.T) {
		c := webtest.New(t)
		c.Server.App.Config.BasicAuthUsername = ""
		c.Server.App.Config.BasicAuthPassword = "pass"

		res := c.Get("/sources")

		// Should succeed because basic auth is not required
		if res.Status == http.StatusUnauthorized && res.Header.Get("WWW-Authenticate") != "" {
			t.Errorf("should not require basic auth when username is not set")
		}
	})

	t.Run("treats empty strings as not being set when using basic auth", func(t *testing.T) {
		c := webtest.New(t)
		c.Server.App.Config.BasicAuthUsername = ""
		c.Server.App.Config.BasicAuthPassword = "pass"

		res := c.Get("/sources")

		// Should not require basic auth because username is empty
		if res.Status == http.StatusUnauthorized && res.Header.Get("WWW-Authenticate") != "" {
			t.Errorf("should treat empty username as not set")
		}
	})
}

func TestTokenProtectedRoute(t *testing.T) {
	t.Run("allows access when the route token is correct", func(t *testing.T) {
		c := webtest.New(t)
		token, _ := c.App.GetSettingBang(c.Ctx, "route_token")
		tokenStr := token.(string)

		res := c.Get("/sources/opml", map[string]string{"route_token": tokenStr})

		// Should not get a 401
		if res.Status == http.StatusUnauthorized {
			t.Errorf("expected access allowed, got status %d", res.Status)
		}
	})

	t.Run("does not allow access when the route token is incorrect", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/sources/opml", map[string]string{"route_token": "incorrect"})

		if res.Status != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, res.Status)
		}
		if res.Body != "Unauthorized" {
			t.Errorf("expected body 'Unauthorized', got %q", res.Body)
		}
	})

	t.Run("does not allow access when the route token is missing", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/sources/opml")

		if res.Status != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, res.Status)
		}
		if res.Body != "Unauthorized" {
			t.Errorf("expected body 'Unauthorized', got %q", res.Body)
		}
	})
}

// Helper to create a request with Authorization header
func newRequestWithAuth(method, path, auth string) *http.Request {
	req := &http.Request{
		Method: method,
		URL:    &url.URL{Path: path, Scheme: "http", Host: "localhost"},
		Header: make(http.Header),
	}
	req.Header.Set("Authorization", "Basic "+auth)
	return req
}

// allow_iframe_embed/2 deleted the x-frame-options header that
// put_secure_browser_headers added; the Go secureBrowserHeaders never sets it,
// so check a browser-pipeline response through the full stack.
func TestPlugsAllowIframeEmbed(t *testing.T) {
	t.Run("deletes the x-frame-options header", func(t *testing.T) {
		c := webtest.New(t)
		res := c.Get("/")
		if res.Header.Get("X-Content-Type-Options") == "" {
			t.Fatal("expected the browser pipeline's secure headers to be set")
		}
		if v := res.Header.Values("X-Frame-Options"); len(v) != 0 {
			t.Errorf("x-frame-options = %v, want none", v)
		}
	})
}
