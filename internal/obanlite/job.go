// Package obanlite is a Go port of the parts of Oban (2.19, Lite engine) that
// Pinchflat uses. It reads and writes the existing oban_jobs table with the
// same column semantics, so jobs queued by the Elixir app before cutover keep
// running, and worker names stay the Elixir module names.
package obanlite

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
)

// Job states, as stored in oban_jobs.state.
const (
	StateAvailable = "available"
	StateScheduled = "scheduled"
	StateExecuting = "executing"
	StateRetryable = "retryable"
	StateCompleted = "completed"
	StateDiscarded = "discarded"
	StateCancelled = "cancelled"
)

// AllStates lists every job state.
var AllStates = []string{StateAvailable, StateScheduled, StateExecuting, StateRetryable, StateCompleted, StateDiscarded, StateCancelled}

// Job is a row of oban_jobs.
type Job struct {
	ID          int64               `db:"id"`
	State       string              `db:"state"`
	Queue       string              `db:"queue"`
	Worker      string              `db:"worker"`
	Args        RawJSON             `db:"args"`
	Meta        RawJSON             `db:"meta"`
	Tags        db.JSON[[]string]   `db:"tags"`
	Errors      RawJSON             `db:"errors"`
	Attempt     int                 `db:"attempt"`
	MaxAttempts int                 `db:"max_attempts"`
	Priority    int                 `db:"priority"`
	InsertedAt  db.UTCDateTimeUsec  `db:"inserted_at"`
	ScheduledAt db.UTCDateTimeUsec  `db:"scheduled_at"`
	AttemptedAt *db.UTCDateTimeUsec `db:"attempted_at"`
	AttemptedBy db.JSON[[]string]   `db:"attempted_by"`
	CancelledAt *db.UTCDateTimeUsec `db:"cancelled_at"`
	CompletedAt *db.UTCDateTimeUsec `db:"completed_at"`
	DiscardedAt *db.UTCDateTimeUsec `db:"discarded_at"`

	// Conflict is true when Insert found an existing job matching the
	// worker's unique options and returned it instead of inserting.
	Conflict bool `db:"-"`
}

// jobColumns is the column list for SELECTs into Job.
const jobColumns = `id, state, queue, worker, args, meta, tags, errors, attempt, max_attempts, priority,
	inserted_at, scheduled_at, attempted_at, attempted_by, cancelled_at, completed_at, discarded_at`

// DecodeArgs unmarshals the job's args into v (a struct with json tags or a map).
func (j *Job) DecodeArgs(v any) error {
	return json.Unmarshal(j.Args, v)
}

// ArgsMap returns args as a map with numbers decoded as json.Number.
func (j *Job) ArgsMap() map[string]any {
	m := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader(j.Args))
	dec.UseNumber()
	_ = dec.Decode(&m)
	return m
}

// HasTag reports whether the job carries tag.
func (j *Job) HasTag(tag string) bool {
	for _, t := range j.Tags.V {
		if t == tag {
			return true
		}
	}
	return false
}

// RawJSON holds a JSON column verbatim.
type RawJSON json.RawMessage

func (r RawJSON) Value() (driver.Value, error) {
	if len(r) == 0 {
		return nil, nil
	}
	return string(r), nil
}

func (r *RawJSON) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*r = nil
	case string:
		*r = RawJSON(v)
	case []byte:
		*r = append(RawJSON(nil), v...)
	default:
		return fmt.Errorf("cannot scan %T into RawJSON", src)
	}
	return nil
}

func (r RawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

func (r *RawJSON) UnmarshalJSON(b []byte) error {
	*r = append(RawJSON(nil), b...)
	return nil
}

// EncodeArgs normalises args the way Jason encodes an Elixir map: compact,
// keys sorted, no HTML escaping. v may be a struct, a map, or nil.
func EncodeArgs(v any) (RawJSON, error) {
	if v == nil {
		return RawJSON("{}"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("job args must encode to a JSON object: %w", err)
	}
	if m == nil { // nil map or JSON null
		m = map[string]any{}
	}
	s, err := db.EncodeJSON(m)
	if err != nil {
		return nil, err
	}
	return RawJSON(s), nil
}

func nowUsec() db.UTCDateTimeUsec { return db.NowUsec() }

func secondsFromNow(s int) db.UTCDateTimeUsec {
	return db.UTCDateTimeUsec{Time: time.Now().UTC().Add(time.Duration(s) * time.Second).Truncate(time.Microsecond)}
}
