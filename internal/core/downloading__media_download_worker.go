package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// MediaDownloadWorker handles media download jobs.

const MediaDownloadWorkerName = "Pinchflat.Downloading.MediaDownloadWorker"

var mediaDownloadWorkerOpts = obanlite.WorkerOpts{
	Queue:    "media_fetching",
	Priority: 5,
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable", "executing"},
	},
	Tags: []string{"media_item", "media_fetching", "show_in_dashboard"},
}

// MediaDownloadWorkerKickoffWithTask/3
func (a *App) MediaDownloadWorkerKickoffWithTask(ctx context.Context, mediaItem *MediaItem, jobArgs Attrs, jobOpts KW) (*Task, error) {
	panic("unported: Pinchflat.Downloading.MediaDownloadWorker.kickoff_with_task/3")
}

// MediaDownloadWorkerPerform/1
func (a *App) MediaDownloadWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.Downloading.MediaDownloadWorker.perform/1")
}
