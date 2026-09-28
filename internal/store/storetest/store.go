// Package storetest is the Go port of test/support for internal/store: a
// migrated per-test database plus fixtures for the entity types.
package storetest

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// TestStore is a Store backed by a fresh migrated database, with Oban in
// manual mode (jobs inserted, never run).
type TestStore struct {
	*store.Store
	Ctx context.Context
}

// Tests log at :critical in Elixir (config/test.exs); keep test output quiet.
// Set PINCHFLAT_TEST_LOGS=1 to see logs.
func init() {
	if os.Getenv("PINCHFLAT_TEST_LOGS") == "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
}

var (
	templateOnce sync.Once
	templatePath string
	templateErr  error
)

// migratedTemplate migrates one database per test binary; each test gets a
// copy (replacing the Ecto SQL sandbox).
func migratedTemplate() (string, error) {
	templateOnce.Do(func() {
		dir, err := os.MkdirTemp("", "pinchflat-store-template-")
		if err != nil {
			templateErr = err
			return
		}
		templatePath = filepath.Join(dir, "template.db")
		d, err := db.Open(templatePath, db.Options{JournalMode: "delete"})
		if err != nil {
			templateErr = err
			return
		}
		_, templateErr = d.Migrate(context.Background())
		d.Close()
	})
	return templatePath, templateErr
}

// NewStore returns a Store backed by a fresh migrated database.
func NewStore(t testing.TB) *TestStore {
	t.Helper()
	tmpl, err := migratedTemplate()
	if err != nil {
		t.Fatalf("migrate template: %v", err)
	}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "pinchflat.db")
	if err := copyFile(tmpl, dbPath); err != nil {
		t.Fatal(err)
	}
	d, err := db.Open(dbPath, db.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	return &TestStore{
		Store: &store.Store{DB: d, Oban: obanlite.New(d)},
		Ctx:   context.Background(),
	}
}

// RepoPath returns a path relative to the repository root.
func RepoPath(rel string) string { return filepath.Join(dbtest.RepoRoot(), rel) }

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
