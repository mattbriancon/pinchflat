package core

import (
	"context"

	sq "github.com/Masterminds/squirrel"
)

// ListMediaProfiles/0
func (a *App) ProfilesListMediaProfiles(ctx context.Context) ([]*MediaProfile, error) {
	return All[MediaProfile](ctx, a.Q(ctx), From[MediaProfile](""))
}

// GetMediaProfileBang/1
func (a *App) ProfilesGetMediaProfile(ctx context.Context, id int64) (*MediaProfile, error) {
	return MustOne[MediaProfile](ctx, a.Q(ctx), From[MediaProfile]("").Where(sq.Eq{"id": id}))
}

// CreateMediaProfile/1
func (a *App) ProfilesCreateMediaProfile(ctx context.Context, attrs Attrs) (*MediaProfile, error) {
	profile := NewMediaProfile()
	cs := MediaProfileChangeset(profile, attrs)
	return Insert[MediaProfile](ctx, a.Q(ctx), cs)
}

// UpdateMediaProfile/2
func (a *App) ProfilesUpdateMediaProfile(ctx context.Context, profile *MediaProfile, attrs Attrs) (*MediaProfile, error) {
	cs := MediaProfileChangeset(profile, attrs)
	return Update[MediaProfile](ctx, a.Q(ctx), cs)
}

// DeleteMediaProfile/1 and DeleteMediaProfile/2
func (a *App) ProfilesDeleteMediaProfile(ctx context.Context, profile *MediaProfile, opts KW) (*MediaProfile, error) {
	deleteFiles := opts.GetOr("delete_files", false).(bool)

	sources, err := a.SourcesListSourcesFor(ctx, profile)
	if err != nil {
		return nil, err
	}

	for _, source := range sources {
		deleteOpts := KW{}
		if deleteFiles {
			deleteOpts = append(deleteOpts, Opt("delete_files", true))
		}
		if _, err := a.SourcesDeleteSource(ctx, source, deleteOpts); err != nil {
			return nil, err
		}
	}

	if err := Delete(ctx, a.Q(ctx), profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// ChangeMediaProfile/1 and ChangeMediaProfile/2
func (a *App) ProfilesChangeMediaProfile(ctx context.Context, profile *MediaProfile, attrs Attrs) *Changeset {
	return MediaProfileChangeset(profile, attrs)
}
