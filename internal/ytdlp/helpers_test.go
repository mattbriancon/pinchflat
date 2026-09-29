package ytdlp_test

import "testing"

// must fails the test immediately if err is non-nil.
func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
