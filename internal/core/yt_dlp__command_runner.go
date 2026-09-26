package core

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// YtDlpCommandRunner is Pinchflat.YtDlp.CommandRunner, the real yt-dlp
// runner. It implements YtDlpRunner (app.go).
type YtDlpCommandRunner struct {
	App *App
}

var _ YtDlpRunner = (*YtDlpCommandRunner)(nil)

// run/5 — {:ok, output} | {:error, output, status} (as *CommandError)
func (r *YtDlpCommandRunner) Run(ctx context.Context, url string, actionName string, commandOpts KW, outputTemplate string, addlOpts KW) (string, error) {
	slog.Debug("Running yt-dlp command for action: " + actionName)

	outputFilepath, err := r.generateOutputFilepath(ctx, addlOpts)
	if err != nil {
		return "", err
	}

	// Build options as mixed []interface{} to support both KV pairs and bare strings
	// This allows us to do: --print-to-file <template> <filepath>
	userConfiguredOpts := ytDlpCommandRunnerCookieFileOptions(ctx, r.App, addlOpts)
	userConfiguredOpts = append(userConfiguredOpts, ytDlpCommandRunnerRateLimitOptions(ctx, r.App, addlOpts)...)
	userConfiguredOpts = append(userConfiguredOpts, ytDlpCommandRunnerMiscOptions(ctx, r.App)...)

	allOpts := append(KW{}, commandOpts...)
	allOpts = append(allOpts, userConfiguredOpts...)
	allOpts = append(allOpts, ytDlpCommandRunnerGlobalOptions(r.App)...)

	// Convert to []interface{} and insert print-to-file options in the right place
	var optItems []interface{}
	for _, kv := range allOpts {
		optItems = append(optItems, kv)
	}
	// Add print-to-file options as individual items
	optItems = append(optItems, KV{Key: "print_to_file", Value: outputTemplate, Flag: false})
	optItems = append(optItems, outputFilepath) // bare string

	formattedCommandOpts := append([]string{url}, CliUtilsParseOptions(optItems)...)

	output, status, err := r.App.CliUtilsWrapCmd(ctx, r.App.Config.YtDlpExecutable, formattedCommandOpts, KW{Flag("stderr_to_stdout")}, KW{})
	if err != nil {
		return "", err
	}

	// yt-dlp exit codes:
	//   0 = Everything is successful
	//   100 = yt-dlp must restart for update to complete
	//   101 = Download cancelled by --max-downloads etc
	//     2 = Error in user-provided options
	//     1 = Any other error
	if status == 0 || status == 101 {
		content, err := os.ReadFile(outputFilepath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	return "", &CommandError{Output: output, Status: status}
}

// version/0
func (r *YtDlpCommandRunner) Version(ctx context.Context) (string, error) {
	output, status, err := r.App.CliUtilsWrapCmd(ctx, r.App.Config.YtDlpExecutable, []string{"--version"}, KW{}, KW{})
	if err != nil {
		return "", err
	}

	if status == 0 {
		return strings.TrimSpace(output), nil
	}

	return "", &CommandError{Output: output, Status: status}
}

// update/0
func (r *YtDlpCommandRunner) Update(ctx context.Context) (string, error) {
	output, status, err := r.App.CliUtilsWrapCmd(ctx, r.App.Config.YtDlpExecutable, []string{"--update"}, KW{}, KW{})
	if err != nil {
		return "", err
	}

	if status == 0 {
		return strings.TrimSpace(output), nil
	}

	return "", &CommandError{Output: output, Status: status}
}

func (r *YtDlpCommandRunner) generateOutputFilepath(ctx context.Context, addlOpts KW) (string, error) {
	if filepath, ok := addlOpts.Get("output_filepath"); ok {
		return filepath.(string), nil
	}
	return r.App.FilesystemUtilsGenerateMetadataTmpfile(ctx, "json")
}

func ytDlpCommandRunnerGlobalOptions(a *App) KW {
	// Use tmpfile_directory from config, same as Elixir: Path.join(tmpfile_directory, "yt-dlp-cache")
	cacheDir := filepath.Join(a.Config.TmpfileDirectory, "yt-dlp-cache")
	return KW{
		Flag("windows_filenames"),
		Flag("quiet"),
		Opt("cache_dir", cacheDir),
	}
}

func ytDlpCommandRunnerCookieFileOptions(ctx context.Context, a *App, addlOpts KW) KW {
	if !addlOpts.Bool("use_cookies") {
		return KW{}
	}
	return ytDlpCommandRunnerAddCookieFile(ctx, a)
}

func ytDlpCommandRunnerAddCookieFile(ctx context.Context, a *App) KW {
	result := KW{}
	baseDir := a.Config.ExtrasDirectory
	cookiesPath := filepath.Join(baseDir, "cookies.txt")

	if FilesystemUtilsExistsAndNonempty(ctx, cookiesPath) {
		result = append(result, Opt("cookies", cookiesPath))
	}

	return result
}

func ytDlpCommandRunnerRateLimitOptions(ctx context.Context, a *App, addlOpts KW) KW {
	result := KW{}

	throughputLimitAny, err := a.SettingsGetBang(ctx, "download_throughput_limit")
	if err == nil && throughputLimitAny != nil {
		if throughputLimit, ok := throughputLimitAny.(string); ok && throughputLimit != "" {
			result = append(result, Opt("limit_rate", throughputLimit))
		}
	}

	sleepIntervalOpts := ytDlpCommandRunnerSleepIntervalOpts(ctx, a, addlOpts)
	result = append(result, sleepIntervalOpts...)

	return result
}

func ytDlpCommandRunnerSleepIntervalOpts(ctx context.Context, a *App, addlOpts KW) KW {
	result := KW{}

	sleepIntervalAny, err := a.SettingsGetBang(ctx, "extractor_sleep_interval_seconds")
	if err != nil || sleepIntervalAny == nil {
		return result
	}

	sleepInterval, _ := sleepIntervalAny.(int)

	if sleepInterval <= 0 || addlOpts.Bool("skip_sleep_interval") {
		return result
	}

	// Jitter is computed separately for each option, as in Elixir.
	return KW{
		Opt("sleep_requests", NumberUtilsAddJitter(sleepInterval, 0.5)),
		Opt("sleep_interval", NumberUtilsAddJitter(sleepInterval, 0.5)),
		Opt("sleep_subtitles", NumberUtilsAddJitter(sleepInterval, 0.5)),
	}
}

func ytDlpCommandRunnerMiscOptions(ctx context.Context, a *App) KW {
	restrictFilenamesAny, err := a.SettingsGetBang(ctx, "restrict_filenames")
	if err != nil || restrictFilenamesAny == nil {
		return KW{}
	}

	if restrictFilenames, ok := restrictFilenamesAny.(bool); ok && restrictFilenames {
		return KW{Flag("restrict_filenames")}
	}

	return KW{}
}
