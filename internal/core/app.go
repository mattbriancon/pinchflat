package core

// App replaces the Elixir application environment: the Repo, Oban, the
// Application.get_env config, and the swappable runner modules
// (yt_dlp_runner, apprise_runner, user_script_runner, http_client).
// Any ported function that touched one of those is a method on *App.
//
// Hand-written W0 infrastructure; not a manifest row.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// Config mirrors the :pinchflat application env (config/*.exs).
type Config struct {
	Env                     string // "prod", "dev" or "test" (config_env())
	YtDlpExecutable         string
	AppriseExecutable       string
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
	DB     *db.DB
	Oban   *obanlite.Oban
	Config Config

	YtDlp       YtDlpRunner
	Apprise     AppriseRunner
	UserScripts UserScriptRunner
	HTTP        HTTPClient
}

// Q returns the querier to use: the transaction in ctx if one was started
// with InTx, else the database. Always write `a.Q(ctx)` where Elixir used Repo.
func (a *App) Q(ctx context.Context) db.Querier {
	if tx, ok := ctx.Value(txKey{}).(*db.Tx); ok {
		return tx
	}
	return a.DB
}

type txKey struct{}

// InTx runs fn in a transaction (Repo.transaction/1). Calls to a.Q(ctx)
// inside fn use the transaction. Nested calls reuse the outer transaction.
func (a *App) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*db.Tx); ok {
		return fn(ctx)
	}
	return a.DB.InTx(ctx, func(tx *db.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// --- Behaviours (formerly Mox-mocked modules) ---

// YtDlpRunner is Pinchflat.YtDlp.YtDlpCommandRunner.
type YtDlpRunner interface {
	// Run runs yt-dlp against url for action (the atom in Elixir, e.g.
	// "get_media_attributes_for_collection"), with CLI options, an output
	// template and additional options (use_cookies, skip_sleep_interval,
	// output_filepath). Failure returns *CommandError.
	Run(ctx context.Context, url string, action string, opts KW, outputTemplate string, addlOpts KW) (string, error)
	Version(ctx context.Context) (string, error)
	Update(ctx context.Context) (string, error)
}

// AppriseRunner is Pinchflat.Lifecycle.Notifications.AppriseCommandRunner.
// endpoints is one server string or a list, as in Elixir.
type AppriseRunner interface {
	Run(ctx context.Context, endpoints []string, opts KW) error
	Version(ctx context.Context) (string, error)
}

// UserScriptRunner is Pinchflat.Lifecycle.UserScripts.UserScriptCommandRunner.
// event is the atom name ("media_downloaded"); data is JSON-encoded.
type UserScriptRunner interface {
	Run(ctx context.Context, event string, data any) error
}

// HTTPClient is Pinchflat.HTTP.HTTPBehaviour.
type HTTPClient interface {
	Get(ctx context.Context, url string, headers KW, opts KW) (string, error)
}

// CommandError is Elixir's {:error, output, status} from a command runner.
type CommandError struct {
	Output string
	Status int
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command exited %d: %s", e.Status, strings.TrimSpace(e.Output))
}

// --- Keyword lists ---

// KV is one keyword-list entry. A bare atom in the list (e.g. :no_warnings)
// is a KV with Flag set and no Value.
type KV struct {
	Key   string
	Value any
	Flag  bool
}

// KW ports an Elixir keyword list, keeping order and duplicates. Use it for
// yt-dlp option lists and for `opts \\ []` function options.
type KW []KV

// Flag builds a bare-atom entry: [:no_warnings] -> KW{Flag("no_warnings")}.
func Flag(key string) KV { return KV{Key: key, Flag: true} }

// Opt builds a key/value entry: [output: "x"] -> KW{Opt("output", "x")}.
func Opt(key string, value any) KV { return KV{Key: key, Value: value} }

// Get returns the first value for key (Keyword.get/3 without default).
func (kw KW) Get(key string) (any, bool) {
	for _, kv := range kw {
		if kv.Key == key && !kv.Flag {
			return kv.Value, true
		}
	}
	return nil, false
}

// GetOr is Keyword.get/3 with a default.
func (kw KW) GetOr(key string, def any) any {
	if v, ok := kw.Get(key); ok {
		return v
	}
	return def
}

// Bool returns a boolean option, false if absent.
func (kw KW) Bool(key string) bool {
	v, ok := kw.Get(key)
	b, _ := v.(bool)
	return ok && b
}

// String returns a string option, "" if absent.
func (kw KW) String(key string) string {
	v, _ := kw.Get(key)
	s, _ := v.(string)
	return s
}

// HasFlag reports whether the bare atom key is present (`:foo in opts`).
func (kw KW) HasFlag(key string) bool {
	for _, kv := range kw {
		if kv.Key == key && kv.Flag {
			return true
		}
	}
	return false
}

// Has reports whether key is present as a flag or key/value.
func (kw KW) Has(key string) bool {
	for _, kv := range kw {
		if kv.Key == key {
			return true
		}
	}
	return false
}

// Contains reports whether the exact entry is present (`{:k, v} in opts`).
func (kw KW) Contains(e KV) bool {
	for _, kv := range kw {
		if kv.Key == e.Key && kv.Flag == e.Flag && fmt.Sprint(kv.Value) == fmt.Sprint(e.Value) {
			return true
		}
	}
	return false
}

// Put replaces all entries for key with one value (Keyword.put/3).
func (kw KW) Put(key string, value any) KW {
	out := kw.Delete(key)
	return append(out, Opt(key, value))
}

// Delete removes all entries for key (Keyword.delete/2).
func (kw KW) Delete(key string) KW {
	out := make(KW, 0, len(kw))
	for _, kv := range kw {
		if kv.Key != key {
			out = append(out, kv)
		}
	}
	return out
}
