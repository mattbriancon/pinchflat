package core

import (
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
