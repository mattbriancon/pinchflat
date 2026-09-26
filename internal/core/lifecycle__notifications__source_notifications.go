package core

import (
	"context"
)

// SourceNotifications provides utilities for sending notifications about sources.

// SourceNotificationsWrapNewMediaNotification/3
func (a *App) SourceNotificationsWrapNewMediaNotification(ctx context.Context, servers []string, source *Source, fn func() (any, error)) (any, error) {
	panic("unported: Pinchflat.Lifecycle.Notifications.SourceNotifications.wrap_new_media_notification/3")
}

// SourceNotificationsSendNewMediaNotification/3
func (a *App) SourceNotificationsSendNewMediaNotification(ctx context.Context, servers []string, source *Source, changedCount int) error {
	panic("unported: Pinchflat.Lifecycle.Notifications.SourceNotifications.send_new_media_notification/3")
}
