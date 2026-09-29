package app_test

import (
	"os"
	"path/filepath"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/cmdrun"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestPreJobStartupTasks_EnsureTmpfileDirectory(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	ta.YtDlpMock.Version.Stub(func() (string, error) {
		return "1", nil
	})
	ta.UserScriptMock.Run.Stub(func(event string, data any) error {
		return nil
	})

	tmpfileDir := ta.Config.TmpfileDirectory
	os.RemoveAll(tmpfileDir)

	if _, err := os.Stat(tmpfileDir); err == nil {
		t.Error("tmpfile directory should not exist")
	}

	if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
		t.Errorf("PreJobStartupTasksInit failed: %v", err)
	}

	if _, err := os.Stat(tmpfileDir); err != nil {
		t.Error("tmpfile directory should exist")
	}
}

func TestPreJobStartupTasks_ResetExecutingJobs(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	ta.YtDlpMock.Version.Stub(func() (string, error) {
		return "1", nil
	})
	ta.UserScriptMock.Run.Stub(func(event string, data any) error {
		return nil
	})

	job := apptest.JobFixture(t, ta)

	store.Exec(ta.Ctx, ta.Q(ta.Ctx), store.SQ.Update("oban_jobs").Set("state", "executing").Where(sq.Eq{"id": job.ID}))

	var reloadedJob obanlite.Job
	ta.Q(ta.Ctx).GetContext(ta.Ctx, &reloadedJob, "SELECT * FROM oban_jobs WHERE id = ?", job.ID)
	if reloadedJob.State != "executing" {
		t.Errorf("expected job state executing, got %s", reloadedJob.State)
	}

	if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
		t.Errorf("PreJobStartupTasksInit failed: %v", err)
	}

	ta.Q(ta.Ctx).GetContext(ta.Ctx, &reloadedJob, "SELECT * FROM oban_jobs WHERE id = ?", job.ID)
	if reloadedJob.State != "retryable" {
		t.Errorf("expected job state retryable, got %s", reloadedJob.State)
	}
}

func TestPreJobStartupTasks_CreateBlankYtDlpFiles(t *testing.T) {
	t.Run("cookie file", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		filepath := filepath.Join(ta.Config.ExtrasDirectory, "cookies.txt")
		os.Remove(filepath)

		if _, err := os.Stat(filepath); err == nil {
			t.Error("file should not exist")
		}

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			t.Error("file should exist")
		}
	})

	t.Run("config file", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		filepath := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs", "base-config.txt")
		os.Remove(filepath)

		if _, err := os.Stat(filepath); err == nil {
			t.Error("file should not exist")
		}

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			t.Error("file should exist")
		}
	})
}

func TestPreJobStartupTasks_CreateBlankUserScriptFile(t *testing.T) {
	t.Run("creates file", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		filepath := filepath.Join(ta.Config.ExtrasDirectory, "user-scripts", "lifecycle")
		os.Remove(filepath)

		if _, err := os.Stat(filepath); err == nil {
			t.Error("file should not exist")
		}

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			t.Error("file should exist")
		}
	})

	t.Run("sets permissions", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		filepath := filepath.Join(ta.Config.ExtrasDirectory, "user-scripts", "lifecycle")
		os.Remove(filepath)

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		stat, err := os.Stat(filepath)
		if err != nil {
			t.Error("file should exist")
		}

		if stat.Mode().Perm() != 0o755 {
			t.Errorf("expected permissions 0755, got %o", stat.Mode().Perm())
		}
	})
}

func TestPreJobStartupTasks_ApplyDefaultSettings(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	os.RemoveAll(ta.Config.TmpfileDirectory)
	ta.SetSetting(ta.Ctx, store.KW{store.Opt("yt_dlp_version", nil)})

	val, _ := ta.GetSetting(ta.Ctx, "yt_dlp_version")
	if val != nil {
		t.Errorf("expected yt_dlp_version nil, got %v", val)
	}

	ta.YtDlpMock.Version.Stub(func() (string, error) {
		return "1", nil
	})
	ta.UserScriptMock.Run.Stub(func(event string, data any) error {
		return nil
	})

	if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
		t.Errorf("PreJobStartupTasksInit failed: %v", err)
	}

	val, _ = ta.GetSetting(ta.Ctx, "yt_dlp_version")
	if val != "1" {
		t.Errorf("expected yt_dlp_version '1', got %v", val)
	}
}

func TestPreJobStartupTasks_RunAppInitScript(t *testing.T) {
	t.Run("calls app_init", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})

		called := false
		ta.UserScriptMock.Run.Expect(func(event string, data any) error {
			called = true
			if event != "app_init" {
				t.Errorf("expected event app_init, got %s", event)
			}
			return nil
		})

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if !called {
			t.Error("UserScriptMock.Run should have been called")
		}
	})

	t.Run("boots on script error", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Expect(func(event string, data any) error {
			return &cmdrun.Error{Output: "boom", Status: 1}
		})

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}
	})
}
