package obanlite_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

const downloadWorker = "Pinchflat.Downloading.MediaDownloadWorker"

// Worker options copied from the Elixir `use Oban.Worker` declarations.
func register(o *obanlite.Oban, w obanlite.Worker) {
	o.Register(downloadWorker, obanlite.WorkerOpts{
		Queue: "media_fetching", Priority: 5,
		Unique: &obanlite.UniqueOpts{Period: obanlite.Infinity, States: []string{"available", "scheduled", "retryable", "executing"}},
		Tags:   []string{"media_item", "media_fetching", "show_in_dashboard"},
	}, w)
	o.Register("Pinchflat.Metadata.SourceMetadataStorageWorker", obanlite.WorkerOpts{
		Queue: "remote_metadata", MaxAttempts: 3,
		Tags: []string{"media_source", "source_metadata", "remote_metadata", "show_in_dashboard"},
	}, w)
	o.Register("Pinchflat.YtDlp.UpdateWorker", obanlite.WorkerOpts{Queue: "local_data", Tags: []string{"local_data"}}, w)
}

var noop = obanlite.WorkerFunc(func(context.Context, *obanlite.Job) error { return nil })

func TestReadsEveryElixirJob(t *testing.T) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	o := obanlite.New(d)
	jobs, err := o.Jobs(context.Background(), "")
	must(t, err)
	if len(jobs) != 13 {
		t.Fatalf("want 13 jobs, got %d", len(jobs))
	}
	states := map[string]bool{}
	for _, j := range jobs {
		states[j.State] = true
		if j.InsertedAt.IsZero() || j.ScheduledAt.IsZero() {
			t.Errorf("job %d: timestamps not parsed", j.ID)
		}
	}
	for _, s := range obanlite.AllStates {
		if !states[s] {
			t.Errorf("fixture has no %s job", s)
		}
	}
}

// A job inserted by Go must be stored exactly as Oban stored the same job.
func TestInsertMatchesElixirEncoding(t *testing.T) {
	elixir := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	var want struct {
		State, Queue, Worker, Args, Meta, Tags, Errors, AttemptedBy string
		Attempt, MaxAttempts, Priority                              int
		InsertedAt, ScheduledAt                                     string
	}
	q := `SELECT state, queue, worker, args, meta, tags, errors, attempted_by AS attemptedby, attempt, max_attempts AS maxattempts, priority,
		inserted_at AS insertedat, scheduled_at AS scheduledat FROM oban_jobs WHERE id = ?`
	if err := elixir.Get(&want, q, 3); err != nil { // MediaDownloadWorker %{id: 4, quality_upgrade?: true}
		t.Fatal(err)
	}

	d := dbtest.New(t)
	o := obanlite.New(d)
	register(o, noop)
	job, err := o.Insert(context.Background(), nil, obanlite.NewJob(downloadWorker, map[string]any{"quality_upgrade?": true, "id": 4}))
	must(t, err)
	got := want
	must(t, d.Get(&got, q, job.ID))
	// Timestamps differ in value but must share the format.
	if len(got.InsertedAt) != len(want.InsertedAt) || got.InsertedAt[10] != want.InsertedAt[10] {
		t.Errorf("inserted_at format: got %q want like %q", got.InsertedAt, want.InsertedAt)
	}
	if len(got.ScheduledAt) != len(want.ScheduledAt) {
		t.Errorf("scheduled_at format: got %q want like %q", got.ScheduledAt, want.ScheduledAt)
	}
	got.InsertedAt, got.ScheduledAt = want.InsertedAt, want.ScheduledAt
	if got != want {
		t.Errorf("stored job differs:\n got %+v\nwant %+v", got, want)
	}
}

func TestScheduledInsertMatchesElixirFormat(t *testing.T) {
	d := dbtest.New(t)
	o := obanlite.New(d)
	register(o, noop)
	spec := obanlite.NewJob(downloadWorker, map[string]any{"id": 1})
	spec.ScheduleIn = 3600
	job, err := o.Insert(context.Background(), nil, spec)
	must(t, err)
	var raw string
	d.Get(&raw, `SELECT scheduled_at FROM oban_jobs WHERE id = ?`, job.ID)
	// Elixir: 2026-09-26T23:00:38.921177Z
	if job.State != "scheduled" || len(raw) != 27 || !strings.HasSuffix(raw, "Z") || raw[10] != 'T' {
		t.Errorf("state %s scheduled_at %q", job.State, raw)
	}
}

func TestUniqueConflictWithElixirJob(t *testing.T) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	o := obanlite.New(d)
	register(o, noop)
	ctx := context.Background()

	// Job 1 is MediaDownloadWorker %{id: 2}, available.
	job, err := o.Insert(ctx, nil, obanlite.NewJob(downloadWorker, map[string]any{"id": 2}))
	must(t, err)
	if !job.Conflict || job.ID != 1 {
		t.Fatalf("want conflict with job 1, got conflict=%v id=%d", job.Conflict, job.ID)
	}

	// Different args (extra key) is a different job: args compared both ways.
	job, err = o.Insert(ctx, nil, obanlite.NewJob(downloadWorker, map[string]any{"id": 2, "force": true}))
	if err != nil || job.Conflict {
		t.Fatalf("want new job, got conflict=%v err=%v", job.Conflict, err)
	}

	// Completed jobs aren't in the worker's unique states.
	mustOK(t)(d.Exec(`UPDATE oban_jobs SET state = 'completed' WHERE id = 1`))
	job, err = o.Insert(ctx, nil, obanlite.NewJob(downloadWorker, map[string]any{"id": 2}))
	if err != nil || job.Conflict {
		t.Fatalf("want new job after completion, got conflict=%v err=%v", job.Conflict, err)
	}
}

func setup(t *testing.T, w obanlite.Worker) (*obanlite.Oban, *db.DB) {
	d := dbtest.New(t)
	o := obanlite.New(d)
	register(o, w)
	return o, d
}

func insertAndRun(t *testing.T, o *obanlite.Oban, worker string) *obanlite.Job {
	t.Helper()
	job, err := o.Insert(context.Background(), nil, obanlite.NewJob(worker, map[string]any{"id": 1}))
	must(t, err)
	job, err = o.PerformExisting(context.Background(), job.ID)
	must(t, err)
	return job
}

func TestOutcomes(t *testing.T) {
	t.Run("success completes", func(t *testing.T) {
		o, _ := setup(t, noop)
		job := insertAndRun(t, o, downloadWorker)
		if job.State != "completed" || job.CompletedAt == nil || job.Attempt != 1 {
			t.Fatalf("%+v", job)
		}
	})
	t.Run("error retries with backoff and records the error", func(t *testing.T) {
		o, _ := setup(t, obanlite.WorkerFunc(func(context.Context, *obanlite.Job) error { return errors.New("boom") }))
		job := insertAndRun(t, o, downloadWorker)
		if job.State != "retryable" {
			t.Fatalf("state %s", job.State)
		}
		if d := time.Until(job.ScheduledAt.Time); d < 15*time.Second || d > 20*time.Second {
			t.Errorf("first retry should be ~17s out, got %s", d)
		}
		if !strings.Contains(string(job.Errors), "boom") || !strings.Contains(string(job.Errors), `\"attempt\":1`) {
			t.Errorf("errors %s", job.Errors)
		}
	})
	t.Run("panic is an error", func(t *testing.T) {
		o, _ := setup(t, obanlite.WorkerFunc(func(context.Context, *obanlite.Job) error { panic("kaboom") }))
		if job := insertAndRun(t, o, downloadWorker); job.State != "retryable" {
			t.Fatalf("state %s", job.State)
		}
	})
	t.Run("last attempt discards", func(t *testing.T) {
		o, d := setup(t, obanlite.WorkerFunc(func(context.Context, *obanlite.Job) error { return errors.New("boom") }))
		job, _ := o.Insert(context.Background(), nil, obanlite.NewJob("Pinchflat.Metadata.SourceMetadataStorageWorker", map[string]any{"id": 1}))
		d.Exec(`UPDATE oban_jobs SET attempt = 2 WHERE id = ?`, job.ID)
		job, _ = o.PerformExisting(context.Background(), job.ID)
		if job.State != "discarded" || job.DiscardedAt == nil {
			t.Fatalf("%+v", job)
		}
	})
}

func TestCancelJobStopsRunningWorker(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan error, 1)
	o, _ := setup(t, obanlite.WorkerFunc(func(ctx context.Context, _ *obanlite.Job) error {
		close(started)
		<-ctx.Done()
		stopped <- ctx.Err()
		return ctx.Err()
	}))
	ctx := context.Background()
	job, _ := o.Insert(ctx, nil, obanlite.NewJob(downloadWorker, map[string]any{"id": 1}))
	done := make(chan struct{})
	go func() { o.PerformExisting(ctx, job.ID); close(done) }()
	<-started
	must(t, o.CancelJob(ctx, job.ID))
	if err := <-stopped; err == nil {
		t.Fatal("worker context was not cancelled")
	}
	<-done
	job, _ = o.GetJob(ctx, job.ID)
	if job.State != "cancelled" {
		t.Fatalf("state %s", job.State)
	}
}

func TestStageAndPrune(t *testing.T) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	o := obanlite.New(d)
	ctx := context.Background()

	// Job 4 was captured as "scheduled an hour from now"; pin it far in the
	// future so the test doesn't depend on when it runs.
	d.Exec(`UPDATE oban_jobs SET scheduled_at = '2999-01-01T00:00:00.000000Z' WHERE id = 4`)
	// Make the scheduled SourceDeletionWorker job (id 10) due.
	d.Exec(`UPDATE oban_jobs SET scheduled_at = '2020-01-01T00:00:00.000000Z' WHERE id = 10`)
	must(t, o.Stage(ctx))
	if j, _ := o.GetJob(ctx, 10); j.State != "available" {
		t.Fatalf("job 10 state %s", j.State)
	}
	if j, _ := o.GetJob(ctx, 4); j.State != "scheduled" {
		t.Fatalf("future job 4 should stay scheduled, got %s", j.State)
	}

	// Age the completed job and give it a task; pruning cascades to tasks.
	d.Exec(`UPDATE oban_jobs SET scheduled_at = '2020-01-01 00:00:00' WHERE id = 9`)
	d.Exec(`INSERT INTO tasks (job_id, inserted_at, updated_at) VALUES (9, 'x', 'x')`)
	n, err := o.Prune(ctx, 30*24*time.Hour, 10000)
	if err != nil || n != 1 {
		t.Fatalf("pruned %d err %v", n, err)
	}
	var tasks int
	d.Get(&tasks, `SELECT count(*) FROM tasks WHERE job_id = 9`)
	if tasks != 0 {
		t.Fatal("tasks for pruned job should cascade-delete")
	}
}

func TestCronInsertsMatchingEntries(t *testing.T) {
	o, _ := setup(t, noop)
	crons, err := obanlite.ParseCrontab([]obanlite.CronEntry{
		{Expr: "0 1 * * *", Worker: "Pinchflat.YtDlp.UpdateWorker"},
		{Expr: "0 2 * * *", Worker: downloadWorker},
	})
	must(t, err)
	o.InsertCronJobs(context.Background(), crons, time.Date(2025, 1, 1, 1, 0, 0, 0, time.UTC))
	job := o.AssertEnqueued(t, obanlite.Match{Worker: "Pinchflat.YtDlp.UpdateWorker"})
	if string(job.Meta) != `{"cron":true,"cron_expr":"0 1 * * *","cron_tz":"Etc/UTC"}` {
		t.Errorf("meta %s", job.Meta)
	}
	o.RefuteEnqueued(t, obanlite.Match{Worker: downloadWorker})
}

func TestStartRunsElixirQueuedJobs(t *testing.T) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	o := obanlite.New(d)
	var ran atomic.Int32
	register(o, obanlite.WorkerFunc(func(_ context.Context, j *obanlite.Job) error {
		var args struct {
			ID int `json:"id"`
		}
		if err := j.DecodeArgs(&args); err != nil || args.ID == 0 {
			t.Errorf("bad args %s", j.Args)
		}
		ran.Add(1)
		return nil
	}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		o.Start(ctx, obanlite.Config{Queues: map[string]int{"media_fetching": 2}, PollInterval: 20 * time.Millisecond})
		close(done)
	}()
	deadline := time.After(5 * time.Second)
	for ran.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("only %d of 3 Elixir-queued download jobs ran", ran.Load())
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	<-done
	var completed int
	d.Get(&completed, `SELECT count(*) FROM oban_jobs WHERE worker = ? AND state = 'completed'`, downloadWorker)
	if completed != 3 {
		t.Fatalf("want 3 completed, got %d", completed)
	}
}

func TestAssertEnqueuedArgsSubset(t *testing.T) {
	o, _ := setup(t, noop)
	o.Insert(context.Background(), nil, obanlite.NewJob(downloadWorker, map[string]any{"id": 7, "force": true}))
	o.AssertEnqueued(t, obanlite.Match{Worker: downloadWorker, Args: map[string]any{"id": 7}})
	o.AssertEnqueued(t, obanlite.Match{Args: map[string]any{"force": "_"}})
	o.RefuteEnqueued(t, obanlite.Match{Args: map[string]any{"id": 8}})
}
