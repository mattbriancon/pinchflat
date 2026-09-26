package core

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// TasksListTasks/0
func (a *App) TasksListTasks(ctx context.Context) ([]*Task, error) {
	return All[Task](ctx, a.Q(ctx), From[Task]())
}

// TasksListTasksFor/3
func (a *App) TasksListTasksFor(ctx context.Context, record any, workerName *string, jobStates []string) ([]*Task, error) {
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

	return All[Task](ctx, a.Q(ctx), q)
}

// TasksGetTaskBang/1
func (a *App) TasksGetTaskBang(ctx context.Context, id int64) (*Task, error) {
	task, err := MustOne[Task](ctx, a.Q(ctx), From[Task]().Where(sq.Eq{"id": id}))
	if err != nil {
		return nil, err
	}
	return task, nil
}

// TasksCreateTask/1 (with map)
func (a *App) TasksCreateTask(ctx context.Context, attrs Attrs) (*Task, error) {
	task := NewTask()
	cs := TaskChangeset(task, attrs)
	t, err := Insert[Task](ctx, a.Q(ctx), cs)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// TasksCreateTaskWithRecord/2 (with job and record)
func (a *App) TasksCreateTaskWithRecord(ctx context.Context, job *obanlite.Job, attachedRecord any) (*Task, error) {
	var attachedRecordAttr Attrs

	switch r := attachedRecord.(type) {
	case *Source:
		attachedRecordAttr = Attrs{"source_id": r.ID}
	case *MediaItem:
		attachedRecordAttr = Attrs{"media_item_id": r.ID}
	default:
		return nil, fmt.Errorf("unsupported attached record type")
	}

	attrs := Attrs{"job_id": job.ID}
	for k, v := range attachedRecordAttr {
		attrs[k] = v
	}

	return a.TasksCreateTask(ctx, attrs)
}

// TasksCreateJobWithTask/2
func (a *App) TasksCreateJobWithTask(ctx context.Context, jobSpec obanlite.JobSpec, taskAttachedRecord any) (*Task, error) {
	job, duplicate, err := a.InsertUniqueJob(ctx, jobSpec)
	if err != nil {
		return nil, err
	}

	if duplicate {
		return nil, fmt.Errorf("duplicate_job")
	}

	return a.TasksCreateTaskWithRecord(ctx, job, taskAttachedRecord)
}

// TasksDeleteTask/1
func (a *App) TasksDeleteTask(ctx context.Context, task *Task) (*Task, error) {
	if err := a.Oban.CancelJob(ctx, task.JobID); err != nil {
		return nil, err
	}

	if err := Delete[Task](ctx, a.Q(ctx), task); err != nil {
		return nil, err
	}

	return task, nil
}

// TasksDeleteTasksFor/3
func (a *App) TasksDeleteTasksFor(ctx context.Context, record any, workerName *string, jobStates []string) error {
	tasks, err := a.TasksListTasksFor(ctx, record, workerName, jobStates)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if _, err := a.TasksDeleteTask(ctx, task); err != nil {
			return err
		}
	}

	return nil
}

// TasksDeletePendingTasksFor/3
func (a *App) TasksDeletePendingTasksFor(ctx context.Context, record any, workerName *string, opts KW) error {
	includeExecuting := opts.Bool("include_executing")

	jobStates := []string{obanlite.StateAvailable, obanlite.StateScheduled, obanlite.StateRetryable}
	if includeExecuting {
		jobStates = append(jobStates, obanlite.StateExecuting)
	}

	return a.TasksDeleteTasksFor(ctx, record, workerName, jobStates)
}

// TasksChangeTask/2
func (a *App) TasksChangeTask(ctx context.Context, task *Task, attrs Attrs) *Changeset {
	return TaskChangeset(task, attrs)
}
