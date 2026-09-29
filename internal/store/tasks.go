package store

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// ListTasks returns every task.
func (s *Store) ListTasks(ctx context.Context) ([]*Task, error) {
	return All[Task](ctx, s.Q(ctx), From[Task]())
}

// ListTasksFor returns record's tasks in jobStates (all Oban states if nil),
// optionally filtered to workerName.
func (s *Store) ListTasksFor(ctx context.Context, record any, workerName *string, jobStates []string) ([]*Task, error) {
	if jobStates == nil { // job_states \\ Oban.Job.states()
		jobStates = obanlite.AllStates
	}
	var recordType string
	var recordID int64

	switch r := record.(type) {
	case *Source:
		recordType = "source_id"
		recordID = r.ID
	case *MediaItem:
		recordType = "media_item_id"
		recordID = r.ID
	default:
		return nil, fmt.Errorf("unsupported record type")
	}

	q := TasksQueryNew().
		LeftJoin("oban_jobs AS j ON j.id = t.job_id").
		Where(sq.Eq{fmt.Sprintf("t.%s", recordType): recordID}).
		Where(sq.Eq{"j.state": jobStates})

	if workerName != nil {
		// Workers match on the string ENDING with the passed worker name
		// preceded with a dot: "%.{workerName}"
		q = q.Where(sq.Like{"j.worker": fmt.Sprintf("%%.%s", *workerName)})
	}

	return All[Task](ctx, s.Q(ctx), q)
}

// GetTaskBang returns ErrNotFound if the task doesn't exist.
func (s *Store) GetTaskBang(ctx context.Context, id int64) (*Task, error) {
	task, err := MustOne[Task](ctx, s.Q(ctx), From[Task]().Where(sq.Eq{"id": id}))
	if err != nil {
		return nil, err
	}
	return task, nil
}

// CreateTask creates a task for job and, optionally, a source or media item.
func (s *Store) CreateTask(ctx context.Context, jobID int64, sourceID, mediaItemID *int64) (*Task, error) {
	task := &Task{JobID: jobID, SourceID: sourceID, MediaItemID: mediaItemID}
	if err := Insert(ctx, s.Q(ctx), task); err != nil {
		return nil, err
	}
	return task, nil
}

// CreateTaskWithRecord creates a task attached to job and record.
func (s *Store) CreateTaskWithRecord(ctx context.Context, job *obanlite.Job, attachedRecord any) (*Task, error) {
	var sourceID, mediaItemID *int64

	switch r := attachedRecord.(type) {
	case *Source:
		sourceID = &r.ID
	case *MediaItem:
		mediaItemID = &r.ID
	default:
		return nil, fmt.Errorf("unsupported attached record type")
	}

	return s.CreateTask(ctx, job.ID, sourceID, mediaItemID)
}

// CreateJobWithTask inserts jobSpec's job (reporting a duplicate_job error
// for a duplicate unique job) and attaches a task to it and to
// taskAttachedRecord.
func (s *Store) CreateJobWithTask(ctx context.Context, jobSpec obanlite.JobSpec, taskAttachedRecord any) (*Task, error) {
	job, duplicate, err := s.InsertUniqueJob(ctx, jobSpec)
	if err != nil {
		return nil, err
	}

	if duplicate {
		return nil, fmt.Errorf("duplicate_job")
	}

	return s.CreateTaskWithRecord(ctx, job, taskAttachedRecord)
}

// DeleteTask cancels task's job and deletes the task.
func (s *Store) DeleteTask(ctx context.Context, task *Task) (*Task, error) {
	if err := s.Oban.CancelJob(ctx, task.JobID); err != nil {
		return nil, err
	}

	if err := Delete[Task](ctx, s.Q(ctx), task); err != nil {
		return nil, err
	}

	return task, nil
}

// DeleteTasksFor deletes record's tasks in jobStates (see ListTasksFor).
func (s *Store) DeleteTasksFor(ctx context.Context, record any, workerName *string, jobStates []string) error {
	tasks, err := s.ListTasksFor(ctx, record, workerName, jobStates)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if _, err := s.DeleteTask(ctx, task); err != nil {
			return err
		}
	}

	return nil
}

// DeletePendingTasksFor deletes record's pending (available/scheduled/
// retryable, plus executing when requested) tasks.
func (s *Store) DeletePendingTasksFor(ctx context.Context, record any, workerName *string, includeExecuting bool) error {
	jobStates := []string{obanlite.StateAvailable, obanlite.StateScheduled, obanlite.StateRetryable}
	if includeExecuting {
		jobStates = append(jobStates, obanlite.StateExecuting)
	}

	return s.DeleteTasksFor(ctx, record, workerName, jobStates)
}
