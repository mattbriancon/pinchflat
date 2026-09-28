package core

// App replaces the Elixir application environment: the Repo, Oban, the
// Application.get_env config, and the swappable runner modules
// (yt_dlp_runner, user_script_runner, http_client).
// Any ported function that touched one of those is a method on *App.
//
// Hand-written W0 infrastructure; not a manifest row.

import (
	"context"
	"time"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// Config mirrors the :pinchflat application env (config/*.exs).
type Config struct {
	Env                     string // "prod", "dev" or "test" (config_env())
	YtDlpExecutable         string
	MediaDirectory          string
	MetadataDirectory       string
	ExtrasDirectory         string
	TmpfileDirectory        string
	LogPath                 string
	DatabasePath            string
	BasicAuthUsername       string
	BasicAuthPassword       string
	ExposeFeedEndpoints     bool
	FileWatcherPollInterval time.Duration
	Timezone                string
	BaseRoutePath           string
}

// App holds the dependencies every context, worker and helper needs.
type App struct {
	*store.Store
	Config Config

	YtDlp       YtDlpRunner
	UserScripts UserScriptRunner
	HTTP        HTTPClient
}

// --- Behaviours (formerly Mox-mocked modules) ---

// YtDlpRunner is Pinchflat.YtDlp.YtDlpCommandRunner.
type YtDlpRunner interface {
	// Run runs yt-dlp against url for action (the atom in Elixir, e.g.
	// "get_media_attributes_for_collection"), with CLI options, an output
	// template and additional options (use_cookies, skip_sleep_interval,
	// output_filepath). Failure returns *CommandError.
	Run(ctx context.Context, url string, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error)
	Version(ctx context.Context) (string, error)
	Update(ctx context.Context) (string, error)
}

// UserScriptRunner is Pinchflat.Lifecycle.UserScripts.UserScriptCommandRunner.
// event is the atom name ("media_downloaded"); data is JSON-encoded.
type UserScriptRunner interface {
	Run(ctx context.Context, event string, data any) error
}

// HTTPClient is Pinchflat.HTTP.HTTPBehaviour.
type HTTPClient interface {
	Get(ctx context.Context, url string, headers store.KW, opts store.KW) (string, error)
}
