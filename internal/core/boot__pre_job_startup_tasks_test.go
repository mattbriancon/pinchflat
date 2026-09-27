package core_test

import (
	"os"
	"path/filepath"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestPreJobStartupTasks_EnsureTmpfileDirectory(t *testing.T) {
	t.Run("creates the tmpfile directory if it doesn't exist", func(t *testing.T) {
		ta := coretest.NewApp(t)

		// Setup stubs
		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		tmpfileDir := ta.Config.TmpfileDirectory

		os.RemoveAll(tmpfileDir)

		if _, err := os.Stat(tmpfileDir); err == nil {
			t.Errorf("Expected tmpfile directory to not exist, but it does")
		}

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if _, err := os.Stat(tmpfileDir); err != nil {
			t.Errorf("Expected tmpfile directory to exist, but it doesn't: %v", err)
		}
	})
}

func TestPreJobStartupTasks_ResetExecutingJobs(t *testing.T) {
	t.Run("resets executing jobs", func(t *testing.T) {
		ta := coretest.NewApp(t)

		// Setup stubs
		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		job := coretest.JobFixture(t, ta)

		// Update job to executing state
		core.Exec(ta.Ctx, ta.Q(ta.Ctx), core.SQ.Update("oban_jobs").Set("state", "executing").Where(sq.Eq{"id": job.ID}))

		var reloadedJob obanlite.Job
		ta.Q(ta.Ctx).GetContext(ta.Ctx, &reloadedJob, "SELECT * FROM oban_jobs WHERE id = ?", job.ID)
		if reloadedJob.State != "executing" {
			t.Errorf("Expected job state to be 'executing', got %s", reloadedJob.State)
		}

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		ta.Q(ta.Ctx).GetContext(ta.Ctx, &reloadedJob, "SELECT * FROM oban_jobs WHERE id = ?", job.ID)
		if reloadedJob.State != "retryable" {
			t.Errorf("Expected job state to be 'retryable', got %s", reloadedJob.State)
		}
	})
}

func TestPreJobStartupTasks_CreateBlankYtDlpFiles(t *testing.T) {
	t.Run("creates a blank cookie file", func(t *testing.T) {
		ta := coretest.NewApp(t)

		// Setup stubs
		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		baseDir := ta.Config.ExtrasDirectory
		filepath := filepath.Join(baseDir, "cookies.txt")

		os.Remove(filepath)

		if _, err := os.Stat(filepath); err == nil {
			t.Errorf("Expected file to not exist, but it does")
		}

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			t.Errorf("Expected file to exist, but it doesn't: %v", err)
		}
	})

	t.Run("creates a blank yt-dlp config file", func(t *testing.T) {
		ta := coretest.NewApp(t)

		// Setup stubs
		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		baseDir := ta.Config.ExtrasDirectory
		filepath := filepath.Join(baseDir, "yt-dlp-configs", "base-config.txt")

		os.Remove(filepath)

		if _, err := os.Stat(filepath); err == nil {
			t.Errorf("Expected file to not exist, but it does")
		}

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			t.Errorf("Expected file to exist, but it doesn't: %v", err)
		}
	})
}

func TestPreJobStartupTasks_CreateBlankUserScriptFile(t *testing.T) {
	t.Run("creates a blank script file", func(t *testing.T) {
		ta := coretest.NewApp(t)

		// Setup stubs
		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		baseDir := ta.Config.ExtrasDirectory
		filepath := filepath.Join(baseDir, "user-scripts", "lifecycle")

		os.Remove(filepath)

		if _, err := os.Stat(filepath); err == nil {
			t.Errorf("Expected file to not exist, but it does")
		}

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			t.Errorf("Expected file to exist, but it doesn't: %v", err)
		}
	})

	t.Run("gives it 755 permissions", func(t *testing.T) {
		ta := coretest.NewApp(t)

		// Setup stubs
		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		baseDir := ta.Config.ExtrasDirectory
		filepath := filepath.Join(baseDir, "user-scripts", "lifecycle")

		os.Remove(filepath)

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		stat, err := os.Stat(filepath)
		if err != nil {
			t.Errorf("Expected file to exist, but it doesn't: %v", err)
		}

		if stat.Mode().Perm() != 0o755 {
			t.Errorf("Expected permissions to be 0755, got %o", stat.Mode().Perm())
		}
	})
}

func TestPreJobStartupTasks_ApplyDefaultSettings(t *testing.T) {
	t.Run("sets yt_dlp version", func(t *testing.T) {
		ta := coretest.NewApp(t)

		os.RemoveAll(ta.Config.TmpfileDirectory)
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("yt_dlp_version", nil)})

		val, _ := ta.SettingsGet(ta.Ctx, "yt_dlp_version")
		if val != nil {
			t.Errorf("Expected yt_dlp_version to be nil, got %v", val)
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

		val, _ = ta.SettingsGet(ta.Ctx, "yt_dlp_version")
		if val != "1" {
			t.Errorf("Expected yt_dlp_version to be '1', got %v", val)
		}
	})

}

func TestPreJobStartupTasks_RunAppInitScript(t *testing.T) {
	t.Run("calls the app_init user script runner", func(t *testing.T) {
		ta := coretest.NewApp(t)

		ta.YtDlpMock.Version.Stub(func() (string, error) {
			return "1", nil
		})
		// Verify the app_init event is called with empty data
		called := false
		ta.UserScriptMock.Run.Expect(func(event string, data any) error {
			called = true
			if event != "app_init" {
				t.Errorf("Expected event to be 'app_init', got %s", event)
			}
			return nil
		})

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}

		if !called {
			t.Errorf("Expected UserScriptMock.Run to be called, but it wasn't")
		}
	})

	t.Run("boots even if the app_init script exits non-zero", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.YtDlpMock.Version.Stub(func() (string, error) { return "1", nil })
		ta.UserScriptMock.Run.Expect(func(event string, data any) error {
			return &fsutil.CommandError{Output: "boom", Status: 1}
		})

		if err := ta.PreJobStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PreJobStartupTasksInit failed: %v", err)
		}
	})
}
