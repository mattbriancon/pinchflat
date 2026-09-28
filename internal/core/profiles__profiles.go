package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// DeleteMediaProfile/1 and DeleteMediaProfile/2
func (a *App) ProfilesDeleteMediaProfile(ctx context.Context, profile *store.MediaProfile, opts store.KW) (*store.MediaProfile, error) {
	deleteFiles := opts.GetOr("delete_files", false).(bool)

	sources, err := a.ListSourcesFor(ctx, profile)
	if err != nil {
		return nil, err
	}

	for _, source := range sources {
		deleteOpts := store.KW{}
		if deleteFiles {
			deleteOpts = append(deleteOpts, store.Opt("delete_files", true))
		}
		if _, err := a.SourcesDeleteSource(ctx, source, deleteOpts); err != nil {
			return nil, err
		}
	}

	if err := store.Delete(ctx, a.Q(ctx), profile); err != nil {
		return nil, err
	}
	return profile, nil
}
