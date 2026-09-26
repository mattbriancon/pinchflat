package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestYtDlpCommandRunner_Run(t *testing.T) {
	t.Run("returns the output and status when the command succeeds", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		// Create a real command runner
		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("considers a 101 exit code as being successful", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		// Wrap the executable to use 101_exit_code.sh
		originalExec := ta.Config.YtDlpExecutable
		ta.Config.YtDlpExecutable = coretest.RepoPath("test/support/scripts/yt-dlp-mocks/101_exit_code.sh")
		defer func() { ta.Config.YtDlpExecutable = originalExec }()

		runner := &core.YtDlpCommandRunner{App: ta.App}
		_, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("includes the media url as the first argument", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{core.Flag("ignore_errors")}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, mediaURL+" --ignore-errors") {
			t.Errorf("expected output to contain %q, got %q", mediaURL+" --ignore-errors", output)
		}
	})

	t.Run("automatically includes the --print-to-file flag", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "%(id)s", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--print-to-file %(id)s /tmp/") {
			t.Errorf("expected output to contain print-to-file flag, got %q", output)
		}
	})

	t.Run("returns the output and status when the command fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		originalExec := ta.Config.YtDlpExecutable
		ta.Config.YtDlpExecutable = "/bin/false"
		defer func() { ta.Config.YtDlpExecutable = originalExec }()

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
		if output != "" {
			t.Errorf("expected empty output on error, got %q", output)
		}
		if cmdErr, ok := err.(*core.CommandError); !ok || cmdErr.Status != 1 {
			t.Errorf("expected CommandError with status 1, got %T: %v", err, err)
		}
	})

	t.Run("optionally lets you specify an output_filepath", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"
		outputPath := filepath.Join(t.TempDir(), "yt-dlp-output.json")

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "%(id)s", core.KW{core.Opt("output_filepath", outputPath)})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--print-to-file %(id)s "+outputPath) {
			t.Errorf("expected output to contain output path, got %q", output)
		}
	})
}

func TestYtDlpCommandRunner_RunExternalFileOptions(t *testing.T) {
	t.Run("includes cookie options when cookies.txt exists and enabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"
		cookieFile := filepath.Join(ta.Config.ExtrasDirectory, "cookies.txt")

		if err := os.WriteFile(cookieFile, []byte("cookie data"), 0644); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{core.Opt("use_cookies", true)})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--cookies "+cookieFile) {
			t.Errorf("expected output to contain cookies option, got %q", output)
		}
	})

	t.Run("doesn't include cookie options when cookies.txt exists but disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"
		cookieFile := filepath.Join(ta.Config.ExtrasDirectory, "cookies.txt")

		if err := os.WriteFile(cookieFile, []byte("cookie data"), 0644); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{core.Opt("use_cookies", false)})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--cookies "+cookieFile) {
			t.Errorf("expected output to NOT contain cookies option, got %q", output)
		}
	})

	t.Run("doesn't include cookie options when cookies.txt blank", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"
		cookieFile := filepath.Join(ta.Config.ExtrasDirectory, "cookies.txt")

		if err := os.WriteFile(cookieFile, []byte(" \n \n "), 0644); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{core.Opt("use_cookies", true)})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--cookies") {
			t.Errorf("expected output to NOT contain cookies option, got %q", output)
		}
	})

	t.Run("doesn't include cookie options when cookies.txt doesn't exist", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"
		cookieFile := filepath.Join(ta.Config.ExtrasDirectory, "cookies.txt")

		// Ensure it doesn't exist
		os.Remove(cookieFile)

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--cookies") {
			t.Errorf("expected output to NOT contain cookies option, got %q", output)
		}
	})
}

func TestYtDlpCommandRunner_RunRateLimitOptions(t *testing.T) {
	t.Run("includes sleep interval options by default", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		if _, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("extractor_sleep_interval_seconds", 5)}); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--sleep-interval") {
			t.Errorf("expected output to contain --sleep-interval, got %q", output)
		}
		if !strings.Contains(output, "--sleep-requests") {
			t.Errorf("expected output to contain --sleep-requests, got %q", output)
		}
		if !strings.Contains(output, "--sleep-subtitles") {
			t.Errorf("expected output to contain --sleep-subtitles, got %q", output)
		}
	})

	t.Run("doesn't include sleep interval options when skip_sleep_interval is true", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{core.Flag("skip_sleep_interval")})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--sleep-interval") {
			t.Errorf("expected output to NOT contain --sleep-interval, got %q", output)
		}
	})

	t.Run("doesn't include sleep interval options when extractor_sleep_interval_seconds is 0", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		if _, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("extractor_sleep_interval_seconds", 0)}); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--sleep-interval") {
			t.Errorf("expected output to NOT contain --sleep-interval, got %q", output)
		}
	})

	t.Run("includes limit_rate option when specified", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		if _, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("download_throughput_limit", "100K")}); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--limit-rate 100K") {
			t.Errorf("expected output to contain --limit-rate 100K, got %q", output)
		}
	})

	t.Run("doesn't include limit_rate option when download_throughput_limit is nil", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		if _, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("download_throughput_limit", nil)}); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--limit-rate") {
			t.Errorf("expected output to NOT contain --limit-rate, got %q", output)
		}
	})
}

func TestYtDlpCommandRunner_RunGlobalOptions(t *testing.T) {
	t.Run("creates windows-safe filenames", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--windows-filenames") {
			t.Errorf("expected output to contain --windows-filenames, got %q", output)
		}
	})

	t.Run("runs quietly", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--quiet") {
			t.Errorf("expected output to contain --quiet, got %q", output)
		}
	})

	t.Run("sets the cache directory", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--cache-dir") {
			t.Errorf("expected output to contain --cache-dir, got %q", output)
		}
	})
}

func TestYtDlpCommandRunner_RunMiscOptions(t *testing.T) {
	t.Run("includes --restrict-filenames when enabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		if _, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("restrict_filenames", true)}); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--restrict-filenames") {
			t.Errorf("expected output to contain --restrict-filenames, got %q", output)
		}
	})

	t.Run("doesn't include --restrict-filenames when disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=-LHXuyzpex0"

		if _, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("restrict_filenames", false)}); err != nil {
			t.Fatal(err)
		}

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Run(ta.Ctx, mediaURL, "foo", core.KW{}, "", core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(output, "--restrict-filenames") {
			t.Errorf("expected output to NOT contain --restrict-filenames, got %q", output)
		}
	})
}

func TestYtDlpCommandRunner_Version(t *testing.T) {
	t.Run("adds the version arg", func(t *testing.T) {
		ta := coretest.NewApp(t)

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Version(ta.Ctx)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--version") {
			t.Errorf("expected output to contain --version, got %q", output)
		}
	})
}

func TestYtDlpCommandRunner_Update(t *testing.T) {
	t.Run("adds the update arg", func(t *testing.T) {
		ta := coretest.NewApp(t)

		runner := &core.YtDlpCommandRunner{App: ta.App}
		output, err := runner.Update(ta.Ctx)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "--update") {
			t.Errorf("expected output to contain --update, got %q", output)
		}
	})
}
