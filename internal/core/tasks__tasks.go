package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// TasksListTasks/0
func (a *App) TasksListTasks(ctx context.Context) ([]*Task, error) {
	panic("unported: Pinchflat.Tasks.list_tasks/0")
}

// TasksListTasksFor/3
func (a *App) TasksListTasksFor(ctx context.Context, record any, workerName *string, jobStates []string) ([]*Task, error) {
	panic("unported: Pinchflat.Tasks.list_tasks_for/3")
}

// TasksGetTaskBang/1
func (a *App) TasksGetTaskBang(ctx context.Context, id int64) (*Task, error) {
	panic("unported: Pinchflat.Tasks.get_task!/1")
}

// TasksCreateTask/1 (with map)
func (a *App) TasksCreateTask(ctx context.Context, attrs Attrs) (*Task, error) {
	panic("unported: Pinchflat.Tasks.create_task/1")
}

// TasksCreateTask/2 (with job and record)
func (a *App) TasksCreateTaskWithRecord(ctx context.Context, job *obanlite.Job, attachedRecord any) (*Task, error) {
	panic("unported: Pinchflat.Tasks.create_task/2")
}

// TasksCreateJobWithTask/2
func (a *App) TasksCreateJobWithTask(ctx context.Context, jobAttrs KW, taskAttachedRecord any) (*Task, error) {
	panic("unported: Pinchflat.Tasks.create_job_with_task/2")
}

// TasksDeleteTask/1
func (a *App) TasksDeleteTask(ctx context.Context, task *Task) (*Task, error) {
	panic("unported: Pinchflat.Tasks.delete_task/1")
}

// TasksDeleteTasksFor/3
func (a *App) TasksDeleteTasksFor(ctx context.Context, record any, workerName *string, jobStates []string) error {
	panic("unported: Pinchflat.Tasks.delete_tasks_for/3")
}

// TasksDeletePendingTasksFor/3
func (a *App) TasksDeletePendingTasksFor(ctx context.Context, record any, workerName *string, opts KW) error {
	panic("unported: Pinchflat.Tasks.delete_pending_tasks_for/3")
}

// TasksChangeTask/2
func (a *App) TasksChangeTask(ctx context.Context, task *Task, attrs Attrs) *Changeset {
	panic("unported: Pinchflat.Tasks.change_task/2")
}
