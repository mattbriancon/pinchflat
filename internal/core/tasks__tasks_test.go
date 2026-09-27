package core_test

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestTasks_Schema(t *testing.T) {
	t.Run("deletes a task when the job gets deleted", func(t *testing.T) {
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		// Delete the job (which should cascade to the task)
		_, err := core.Exec(ta.Ctx, ta.Q(ta.Ctx), core.SQ.Delete("oban_jobs").Where(sq.Eq{"id": task.JobID}))
		if err != nil {
			t.Fatalf("deleting job failed: %v", err)
		}

		// Verify the task was deleted (should raise an error)
		_, err = core.MustOne[core.Task](ta.Ctx, ta.Q(ta.Ctx), core.From[core.Task]().Where(sq.Eq{"id": task.ID}))
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("does not delete the other record when a job gets deleted", func(t *testing.T) {
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		// Get source before deleting job
		if task.SourceID == nil {
			t.Fatalf("task should have source_id")
		}
		source, err := core.MustOne[core.Source](ta.Ctx, ta.Q(ta.Ctx), core.From[core.Source]().Where(sq.Eq{"id": *task.SourceID}))
		if err != nil {
			t.Fatalf("getting source failed: %v", err)
		}

		// Delete the job (which should cascade to the task)
		_, err = core.Exec(ta.Ctx, ta.Q(ta.Ctx), core.SQ.Delete("oban_jobs").Where(sq.Eq{"id": task.JobID}))
		if err != nil {
			t.Fatalf("deleting job failed: %v", err)
		}

		// Verify the source still exists
		reloadedSource, err := core.MustOne[core.Source](ta.Ctx, ta.Q(ta.Ctx), core.From[core.Source]().Where(sq.Eq{"id": source.ID}))
		if err != nil {
			t.Errorf("expected source to still exist, but got error: %v", err)
		}
		if reloadedSource.ID != source.ID {
			t.Errorf("expected same source, got different one")
		}
	})
}

func TestTasks_ListTasks(t *testing.T) {
	t.Run("returns all tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		tasks, err := ta.TasksListTasks(ta.Ctx)
		if err != nil {
			t.Fatalf("TasksListTasks failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != task.ID {
			t.Errorf("expected task ID %d, got %d", task.ID, tasks[0].ID)
		}
	})
}

func TestTasks_ListTasksFor(t *testing.T) {
	t.Run("lets you specify which record type/ID to join on", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		tasks, err := ta.TasksListTasksFor(ta.Ctx, source, nil, []string{"available"})
		if err != nil {
			t.Fatalf("TasksListTasksFor failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != task.ID {
			t.Errorf("expected task ID %d, got %d", task.ID, tasks[0].ID)
		}
	})

	t.Run("lets you specify which job states to include", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		_ = coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		// Should find task with available state
		tasks, err := ta.TasksListTasksFor(ta.Ctx, source, nil, []string{"available"})
		if err != nil {
			t.Fatalf("TasksListTasksFor available failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task with available state, got %d", len(tasks))
		}

		// Should not find task with cancelled state
		tasks, err = ta.TasksListTasksFor(ta.Ctx, source, nil, []string{"cancelled"})
		if err != nil {
			t.Fatalf("TasksListTasksFor cancelled failed: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks with cancelled state, got %d", len(tasks))
		}
	})

	t.Run("lets you specify which worker to include", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		_ = coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		// Default pending states: available, scheduled, retryable
		pendingStates := []string{"available", "scheduled", "retryable"}

		// Should find task with TestJobWorker
		tasks, err := ta.TasksListTasksFor(ta.Ctx, source, core.Ptr("TestJobWorker"), pendingStates)
		if err != nil {
			t.Fatalf("TasksListTasksFor with worker failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task with TestJobWorker, got %d", len(tasks))
		}

		// Should not find task with FooBarWorker
		tasks, err = ta.TasksListTasksFor(ta.Ctx, source, core.Ptr("FooBarWorker"), pendingStates)
		if err != nil {
			t.Fatalf("TasksListTasksFor with FooBarWorker failed: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks with FooBarWorker, got %d", len(tasks))
		}
	})

	t.Run("includes all workers if no worker is specified", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		// Default pending states: available, scheduled, retryable
		pendingStates := []string{"available", "scheduled", "retryable"}

		// Should find task without specifying worker
		tasks, err := ta.TasksListTasksFor(ta.Ctx, source, nil, pendingStates)
		if err != nil {
			t.Fatalf("TasksListTasksFor without worker filter failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != task.ID {
			t.Errorf("expected task ID %d, got %d", task.ID, tasks[0].ID)
		}
	})
}

func TestTasks_GetTaskBang(t *testing.T) {
	t.Run("returns the task with given id", func(t *testing.T) {
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		retrievedTask, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err != nil {
			t.Fatalf("TasksGetTaskBang failed: %v", err)
		}
		if retrievedTask.ID != task.ID {
			t.Errorf("expected task ID %d, got %d", task.ID, retrievedTask.ID)
		}
	})
}

func TestTasks_CreateTask(t *testing.T) {
	t.Run("creation with valid data creates a task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		job := coretest.JobFixture(t, ta)
		attrs := core.Attrs{"job_id": job.ID}

		task, err := ta.TasksCreateTask(ta.Ctx, attrs)
		if err != nil {
			t.Fatalf("TasksCreateTask failed: %v", err)
		}
		if task == nil {
			t.Fatalf("expected Task, got nil")
		}
		if task.JobID != job.ID {
			t.Errorf("expected job_id %d, got %d", job.ID, task.JobID)
		}
	})

	t.Run("creation with invalid data returns error changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		invalidAttrs := core.Attrs{"job_id": nil}
		task, err := ta.TasksCreateTask(ta.Ctx, invalidAttrs)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if task != nil {
			t.Errorf("expected nil task, got %v", task)
		}
	})

	t.Run("accepts a job and source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		job := coretest.JobFixture(t, ta)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		task, err := ta.TasksCreateTaskWithRecord(ta.Ctx, job, source)
		if err != nil {
			t.Fatalf("TasksCreateTaskWithRecord failed: %v", err)
		}
		if task.JobID != job.ID {
			t.Errorf("expected job_id %d, got %d", job.ID, task.JobID)
		}
		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("expected source_id %d, got %v", source.ID, task.SourceID)
		}
	})

	t.Run("accepts a job and media item", func(t *testing.T) {
		ta := coretest.NewApp(t)
		job := coretest.JobFixture(t, ta)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		task, err := ta.TasksCreateTaskWithRecord(ta.Ctx, job, mediaItem)
		if err != nil {
			t.Fatalf("TasksCreateTaskWithRecord failed: %v", err)
		}
		if task.JobID != job.ID {
			t.Errorf("expected job_id %d, got %d", job.ID, task.JobID)
		}
		if task.MediaItemID == nil || *task.MediaItemID != mediaItem.ID {
			t.Errorf("expected media_item_id %d, got %v", mediaItem.ID, task.MediaItemID)
		}
	})
}

func TestTasks_CreateJobWithTask(t *testing.T) {
	t.Run("enqueues the given job", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{})
		task, err := ta.TasksCreateJobWithTask(ta.Ctx, spec, mediaItem)
		if err != nil {
			t.Fatalf("TasksCreateJobWithTask failed: %v", err)
		}
		if task == nil {
			t.Fatalf("expected Task, got nil")
		}

		// Verify job was enqueued
		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: coretest.TestJobWorkerName})
		if len(enqueued) != 1 {
			t.Errorf("expected 1 job enqueued, got %d", len(enqueued))
		}
	})

	t.Run("creates a task record if successful", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{})
		task, err := ta.TasksCreateJobWithTask(ta.Ctx, spec, source)
		if err != nil {
			t.Fatalf("TasksCreateJobWithTask failed: %v", err)
		}
		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("expected source_id %d, got %v", source.ID, task.SourceID)
		}
	})

	t.Run("returns an error if the job already exists", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{"foo": "bar"})
		spec.Unique = &obanlite.UniqueOpts{Period: obanlite.Infinity}

		task1, err := ta.TasksCreateJobWithTask(ta.Ctx, spec, source)
		if err != nil {
			t.Fatalf("first TasksCreateJobWithTask failed: %v", err)
		}
		if task1 == nil {
			t.Fatalf("expected Task, got nil")
		}

		_, err = ta.TasksCreateJobWithTask(ta.Ctx, spec, source)
		if err == nil {
			t.Errorf("expected error for duplicate job, got nil")
		}
		if err.Error() != "duplicate_job" {
			t.Errorf("expected error 'duplicate_job', got '%v'", err)
		}
	})

	t.Run("returns an error if the job fails to enqueue", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		// Pass an invalid JobSpec (empty worker should fail)
		spec := obanlite.JobSpec{Worker: ""}
		_, err := ta.TasksCreateJobWithTask(ta.Ctx, spec, source)
		if err == nil {
			t.Errorf("expected error for invalid job spec, got nil")
		}
	})
}

func TestTasks_DeleteTask(t *testing.T) {
	t.Run("deletion deletes the task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		_, err := ta.TasksDeleteTask(ta.Ctx, task)
		if err != nil {
			t.Fatalf("TasksDeleteTask failed: %v", err)
		}

		// Verify the task was deleted
		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletion also cancels the attached job", func(t *testing.T) {
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})
		jobID := task.JobID

		_, err := ta.TasksDeleteTask(ta.Ctx, task)
		if err != nil {
			t.Fatalf("TasksDeleteTask failed: %v", err)
		}

		// Verify the job was cancelled by querying the database
		var state string
		err = ta.Q(ta.Ctx).GetContext(ta.Ctx, &state, "SELECT state FROM oban_jobs WHERE id = ?", jobID)
		if err != nil {
			t.Fatalf("failed to query job state: %v", err)
		}
		if state != "cancelled" {
			t.Errorf("expected job state 'cancelled', got '%s'", state)
		}
	})
}

func TestTasks_DeleteTasksFor(t *testing.T) {
	t.Run("deletes tasks attached to a source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}
		err := ta.TasksDeleteTasksFor(ta.Ctx, source, nil, pendingStates)
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor failed: %v", err)
		}

		// Verify the task was deleted
		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletes the tasks attached to a media_item", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}
		err := ta.TasksDeleteTasksFor(ta.Ctx, mediaItem, nil, pendingStates)
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor failed: %v", err)
		}

		// Verify the task was deleted
		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletion can specify which worker to include", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}

		// Should not delete with FooBarWorker filter
		err := ta.TasksDeleteTasksFor(ta.Ctx, mediaItem, core.Ptr("FooBarWorker"), pendingStates)
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor with FooBarWorker filter failed: %v", err)
		}
		retrievedTask, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrievedTask == nil {
			t.Errorf("expected task to still exist after filtering by FooBarWorker")
		}

		// Should delete with TestJobWorker filter
		err = ta.TasksDeleteTasksFor(ta.Ctx, mediaItem, core.Ptr("TestJobWorker"), pendingStates)
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor with TestJobWorker filter failed: %v", err)
		}
		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletion can specify which states to include", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		// Should not delete with executing state filter
		err := ta.TasksDeleteTasksFor(ta.Ctx, source, nil, []string{"executing"})
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor with executing state filter failed: %v", err)
		}
		retrievedTask, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrievedTask == nil {
			t.Errorf("expected task to still exist after filtering by executing state")
		}

		// Should delete with available state filter
		err = ta.TasksDeleteTasksFor(ta.Ctx, source, nil, []string{"available"})
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor with available state filter failed: %v", err)
		}
		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletion does not impact unintended records", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}

		// Delete tasks for different sources
		err := ta.TasksDeleteTasksFor(ta.Ctx, coretest.SourceFixture(t, ta, core.Attrs{}), nil, pendingStates)
		if err != nil {
			t.Fatalf("first TasksDeleteTasksFor failed: %v", err)
		}

		err = ta.TasksDeleteTasksFor(ta.Ctx, coretest.SourceFixture(t, ta, core.Attrs{}), core.Ptr("FooBarWorker"), pendingStates)
		if err != nil {
			t.Fatalf("second TasksDeleteTasksFor failed: %v", err)
		}

		err = ta.TasksDeleteTasksFor(ta.Ctx, coretest.SourceFixture(t, ta, core.Attrs{}), core.Ptr("TestJobWorker"), pendingStates)
		if err != nil {
			t.Fatalf("third TasksDeleteTasksFor failed: %v", err)
		}

		// Verify the original task still exists
		retrievedTask, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err != nil || retrievedTask == nil {
			t.Errorf("expected task to still exist")
		}
	})
}

func TestTasks_DeletePendingTasksFor(t *testing.T) {
	t.Run("deletes pending tasks attached to a source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		err := ta.TasksDeletePendingTasksFor(ta.Ctx, source, nil, core.KW{})
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor failed: %v", err)
		}

		// Verify the task was deleted
		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("does not delete non-pending tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		// Cancel the job to make it non-pending
		ta.Oban.CancelJob(ta.Ctx, task.JobID)

		err := ta.TasksDeletePendingTasksFor(ta.Ctx, source, nil, core.KW{})
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor failed: %v", err)
		}

		// Verify the task still exists
		retrievedTask, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err != nil || retrievedTask == nil {
			t.Errorf("expected task to still exist")
		}
	})

	t.Run("works on media_items", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		pendingTask := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})
		cancelledTask := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})

		// Cancel one task
		ta.Oban.CancelJob(ta.Ctx, cancelledTask.JobID)

		err := ta.TasksDeletePendingTasksFor(ta.Ctx, mediaItem, nil, core.KW{})
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor failed: %v", err)
		}

		// Verify the pending task was deleted
		_, err = ta.TasksGetTaskBang(ta.Ctx, pendingTask.ID)
		if err == nil {
			t.Errorf("expected pending task to be deleted")
		}

		// Verify the cancelled task still exists
		retrievedTask, err := ta.TasksGetTaskBang(ta.Ctx, cancelledTask.ID)
		if err != nil || retrievedTask == nil {
			t.Errorf("expected cancelled task to still exist")
		}
	})

	t.Run("deletion can specify which worker to include", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})

		// Should not delete with FooBarWorker filter
		err := ta.TasksDeletePendingTasksFor(ta.Ctx, mediaItem, core.Ptr("FooBarWorker"), core.KW{})
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor with FooBarWorker filter failed: %v", err)
		}
		retrievedTask, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrievedTask == nil {
			t.Errorf("expected task to still exist after filtering by FooBarWorker")
		}

		// Should delete with TestJobWorker filter
		err = ta.TasksDeletePendingTasksFor(ta.Ctx, mediaItem, core.Ptr("TestJobWorker"), core.KW{})
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor with TestJobWorker filter failed: %v", err)
		}
		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted")
		}
	})

	t.Run("deletion can optionally include executing tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		// Set the job to executing state
		_, err := core.Exec(ta.Ctx, ta.Q(ta.Ctx), core.SQ.Update("oban_jobs").Set("state", "executing").Where(sq.Eq{"id": task.JobID}))
		if err != nil {
			t.Fatalf("failed to update job state: %v", err)
		}

		// Should not delete with include_executing=false
		err = ta.TasksDeletePendingTasksFor(ta.Ctx, source, nil, core.KW{core.Opt("include_executing", false)})
		if err != nil {
			t.Fatalf("first TasksDeletePendingTasksFor failed: %v", err)
		}
		retrievedTask, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrievedTask == nil {
			t.Errorf("expected task to still exist when include_executing=false")
		}

		// Should delete with include_executing=true
		err = ta.TasksDeletePendingTasksFor(ta.Ctx, source, nil, core.KW{core.Opt("include_executing", true)})
		if err != nil {
			t.Fatalf("second TasksDeletePendingTasksFor failed: %v", err)
		}
		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted when include_executing=true")
		}
	})
}

func TestTasks_ChangeTask(t *testing.T) {
	t.Run("returns a task changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		changeset := ta.TasksChangeTask(ta.Ctx, task, core.Attrs{})
		if changeset == nil {
			t.Fatalf("expected Changeset, got nil")
		}
	})
}
