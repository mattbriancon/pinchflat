// Command pinchflat is the Go port of the Pinchflat application
// (lib/pinchflat/application.ex plus rel/overlays/bin/docker_start and
// migrate).
//
//	pinchflat                      check permissions, migrate, run (docker_start)
//	pinchflat migrate              apply pending migrations and exit (bin/migrate)
//	pinchflat check-permissions    Release.check_file_permissions
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // TZ works without system zoneinfo (replaces TZ_DATA_DIR)

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web"
	"gopkg.in/natefinch/lumberjack.v2"
)

// version is set at build time: -ldflags "-X main.version=2025.9.26"
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	s, err := loadSettings()
	if err != nil {
		return err
	}
	setupLogging(s)
	for _, w := range earlyWarnings {
		slog.Warn(w)
	}

	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "check-permissions":
		return checkFilePermissions(s.PermissionCheckDir)
	case "migrate":
		d, err := openDB(s)
		if err != nil {
			return err
		}
		defer d.Close()
		return migrate(d)
	case "", "start":
		return start(s)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// start is docker_start: permissions, umask, migrate, then the app.
func start(s Settings) error {
	if err := checkFilePermissions(s.PermissionCheckDir); err != nil {
		return errors.New("Filesystem error. Exiting.")
	}
	if s.Umask != "" {
		mask, err := strconv.ParseUint(s.Umask, 8, 32)
		if err != nil {
			return fmt.Errorf("invalid UMASK %q: %w", s.Umask, err)
		}
		slog.Info("Setting umask to " + s.Umask)
		syscall.Umask(int(mask))
	}

	d, err := openDB(s)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := migrate(d); err != nil {
		return err
	}

	a := &app.App{Store: &store.Store{DB: d, Oban: obanlite.New(d)}, Config: s.Core}
	a.YtDlp = app.NewYtDlpCommandRunner(a)
	a.UserScripts = &app.UserScriptsCommandRunner{App: a}
	a.HTTP = &app.HTTPClientImpl{}
	a.RegisterWorkers()
	a.Oban.OnEvent = logJobEvent

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Supervision order from application.ex: Repo, PreJobStartupTasks,
	// Oban, Endpoint, PostBootStartupTasks.
	if err := a.PreJobStartupTasksInit(ctx); err != nil {
		slog.Error("pre-job startup tasks failed", "err", err)
	}

	obanDone := make(chan error, 1)
	go func() { obanDone <- a.Oban.Start(ctx, obanConfig(s)) }()

	srv := &http.Server{Handler: newHandler(a, s), ReadHeaderTimeout: 30 * time.Second}
	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(s.Port))
	if s.EnableIPv6 {
		addr = net.JoinHostPort("::", strconv.Itoa(s.Port))
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	slog.Info("Running Pinchflat " + version + " at " + addr)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server", "err", err)
			stop()
		}
	}()

	if err := a.PostBootStartupTasksInit(ctx); err != nil {
		slog.Error("post-boot startup tasks failed", "err", err)
	}

	<-ctx.Done()
	slog.Info("Shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	<-obanDone
	return nil
}

// obanConfig ports the Oban config in config/runtime.exs.
func obanConfig(s Settings) obanlite.Config {
	now := time.Now().UTC()
	return obanlite.Config{
		Queues: map[string]int{
			"default":                   10,
			"fast_indexing":             s.YtDlpWorkerCount,
			"media_collection_indexing": s.YtDlpWorkerCount,
			"media_fetching":            s.YtDlpWorkerCount,
			"remote_metadata":           s.YtDlpWorkerCount,
			"local_data":                8,
		},
		// Keep old jobs for 30 days for display in the UI
		PruneMaxAge: 30 * 24 * time.Hour,
		Crontab: []obanlite.CronEntry{
			// Staggered per instance so every install doesn't update yt-dlp at once.
			{Expr: fmt.Sprintf("%d %d * * *", now.Minute(), now.Hour()), Worker: app.UpdateWorkerName},
			{Expr: "0 1 * * *", Worker: app.MediaRetentionWorkerName},
			{Expr: "0 2 * * *", Worker: app.MediaQualityUpgradeWorkerName},
		},
	}
}

func openDB(s Settings) (*db.DB, error) {
	if err := os.MkdirAll(filepath.Dir(s.Core.DatabasePath), 0o755); err != nil {
		return nil, err
	}
	return db.Open(s.Core.DatabasePath, db.Options{JournalMode: s.JournalMode})
}

func migrate(d *db.DB) error {
	ran, err := d.Migrate(context.Background())
	for _, v := range ran {
		slog.Info("Migrated " + strconv.FormatInt(v, 10))
	}
	return err
}

// setupLogging logs to stdout and to LOG_PATH, rotated like the Elixir
// :logger_std_h handler (10 MB, 5 files).
func setupLogging(s Settings) {
	level := slog.LevelDebug
	switch strings.ToLower(s.LogLevel) {
	case "info", "notice":
		level = slog.LevelInfo
	case "warning", "warn":
		level = slog.LevelWarn
	case "error", "critical", "alert", "emergency":
		level = slog.LevelError
	}
	var out io.Writer = os.Stdout
	if s.Core.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(s.Core.LogPath), 0o755); err == nil {
			out = io.MultiWriter(os.Stdout, &lumberjack.Logger{Filename: s.Core.LogPath, MaxSize: 10, MaxBackups: 5})
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: level})))
}

// logJobEvent replaces Oban.Telemetry.attach_default_logger/0.
func logJobEvent(event string, job *obanlite.Job, err error, dur time.Duration) {
	attrs := []any{"event", "job:" + event, "worker", job.Worker, "queue", job.Queue, "id", job.ID, "attempt", job.Attempt}
	switch event {
	case "stop", "exception":
		outcome := "completed"
		if event == "exception" {
			outcome = "failed"
		} else if err != nil {
			outcome = "cancelled"
		}
		web.JobsTotal.WithLabelValues(job.Queue, job.Worker, outcome).Inc()
		web.JobDuration.WithLabelValues(job.Queue, job.Worker).Observe(dur.Seconds())
	}
	switch event {
	case "start":
		slog.Info("job started", attrs...)
	case "stop":
		slog.Info("job stopped", append(attrs, "duration", dur)...)
	case "exception":
		slog.Error("job failed", append(attrs, "duration", dur, "error", err)...)
	}
}
