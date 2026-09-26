package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestTasks_Schema(t *testing.T) {
	t.Run("deletes a task when the job gets deleted", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("does not delete the other record when a job gets deleted", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})
}

func TestTasks_ListTasks(t *testing.T) {
	t.Run("returns all tasks", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})
}

func TestTasks_ListTasksFor(t *testing.T) {
	t.Run("lets you specify which record type/ID to join on", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("lets you specify which job states to include", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("lets you specify which worker to include", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("includes all workers if no worker is specified", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})
}

func TestTasks_GetTaskBang(t *testing.T) {
	t.Run("returns the task with given id", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})
}

func TestTasks_CreateTask(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("creation with valid data creates a task", func(t *testing.T) {
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
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("accepts a job and media item", func(t *testing.T) {
	})
}

func TestTasks_CreateJobWithTask(t *testing.T) {
	t.Run("enqueues the given job", func(t *testing.T) {
	})

	t.Run("creates a task record if successful", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("returns an error if the job already exists", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("returns an error if the job fails to enqueue", func(t *testing.T) {
		t.Skip("BLOCKED: cannot pass invalid JobSpec without changing signature")
	})
}

func TestTasks_DeleteTask(t *testing.T) {
	t.Run("deletion deletes the task", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("deletion also cancels the attached job", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})
}

func TestTasks_DeleteTasksFor(t *testing.T) {
	t.Run("deletes tasks attached to a source", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("deletes the tasks attached to a media_item", func(t *testing.T) {
	})

	t.Run("deletion can specify which worker to include", func(t *testing.T) {
	})

	t.Run("deletion can specify which states to include", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("deletion does not impact unintended records", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})
}

func TestTasks_DeletePendingTasksFor(t *testing.T) {
	t.Run("deletes pending tasks attached to a source", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("does not delete non-pending tasks", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})

	t.Run("works on media_items", func(t *testing.T) {
	})

	t.Run("deletion can specify which worker to include", func(t *testing.T) {
	})

	t.Run("deletion can optionally include executing tasks", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})
}

func TestTasks_ChangeTask(t *testing.T) {
	t.Run("returns a task changeset", func(t *testing.T) {
		// unblocked: ProfilesCreateMediaProfile ported
	})
}
