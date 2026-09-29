package obanlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
)

// Oban is the job system: a worker registry plus, once started, queue
// runners, stager, pruner and cron.
type Oban struct {
	db      *db.DB
	node    string
	workers map[string]*registered

	mu      sync.Mutex
	running map[int64]context.CancelFunc
	pokes   map[string]chan struct{}

	// OnEvent, if set, is called for job lifecycle events ("start",
	// "stop", "exception"). Used for metrics.
	OnEvent func(event string, job *Job, err error, dur time.Duration)
}

// New creates an Oban bound to d. Register workers before Start.
func New(d *db.DB) *Oban {
	node := os.Getenv("DYNO")
	if node == "" {
		node, _ = os.Hostname()
	}
	return &Oban{
		db:      d,
		node:    node,
		workers: map[string]*registered{},
		running: map[int64]context.CancelFunc{},
		pokes:   map[string]chan struct{}{},
	}
}

// DB returns the database the Oban instance uses.
func (o *Oban) DB() *db.DB { return o.db }

// Register adds a worker under name, which must be the Elixir module name
// (e.g. "Pinchflat.Downloading.MediaDownloadWorker") so existing rows run.
func (o *Oban) Register(name string, opts WorkerOpts, w Worker) {
	o.workers[name] = &registered{name: name, opts: opts.withDefaults(), worker: w}
}

// WorkerOptsFor returns the registered options for a worker name.
func (o *Oban) WorkerOptsFor(name string) (WorkerOpts, bool) {
	r, ok := o.workers[name]
	if !ok {
		return WorkerOpts{}, false
	}
	return r.opts, true
}

// JobSpec describes a job to insert (Oban's Worker.new/2).
type JobSpec struct {
	Worker string
	Args   any
	// Optional overrides of the worker's defaults.
	Queue       string
	Priority    *int
	MaxAttempts int
	Tags        []string
	Meta        map[string]any
	// ScheduleIn (seconds) or ScheduledAt make the job "scheduled".
	ScheduleIn  int
	ScheduledAt *time.Time
	// Unique overrides the worker's unique options.
	Unique *UniqueOpts
}

// NewJob builds a JobSpec for worker with args.
func NewJob(worker string, args any) JobSpec { return JobSpec{Worker: worker, Args: args} }

// Insert inserts a job, honouring unique options. When an existing job
// matches, it is returned with Conflict set and nothing is inserted (the
// engine's insert_job/3). Pass a *db.Tx as q to insert inside a transaction,
// or nil to use the Oban database.
func (o *Oban) Insert(ctx context.Context, q db.Querier, spec JobSpec) (*Job, error) {
	if q == nil {
		q = o.db
	}
	reg, ok := o.workers[spec.Worker]
	if !ok {
		return nil, fmt.Errorf("obanlite: unknown worker %q", spec.Worker)
	}
	opts := reg.opts
	if spec.Unique != nil {
		opts.Unique = WorkerOpts{Unique: spec.Unique}.withDefaults().Unique
	}

	args, err := EncodeArgs(spec.Args)
	if err != nil {
		return nil, err
	}
	meta, err := EncodeArgs(spec.Meta)
	if err != nil {
		return nil, err
	}
	queue := opts.Queue
	if spec.Queue != "" {
		queue = spec.Queue
	}
	priority := opts.Priority
	if spec.Priority != nil {
		priority = *spec.Priority
	}
	maxAttempts := opts.MaxAttempts
	if spec.MaxAttempts != 0 {
		maxAttempts = spec.MaxAttempts
	}
	tags := opts.Tags
	if spec.Tags != nil {
		tags = spec.Tags
	}

	if u := opts.Unique; u != nil {
		existing, err := o.fetchUnique(ctx, q, u, spec.Worker, queue, args, meta)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			existing.Conflict = true
			return existing, nil
		}
	}

	// Like Ecto, only non-default columns are sent; inserted_at and (for
	// immediate jobs) scheduled_at come from the column defaults
	// (CURRENT_TIMESTAMP), exactly as rows written by Elixir.
	cols := []string{"state", "queue", "worker", "args", "meta", "tags", "max_attempts", "priority"}
	state := StateAvailable
	var scheduledAt *db.UTCDateTimeUsec
	switch {
	case spec.ScheduledAt != nil:
		t := db.UTCDateTimeUsec{Time: spec.ScheduledAt.UTC().Truncate(time.Microsecond)}
		scheduledAt, state = &t, StateScheduled
	case spec.ScheduleIn > 0:
		t := secondsFromNow(spec.ScheduleIn)
		scheduledAt, state = &t, StateScheduled
	}
	vals := []any{state, queue, spec.Worker, args, meta, db.NewJSON(tags), maxAttempts, priority}
	if scheduledAt != nil {
		cols = append(cols, "scheduled_at")
		vals = append(vals, *scheduledAt)
	}
	query := fmt.Sprintf(`INSERT INTO oban_jobs (%s) VALUES (%s) RETURNING %s`,
		strings.Join(cols, ", "), strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", "), JobColumns)

	job := &Job{}
	if err := q.QueryRowxContext(ctx, query, vals...).StructScan(job); err != nil {
		return nil, fmt.Errorf("insert job: %w", err)
	}
	if state == StateAvailable {
		o.poke(queue)
	}
	return job, nil
}

// fetchUnique ports Oban.Engines.Lite.fetch_unique/2.
func (o *Oban) fetchUnique(ctx context.Context, q db.Querier, u *UniqueOpts, worker, queue string, args, meta RawJSON) (*Job, error) {
	period := u.Period
	if period == Infinity || period > 60*60*24*365*99 {
		period = 60 * 60 * 24 * 365 * 99
	}
	since := secondsFromNow(-period)

	where := []string{
		`state IN (` + placeholders(len(u.States)) + `)`,
		fmt.Sprintf(`datetime(%s) >= datetime(?)`, u.Timestamp),
	}
	params := []any{}
	for _, s := range u.States {
		params = append(params, s)
	}
	params = append(params, since)

	for _, f := range u.Fields {
		switch f {
		case "args", "meta":
			raw := args
			if f == "meta" {
				raw = meta
			}
			value, err := restrictKeys(raw, u.Keys)
			if err != nil {
				return nil, err
			}
			switch {
			case value == "{}":
				where = append(where, f+" = ?")
				params = append(params, value)
			case len(u.Keys) == 0:
				where = append(where, jsonContains(f, "?")+" AND "+jsonContains("?", f))
				params = append(params, value, value)
			default:
				where = append(where, jsonContains(f, "?"))
				params = append(params, value)
			}
		case "queue":
			where = append(where, "queue = ?")
			params = append(params, queue)
		case "worker":
			where = append(where, "worker = ?")
			params = append(params, worker)
		}
	}

	job := &Job{}
	err := q.QueryRowxContext(ctx, `SELECT `+JobColumns+` FROM oban_jobs WHERE `+strings.Join(where, " AND ")+` LIMIT 1`, params...).StructScan(job)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("unique check: %w", err)
	}
	return job, nil
}

// jsonContains is Oban Lite's json_contains macro: every key of object has
// an equal value in column.
func jsonContains(column, object string) string {
	return fmt.Sprintf(`(SELECT 0 NOT IN (SELECT json_extract(%s, '$.' || t.key) = t.value FROM json_each(%s) t))`, column, object)
}

func restrictKeys(raw RawJSON, keys []string) (string, error) {
	if len(keys) == 0 {
		return string(raw), nil
	}
	m := (&Job{Args: raw}).ArgsMap()
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return db.EncodeJSON(out)
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

// CancelJob cancels a job (Oban.cancel_job/1). A job executing in this
// process has its context cancelled, which kills any command it's running.
func (o *Oban) CancelJob(ctx context.Context, id int64) error {
	_, err := o.db.ExecContext(ctx, `UPDATE oban_jobs SET state = 'cancelled', cancelled_at = ?
		WHERE id = ? AND state NOT IN ('cancelled', 'completed', 'discarded')`, nowUsec(), id)
	if err != nil {
		return err
	}
	o.mu.Lock()
	cancel := o.running[id]
	o.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// GetJob loads a job by id.
func (o *Oban) GetJob(ctx context.Context, id int64) (*Job, error) {
	job := &Job{}
	if err := o.db.GetContext(ctx, job, `SELECT `+JobColumns+` FROM oban_jobs WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return job, nil
}

// Jobs returns jobs matching an optional SQL condition, newest first.
func (o *Oban) Jobs(ctx context.Context, where string, args ...any) ([]*Job, error) {
	q := `SELECT ` + JobColumns + ` FROM oban_jobs`
	if where != "" {
		q += ` WHERE ` + where
	}
	q += ` ORDER BY id DESC`
	var jobs []*Job
	err := o.db.SelectContext(ctx, &jobs, q, args...)
	return jobs, err
}

// QueueNames returns the queues of all registered workers, sorted.
func (o *Oban) QueueNames() []string {
	set := map[string]bool{}
	for _, r := range o.workers {
		set[r.opts.Queue] = true
	}
	var out []string
	for q := range set {
		out = append(out, q)
	}
	sort.Strings(out)
	return out
}

func (o *Oban) poke(queue string) {
	o.mu.Lock()
	ch := o.pokes[queue]
	o.mu.Unlock()
	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
