package app

// Central Oban worker registry (replaces `use Oban.Worker` discovery).
// Hand-written W0 infrastructure.

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// WorkerNames lists every worker, by Elixir module name.
var WorkerNames = []string{
	MediaDownloadWorkerName,
	MediaQualityUpgradeWorkerName,
	MediaRetentionWorkerName,
	FastIndexingWorkerName,
	FileSyncingWorkerName,
	SourceMetadataStorageWorkerName,
	MediaProfileDeletionWorkerName,
	MediaCollectionIndexingWorkerName,
	SourceDeletionWorkerName,
	UpdateWorkerName,
}

// RegisterWorkers registers every worker with a.Oban.
func (a *App) RegisterWorkers() {
	reg := func(name string, opts obanlite.WorkerOpts, perform func(context.Context, *obanlite.Job) error) {
		a.Oban.Register(name, opts, obanlite.WorkerFunc(perform))
	}
	reg(MediaDownloadWorkerName, mediaDownloadWorkerOpts, a.MediaDownloadWorkerPerform)
	reg(MediaQualityUpgradeWorkerName, mediaQualityUpgradeWorkerOpts, a.MediaQualityUpgradeWorkerPerform)
	reg(MediaRetentionWorkerName, mediaRetentionWorkerOpts, a.MediaRetentionWorkerPerform)
	reg(FastIndexingWorkerName, fastIndexingWorkerOpts, a.FastIndexingWorkerPerform)
	reg(FileSyncingWorkerName, fileSyncingWorkerOpts, a.FileSyncingWorkerPerform)
	reg(SourceMetadataStorageWorkerName, sourceMetadataStorageWorkerOpts, a.SourceMetadataStorageWorkerPerform)
	reg(MediaProfileDeletionWorkerName, mediaProfileDeletionWorkerOpts, a.MediaProfileDeletionWorkerPerform)
	reg(MediaCollectionIndexingWorkerName, mediaCollectionIndexingWorkerOpts, a.MediaCollectionIndexingWorkerPerform)
	reg(SourceDeletionWorkerName, sourceDeletionWorkerOpts, a.SourceDeletionWorkerPerform)
	reg(UpdateWorkerName, updateWorkerOpts, a.UpdateWorkerPerform)
}
