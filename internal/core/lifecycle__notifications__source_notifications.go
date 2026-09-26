package core

import (
	"context"
	"fmt"
	"log/slog"
)

// SourceNotifications provides utilities for sending notifications about sources.

// SourceNotificationsWrapNewMediaNotification/3
func (a *App) SourceNotificationsWrapNewMediaNotification(ctx context.Context, servers []string, source *Source, fn func() (any, error)) (any, error) {
	beforeCount := sourceNotificationsRelevantMediaItemCount(ctx, a, source)
	retval, err := fn()
	if err != nil {
		return retval, err
	}
	afterCount := sourceNotificationsRelevantMediaItemCount(ctx, a, source)

	_ = a.SourceNotificationsSendNewMediaNotification(ctx, servers, source, afterCount-beforeCount)

	return retval, nil
}

// SourceNotificationsSendNewMediaNotification/3
func (a *App) SourceNotificationsSendNewMediaNotification(ctx context.Context, servers []string, source *Source, changedCount int) error {
	if changedCount <= 0 {
		return nil
	}

	opts := KW{
		Opt("title", "[Pinchflat] New media found"),
		Opt("body", fmt.Sprintf("Found %d new media item(s) for %s. Downloading them now", changedCount, source.CustomName)),
	}

	if err := a.Apprise.Run(ctx, servers, opts); err != nil {
		slog.Error("Failed to send new media notification", "source_id", source.ID, "error", err)
	} else {
		slog.Info("Sent new media notification", "source_id", source.ID)
	}

	return nil
}

func sourceNotificationsRelevantMediaItemCount(ctx context.Context, a *App, source *Source) int {
	if !source.DownloadMedia {
		return 0
	}
	return sourceNotificationsPendingMediaItemCount(ctx, a, source) + sourceNotificationsDownloadedMediaItemCount(ctx, a, source)
}

func sourceNotificationsPendingMediaItemCount(ctx context.Context, a *App, source *Source) int {
	q := MediaQueryNew().
		RequireAssoc("media_profile").
		Where(MediaQueryForSource(source.ID)).
		Where(MediaQueryPending())

	count, _ := Scalar[int](ctx, a.Q(ctx), SQ.Select("COUNT(*)").FromSelect(q.B, "mi"))
	return count
}

func sourceNotificationsDownloadedMediaItemCount(ctx context.Context, a *App, source *Source) int {
	q := MediaQueryNew().
		Where(MediaQueryForSource(source.ID)).
		Where(MediaQueryDownloaded())

	count, _ := Scalar[int](ctx, a.Q(ctx), SQ.Select("COUNT(*)").FromSelect(q.B, "mi"))
	return count
}
