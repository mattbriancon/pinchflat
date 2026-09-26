package core

import (
	"context"
	"fmt"
	"log/slog"

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
	upgradableMedia, err := a.MediaListUpgradeableMediaItems(ctx)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Redownloading %d media items", len(upgradableMedia)))

	for _, mediaItem := range upgradableMedia {
		_, err := a.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, Attrs{"quality_upgrade?": true}, KW{})
		if err != nil {
			return err
		}
	}

	return nil
}
