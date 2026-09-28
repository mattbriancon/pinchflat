package app_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestPostBootStartupTasks_UpdateYtDlp(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.UpdateWorkerName})) != 0 {
		t.Errorf("expected no jobs initially")
	}

	if err := ta.PostBootStartupTasksInit(ta.Ctx); err != nil {
		t.Fatalf("PostBootStartupTasksInit failed: %v", err)
	}

	if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.UpdateWorkerName})) != 1 {
		t.Errorf("expected 1 job enqueued")
	}
}
