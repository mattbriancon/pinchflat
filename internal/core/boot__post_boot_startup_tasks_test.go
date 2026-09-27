package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestPostBootStartupTasks_UpdateYtDlp(t *testing.T) {
	t.Run("enqueues an update job", func(t *testing.T) {
		ta := coretest.NewApp(t)

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.UpdateWorkerName})
		if len(enqueued) != 0 {
			t.Errorf("Expected no jobs enqueued, got %d", len(enqueued))
		}

		if err := ta.PostBootStartupTasksInit(ta.Ctx); err != nil {
			t.Errorf("PostBootStartupTasksInit failed: %v", err)
		}

		enqueued = ta.Oban.Enqueued(t, obanlite.Match{Worker: core.UpdateWorkerName})
		if len(enqueued) != 1 {
			t.Errorf("Expected 1 job enqueued, got %d", len(enqueued))
		}
	})
}
