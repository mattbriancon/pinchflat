package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// PreJobStartupTasks runs before the job runner starts. The GenServer
// wrapper is gone: main calls PreJobStartupTasksInit directly (and skips
// it in the test env, like the Elixir init(%{env: :test}) clause).

// init/1
func (a *App) PreJobStartupTasksInit(ctx context.Context) error {
	if err := ensureTmpfileDirectory(ctx, a); err != nil {
		return err
	}

	if err := resetExecutingJobs(ctx, a); err != nil {
		return err
	}

	if err := createBlankYtDlpFiles(ctx, a); err != nil {
		return err
	}

	if err := createBlankUserScriptFile(ctx, a); err != nil {
		return err
	}

	if err := applyDefaultSettings(ctx, a); err != nil {
		return err
	}

	if err := runAppInitScript(ctx, a); err != nil {
		return err
	}

	return nil
}

// ensureTmpfileDirectory/0
func ensureTmpfileDirectory(ctx context.Context, a *App) error {
	tmpfileDir := a.Config.TmpfileDirectory

	if _, err := os.Stat(tmpfileDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(tmpfileDir, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

// resetExecutingJobs/0
func resetExecutingJobs(ctx context.Context, a *App) error {
	query := store.SQ.Update("oban_jobs").
		Set("state", "retryable").
		Where(sq.Eq{"state": "executing"})

	count, err := store.Exec(ctx, a.Q(ctx), query)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Reset %d executing jobs", count))

	return nil
}

// createBlankYtDlpFiles/0
func createBlankYtDlpFiles(ctx context.Context, a *App) error {
	files := []string{"cookies.txt", "yt-dlp-configs/base-config.txt"}
	baseDir := a.Config.ExtrasDirectory

	for _, file := range files {
		filepath := filepath.Join(baseDir, file)

		if _, err := os.Stat(filepath); err != nil {
			if os.IsNotExist(err) {
				slog.Info("Creating blank file: " + filepath)

				if err := fsutil.WriteFileAll(filepath, ""); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}

// createBlankUserScriptFile/0
func createBlankUserScriptFile(ctx context.Context, a *App) error {
	baseDir := a.Config.ExtrasDirectory
	filepath := filepath.Join(baseDir, "user-scripts", "lifecycle")

	if _, err := os.Stat(filepath); err != nil {
		if os.IsNotExist(err) {
			slog.Info("Creating blank file and making it executable: " + filepath)

			if err := fsutil.WriteFileAll(filepath, ""); err != nil {
				return err
			}

			if err := os.Chmod(filepath, 0o755); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

// applyDefaultSettings/0
func applyDefaultSettings(ctx context.Context, a *App) error {
	ytDlpVersion, err := a.YtDlp.Version(ctx)
	if err != nil {
		return err
	}

	if _, err := a.SetSetting(ctx, store.KW{store.Opt("yt_dlp_version", ytDlpVersion)}); err != nil {
		return err
	}

	return nil
}

// runAppInitScript/0
// Elixir ignores the script's exit code, so a non-zero exit must not stop boot.
func runAppInitScript(ctx context.Context, a *App) error {
	err := a.UserScripts.Run(ctx, "app_init", store.Attrs{})
	var cmdErr *fsutil.CommandError
	if errors.As(err, &cmdErr) {
		slog.Warn("app_init user script exited non-zero", "status", cmdErr.Status)
		return nil
	}
	return err
}
