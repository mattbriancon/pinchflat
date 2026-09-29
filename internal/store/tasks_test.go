package store_test

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/store/storetest"
)

func TestTasks_Schema(t *testing.T) {
	t.Run("deletes a task when the job gets deleted", func(t *testing.T) {
		ts := storetest.NewStore(t)
		task := storetest.TaskFixture(t, ts, store.Attrs{})

		// Delete the job (which should cascade to the task)
		_, err := store.Exec(ts.Ctx, ts.Q(ts.Ctx), store.SQ.Delete("oban_jobs").Where(sq.Eq{"id": task.JobID}))
		if err != nil {
			t.Fatalf("deleting job failed: %v", err)
		}

		// Verify the task was deleted (should raise an error)
		_, err = store.MustOne[store.Task](ts.Ctx, ts.Q(ts.Ctx), store.From[store.Task]().Where(sq.Eq{"id": task.ID}))
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("does not delete the other record when a job gets deleted", func(t *testing.T) {
		ts := storetest.NewStore(t)
		task := storetest.TaskFixture(t, ts, store.Attrs{})

		// Get source before deleting job
		if task.SourceID == nil {
			t.Fatalf("task should have source_id")
		}
		source, err := store.MustOne[store.Source](ts.Ctx, ts.Q(ts.Ctx), store.From[store.Source]().Where(sq.Eq{"id": *task.SourceID}))
		if err != nil {
			t.Fatalf("getting source failed: %v", err)
		}

		// Delete the job (which should cascade to the task)
		_, err = store.Exec(ts.Ctx, ts.Q(ts.Ctx), store.SQ.Delete("oban_jobs").Where(sq.Eq{"id": task.JobID}))
		if err != nil {
			t.Fatalf("deleting job failed: %v", err)
		}

		// Verify the source still exists
		reloadedSource, err := store.MustOne[store.Source](ts.Ctx, ts.Q(ts.Ctx), store.From[store.Source]().Where(sq.Eq{"id": source.ID}))
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
		ts := storetest.NewStore(t)
		task := storetest.TaskFixture(t, ts, store.Attrs{})

		tasks, err := ts.ListTasks(ts.Ctx)
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
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		tasks, err := ts.ListTasksFor(ts.Ctx, source, nil, []string{"available"})
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
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		_ = storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		// Should find task with available state
		tasks, err := ts.ListTasksFor(ts.Ctx, source, nil, []string{"available"})
		if err != nil {
			t.Fatalf("TasksListTasksFor available failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task with available state, got %d", len(tasks))
		}

		// Should not find task with cancelled state
		tasks, err = ts.ListTasksFor(ts.Ctx, source, nil, []string{"cancelled"})
		if err != nil {
			t.Fatalf("TasksListTasksFor cancelled failed: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks with cancelled state, got %d", len(tasks))
		}
	})

	t.Run("lets you specify which worker to include", func(t *testing.T) {
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		_ = storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		// Default pending states: available, scheduled, retryable
		pendingStates := []string{"available", "scheduled", "retryable"}

		// Should find task with TestJobWorker
		tasks, err := ts.ListTasksFor(ts.Ctx, source, store.Ptr("TestJobWorker"), pendingStates)
		if err != nil {
			t.Fatalf("TasksListTasksFor with worker failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task with TestJobWorker, got %d", len(tasks))
		}

		// Should not find task with FooBarWorker
		tasks, err = ts.ListTasksFor(ts.Ctx, source, store.Ptr("FooBarWorker"), pendingStates)
		if err != nil {
			t.Fatalf("TasksListTasksFor with FooBarWorker failed: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks with FooBarWorker, got %d", len(tasks))
		}
	})

	t.Run("includes all workers if no worker is specified", func(t *testing.T) {
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		// Default pending states: available, scheduled, retryable
		pendingStates := []string{"available", "scheduled", "retryable"}

		// Should find task without specifying worker
		tasks, err := ts.ListTasksFor(ts.Ctx, source, nil, pendingStates)
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
		ts := storetest.NewStore(t)
		task := storetest.TaskFixture(t, ts, store.Attrs{})

		retrievedTask, err := ts.GetTaskBang(ts.Ctx, task.ID)
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
		ts := storetest.NewStore(t)
		job := storetest.JobFixture(t, ts)

		task, err := ts.CreateTask(ts.Ctx, job.ID, nil, nil)
		if err != nil {
			t.Fatalf("TasksCreateTask failed: %v", err)
		}
		if task == nil {
			t.Fatalf("expected store.Task, got nil")
		}
		if task.JobID != job.ID {
			t.Errorf("expected job_id %d, got %d", job.ID, task.JobID)
		}
	})

	t.Run("creation with valid source_id creates a task", func(t *testing.T) {
		ts := storetest.NewStore(t)
		job := storetest.JobFixture(t, ts)
		source := storetest.SourceFixture(t, ts, store.Attrs{})

		task, err := ts.CreateTask(ts.Ctx, job.ID, &source.ID, nil)
		if err != nil {
			t.Fatalf("TasksCreateTask failed: %v", err)
		}
		if task == nil {
			t.Fatalf("expected store.Task, got nil")
		}
		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("expected source_id %d, got %v", source.ID, task.SourceID)
		}
	})

	t.Run("accepts a job and source", func(t *testing.T) {
		ts := storetest.NewStore(t)
		job := storetest.JobFixture(t, ts)
		source := storetest.SourceFixture(t, ts, store.Attrs{})

		task, err := ts.CreateTaskWithRecord(ts.Ctx, job, source)
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
		ts := storetest.NewStore(t)
		job := storetest.JobFixture(t, ts)
		mediaItem := storetest.MediaItemFixture(t, ts, store.MediaItemParams{})

		task, err := ts.CreateTaskWithRecord(ts.Ctx, job, mediaItem)
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
		ts := storetest.NewStore(t)
		ts.Oban.Register(storetest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, storetest.TestJobWorker{})
		mediaItem := storetest.MediaItemFixture(t, ts, store.MediaItemParams{})

		spec := obanlite.NewJob(storetest.TestJobWorkerName, map[string]any{})
		task, err := ts.CreateJobWithTask(ts.Ctx, spec, mediaItem)
		if err != nil {
			t.Fatalf("TasksCreateJobWithTask failed: %v", err)
		}
		if task == nil {
			t.Fatalf("expected store.Task, got nil")
		}

		// Verify job was enqueued
		enqueued := ts.Oban.Enqueued(t, obanlite.Match{Worker: storetest.TestJobWorkerName})
		if len(enqueued) != 1 {
			t.Errorf("expected 1 job enqueued, got %d", len(enqueued))
		}
	})

	t.Run("creates a task record if successful", func(t *testing.T) {
		ts := storetest.NewStore(t)
		ts.Oban.Register(storetest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, storetest.TestJobWorker{})
		source := storetest.SourceFixture(t, ts, store.Attrs{})

		spec := obanlite.NewJob(storetest.TestJobWorkerName, map[string]any{})
		task, err := ts.CreateJobWithTask(ts.Ctx, spec, source)
		if err != nil {
			t.Fatalf("TasksCreateJobWithTask failed: %v", err)
		}
		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("expected source_id %d, got %v", source.ID, task.SourceID)
		}
	})

	t.Run("returns an error if the job already exists", func(t *testing.T) {
		ts := storetest.NewStore(t)
		ts.Oban.Register(storetest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, storetest.TestJobWorker{})
		source := storetest.SourceFixture(t, ts, store.Attrs{})

		spec := obanlite.NewJob(storetest.TestJobWorkerName, map[string]any{"foo": "bar"})
		spec.Unique = &obanlite.UniqueOpts{Period: obanlite.Infinity}

		task1, err := ts.CreateJobWithTask(ts.Ctx, spec, source)
		if err != nil {
			t.Fatalf("first TasksCreateJobWithTask failed: %v", err)
		}
		if task1 == nil {
			t.Fatalf("expected store.Task, got nil")
		}

		_, err = ts.CreateJobWithTask(ts.Ctx, spec, source)
		if err == nil {
			t.Errorf("expected error for duplicate job, got nil")
		}
		if err.Error() != "duplicate_job" {
			t.Errorf("expected error 'duplicate_job', got '%v'", err)
		}
	})

	t.Run("returns an error if the job fails to enqueue", func(t *testing.T) {
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})

		// Pass an invalid JobSpec (empty worker should fail)
		spec := obanlite.JobSpec{Worker: ""}
		_, err := ts.CreateJobWithTask(ts.Ctx, spec, source)
		if err == nil {
			t.Errorf("expected error for invalid job spec, got nil")
		}
	})
}

func TestTasks_DeleteTask(t *testing.T) {
	t.Run("deletion deletes the task", func(t *testing.T) {
		ts := storetest.NewStore(t)
		task := storetest.TaskFixture(t, ts, store.Attrs{})

		_, err := ts.DeleteTask(ts.Ctx, task)
		if err != nil {
			t.Fatalf("TasksDeleteTask failed: %v", err)
		}

		// Verify the task was deleted
		_, err = ts.GetTaskBang(ts.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletion also cancels the attached job", func(t *testing.T) {
		ts := storetest.NewStore(t)
		task := storetest.TaskFixture(t, ts, store.Attrs{})
		jobID := task.JobID

		_, err := ts.DeleteTask(ts.Ctx, task)
		if err != nil {
			t.Fatalf("TasksDeleteTask failed: %v", err)
		}

		// Verify the job was cancelled by querying the database
		var state string
		err = ts.Q(ts.Ctx).GetContext(ts.Ctx, &state, "SELECT state FROM oban_jobs WHERE id = ?", jobID)
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
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}
		err := ts.DeleteTasksFor(ts.Ctx, source, nil, pendingStates)
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor failed: %v", err)
		}

		// Verify the task was deleted
		_, err = ts.GetTaskBang(ts.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletes the tasks attached to a media_item", func(t *testing.T) {
		ts := storetest.NewStore(t)
		mediaItem := storetest.MediaItemFixture(t, ts, store.MediaItemParams{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"media_item_id": mediaItem.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}
		err := ts.DeleteTasksFor(ts.Ctx, mediaItem, nil, pendingStates)
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor failed: %v", err)
		}

		// Verify the task was deleted
		_, err = ts.GetTaskBang(ts.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletion can specify which worker to include", func(t *testing.T) {
		ts := storetest.NewStore(t)
		mediaItem := storetest.MediaItemFixture(t, ts, store.MediaItemParams{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"media_item_id": mediaItem.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}

		// Should not delete with FooBarWorker filter
		err := ts.DeleteTasksFor(ts.Ctx, mediaItem, store.Ptr("FooBarWorker"), pendingStates)
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor with FooBarWorker filter failed: %v", err)
		}
		retrievedTask, _ := ts.GetTaskBang(ts.Ctx, task.ID)
		if retrievedTask == nil {
			t.Errorf("expected task to still exist after filtering by FooBarWorker")
		}

		// Should delete with TestJobWorker filter
		err = ts.DeleteTasksFor(ts.Ctx, mediaItem, store.Ptr("TestJobWorker"), pendingStates)
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor with TestJobWorker filter failed: %v", err)
		}
		_, err = ts.GetTaskBang(ts.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletion can specify which states to include", func(t *testing.T) {
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		// Should not delete with executing state filter
		err := ts.DeleteTasksFor(ts.Ctx, source, nil, []string{"executing"})
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor with executing state filter failed: %v", err)
		}
		retrievedTask, _ := ts.GetTaskBang(ts.Ctx, task.ID)
		if retrievedTask == nil {
			t.Errorf("expected task to still exist after filtering by executing state")
		}

		// Should delete with available state filter
		err = ts.DeleteTasksFor(ts.Ctx, source, nil, []string{"available"})
		if err != nil {
			t.Fatalf("TasksDeleteTasksFor with available state filter failed: %v", err)
		}
		_, err = ts.GetTaskBang(ts.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("deletion does not impact unintended records", func(t *testing.T) {
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}

		// Delete tasks for different sources
		err := ts.DeleteTasksFor(ts.Ctx, storetest.SourceFixture(t, ts, store.Attrs{}), nil, pendingStates)
		if err != nil {
			t.Fatalf("first TasksDeleteTasksFor failed: %v", err)
		}

		err = ts.DeleteTasksFor(ts.Ctx, storetest.SourceFixture(t, ts, store.Attrs{}), store.Ptr("FooBarWorker"), pendingStates)
		if err != nil {
			t.Fatalf("second TasksDeleteTasksFor failed: %v", err)
		}

		err = ts.DeleteTasksFor(ts.Ctx, storetest.SourceFixture(t, ts, store.Attrs{}), store.Ptr("TestJobWorker"), pendingStates)
		if err != nil {
			t.Fatalf("third TasksDeleteTasksFor failed: %v", err)
		}

		// Verify the original task still exists
		retrievedTask, err := ts.GetTaskBang(ts.Ctx, task.ID)
		if err != nil || retrievedTask == nil {
			t.Errorf("expected task to still exist")
		}
	})
}

func TestTasks_DeletePendingTasksFor(t *testing.T) {
	t.Run("deletes pending tasks attached to a source", func(t *testing.T) {
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		err := ts.DeletePendingTasksFor(ts.Ctx, source, nil, false)
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor failed: %v", err)
		}

		// Verify the task was deleted
		_, err = ts.GetTaskBang(ts.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted, but it still exists")
		}
	})

	t.Run("does not delete non-pending tasks", func(t *testing.T) {
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		// Cancel the job to make it non-pending
		ts.Oban.CancelJob(ts.Ctx, task.JobID)

		err := ts.DeletePendingTasksFor(ts.Ctx, source, nil, false)
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor failed: %v", err)
		}

		// Verify the task still exists
		retrievedTask, err := ts.GetTaskBang(ts.Ctx, task.ID)
		if err != nil || retrievedTask == nil {
			t.Errorf("expected task to still exist")
		}
	})

	t.Run("works on media_items", func(t *testing.T) {
		ts := storetest.NewStore(t)
		mediaItem := storetest.MediaItemFixture(t, ts, store.MediaItemParams{})
		pendingTask := storetest.TaskFixture(t, ts, store.Attrs{"media_item_id": mediaItem.ID})
		cancelledTask := storetest.TaskFixture(t, ts, store.Attrs{"media_item_id": mediaItem.ID})

		// Cancel one task
		ts.Oban.CancelJob(ts.Ctx, cancelledTask.JobID)

		err := ts.DeletePendingTasksFor(ts.Ctx, mediaItem, nil, false)
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor failed: %v", err)
		}

		// Verify the pending task was deleted
		_, err = ts.GetTaskBang(ts.Ctx, pendingTask.ID)
		if err == nil {
			t.Errorf("expected pending task to be deleted")
		}

		// Verify the cancelled task still exists
		retrievedTask, err := ts.GetTaskBang(ts.Ctx, cancelledTask.ID)
		if err != nil || retrievedTask == nil {
			t.Errorf("expected cancelled task to still exist")
		}
	})

	t.Run("deletion can specify which worker to include", func(t *testing.T) {
		ts := storetest.NewStore(t)
		mediaItem := storetest.MediaItemFixture(t, ts, store.MediaItemParams{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"media_item_id": mediaItem.ID})

		// Should not delete with FooBarWorker filter
		err := ts.DeletePendingTasksFor(ts.Ctx, mediaItem, store.Ptr("FooBarWorker"), false)
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor with FooBarWorker filter failed: %v", err)
		}
		retrievedTask, _ := ts.GetTaskBang(ts.Ctx, task.ID)
		if retrievedTask == nil {
			t.Errorf("expected task to still exist after filtering by FooBarWorker")
		}

		// Should delete with TestJobWorker filter
		err = ts.DeletePendingTasksFor(ts.Ctx, mediaItem, store.Ptr("TestJobWorker"), false)
		if err != nil {
			t.Fatalf("TasksDeletePendingTasksFor with TestJobWorker filter failed: %v", err)
		}
		_, err = ts.GetTaskBang(ts.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted")
		}
	})

	t.Run("deletion can optionally include executing tasks", func(t *testing.T) {
		ts := storetest.NewStore(t)
		source := storetest.SourceFixture(t, ts, store.Attrs{})
		task := storetest.TaskFixture(t, ts, store.Attrs{"source_id": source.ID})

		// Set the job to executing state
		_, err := store.Exec(ts.Ctx, ts.Q(ts.Ctx), store.SQ.Update("oban_jobs").Set("state", "executing").Where(sq.Eq{"id": task.JobID}))
		if err != nil {
			t.Fatalf("failed to update job state: %v", err)
		}

		// Should not delete with include_executing=false
		err = ts.DeletePendingTasksFor(ts.Ctx, source, nil, false)
		if err != nil {
			t.Fatalf("first TasksDeletePendingTasksFor failed: %v", err)
		}
		retrievedTask, _ := ts.GetTaskBang(ts.Ctx, task.ID)
		if retrievedTask == nil {
			t.Errorf("expected task to still exist when include_executing=false")
		}

		// Should delete with include_executing=true
		err = ts.DeletePendingTasksFor(ts.Ctx, source, nil, true)
		if err != nil {
			t.Fatalf("second TasksDeletePendingTasksFor failed: %v", err)
		}
		_, err = ts.GetTaskBang(ts.Ctx, task.ID)
		if err == nil {
			t.Errorf("expected task to be deleted when include_executing=true")
		}
	})
}

func TestTasks_ChangeTask(t *testing.T) {
	t.Run("returns a task changeset", func(t *testing.T) {
		ts := storetest.NewStore(t)
		task := storetest.TaskFixture(t, ts, store.Attrs{})

		changeset := ts.ChangeTask(ts.Ctx, task, store.Attrs{})
		if changeset == nil {
			t.Fatalf("expected store.Changeset, got nil")
		}
	})
}
