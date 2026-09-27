// Package coretest is the Go port of test/support: DataCase setup, Mox
// mocks, fixtures and testing helper methods.
package coretest

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// TestApp is what a ported test gets from DataCase: the App plus its mocks.
type TestApp struct {
	*core.App
	YtDlpMock      *YtDlpMock
	AppriseMock    *AppriseMock
	UserScriptMock *UserScriptMock
	HTTPMock       *HTTPMock
	Ctx            context.Context
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
		dir, err := os.MkdirTemp("", "pinchflat-template-")
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

// NewApp returns an App backed by a fresh migrated database, with mocks
// installed for every runner (as config/test.exs + test_helper.exs do), Oban
// in manual mode (jobs inserted, never run), and per-test directories in
// place of /tmp/test/{media,metadata,extras,tmpfiles}.
func NewApp(t testing.TB) *TestApp {
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

	root := dbtest.RepoRoot()
	testDataDir := dir // per-test, replacing the shared /tmp/test/* of config/test.exs
	cfg := core.Config{
		Env:                     "test",
		YtDlpExecutable:         filepath.Join(root, "testdata/support/scripts/yt-dlp-mocks/repeater.sh"),
		AppriseExecutable:       filepath.Join(root, "testdata/support/scripts/yt-dlp-mocks/repeater.sh"),
		MediaDirectory:          filepath.Join(testDataDir, "media"),
		MetadataDirectory:       filepath.Join(testDataDir, "metadata"),
		ExtrasDirectory:         filepath.Join(dir, "extras"),
		TmpfileDirectory:        filepath.Join(dir, "tmpfiles"),
		LogPath:                 filepath.Join(dir, "logs", "pinchflat.log"),
		DatabasePath:            dbPath,
		FileWatcherPollInterval: 50 * time.Millisecond,
		Timezone:                "UTC",
		BaseRoutePath:           "/",
	}
	for _, p := range []string{cfg.MediaDirectory, cfg.MetadataDirectory, cfg.ExtrasDirectory, cfg.TmpfileDirectory, filepath.Dir(cfg.LogPath)} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ta := &TestApp{
		YtDlpMock:      NewYtDlpMock(t),
		AppriseMock:    NewAppriseMock(t),
		UserScriptMock: NewUserScriptMock(t),
		HTTPMock:       NewHTTPMock(t),
		Ctx:            context.Background(),
	}
	ta.App = &core.App{
		DB:          d,
		Oban:        obanlite.New(d),
		Config:      cfg,
		YtDlp:       ytDlpRunner{ta.YtDlpMock},
		Apprise:     appriseRunner{ta.AppriseMock},
		UserScripts: userScriptRunner{ta.UserScriptMock},
		HTTP:        httpClient{ta.HTTPMock},
	}
	ta.App.RegisterWorkers()
	return ta
}

// RepoPath returns a path relative to the repository root (File.cwd! in the
// Elixir tests).
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
