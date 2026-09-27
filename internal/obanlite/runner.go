package obanlite

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// CronEntry is one crontab line (Oban.Plugins.Cron). Expressions are
// evaluated in UTC, like Oban's default.
type CronEntry struct {
	Expr   string
	Worker string
	Args   any
}

// Config configures Start.
type Config struct {
	// Queues maps queue name to concurrency limit.
	Queues map[string]int
	// Crontab entries to insert on schedule.
	Crontab []CronEntry
	// PruneMaxAge: finished jobs older than this are deleted (Pruner).
	// Zero disables pruning.
	PruneMaxAge time.Duration
	// PollInterval for staging and fetching. Defaults to 1s (Oban's stager).
	PollInterval time.Duration
}

// Start runs queues, stager, pruner and cron until ctx is cancelled, then
// waits for executing jobs to finish (their contexts are cancelled too).
func (o *Oban) Start(ctx context.Context, cfg Config) error {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = time.Second
	}
	crons, err := ParseCrontab(cfg.Crontab)
	if err != nil {
		return err
	}
	for _, c := range crons {
		if _, ok := o.workers[c.Worker]; !ok {
			return fmt.Errorf("cron %q: unknown worker %s", c.Expr, c.Worker)
		}
	}

	var wg sync.WaitGroup
	for queue, limit := range cfg.Queues {
		ch := make(chan struct{}, 1)
		o.mu.Lock()
		o.pokes[queue] = ch
		o.mu.Unlock()
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.runQueue(ctx, queue, limit, cfg.PollInterval, ch)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		o.loop(ctx, cfg.PollInterval, func() {
			if err := o.Stage(ctx); err != nil && ctx.Err() == nil {
				slog.Error("oban stage", "err", err)
			}
		})
	}()

	if cfg.PruneMaxAge > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.loop(ctx, 30*time.Second, func() {
				if _, err := o.Prune(ctx, cfg.PruneMaxAge, 10_000); err != nil && ctx.Err() == nil {
					slog.Error("oban prune", "err", err)
				}
			})
		}()
	}

	if len(crons) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.runCron(ctx, crons)
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

func (o *Oban) loop(ctx context.Context, every time.Duration, fn func()) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		fn()
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Stage moves due scheduled/retryable jobs to available (stage_jobs/3).
func (o *Oban) Stage(ctx context.Context) error {
	var staged []struct {
		ID    int64  `db:"id"`
		Queue string `db:"queue"`
	}
	err := o.db.SelectContext(ctx, &staged, `SELECT id, queue FROM oban_jobs
		WHERE state IN ('scheduled', 'retryable') AND scheduled_at <= ? LIMIT 5000`, nowUsec())
	if err != nil || len(staged) == 0 {
		return err
	}
	ids := make([]any, len(staged))
	for i, s := range staged {
		ids[i] = s.ID
	}
	if _, err := o.db.ExecContext(ctx, `UPDATE oban_jobs SET state = 'available' WHERE id IN (`+placeholders(len(ids))+`)`, ids...); err != nil {
		return err
	}
	for _, s := range staged {
		o.poke(s.Queue)
	}
	return nil
}

// Prune deletes finished jobs older than maxAge (prune_jobs/3). Tasks rows
// go with them through the tasks.job_id ON DELETE CASCADE.
func (o *Oban) Prune(ctx context.Context, maxAge time.Duration, limit int) (int64, error) {
	cutoff := secondsFromNow(-int(maxAge.Seconds()))
	res, err := o.db.ExecContext(ctx, `DELETE FROM oban_jobs WHERE id IN (
		SELECT id FROM oban_jobs WHERE
			(state = 'completed' AND scheduled_at < ?) OR
			(state = 'cancelled' AND cancelled_at < ?) OR
			(state = 'discarded' AND discarded_at < ?)
		LIMIT ?)`, cutoff, cutoff, cutoff, limit)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (o *Oban) runQueue(ctx context.Context, queue string, limit int, poll time.Duration, poke chan struct{}) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, limit)
	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		demand := limit - len(sem)
		if demand > 0 {
			jobs, err := o.fetch(ctx, queue, demand)
			if err != nil && ctx.Err() == nil {
				slog.Error("oban fetch", "queue", queue, "err", err)
			}
			for _, job := range jobs {
				sem <- struct{}{}
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-sem; o.poke(queue) }()
					o.Execute(ctx, job)
				}()
			}
		}
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-t.C:
		case <-poke:
		}
	}
}

// fetch claims up to demand available jobs from queue (fetch_jobs/3).
func (o *Oban) fetch(ctx context.Context, queue string, demand int) ([]*Job, error) {
	var jobs []*Job
	err := o.db.SelectContext(ctx, &jobs, `UPDATE oban_jobs
		SET state = 'executing', attempted_at = ?, attempted_by = json_array(?), attempt = attempt + 1
		WHERE id IN (
			SELECT id FROM oban_jobs
			WHERE state = 'available' AND queue = ? AND attempt < max_attempts
			ORDER BY priority ASC, scheduled_at ASC, id ASC
			LIMIT ?)
		RETURNING `+jobColumns, nowUsec(), o.node, queue, demand)
	return jobs, err
}

// Execute runs one already-claimed job and records its outcome. Exposed for
// tests; normally called by the queue runners.
func (o *Oban) Execute(parent context.Context, job *Job) {
	reg, ok := o.workers[job.Worker]
	if !ok {
		o.recordError(job, fmt.Errorf("unknown worker %s", job.Worker), true)
		return
	}

	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	o.mu.Lock()
	o.running[job.ID] = cancel
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		delete(o.running, job.ID)
		o.mu.Unlock()
	}()

	start := time.Now()
	o.emit("start", job, nil, 0)
	err := safePerform(ctx, reg.worker, job)
	dur := time.Since(start)

	// Cancelled from outside (CancelJob or shutdown): CancelJob already
	// set the state; on shutdown leave it executing for the boot rescue.
	if ctx.Err() != nil && parent.Err() == nil {
		o.emit("stop", job, err, dur)
		return
	}
	if parent.Err() != nil {
		return
	}

	bg := context.WithoutCancel(parent)
	switch {
	case err == nil:
		_, err2 := o.db.ExecContext(bg, `UPDATE oban_jobs SET state = 'completed', completed_at = ? WHERE id = ?`, nowUsec(), job.ID)
		logIf(err2)
		o.emit("stop", job, nil, dur)
	default:
		o.recordError(job, err, job.Attempt >= job.MaxAttempts)
		o.emit("exception", job, err, dur)
	}
}

func (o *Oban) recordError(job *Job, err error, discard bool) {
	bg := context.Background()
	var err2 error
	if discard {
		_, err2 = o.db.ExecContext(bg, `UPDATE oban_jobs SET state = 'discarded', discarded_at = ?, errors = json_insert(errors, '$[#]', ?) WHERE id = ?`,
			nowUsec(), formatAttempt(job, err), job.ID)
	} else {
		_, err2 = o.db.ExecContext(bg, `UPDATE oban_jobs SET state = 'retryable', scheduled_at = ?, errors = json_insert(errors, '$[#]', ?) WHERE id = ?`,
			secondsFromNow(Backoff(job.Attempt, job.MaxAttempts)), formatAttempt(job, err), job.ID)
	}
	logIf(err2)
}

// formatAttempt mirrors Oban.Job.format_attempt/1. Like the Lite engine, the
// encoded object is appended with json_insert and a text parameter, so it is
// stored as a JSON string element.
func formatAttempt(job *Job, err error) string {
	s, _ := EncodeArgs(map[string]any{
		"attempt": job.Attempt,
		"at":      nowUsec().Format("2006-01-02T15:04:05.000000Z"),
		"error":   "** (Pinchflat.Error) " + err.Error(),
	})
	return string(s)
}

func safePerform(ctx context.Context, w Worker, job *Job) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v\n%s", p, debug.Stack())
		}
	}()
	return w.Perform(ctx, job)
}

func (o *Oban) emit(event string, job *Job, err error, dur time.Duration) {
	if o.OnEvent != nil {
		o.OnEvent(event, job, err, dur)
	}
}

func logIf(err error) {
	if err != nil {
		slog.Error("oban update", "err", err)
	}
}

// ParseCrontab parses each entry's standard 5-field cron expression.
func ParseCrontab(entries []CronEntry) ([]parsedCron, error) {
	var out []parsedCron
	for _, c := range entries {
		s, err := cron.ParseStandard(c.Expr)
		if err != nil {
			return nil, fmt.Errorf("cron %q: %w", c.Expr, err)
		}
		out = append(out, parsedCron{CronEntry: c, sched: s})
	}
	return out, nil
}

type parsedCron struct {
	CronEntry
	sched cron.Schedule
}

func (o *Oban) runCron(ctx context.Context, crons []parsedCron) {
	for {
		now := time.Now().UTC()
		next := now.Truncate(time.Minute).Add(time.Minute)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
		}
		o.InsertCronJobs(ctx, crons, next)
	}
}

// InsertCronJobs inserts the jobs whose expression matches minute.
func (o *Oban) InsertCronJobs(ctx context.Context, crons []parsedCron, minute time.Time) {
	for _, c := range crons {
		if !c.sched.Next(minute.Add(-time.Second)).Equal(minute) {
			continue
		}
		spec := NewJob(c.Worker, c.Args)
		spec.Meta = map[string]any{"cron": true, "cron_expr": c.Expr, "cron_tz": "Etc/UTC"}
		if _, err := o.Insert(ctx, nil, spec); err != nil {
			slog.Error("oban cron insert", "worker", c.Worker, "err", err)
		}
	}
}
