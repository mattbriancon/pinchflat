package ytdlp_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

const mediaURL = "https://www.youtube.com/watch?v=-LHXuyzpex0"

// newTestRunner returns a Runner pointed at the repeater.sh mock, which
// echoes back whatever args it's called with (see
// testdata/support/scripts/yt-dlp-mocks/repeater.sh).
func newTestRunner(t *testing.T, settings ytdlp.Settings) *ytdlp.Runner {
	t.Helper()
	dir := t.TempDir()
	for _, p := range []string{filepath.Join(dir, "tmpfiles"), filepath.Join(dir, "extras")} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return &ytdlp.Runner{
		Executable:   filepath.Join(dbtest.RepoRoot(), "testdata/support/scripts/yt-dlp-mocks/repeater.sh"),
		TmpDir:       filepath.Join(dir, "tmpfiles"),
		CookieDir:    filepath.Join(dir, "extras"),
		SettingsFunc: func(context.Context) ytdlp.Settings { return settings },
	}
}

func TestRunnerRun(t *testing.T) {
	t.Run("returns the output and status when the command succeeds", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("considers a 101 exit code as being successful", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		runner.Executable = filepath.Join(dbtest.RepoRoot(), "testdata/support/scripts/yt-dlp-mocks/101_exit_code.sh")
		_, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("includes the media url as the first argument", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		output, err := runner.Run(context.Background(), mediaURL, "foo", ytdlp.Args{}.Flag("ignore_errors"), "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, mediaURL+" --ignore-errors") {
			t.Errorf("expected output to contain %q, got %q", mediaURL+" --ignore-errors", output)
		}
	})

	t.Run("automatically includes the --print-to-file flag", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "%(id)s", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--print-to-file %(id)s "+runner.TmpDir) {
			t.Errorf("expected output to contain print-to-file flag, got %q", output)
		}
	})

	t.Run("returns the output and status when the command fails", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		runner.Executable = "/bin/false"
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err == nil {
			t.Fatal("expected an error")
		}
		if output != "" {
			t.Errorf("expected empty output on error, got %q", output)
		}
		if cmdErr, ok := err.(*fsutil.CommandError); !ok || cmdErr.Status != 1 {
			t.Errorf("expected CommandError with status 1, got %T: %v", err, err)
		}
	})

	t.Run("optionally lets you specify an output_filepath", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		outputPath := filepath.Join(t.TempDir(), "yt-dlp-output.json")
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "%(id)s", ytdlp.CallOptions{OutputFilepath: outputPath})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--print-to-file %(id)s "+outputPath) {
			t.Errorf("expected output to contain output path, got %q", output)
		}
	})
}

func TestRunnerRunCookieFileOptions(t *testing.T) {
	t.Run("includes cookie options when cookies.txt exists and enabled", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		cookieFile := filepath.Join(runner.CookieDir, "cookies.txt")
		if err := os.WriteFile(cookieFile, []byte("cookie data"), 0644); err != nil {
			t.Fatal(err)
		}

		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{UseCookies: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--cookies "+cookieFile) {
			t.Errorf("expected output to contain cookies option, got %q", output)
		}
	})

	t.Run("doesn't include cookie options when disabled", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		cookieFile := filepath.Join(runner.CookieDir, "cookies.txt")
		if err := os.WriteFile(cookieFile, []byte("cookie data"), 0644); err != nil {
			t.Fatal(err)
		}

		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{UseCookies: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--cookies "+cookieFile) {
			t.Errorf("expected output to NOT contain cookies option, got %q", output)
		}
	})

	t.Run("doesn't include cookie options when cookies.txt is blank", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		cookieFile := filepath.Join(runner.CookieDir, "cookies.txt")
		if err := os.WriteFile(cookieFile, []byte(" \n \n "), 0644); err != nil {
			t.Fatal(err)
		}

		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{UseCookies: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--cookies") {
			t.Errorf("expected output to NOT contain cookies option, got %q", output)
		}
	})

	t.Run("doesn't include cookie options when cookies.txt doesn't exist", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{UseCookies: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--cookies") {
			t.Errorf("expected output to NOT contain cookies option, got %q", output)
		}
	})
}

func TestRunnerRunRateLimitOptions(t *testing.T) {
	t.Run("includes sleep interval options by default", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{SleepIntervalSeconds: 5})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, want := range []string{"--sleep-interval", "--sleep-requests", "--sleep-subtitles"} {
			if !strings.Contains(output, want) {
				t.Errorf("expected output to contain %s, got %q", want, output)
			}
		}
	})

	t.Run("doesn't include sleep interval options when skip_sleep_interval is true", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{SleepIntervalSeconds: 5})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{SkipSleepInterval: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--sleep-interval") {
			t.Errorf("expected output to NOT contain --sleep-interval, got %q", output)
		}
	})

	t.Run("doesn't include sleep interval options when the sleep interval is 0", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{SleepIntervalSeconds: 0})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--sleep-interval") {
			t.Errorf("expected output to NOT contain --sleep-interval, got %q", output)
		}
	})

	t.Run("includes limit_rate option when specified", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{ThroughputLimit: "100K"})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--limit-rate 100K") {
			t.Errorf("expected output to contain --limit-rate 100K, got %q", output)
		}
	})

	t.Run("doesn't include limit_rate option when unset", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--limit-rate") {
			t.Errorf("expected output to NOT contain --limit-rate, got %q", output)
		}
	})
}

func TestRunnerRunGlobalOptions(t *testing.T) {
	runner := newTestRunner(t, ytdlp.Settings{})
	output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"--windows-filenames", "--quiet", "--cache-dir"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %s, got %q", want, output)
		}
	}
}

func TestRunnerRunMiscOptions(t *testing.T) {
	t.Run("includes restrict_filenames flag when enabled", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{RestrictFilenames: true})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--restrict-filenames") {
			t.Errorf("expected output to contain --restrict-filenames, got %q", output)
		}
	})

	t.Run("doesn't include restrict_filenames flag when disabled", func(t *testing.T) {
		runner := newTestRunner(t, ytdlp.Settings{RestrictFilenames: false})
		output, err := runner.Run(context.Background(), mediaURL, "foo", nil, "", ytdlp.CallOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--restrict-filenames") {
			t.Errorf("expected output to NOT contain --restrict-filenames, got %q", output)
		}
	})
}

func TestRunnerVersion(t *testing.T) {
	runner := newTestRunner(t, ytdlp.Settings{})
	output, err := runner.Version(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "--version") {
		t.Errorf("expected output to contain --version, got %q", output)
	}
}

func TestRunnerUpdate(t *testing.T) {
	runner := newTestRunner(t, ytdlp.Settings{})
	output, err := runner.Update(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "--update") {
		t.Errorf("expected output to contain --update, got %q", output)
	}
}
