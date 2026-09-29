package main

// Port of config/runtime.exs (prod branch) and the defaults in config.exs.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
)

// Settings is everything main reads from the environment.
type Settings struct {
	Core               app.Config
	JournalMode        string
	Port               int
	EnableIPv6         bool
	EnablePrometheus   bool
	LogLevel           string
	SecretKeyBase      string
	YtDlpWorkerCount   int
	Umask              string
	PermissionCheckDir []string
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

// present mirrors `String.length(System.get_env(key, "")) > 0`.
func present(key string) bool { return os.Getenv(key) != "" }

func loadSettings() (Settings, error) {
	mediaPath := getenv("MEDIA_PATH", "/downloads")
	configPath := getenv("CONFIG_PATH", "/config")
	extras := getenv("EXTRAS_PATH", filepath.Join(configPath, "extras"))

	yt, _ := exec.LookPath("yt-dlp")

	s := Settings{
		Core: app.Config{
			Env:                     "prod",
			YtDlpExecutable:         yt,
			MediaDirectory:          mediaPath,
			MetadataDirectory:       getenv("METADATA_PATH", filepath.Join(configPath, "metadata")),
			ExtrasDirectory:         extras,
			TmpfileDirectory:        getenv("TMPFILE_PATH", filepath.Join(os.TempDir(), "pinchflat", "data")),
			LogPath:                 getenv("LOG_PATH", filepath.Join(configPath, "logs", "pinchflat.log")),
			DatabasePath:            getenv("DATABASE_PATH", filepath.Join(configPath, "db", "pinchflat.db")),
			BasicAuthUsername:       os.Getenv("BASIC_AUTH_USERNAME"),
			BasicAuthPassword:       os.Getenv("BASIC_AUTH_PASSWORD"),
			ExposeFeedEndpoints:     present("EXPOSE_FEED_ENDPOINTS"),
			FileWatcherPollInterval: time.Second,
			Timezone:                timezone(),
			BaseRoutePath:           getenv("BASE_ROUTE_PATH", "/"),
		},
		JournalMode:      getenv("JOURNAL_MODE", "wal"),
		EnableIPv6:       present("ENABLE_IPV6"),
		EnablePrometheus: present("ENABLE_PROMETHEUS"),
		LogLevel:         getenv("LOG_LEVEL", "debug"),
		Umask:            os.Getenv("UMASK"),
	}

	port, err := strconv.Atoi(getenv("PORT", "4000"))
	if err != nil {
		return s, err
	}
	s.Port = port

	workers, err := strconv.Atoi(getenv("YT_DLP_WORKER_CONCURRENCY", "2"))
	if err != nil {
		return s, err
	}
	s.YtDlpWorkerCount = workers

	s.SecretKeyBase = os.Getenv("SECRET_KEY_BASE")
	if s.SecretKeyBase == "" && os.Getenv("RUN_CONTEXT") == "selfhosted" {
		// Same default as the Elixir image; fine on a private network. Set
		// SECRET_KEY_BASE if Pinchflat is internet-facing.
		s.SecretKeyBase = "ZkuQMStdmUzBv+gO3m3XZrtQW76e+AX3QIgTLajw3b/HkTLMEx+DOXr2WZsSS+n8"
	}

	// Release.check_file_permissions checks these (tzdata's dir is gone:
	// Go embeds its own zone data).
	s.PermissionCheckDir = uniq([]string{"/config", "/downloads", "/etc/yt-dlp", "/etc/yt-dlp/plugins",
		s.Core.MediaDirectory, s.Core.TmpfileDirectory, s.Core.ExtrasDirectory, s.Core.MetadataDirectory})
	return s, nil
}

// timezone ports Application.check_and_update_timezone/0.
func timezone() string {
	tz := os.Getenv("TIMEZONE")
	if tz == "" {
		tz = os.Getenv("TZ")
	}
	if tz == "" {
		return "UTC"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		earlyWarnings = append(earlyWarnings, "Invalid timezone "+tz+", defaulting to UTC")
		return "UTC"
	}
	return tz
}

var earlyWarnings []string

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
