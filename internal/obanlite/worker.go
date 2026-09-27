package obanlite

import (
	"context"
	"math"
	"math/rand/v2"
)

// Worker performs jobs. Return nil for success (Oban's :ok), a plain error
// to retry with backoff ({:error, _} or a raise), or one of Cancel/Snooze/
// Discard for the corresponding Oban results.
type Worker interface {
	Perform(ctx context.Context, job *Job) error
}

// WorkerFunc adapts a function to Worker.
type WorkerFunc func(ctx context.Context, job *Job) error

func (f WorkerFunc) Perform(ctx context.Context, job *Job) error { return f(ctx, job) }

// Infinity is the unique period meaning "forever" (Oban's :infinity).
const Infinity = -1

// UniqueOpts mirrors Oban's `unique:` worker option.
type UniqueOpts struct {
	// Period in seconds; Infinity for forever. Oban's default is 60.
	Period int
	// States a matching job must be in. Oban's default is every state
	// except discarded and cancelled.
	States []string
	// Fields compared: any of "args", "meta", "queue", "worker".
	// Default {"args", "queue", "worker"}.
	Fields []string
	// Keys restricts the args/meta comparison to these keys (default all).
	Keys []string
	// Timestamp compared against Period: "inserted_at" (default) or "scheduled_at".
	Timestamp string
}

// WorkerOpts mirrors `use Oban.Worker, ...`.
type WorkerOpts struct {
	Queue       string // default "default"
	MaxAttempts int    // default 20
	Priority    int    // default 0
	Tags        []string
	Unique      *UniqueOpts
}

type registered struct {
	name   string
	opts   WorkerOpts
	worker Worker
}

func (o WorkerOpts) withDefaults() WorkerOpts {
	if o.Queue == "" {
		o.Queue = "default"
	}
	if o.MaxAttempts == 0 {
		o.MaxAttempts = 20
	}
	if o.Tags == nil {
		o.Tags = []string{}
	}
	if u := o.Unique; u != nil {
		c := *u
		if c.Period == 0 {
			c.Period = 60
		}
		if c.States == nil {
			c.States = []string{StateScheduled, StateAvailable, StateExecuting, StateRetryable, StateCompleted}
		}
		if c.Fields == nil {
			c.Fields = []string{"args", "queue", "worker"}
		}
		if c.Timestamp == "" {
			c.Timestamp = "inserted_at"
		}
		o.Unique = &c
	}
	return o
}

// Backoff is Oban.Worker.backoff/1: 15 + 2^attempt seconds (attempt clamped
// to 20 relative to max_attempts), plus up to 10% jitter.
func Backoff(attempt, maxAttempts int) int {
	const clampedMax = 20
	clamped := attempt
	if maxAttempts > clampedMax {
		clamped = int(math.Round(float64(attempt) / float64(maxAttempts) * clampedMax))
	}
	t := 15 + int(math.Pow(2, float64(min(clamped, 100))))
	return t + int(rand.Float64()*0.1*float64(t))
}
