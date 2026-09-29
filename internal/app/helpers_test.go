package app_test

import (
	"net/http"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// must fails the test immediately if err is non-nil.
func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// mustOK checks the error of a call whose other result is not needed:
// mustOK(t)(f()).
func mustOK(t testing.TB) func(any, error) {
	return func(_ any, err error) {
		t.Helper()
		must(t, err)
	}
}

// ytReturns is a yt-dlp mock that always returns s.
func ytReturns(s string) func(url, action string, opts ytdlp.Args, outputTemplate string, addl ytdlp.CallOptions) (string, error) {
	return func(string, string, ytdlp.Args, string, ytdlp.CallOptions) (string, error) { return s, nil }
}

// httpReturns is an HTTP mock that always returns s.
func httpReturns(s string) func(url string, headers http.Header) (string, error) {
	return func(string, http.Header) (string, error) { return s, nil }
}
