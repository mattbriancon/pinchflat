package core_test

// Hand-written W0 compatibility test: each worker's queue, priority,
// max_attempts and tags must match the jobs the Elixir app enqueued.

import (
	"context"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestWorkerOptionsMatchElixirJobs(t *testing.T) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	app := &core.App{DB: d, Oban: obanlite.New(d)}
	app.RegisterWorkers()

	type row struct {
		Worker      string `db:"worker"`
		Queue       string `db:"queue"`
		Priority    int    `db:"priority"`
		MaxAttempts int    `db:"max_attempts"`
		Tags        string `db:"tags"`
	}
	var rows []row
	if err := d.Select(&rows, `SELECT worker, queue, priority, max_attempts, tags FROM oban_jobs GROUP BY worker`); err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(core.WorkerNames) {
		t.Fatalf("fixture has %d workers, Go registers %d", len(rows), len(core.WorkerNames))
	}
	for _, r := range rows {
		t.Run(r.Worker, func(t *testing.T) {
			opts, ok := app.Oban.WorkerOptsFor(r.Worker)
			if !ok {
				t.Fatalf("worker %s not registered", r.Worker)
			}
			tags, _ := obanlite.EncodeArgs(map[string]any{"t": opts.Tags})
			want, _ := obanlite.EncodeArgs(map[string]any{"t": rawJSON(r.Tags)})
			if opts.Queue != r.Queue || opts.Priority != r.Priority || opts.MaxAttempts != r.MaxAttempts || string(tags) != string(want) {
				t.Errorf("Go opts %+v; Elixir job queue=%s priority=%d max_attempts=%d tags=%s", opts, r.Queue, r.Priority, r.MaxAttempts, r.Tags)
			}
		})
	}
	_ = context.Background()
}

type rawJSON string

func (r rawJSON) MarshalJSON() ([]byte, error) { return []byte(r), nil }
