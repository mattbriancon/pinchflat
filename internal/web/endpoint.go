package web

// Hand-written W4 infrastructure.

import (
	"crypto/subtle"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

//go:embed static
var staticFS embed.FS

// staticPaths is PinchflatWeb.static_paths/0.
var staticPaths = []string{"/assets/", "/fonts/", "/images/", "/favicon.ico", "/robots.txt"}

// Handler returns the full HTTP stack (the Endpoint).
func (s *Server) Handler() http.Handler {
	router := s.Router()
	static, _ := fs.Sub(staticFS, "static")
	fileServer := http.FileServer(http.FS(static))
	var metrics http.Handler
	if s.Opts.EnablePrometheus {
		metrics = MetricsHandler()
	}

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Plug.Static
		for _, p := range staticPaths {
			if r.URL.Path == strings.TrimSuffix(p, "/") || strings.HasPrefix(r.URL.Path, p) {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// PromEx.Plug
		if metrics != nil && r.URL.Path == "/metrics" {
			metrics.ServeHTTP(w, r)
			return
		}
		r = methodOverride(r)
		r = s.withPageContext(w, r)
		r = stripTrailingExtension(r)
		router.ServeHTTP(w, r)
	})
	return requestLogger(instrument(h))
}

// methodOverride is Plug.MethodOverride: a POST form with _method=patch|put|delete.
func methodOverride(r *http.Request) *http.Request {
	if r.Method != http.MethodPost {
		return r
	}
	if m := strings.ToUpper(r.FormValue("_method")); m == "PATCH" || m == "PUT" || m == "DELETE" {
		r.Method = m
	}
	return r
}

// withPageContext sets up the per-request Page: flash, base path, the
// client-facing base URL (override_base_url/2) and the onboarding flag.
func (s *Server) withPageContext(w http.ResponseWriter, r *http.Request) *http.Request {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" && !strings.Contains(xf, ",") {
		scheme = xf
	}
	host := r.Host // includes a non-default port
	if h, port, ok := strings.Cut(host, ":"); ok && (port == "80" || port == "443") {
		host = h
	}
	page := &Page{
		Request:  r,
		BasePath: trimBase(s.App.Config.BaseRoutePath),
		BaseURL:  scheme + "://" + host,
		Flash:    s.takeFlash(w, r),
		App:      s.App,
		Version:  s.Opts.Version,
	}
	if v, err := s.App.SettingsGet(r.Context(), "onboarding"); err == nil {
		page.Onboarding, _ = v.(bool)
	}
	r = withPage(r, page)
	page.Request = r
	return r
}

// stripTrailingExtension: some podcast clients require (or add) file
// extensions, so "/sources/:uuid/feed.xml" routes like "/sources/:uuid/feed".
func stripTrailingExtension(r *http.Request) *http.Request {
	p := r.URL.Path
	i := strings.LastIndexByte(p, '/')
	last := p[i+1:]
	if dot := strings.LastIndexByte(last, '.'); dot > 0 {
		r2 := r.Clone(r.Context())
		r2.URL.Path = p[:i+1] + last[:dot]
		r2.URL.RawPath = ""
		return r2
	}
	return r
}

// --- Plugs ---

// basicAuth is Plugs.basic_auth/2: enforced only when both credentials are set.
func (s *Server) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass := s.App.Config.BasicAuthUsername, s.App.Config.BasicAuthPassword
		if user == "" || pass == "" {
			next.ServeHTTP(w, r)
			return
		}
		u, p, ok := r.BasicAuth()
		if ok && subtle.ConstantTimeCompare([]byte(u), []byte(user)) == 1 && subtle.ConstantTimeCompare([]byte(p), []byte(pass)) == 1 {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="Pinchflat"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// maybeBasicAuth is Plugs.maybe_basic_auth/2: skipped when EXPOSE_FEED_ENDPOINTS is set.
func (s *Server) maybeBasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.App.Config.ExposeFeedEndpoints {
			next.ServeHTTP(w, r)
		} else {
			s.basicAuth(next).ServeHTTP(w, r)
		}
	})
}

// tokenProtectedRoute is Plugs.token_protected_route/2.
func (s *Server) tokenProtectedRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := r.URL.Query()["route_token"]
		if ok {
			if want, err := s.App.SettingsGetBang(r.Context(), "route_token"); err == nil && want == token[0] {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	})
}

// secureBrowserHeaders is put_secure_browser_headers minus x-frame-options
// (allow_iframe_embed/2 deletes it).
func secureBrowserHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("referrer-policy", "strict-origin-when-cross-origin")
		h.Set("x-content-type-options", "nosniff")
		h.Set("x-download-options", "noopen")
		h.Set("x-permitted-cross-domain-policies", "none")
		next.ServeHTTP(w, r)
	})
}

// protectFromForgery replaces Phoenix's CSRF token with Go's
// Sec-Fetch-Site/Origin based cross-origin protection.
func protectFromForgery(next http.Handler) http.Handler {
	return http.NewCrossOriginProtection().Handler(next)
}

// --- logging ---

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) { s.status = code; s.ResponseWriter.WriteHeader(code) }

// requestLogger is Plug.Telemetry logging; the healthcheck isn't logged.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		if r.URL.Path != "/healthcheck" {
			slog.Info(r.Method+" "+r.URL.Path, "status", rec.status, "duration", time.Since(start))
		}
	})
}
