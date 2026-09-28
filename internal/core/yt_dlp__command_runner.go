package core

import (
	"context"
	"strconv"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// YtDlpCommandRunner adapts an *ytdlp.Runner (the real yt-dlp process
// runner) to the YtDlpRunner interface, translating this package's store.KW
// option lists to the plain args and CallOptions the runner takes.
type YtDlpCommandRunner struct {
	Runner *ytdlp.Runner
}

var _ YtDlpRunner = (*YtDlpCommandRunner)(nil)

// NewYtDlpCommandRunner builds the real runner from a's config, reading
// per-call settings (download_throughput_limit, extractor_sleep_interval_seconds,
// restrict_filenames) through a.GetSettingBang.
func NewYtDlpCommandRunner(a *App) *YtDlpCommandRunner {
	return &YtDlpCommandRunner{
		Runner: &ytdlp.Runner{
			Executable:   a.Config.YtDlpExecutable,
			TmpDir:       a.Config.TmpfileDirectory,
			CookieDir:    a.Config.ExtrasDirectory,
			SettingsFunc: func(ctx context.Context) ytdlp.Settings { return ytDlpSettingsFor(ctx, a) },
		},
	}
}

func ytDlpSettingsFor(ctx context.Context, a *App) ytdlp.Settings {
	var settings ytdlp.Settings

	if v, err := a.GetSettingBang(ctx, "download_throughput_limit"); err == nil {
		if s, ok := v.(string); ok {
			settings.ThroughputLimit = s
		}
	}
	if v, err := a.GetSettingBang(ctx, "extractor_sleep_interval_seconds"); err == nil {
		if n, ok := v.(int); ok {
			settings.SleepIntervalSeconds = n
		}
	}
	if v, err := a.GetSettingBang(ctx, "restrict_filenames"); err == nil {
		if b, ok := v.(bool); ok {
			settings.RestrictFilenames = b
		}
	}

	return settings
}

// Run runs yt-dlp against url for action, with CLI options, an output
// template and additional options (use_cookies, skip_sleep_interval,
// output_filepath). Failure returns *fsutil.CommandError.
func (r *YtDlpCommandRunner) Run(ctx context.Context, url string, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
	args := CliUtilsParseOptions(opts)
	return r.Runner.Run(ctx, url, action, args, outputTemplate, ytdlp.CallOptions{
		OutputFilepath:    addlOpts.String("output_filepath"),
		UseCookies:        addlOpts.Bool("use_cookies"),
		SkipSleepInterval: addlOpts.HasFlag("skip_sleep_interval"),
	})
}

// Version runs `yt-dlp --version`.
func (r *YtDlpCommandRunner) Version(ctx context.Context) (string, error) {
	return r.Runner.Version(ctx)
}

// Update runs `yt-dlp --update`.
func (r *YtDlpCommandRunner) Update(ctx context.Context) (string, error) { return r.Runner.Update(ctx) }

// CliUtilsParseOptions parses a command option list into CLI args suitable
// for exec: a store.KW, a []store.KV, a []interface{} of mixed store.KV/string items, a
// single store.KV, or a single string. Atom-style keys are kebab-cased and
// prefixed with "--"; keys already starting with "-" are passed through.
func CliUtilsParseOptions(commandOpts interface{}) []string {
	var items []interface{}
	switch v := commandOpts.(type) {
	case store.KW:
		for _, kv := range v {
			items = append(items, kv)
		}
	case []store.KV:
		for _, kv := range v {
			items = append(items, kv)
		}
	case []interface{}:
		items = v
	case store.KV:
		items = []interface{}{v}
	default:
		items = []interface{}{v}
	}

	var result []string
	for _, item := range items {
		result = cliUtilsParseOption(item, result)
	}
	return result
}

// cliUtilsParseOption processes one item, appending to acc.
func cliUtilsParseOption(item interface{}, acc []string) []string {
	switch v := item.(type) {
	case store.KV:
		if v.Flag {
			return append(acc, "--"+fsutil.ToKebabCase(v.Key))
		}
		key := v.Key
		if !isFlagLike(key) {
			key = "--" + fsutil.ToKebabCase(key)
		}
		return append(acc, key, toArgString(v.Value))
	case string:
		return append(acc, v)
	default:
		return append(acc, toArgString(v))
	}
}

func isFlagLike(key string) bool {
	return len(key) > 0 && key[0] == '-'
}

func toArgString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return ""
	}
}
