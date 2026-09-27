// Package webtest is the Go port of test/support/conn_case.ex: a test
// client for the web stack, mirroring Phoenix.ConnTest.
package webtest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/web"
)

// Client is ConnCase's conn: an app with mocks plus a cookie-keeping client.
type Client struct {
	*coretest.TestApp
	Server  *web.Server
	handler http.Handler
	cookies map[string]*http.Cookie
	t       testing.TB
}

// New builds a fresh app (coretest.NewApp) and web server.
func New(t testing.TB) *Client {
	ta := coretest.NewApp(t)
	srv := web.New(ta.App, web.Options{SecretKeyBase: "test-secret", Version: "test"})
	return &Client{TestApp: ta, Server: srv, handler: srv.Handler(), cookies: map[string]*http.Cookie{}, t: t}
}

// Response is a completed request.
type Response struct {
	Status int
	Header http.Header
	Body   string
	c      *Client
}

// Do sends a request (cookies from earlier responses are sent back).
func (c *Client) Do(req *http.Request) *Response {
	c.t.Helper()
	for _, ck := range c.cookies {
		req.AddCookie(ck)
	}
	// Browsers send this on same-origin form posts; CrossOriginProtection
	// allows requests without Origin/Sec-Fetch-Site too.
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	res := rec.Result()
	for _, ck := range res.Cookies() {
		if ck.MaxAge < 0 || ck.Value == "" {
			delete(c.cookies, ck.Name)
		} else {
			c.cookies[ck.Name] = ck
		}
	}
	body, _ := io.ReadAll(res.Body)
	return &Response{Status: res.StatusCode, Header: res.Header, Body: string(body), c: c}
}

// Get is get(conn, path, params).
func (c *Client) Get(path string, params ...map[string]string) *Response {
	c.t.Helper()
	if len(params) > 0 && len(params[0]) > 0 {
		q := url.Values{}
		for k, v := range params[0] {
			q.Set(k, v)
		}
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		path += sep + q.Encode()
	}
	return c.Do(httptest.NewRequest(http.MethodGet, path, nil))
}

// Post is post(conn, path, as: attrs), e.g. Post("/sources", "source", attrs).
// Pass as == "" for flat params.
func (c *Client) Post(path, as string, attrs core.Attrs) *Response {
	return c.send(http.MethodPost, path, as, attrs)
}

// Patch is patch(conn, path, as: attrs), sent like a browser form (_method).
func (c *Client) Patch(path, as string, attrs core.Attrs) *Response {
	return c.send(http.MethodPatch, path, as, attrs)
}

// Delete is delete(conn, path).
func (c *Client) Delete(path string) *Response { return c.send(http.MethodDelete, path, "", nil) }

func (c *Client) send(method, path, as string, attrs core.Attrs) *Response {
	c.t.Helper()
	form := url.Values{}
	if method != http.MethodPost {
		form.Set("_method", strings.ToLower(method))
	}
	encode(form, as, attrs)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.Do(req)
}

// encode flattens attrs to Phoenix-style param names: source[name],
// source[metadata][x], source[list][].
func encode(form url.Values, prefix string, v any) {
	switch x := v.(type) {
	case nil:
		if prefix != "" {
			form.Set(prefix, "")
		}
	case core.Attrs:
		encodeMap(form, prefix, map[string]any(x))
	case map[string]any:
		encodeMap(form, prefix, x)
	case []string:
		for _, s := range x {
			form.Add(prefix+"[]", s)
		}
	case []any:
		for _, s := range x {
			form.Add(prefix+"[]", fmt.Sprint(s))
		}
	default:
		form.Set(prefix, fmt.Sprint(x))
	}
}

func encodeMap(form url.Values, prefix string, m map[string]any) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		name := k
		if prefix != "" {
			name = prefix + "[" + k + "]"
		}
		encode(form, name, m[k])
	}
}

// HTML is html_response(conn, status): fails unless the status matches and
// the response is HTML, and returns the body.
func (r *Response) HTML(t testing.TB, status int) string {
	t.Helper()
	if r.Status != status {
		t.Fatalf("expected status %d, got %d\n%s", status, r.Status, truncate(r.Body))
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected an HTML response, got %q", ct)
	}
	return r.Body
}

// JSON is json_response(conn, status).
func (r *Response) JSON(t testing.TB, status int) map[string]any {
	t.Helper()
	if r.Status != status {
		t.Fatalf("expected status %d, got %d\n%s", status, r.Status, truncate(r.Body))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(r.Body), &m); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, truncate(r.Body))
	}
	return m
}

// RedirectedTo is redirected_to(conn): the Location of a 302 (fails otherwise).
func (r *Response) RedirectedTo(t testing.TB) string {
	t.Helper()
	if r.Status != http.StatusFound {
		t.Fatalf("expected a redirect (302), got %d\n%s", r.Status, truncate(r.Body))
	}
	return r.Header.Get("Location")
}

// Flash is Phoenix.Flash.get(conn.assigns.flash, kind) for the flash set by
// this response (read back from the flash cookie).
func (r *Response) Flash(kind string) string {
	for _, ck := range r.c.cookies {
		if ck.Name == "_pinchflat_flash" {
			return r.c.Server.DecodeFlash(ck.Value)[kind]
		}
	}
	return ""
}

func truncate(s string) string {
	if len(s) > 2000 {
		return s[:2000] + "…"
	}
	return s
}
