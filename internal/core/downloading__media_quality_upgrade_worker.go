package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// MediaQualityUpgradeWorker handles redownloading media for quality upgrades.

const MediaQualityUpgradeWorkerName = "Pinchflat.Downloading.MediaQualityUpgradeWorker"

var mediaQualityUpgradeWorkerOpts = obanlite.WorkerOpts{
	Queue: "media_fetching",
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable", "executing"},
	},
	Tags: []string{"media_item", "media_fetching", "show_in_dashboard"},
}

// MediaQualityUpgradeWorkerPerform/1
func (a *App) MediaQualityUpgradeWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.Downloading.MediaQualityUpgradeWorker.perform/1")
}
