package core_test

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestTasks_Schema(t *testing.T) {
	t.Run("cascade deletes task", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		core.Exec(ta.Ctx, ta.Q(ta.Ctx), core.SQ.Delete("oban_jobs").Where(sq.Eq{"id": task.JobID}))

		_, err := core.MustOne[core.Task](ta.Ctx, ta.Q(ta.Ctx), core.From[core.Task]().Where(sq.Eq{"id": task.ID}))
		if err == nil {
			t.Error("task should be deleted")
		}
	})

	t.Run("preserves related records", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		if task.SourceID == nil {
			t.Fatal("task should have source_id")
		}
		source, _ := core.MustOne[core.Source](ta.Ctx, ta.Q(ta.Ctx), core.From[core.Source]().Where(sq.Eq{"id": *task.SourceID}))

		core.Exec(ta.Ctx, ta.Q(ta.Ctx), core.SQ.Delete("oban_jobs").Where(sq.Eq{"id": task.JobID}))

		reloaded, err := core.MustOne[core.Source](ta.Ctx, ta.Q(ta.Ctx), core.From[core.Source]().Where(sq.Eq{"id": source.ID}))
		if err != nil {
			t.Error("source should still exist")
		}
		if reloaded.ID != source.ID {
			t.Error("source ID should match")
		}
	})
}

func TestTasks_ListTasks(t *testing.T) {
	t.Parallel()
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
}

func TestTasks_ListTasksFor(t *testing.T) {
	t.Run("by record", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		tasks, err := ta.TasksListTasksFor(ta.Ctx, source, nil, []string{"available"})
		if err != nil {
			t.Fatalf("TasksListTasksFor failed: %v", err)
		}
		if len(tasks) != 1 || tasks[0].ID != task.ID {
			t.Error("should find task")
		}
	})

	t.Run("by job state", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		tasks, _ := ta.TasksListTasksFor(ta.Ctx, source, nil, []string{"available"})
		if len(tasks) != 1 {
			t.Error("should find task with available state")
		}

		tasks, _ = ta.TasksListTasksFor(ta.Ctx, source, nil, []string{"cancelled"})
		if len(tasks) != 0 {
			t.Error("should not find task with cancelled state")
		}
	})

	t.Run("by worker", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}

		tasks, _ := ta.TasksListTasksFor(ta.Ctx, source, core.Ptr("TestJobWorker"), pendingStates)
		if len(tasks) != 1 {
			t.Error("should find task with TestJobWorker")
		}

		tasks, _ = ta.TasksListTasksFor(ta.Ctx, source, core.Ptr("FooBarWorker"), pendingStates)
		if len(tasks) != 0 {
			t.Error("should not find task with FooBarWorker")
		}
	})

	t.Run("includes all workers if none specified", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}
		tasks, _ := ta.TasksListTasksFor(ta.Ctx, source, nil, pendingStates)
		if len(tasks) != 1 || tasks[0].ID != task.ID {
			t.Error("should find task without worker filter")
		}
	})
}

func TestTasks_GetTaskBang(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)
	task := coretest.TaskFixture(t, ta, core.Attrs{})

	retrieved, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
	if err != nil {
		t.Fatalf("TasksGetTaskBang failed: %v", err)
	}
	if retrieved.ID != task.ID {
		t.Errorf("expected task ID %d, got %d", task.ID, retrieved.ID)
	}
}

func TestTasks_CreateTask(t *testing.T) {
	t.Run("with valid data", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		job := coretest.JobFixture(t, ta)
		attrs := core.Attrs{"job_id": job.ID}

		task, err := ta.TasksCreateTask(ta.Ctx, attrs)
		if err != nil {
			t.Fatalf("TasksCreateTask failed: %v", err)
		}
		if task == nil || task.JobID != job.ID {
			t.Error("task should be created with correct job_id")
		}
	})

	t.Run("with invalid data", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		task, err := ta.TasksCreateTask(ta.Ctx, core.Attrs{"job_id": nil})
		if err == nil {
			t.Error("expected error for invalid attrs")
		}
		if task != nil {
			t.Error("task should be nil on error")
		}
	})

	t.Run("with source", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		job := coretest.JobFixture(t, ta)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		task, err := ta.TasksCreateTaskWithRecord(ta.Ctx, job, source)
		if err != nil {
			t.Fatalf("TasksCreateTaskWithRecord failed: %v", err)
		}
		if task.JobID != job.ID || task.SourceID == nil || *task.SourceID != source.ID {
			t.Error("task should have correct job_id and source_id")
		}
	})

	t.Run("with media item", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		job := coretest.JobFixture(t, ta)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		task, err := ta.TasksCreateTaskWithRecord(ta.Ctx, job, mediaItem)
		if err != nil {
			t.Fatalf("TasksCreateTaskWithRecord failed: %v", err)
		}
		if task.JobID != job.ID || task.MediaItemID == nil || *task.MediaItemID != mediaItem.ID {
			t.Error("task should have correct job_id and media_item_id")
		}
	})
}

func TestTasks_CreateJobWithTask(t *testing.T) {
	t.Run("enqueues job", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{})
		task, err := ta.TasksCreateJobWithTask(ta.Ctx, spec, mediaItem)
		if err != nil {
			t.Fatalf("TasksCreateJobWithTask failed: %v", err)
		}
		if task == nil {
			t.Fatal("task should be created")
		}

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: coretest.TestJobWorkerName})
		if len(enqueued) != 1 {
			t.Errorf("expected 1 job enqueued, got %d", len(enqueued))
		}
	})

	t.Run("creates task record", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{})
		task, err := ta.TasksCreateJobWithTask(ta.Ctx, spec, source)
		if err != nil {
			t.Fatalf("TasksCreateJobWithTask failed: %v", err)
		}
		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Error("task should have correct source_id")
		}
	})

	t.Run("errors on duplicate job", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{"foo": "bar"})
		spec.Unique = &obanlite.UniqueOpts{Period: obanlite.Infinity}

		task1, err := ta.TasksCreateJobWithTask(ta.Ctx, spec, source)
		if err != nil || task1 == nil {
			t.Fatal("first job should be created")
		}

		_, err = ta.TasksCreateJobWithTask(ta.Ctx, spec, source)
		if err == nil {
			t.Error("expected error for duplicate job")
		}
		if err.Error() != "duplicate_job" {
			t.Errorf("expected 'duplicate_job', got %q", err.Error())
		}
	})

	t.Run("errors on invalid job spec", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		spec := obanlite.JobSpec{Worker: ""}
		_, err := ta.TasksCreateJobWithTask(ta.Ctx, spec, source)
		if err == nil {
			t.Error("expected error for invalid job spec")
		}
	})
}

func TestTasks_DeleteTask(t *testing.T) {
	t.Run("deletes task", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})

		_, err := ta.TasksDeleteTask(ta.Ctx, task)
		if err != nil {
			t.Fatalf("TasksDeleteTask failed: %v", err)
		}

		_, err = ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Error("task should be deleted")
		}
	})

	t.Run("cancels job", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		task := coretest.TaskFixture(t, ta, core.Attrs{})
		jobID := task.JobID

		ta.TasksDeleteTask(ta.Ctx, task)

		var state string
		ta.Q(ta.Ctx).GetContext(ta.Ctx, &state, "SELECT state FROM oban_jobs WHERE id = ?", jobID)
		if state != "cancelled" {
			t.Errorf("expected job state cancelled, got %s", state)
		}
	})
}

func TestTasks_DeleteTasksFor(t *testing.T) {
	t.Run("deletes tasks for source", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		ta.TasksDeleteTasksFor(ta.Ctx, source, nil, []string{"available", "scheduled", "retryable"})

		_, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Error("task should be deleted")
		}
	})

	t.Run("deletes tasks for media item", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})

		ta.TasksDeleteTasksFor(ta.Ctx, mediaItem, nil, []string{"available", "scheduled", "retryable"})

		_, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Error("task should be deleted")
		}
	})

	t.Run("filters by worker", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}

		ta.TasksDeleteTasksFor(ta.Ctx, mediaItem, core.Ptr("FooBarWorker"), pendingStates)
		retrieved, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrieved == nil {
			t.Error("task should still exist with wrong worker filter")
		}

		ta.TasksDeleteTasksFor(ta.Ctx, mediaItem, core.Ptr("TestJobWorker"), pendingStates)
		_, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Error("task should be deleted with correct worker filter")
		}
	})

	t.Run("filters by state", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		ta.TasksDeleteTasksFor(ta.Ctx, source, nil, []string{"executing"})
		retrieved, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrieved == nil {
			t.Error("task should still exist with wrong state filter")
		}

		ta.TasksDeleteTasksFor(ta.Ctx, source, nil, []string{"available"})
		_, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Error("task should be deleted with correct state filter")
		}
	})

	t.Run("doesn't affect other records", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		pendingStates := []string{"available", "scheduled", "retryable"}

		ta.TasksDeleteTasksFor(ta.Ctx, coretest.SourceFixture(t, ta, core.Attrs{}), nil, pendingStates)
		ta.TasksDeleteTasksFor(ta.Ctx, coretest.SourceFixture(t, ta, core.Attrs{}), core.Ptr("FooBarWorker"), pendingStates)
		ta.TasksDeleteTasksFor(ta.Ctx, coretest.SourceFixture(t, ta, core.Attrs{}), core.Ptr("TestJobWorker"), pendingStates)

		retrieved, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err != nil || retrieved == nil {
			t.Error("original task should still exist")
		}
	})
}

func TestTasks_DeletePendingTasksFor(t *testing.T) {
	t.Run("deletes pending tasks for source", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		ta.TasksDeletePendingTasksFor(ta.Ctx, source, nil, core.KW{})

		_, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Error("task should be deleted")
		}
	})

	t.Run("preserves non-pending tasks", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		ta.Oban.CancelJob(ta.Ctx, task.JobID)
		ta.TasksDeletePendingTasksFor(ta.Ctx, source, nil, core.KW{})

		retrieved, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrieved == nil {
			t.Error("cancelled task should still exist")
		}
	})

	t.Run("works on media items", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		pendingTask := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})
		cancelledTask := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})

		ta.Oban.CancelJob(ta.Ctx, cancelledTask.JobID)
		ta.TasksDeletePendingTasksFor(ta.Ctx, mediaItem, nil, core.KW{})

		_, err := ta.TasksGetTaskBang(ta.Ctx, pendingTask.ID)
		if err == nil {
			t.Error("pending task should be deleted")
		}

		retrieved, _ := ta.TasksGetTaskBang(ta.Ctx, cancelledTask.ID)
		if retrieved == nil {
			t.Error("cancelled task should still exist")
		}
	})

	t.Run("filters by worker", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"media_item_id": mediaItem.ID})

		ta.TasksDeletePendingTasksFor(ta.Ctx, mediaItem, core.Ptr("FooBarWorker"), core.KW{})
		retrieved, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrieved == nil {
			t.Error("task should still exist with wrong worker filter")
		}

		ta.TasksDeletePendingTasksFor(ta.Ctx, mediaItem, core.Ptr("TestJobWorker"), core.KW{})
		_, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Error("task should be deleted with correct worker filter")
		}
	})

	t.Run("filters by include_executing", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": source.ID})

		core.Exec(ta.Ctx, ta.Q(ta.Ctx), core.SQ.Update("oban_jobs").Set("state", "executing").Where(sq.Eq{"id": task.JobID}))

		ta.TasksDeletePendingTasksFor(ta.Ctx, source, nil, core.KW{core.Opt("include_executing", false)})
		retrieved, _ := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if retrieved == nil {
			t.Error("executing task should still exist when include_executing=false")
		}

		ta.TasksDeletePendingTasksFor(ta.Ctx, source, nil, core.KW{core.Opt("include_executing", true)})
		_, err := ta.TasksGetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Error("executing task should be deleted when include_executing=true")
		}
	})
}

func TestTasks_ChangeTask(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)
	task := coretest.TaskFixture(t, ta, core.Attrs{})

	cs := ta.TasksChangeTask(ta.Ctx, task, core.Attrs{})
	if cs == nil {
		t.Error("should return changeset")
	}
}
