package core_test

import (
	"fmt"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestSourceNotifications_WrapNewMediaNotification(t *testing.T) {
	servers := []string{"server_1", "server_2"}

	t.Run("sends a notification when the pending count changes", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.AppriseMock.Run.Expect(func(endpoints []string, opts core.KW) error {
			if len(endpoints) != 2 {
				t.Errorf("Expected 2 servers, got %d", len(endpoints))
			}

			title, _ := opts.Get("title")
			body, _ := opts.Get("body")

			if title != "[Pinchflat] New media found" {
				t.Errorf("Expected title '[Pinchflat] New media found', got %v", title)
			}

			expectedBody := fmt.Sprintf("Found 1 new media item(s) for %s. Downloading them now", source.CustomName)
			if body != expectedBody {
				t.Errorf("Expected body %q, got %q", expectedBody, body)
			}

			return nil
		})

		retval, err := ta.App.SourceNotificationsWrapNewMediaNotification(ta.Ctx, servers, source, func() (any, error) {
			coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": nil})
			return nil, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retval != nil {
			t.Errorf("Expected nil return value, got %v", retval)
		}
	})

	t.Run("sends a notification when the downloaded count changes", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.AppriseMock.Run.Expect(func(endpoints []string, opts core.KW) error {
			if len(endpoints) != 2 {
				t.Errorf("Expected 2 servers, got %d", len(endpoints))
			}

			title, _ := opts.Get("title")
			body, _ := opts.Get("body")

			if title != "[Pinchflat] New media found" {
				t.Errorf("Expected title '[Pinchflat] New media found', got %v", title)
			}

			expectedBody := fmt.Sprintf("Found 1 new media item(s) for %s. Downloading them now", source.CustomName)
			if body != expectedBody {
				t.Errorf("Expected body %q, got %q", expectedBody, body)
			}

			return nil
		})

		retval, err := ta.App.SourceNotificationsWrapNewMediaNotification(ta.Ctx, servers, source, func() (any, error) {
			coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": "file.mp4"})
			return nil, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retval != nil {
			t.Errorf("Expected nil return value, got %v", retval)
		}
	})

	t.Run("does not send a notification when the count does not change", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.AppriseMock.Run.ExpectN(0, func(endpoints []string, opts core.KW) error {
			return nil
		})

		retval, err := ta.App.SourceNotificationsWrapNewMediaNotification(ta.Ctx, servers, source, func() (any, error) {
			coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "prevent_download": true, "media_filepath": nil})
			return nil, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retval != nil {
			t.Errorf("Expected nil return value, got %v", retval)
		}
	})

	t.Run("does not send a notification if the source is set to not download media", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"download_media": false})

		ta.AppriseMock.Run.ExpectN(0, func(endpoints []string, opts core.KW) error {
			return nil
		})

		retval, err := ta.App.SourceNotificationsWrapNewMediaNotification(ta.Ctx, servers, source, func() (any, error) {
			coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": nil})
			coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": "file.mp4"})
			return nil, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retval != nil {
			t.Errorf("Expected nil return value, got %v", retval)
		}
	})

	t.Run("returns the value of the function", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.AppriseMock.Run.ExpectN(0, func(endpoints []string, opts core.KW) error {
			return nil
		})

		retval, err := ta.App.SourceNotificationsWrapNewMediaNotification(ta.Ctx, servers, source, func() (any, error) {
			return "value", nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retval != "value" {
			t.Errorf("Expected return value 'value', got %v", retval)
		}
	})
}

func TestSourceNotifications_SendNewMediaNotification(t *testing.T) {
	servers := []string{"server_1", "server_2"}

	t.Run("sends a notification when count is positive", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.AppriseMock.Run.Expect(func(endpoints []string, opts core.KW) error {
			if len(endpoints) != 2 {
				t.Errorf("Expected 2 servers, got %d", len(endpoints))
			}

			title, _ := opts.Get("title")
			body, _ := opts.Get("body")

			if title != "[Pinchflat] New media found" {
				t.Errorf("Expected title '[Pinchflat] New media found', got %v", title)
			}

			if body == nil {
				t.Errorf("Expected body, got none")
			}

			return nil
		})

		if err := ta.App.SourceNotificationsSendNewMediaNotification(ta.Ctx, servers, source, 1); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("does not send a notification when count not positive", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.AppriseMock.Run.ExpectN(0, func(endpoints []string, opts core.KW) error {
			return nil
		})

		if err := ta.App.SourceNotificationsSendNewMediaNotification(ta.Ctx, servers, source, 0); err != nil {
			t.Errorf("Expected no error for count=0, got %v", err)
		}

		if err := ta.App.SourceNotificationsSendNewMediaNotification(ta.Ctx, servers, source, -1); err != nil {
			t.Errorf("Expected no error for count=-1, got %v", err)
		}
	})
}
