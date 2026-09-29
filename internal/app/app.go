package app

// App replaces the Elixir application environment: the Repo, Oban, the
// Application.get_env config, and the swappable runner modules
// (yt_dlp_runner, user_script_runner, http_client).
// Any ported function that touched one of those is a method on *App.
//
// Hand-written W0 infrastructure; not a manifest row.

import (
	"context"
	"net/http"
	"time"

	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
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

// YtDlpRunner runs yt-dlp; *ytdlp.Runner is the real implementation.
type YtDlpRunner interface {
	// Run runs yt-dlp against url for action (used for logging), with CLI
	// args, an output template and per-call options. Failure returns
	// *fsutil.CommandError.
	Run(ctx context.Context, url string, action string, args ytdlp.Args, outputTemplate string, opts ytdlp.CallOptions) (string, error)
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
	Get(ctx context.Context, url string, headers http.Header) (string, error)
}
