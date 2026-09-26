// Package dbtest provides throwaway databases for tests.
package dbtest

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/db"
)

// RepoRoot returns the repository root, found relative to this file.
func RepoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

// ElixirFixture returns the path of a ground-truth file under testdata/elixir.
func ElixirFixture(name string) string {
	return filepath.Join(RepoRoot(), "testdata", "elixir", name)
}

// New returns a freshly migrated, empty database in a temp dir.
func New(t testing.TB) *db.DB {
	t.Helper()
	d := open(t, filepath.Join(t.TempDir(), "pinchflat.db"))
	if _, err := d.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return d
}

// CopyOf opens a private copy of a database file (e.g. the Elixir
// populated.db) so the test can modify it freely.
func CopyOf(t testing.TB, src string) *db.DB {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
	out.Close()
	return open(t, dst)
}

func open(t testing.TB, path string) *db.DB {
	t.Helper()
	d, err := db.Open(path, db.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}
