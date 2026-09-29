package app_test

import "testing"

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
