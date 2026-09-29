package store

import (
	"net/url"
	"strconv"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

type Task struct {
	ID          int64          `db:"id"`
	JobID       int64          `db:"job_id"`
	SourceID    *int64         `db:"source_id"`
	MediaItemID *int64         `db:"media_item_id"`
	InsertedAt  db.UTCDateTime `db:"inserted_at"`
	UpdatedAt   db.UTCDateTime `db:"updated_at"`

	// Associations
	Job       *obanlite.Job `db:"-"`
	Source    *Source       `db:"-"`
	MediaItem *MediaItem    `db:"-"`
}

func (Task) TableName() string { return "tasks" }

func NewTask() *Task {
	return &Task{}
}

var taskAllowedFields = []string{
	"job_id",
	"source_id",
	"media_item_id",
}

var taskRequiredFields = []string{
	"job_id",
}

// TaskParams are the typed parameters for creating a Task.
// Nil pointer fields mean the value was not submitted.
type TaskParams struct {
	JobID       *int64
	SourceID    *int64
	MediaItemID *int64
}

// ParseTaskParams parses form values into TaskParams, returning validation errors.
// This is primarily used for web forms, though tasks are typically created internally.
func ParseTaskParams(values url.Values) (TaskParams, map[string][]string) {
	errs := make(map[string][]string)
	p := TaskParams{}

	if v := values.Get("job_id"); v != "" {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			errs["job_id"] = append(errs["job_id"], "is not a valid integer")
		} else {
			p.JobID = &i
		}
	}

	if v := values.Get("source_id"); v != "" {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			errs["source_id"] = append(errs["source_id"], "is not a valid integer")
		} else {
			p.SourceID = &i
		}
	}

	if v := values.Get("media_item_id"); v != "" {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			errs["media_item_id"] = append(errs["media_item_id"], "is not a valid integer")
		} else {
			p.MediaItemID = &i
		}
	}

	return p, errs
}

// Validate validates the TaskParams and returns a map of field errors.
func (p TaskParams) Validate() map[string][]string {
	errs := make(map[string][]string)

	// Check required fields: job_id
	if p.JobID == nil {
		errs["job_id"] = append(errs["job_id"], "can't be blank")
	}

	return errs
}

// TaskChangeset/2 (kept for backwards compatibility until final cleanup agent removes it)
func TaskChangeset(task *Task, attrs Attrs) *Changeset {
	return Cast(task, attrs, taskAllowedFields).
		ValidateRequired(taskRequiredFields...)
}
