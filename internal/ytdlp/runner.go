// Package ytdlp is the real yt-dlp process runner: building its command
// line, running it and parsing its JSON output. It has no dependency on the
// database or on the rest of the app; callers pass in whatever executable
// path, directories and settings it needs.
package ytdlp

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

// Settings is the subset of the app's settings the runner consults on
// every call.
type Settings struct {
	// ThroughputLimit is yt-dlp's --limit-rate value ("" leaves it unset).
	ThroughputLimit string
	// SleepIntervalSeconds is the extractor sleep interval (0 leaves it
	// unset); each of --sleep-interval/--sleep-requests/--sleep-subtitles
	// gets its own random jitter, as yt-dlp expects.
	SleepIntervalSeconds int
	// RestrictFilenames adds --restrict-filenames when true.
	RestrictFilenames bool
}

// CallOptions are the per-call knobs that used to travel as addlOpts.
type CallOptions struct {
	// OutputFilepath is where --print-to-file writes its output. A tmpfile
	// is generated under TmpDir when this is empty.
	OutputFilepath string
	// UseCookies adds --cookies when a non-empty cookies.txt exists in
	// CookieDir.
	UseCookies bool
	// SkipSleepInterval skips the sleep-interval options for this call.
	SkipSleepInterval bool
}

// Runner is the real yt-dlp process runner.
type Runner struct {
	// Executable is the yt-dlp binary to run.
	Executable string
	// TmpDir is used as the working directory, for yt-dlp's own cache, and
	// for generated --print-to-file output files.
	TmpDir string
	// CookieDir holds cookies.txt, if any.
	CookieDir string
	// SettingsFunc returns the current settings values for a call; may be
	// nil, in which case none of Settings' options are applied.
	SettingsFunc func(ctx context.Context) Settings
}

// Run runs yt-dlp against url for action (used only for logging), with CLI
// args, an output template and per-call options. It always adds cookie,
// rate-limit and misc options (from Settings) plus the app's fixed global
// options, then reads back and returns the --print-to-file output.
//
// yt-dlp exit codes:
//
//	0   = success
//	100 = yt-dlp must restart for update to complete
//	101 = download cancelled by --max-downloads etc (treated as success)
//	2   = error in user-provided options
//	1   = any other error
//
// A non-0/101 status is returned as *fsutil.CommandError.
func (r *Runner) Run(ctx context.Context, url string, action string, args []string, outputTemplate string, opts CallOptions) (string, error) {
	slog.Debug("Running yt-dlp command for action: " + action)

	outputFilepath := opts.OutputFilepath
	if outputFilepath == "" {
		var err error
		outputFilepath, err = fsutil.GenerateTmpfile(r.TmpDir, "json")
		if err != nil {
			return "", err
		}
	}

	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, r.cookieFileArgs(opts.UseCookies)...)
	settings := r.settings(ctx)
	allArgs = append(allArgs, rateLimitArgs(settings, opts.SkipSleepInterval)...)
	if settings.RestrictFilenames {
		allArgs = append(allArgs, "--restrict-filenames")
	}
	allArgs = append(allArgs, r.globalArgs()...)
	allArgs = append(allArgs, "--print-to-file", outputTemplate, outputFilepath)

	formattedArgs := append([]string{url}, allArgs...)

	output, status, err := fsutil.RunCommand(ctx, r.TmpDir, r.Executable, formattedArgs, fsutil.RunOptions{StderrToStdout: true})
	if err != nil {
		return "", err
	}

	if status == 0 || status == 101 {
		content, err := os.ReadFile(outputFilepath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	return "", &fsutil.CommandError{Output: output, Status: status}
}

// Version runs `yt-dlp --version`.
func (r *Runner) Version(ctx context.Context) (string, error) {
	output, status, err := fsutil.RunCommand(ctx, r.TmpDir, r.Executable, []string{"--version"}, fsutil.RunOptions{})
	if err != nil {
		return "", err
	}
	if status == 0 {
		return strings.TrimSpace(output), nil
	}
	return "", &fsutil.CommandError{Output: output, Status: status}
}

// Update runs `yt-dlp --update`.
func (r *Runner) Update(ctx context.Context) (string, error) {
	output, status, err := fsutil.RunCommand(ctx, r.TmpDir, r.Executable, []string{"--update"}, fsutil.RunOptions{})
	if err != nil {
		return "", err
	}
	if status == 0 {
		return strings.TrimSpace(output), nil
	}
	return "", &fsutil.CommandError{Output: output, Status: status}
}

func (r *Runner) cookieFileArgs(useCookies bool) []string {
	if !useCookies {
		return nil
	}
	cookiesPath := filepath.Join(r.CookieDir, "cookies.txt")
	if !fsutil.ExistsAndNonEmpty(cookiesPath) {
		return nil
	}
	return []string{"--cookies", cookiesPath}
}

func rateLimitArgs(settings Settings, skipSleepInterval bool) []string {
	var args []string
	if settings.ThroughputLimit != "" {
		args = append(args, "--limit-rate", settings.ThroughputLimit)
	}

	if settings.SleepIntervalSeconds <= 0 || skipSleepInterval {
		return args
	}

	// Jitter is computed separately for each option, as in Elixir.
	args = append(args,
		"--sleep-requests", strconv.Itoa(fsutil.AddJitter(settings.SleepIntervalSeconds, 0.5)),
		"--sleep-interval", strconv.Itoa(fsutil.AddJitter(settings.SleepIntervalSeconds, 0.5)),
		"--sleep-subtitles", strconv.Itoa(fsutil.AddJitter(settings.SleepIntervalSeconds, 0.5)),
	)
	return args
}

func (r *Runner) globalArgs() []string {
	// Resolves an issue where yt-dlp would attempt to write to a read-only
	// directory if you scanned a new video with --windows-filenames enabled.
	cacheDir := filepath.Join(r.TmpDir, "yt-dlp-cache")
	return []string{"--windows-filenames", "--quiet", "--cache-dir", cacheDir}
}

func (r *Runner) settings(ctx context.Context) Settings {
	if r.SettingsFunc == nil {
		return Settings{}
	}
	return r.SettingsFunc(ctx)
}
