package core

import (
	"context"
)

// ListMediaProfiles/0
func (a *App) ListMediaProfiles(ctx context.Context) ([]*MediaProfile, error) {
	panic("unported: Pinchflat.Profiles.list_media_profiles/0")
}

// GetMediaProfileBang/1
func (a *App) GetMediaProfileBang(ctx context.Context, id int64) (*MediaProfile, error) {
	panic("unported: Pinchflat.Profiles.get_media_profile!/1")
}

// CreateMediaProfile/1
func (a *App) CreateMediaProfile(ctx context.Context, attrs Attrs) (*MediaProfile, error) {
	panic("unported: Pinchflat.Profiles.create_media_profile/1")
}

// UpdateMediaProfile/2
func (a *App) UpdateMediaProfile(ctx context.Context, profile *MediaProfile, attrs Attrs) (*MediaProfile, error) {
	panic("unported: Pinchflat.Profiles.update_media_profile/2")
}

// DeleteMediaProfile/1 and DeleteMediaProfile/2
func (a *App) DeleteMediaProfile(ctx context.Context, profile *MediaProfile, opts ...KW) (*MediaProfile, error) {
	panic("unported: Pinchflat.Profiles.delete_media_profile/2")
}

// ChangeMediaProfile/1 and ChangeMediaProfile/2
func (a *App) ChangeMediaProfile(ctx context.Context, profile *MediaProfile, attrs ...Attrs) *Changeset {
	panic("unported: Pinchflat.Profiles.change_media_profile/2")
}
