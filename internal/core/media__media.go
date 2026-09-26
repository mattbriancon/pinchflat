package core

import "context"

var fieldsToDropOnUpdate = []string{"playlist_index"}

// ListMediaItems/0
func (a *App) MediaListMediaItems(ctx context.Context) ([]*MediaItem, error) {
	panic("unported: Pinchflat.Media.list_media_items/0")
}

// ListUpgradeableMediaItems/0
func (a *App) MediaListUpgradeableMediaItems(ctx context.Context) ([]*MediaItem, error) {
	panic("unported: Pinchflat.Media.list_upgradeable_media_items/0")
}

// ListPendingMediaItemsFor/1
func (a *App) MediaListPendingMediaItemsFor(ctx context.Context, source *Source) ([]*MediaItem, error) {
	panic("unported: Pinchflat.Media.list_pending_media_items_for/1")
}

// PendingDownload/1
func (a *App) MediaPendingDownload(ctx context.Context, mediaItem *MediaItem) (bool, error) {
	panic("unported: Pinchflat.Media.pending_download?/1")
}

// Search/1 and Search/2
func (a *App) MediaSearch(ctx context.Context, searchTerm string, opts KW) ([]*MediaItem, error) {
	panic("unported: Pinchflat.Media.search/2")
}

// GetMediaItem/1
func (a *App) MediaGetMediaItem(ctx context.Context, id int64) (*MediaItem, error) {
	panic("unported: Pinchflat.Media.get_media_item!/1")
}

// CreateMediaItem/1
func (a *App) MediaCreateMediaItem(ctx context.Context, attrs Attrs) (*MediaItem, error) {
	panic("unported: Pinchflat.Media.create_media_item/1")
}

// CreateMediaItemFromBackendAttrs/2
func (a *App) MediaCreateMediaItemFromBackendAttrs(ctx context.Context, source *Source, mediaAttrsStruct any) (*MediaItem, error) {
	panic("unported: Pinchflat.Media.create_media_item_from_backend_attrs/2")
}

// UpdateMediaItem/2
func (a *App) MediaUpdateMediaItem(ctx context.Context, mediaItem *MediaItem, attrs Attrs) (*MediaItem, error) {
	panic("unported: Pinchflat.Media.update_media_item/2")
}

// DeleteMediaItem/1 and DeleteMediaItem/2
func (a *App) MediaDeleteMediaItem(ctx context.Context, mediaItem *MediaItem, opts KW) (*MediaItem, error) {
	panic("unported: Pinchflat.Media.delete_media_item/2")
}

// DeleteMediaFiles/1 and DeleteMediaFiles/2
func (a *App) MediaDeleteMediaFiles(ctx context.Context, mediaItem *MediaItem, addlAttrs Attrs) (*MediaItem, error) {
	panic("unported: Pinchflat.Media.delete_media_files/2")
}

// ChangeMediaItem/1 and ChangeMediaItem/2
func (a *App) MediaChangeMediaItem(ctx context.Context, mediaItem *MediaItem, attrs Attrs) *Changeset {
	panic("unported: Pinchflat.Media.change_media_item/2")
}
