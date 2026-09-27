package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestPostBootStartupTasks_UpdateYtDlp(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)

	if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: core.UpdateWorkerName})) != 0 {
		t.Errorf("expected no jobs initially")
	}

	if err := ta.PostBootStartupTasksInit(ta.Ctx); err != nil {
		t.Fatalf("PostBootStartupTasksInit failed: %v", err)
	}

	if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: core.UpdateWorkerName})) != 1 {
		t.Errorf("expected 1 job enqueued")
	}
}
