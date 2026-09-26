package obanlite

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Test helpers mirroring Oban.Testing in :manual mode: jobs are inserted
// but no queues run, and tests assert on what was enqueued.

// Match selects enqueued jobs. Zero-valued fields are ignored.
type Match struct {
	Worker string
	Queue  string
	// Args must be contained in the job's args (Oban.Testing semantics:
	// every given key equal, nested maps compared recursively, "_" matches
	// any value).
	Args map[string]any
	// ScheduledAt matches within ScheduledDelta (default 1s).
	ScheduledAt    *time.Time
	ScheduledDelta time.Duration
	Priority       *int
	Tags           []string
}

// Enqueued returns jobs in the available or scheduled state matching m,
// newest first.
func (o *Oban) Enqueued(t testing.TB, m Match) []*Job {
	t.Helper()
	jobs, err := o.Jobs(context.Background(), `state IN ('available', 'scheduled')`)
	if err != nil {
		t.Fatalf("list enqueued jobs: %v", err)
	}
	var out []*Job
	for _, j := range jobs {
		if m.matches(j) {
			out = append(out, j)
		}
	}
	return out
}

// AssertEnqueued fails unless a matching job is enqueued, returning it.
func (o *Oban) AssertEnqueued(t testing.TB, m Match) *Job {
	t.Helper()
	jobs := o.Enqueued(t, m)
	if len(jobs) == 0 {
		all, _ := o.Jobs(context.Background(), `state IN ('available', 'scheduled')`)
		var desc []string
		for _, j := range all {
			desc = append(desc, fmt.Sprintf("%s %s %s", j.Worker, j.State, j.Args))
		}
		t.Fatalf("expected a job matching %+v to be enqueued; enqueued jobs:\n  %s", m, strings.Join(desc, "\n  "))
	}
	return jobs[0]
}

// RefuteEnqueued fails if a matching job is enqueued.
func (o *Oban) RefuteEnqueued(t testing.TB, m Match) {
	t.Helper()
	if jobs := o.Enqueued(t, m); len(jobs) > 0 {
		t.Fatalf("expected no job matching %+v to be enqueued, found %d", m, len(jobs))
	}
}

// PerformJob runs a worker directly with args without inserting a job
// (Oban.Testing.perform_job/2). The job's Attempt is 1.
func (o *Oban) PerformJob(ctx context.Context, worker string, args any) error {
	reg, ok := o.workers[worker]
	if !ok {
		return fmt.Errorf("obanlite: unknown worker %q", worker)
	}
	raw, err := EncodeArgs(args)
	if err != nil {
		return err
	}
	job := &Job{
		Worker: worker, Queue: reg.opts.Queue, Args: raw, Attempt: 1,
		MaxAttempts: reg.opts.MaxAttempts, Priority: reg.opts.Priority, State: StateExecuting,
	}
	job.Tags.V = reg.opts.Tags
	return safePerform(ctx, reg.worker, job)
}

// PerformExisting runs an inserted job through the full execution path
// (claim, perform, record outcome), for tests of retry behaviour.
func (o *Oban) PerformExisting(ctx context.Context, id int64) (*Job, error) {
	var job Job
	err := o.db.GetContext(ctx, &job, `UPDATE oban_jobs SET state = 'executing', attempted_at = ?, attempt = attempt + 1
		WHERE id = ? RETURNING `+jobColumns, nowUsec(), id)
	if err != nil {
		return nil, err
	}
	o.Execute(ctx, &job)
	return o.GetJob(ctx, id)
}

func (m Match) matches(j *Job) bool {
	if m.Worker != "" && j.Worker != m.Worker {
		return false
	}
	if m.Queue != "" && j.Queue != m.Queue {
		return false
	}
	if m.Priority != nil && j.Priority != *m.Priority {
		return false
	}
	for _, tag := range m.Tags {
		if !j.HasTag(tag) {
			return false
		}
	}
	if m.ScheduledAt != nil {
		delta := m.ScheduledDelta
		if delta == 0 {
			delta = time.Second
		}
		d := j.ScheduledAt.Sub(*m.ScheduledAt)
		if d < -delta || d > delta {
			return false
		}
	}
	if m.Args != nil {
		want := normalize(m.Args)
		return contains(want, normalize(j.ArgsMap()))
	}
	return true
}

// normalize round-trips through JSON so numbers compare equal regardless of
// Go type.
func normalize(v any) any {
	b, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}

func contains(want, got any) bool {
	wm, ok := want.(map[string]any)
	if !ok {
		return reflect.DeepEqual(want, got)
	}
	gm, ok := got.(map[string]any)
	if !ok {
		return false
	}
	for k, wv := range wm {
		gv, present := gm[k]
		if !present {
			return false
		}
		if wv == "_" {
			continue
		}
		if !contains(wv, gv) {
			return false
		}
	}
	return true
}
