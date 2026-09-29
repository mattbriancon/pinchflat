package app

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// DeleteMediaProfile/1 and DeleteMediaProfile/2
func (a *App) ProfilesDeleteMediaProfile(ctx context.Context, profile *store.MediaProfile, deleteFiles bool) (*store.MediaProfile, error) {
	sources, err := a.ListSourcesFor(ctx, profile)
	if err != nil {
		return nil, err
	}

	for _, source := range sources {
		if _, err := a.SourcesDeleteSource(ctx, source, deleteFiles); err != nil {
			return nil, err
		}
	}

	if err := store.Delete(ctx, a.Q(ctx), profile); err != nil {
		return nil, err
	}
	return profile, nil
}
