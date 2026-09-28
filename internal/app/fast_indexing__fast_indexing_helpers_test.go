package app_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// parseWorkerID extracts an int64 ID from a worker arg value that may be json.Number, int64, or float64.
func parseWorkerID(t *testing.T, idVal any) int64 {
	t.Helper()
	switch v := idVal.(type) {
	case json.Number:
		n, _ := v.Int64()
		return n
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		t.Fatalf("unexpected type for worker ID: %T", idVal)
		return 0
	}
}

func TestFastIndexingHelpers_KickoffIndexingTask(t *testing.T) {
	t.Parallel()
	t.Run("deletes existing tasks", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		existingJob := apptest.JobFixture(t, ta)
		existingTask := apptest.TaskFixture(t, ta, store.Attrs{
			"source_id": source.ID,
			"job_id":    existingJob.ID,
		})

		newTask, err := ta.FastIndexingHelpersKickoffIndexingTask(ta.Ctx, source)
		if err != nil {
			t.Fatalf("kickoff failed: %v", err)
		}

		if newTask == nil || newTask.ID == existingTask.ID {
			t.Error("expected new task to be created, different from existing")
		}
	})

	t.Run("creates task with source ID", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		task, err := ta.FastIndexingHelpersKickoffIndexingTask(ta.Ctx, source)
		if err != nil {
			t.Fatalf("kickoff failed: %v", err)
		}

		if task == nil || task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("task.SourceID=%v, want %d", task.SourceID, source.ID)
		}

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
		if len(workers) != 1 {
			t.Errorf("workers=%d, want 1", len(workers))
		}
		args := workers[0].ArgsMap()
		idVal, ok := args["id"]
		if !ok {
			t.Fatalf("missing id in worker args")
		}
		workerID := parseWorkerID(t, idVal)
		if workerID != source.ID {
			t.Errorf("worker id=%d, want %d", workerID, source.ID)
		}
	})
}

// Helper to set up standard mocks for IndexAndKickoffDownloads tests.
func setupDownloadIndexMocks(t *testing.T, ta *apptest.TestApp, rssResp string) {
	t.Helper()
	ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
		return apptest.MediaAttributesReturnFixture(), nil
	})
	ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
		return rssResp, nil
	})
}

func TestFastIndexingHelpers_IndexAndKickoffDownloads(t *testing.T) {
	t.Parallel()
	t.Run("enqueues worker for new media", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		setupDownloadIndexMocks(t, ta, "<yt:videoId>test_1</yt:videoId>")

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("items=%d, want 1", len(items))
		}

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		if len(workers) != 1 {
			t.Errorf("workers=%d, want 1", len(workers))
		}
		args := workers[0].ArgsMap()
		idVal, ok := args["id"]
		if !ok {
			t.Fatalf("missing id in worker args")
		}
		workerID := parseWorkerID(t, idVal)
		if workerID != items[0].ID {
			t.Errorf("worker id=%d, want %d", workerID, items[0].ID)
		}
		if workers[0].Priority != 0 {
			t.Errorf("priority=%d, want 0", workers[0].Priority)
		}
	})

	t.Run("skips existing media", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id": source.ID,
			"media_id":  "test_1",
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 0 {
			t.Errorf("items=%d, want 0", len(items))
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
	})

	t.Run("enqueues pending media at lower priority", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		pendingItem := apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":      source.ID,
			"media_filepath": nil,
		})
		setupDownloadIndexMocks(t, ta, "<yt:videoId>test_1</yt:videoId>")

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("items=%d, want 1", len(items))
		}

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		if len(workers) != 2 {
			t.Errorf("workers=%d, want 2", len(workers))
		}

		args := workers[0].ArgsMap()
		idVal, ok := args["id"]
		if !ok {
			t.Fatalf("missing id in worker args")
		}
		workerID := parseWorkerID(t, idVal)
		if workerID != pendingItem.ID {
			t.Errorf("worker id=%d, want %d", workerID, pendingItem.ID)
		}
		if workers[0].Priority != 1 {
			t.Errorf("priority=%d, want 1", workers[0].Priority)
		}
	})

	t.Run("returns found media items", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		setupDownloadIndexMocks(t, ta, "<yt:videoId>test_1</yt:videoId>")

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("items=%d, want 1", len(items))
		}
		if items[0].MediaID != "video1" {
			t.Errorf("media_id=%s, want video1", items[0].MediaID)
		}
	})

	t.Run("respects download_media flag", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"download_media": false})
		setupDownloadIndexMocks(t, ta, "<yt:videoId>test_1</yt:videoId>")

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("items=%d, want 1", len(items))
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
	})

	t.Run("tolerates invalid media attributes", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
			return "{}", nil
		})
		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 0 {
			t.Errorf("items=%d, want 0", len(items))
		}
	})

	t.Run("tolerates yt-dlp errors", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
			return "", &fsutil.CommandError{Output: "error", Status: 1}
		})
		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 0 {
			t.Errorf("items=%d, want 0", len(items))
		}
	})

	t.Run("passes download options to yt-dlp", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		optionsChecked := false
		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
			if val, ok := opts.Get("output"); !ok || val == nil {
				t.Error("missing output option")
			}
			found := false
			for _, opt := range opts {
				if opt.Key == "remux_video" {
					found = true
					break
				}
			}
			if !found {
				t.Error("missing remux_video option")
			}
			optionsChecked = true
			return apptest.MediaAttributesReturnFixture(), nil
		})
		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if !optionsChecked {
			t.Error("yt-dlp not called")
		}
	})

	t.Run("filters by media profile rules", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "exclude"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
			output := map[string]any{
				"id":           "video2",
				"title":        "Video 2",
				"original_url": "https://example.com/shorts/video2",
				"live_status":  "not_live",
				"description":  "desc2",
				"aspect_ratio": 0.5,
				"duration":     30.0,
				"upload_date":  "20210101",
				"timestamp":    1600000000,
			}
			jsonStr, _ := db.EncodeJSON(output)
			return jsonStr, nil
		})
		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("items=%d, want 1", len(items))
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
	})
}

func TestFastIndexingHelpers_CookieBehavior(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		behavior     string
		expectedCook any
	}{
		{"uses cookies when all_operations", "all_operations", true},
		{"skips when_needed", "when_needed", false},
		{"skips when disabled", "disabled", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, store.Attrs{"cookie_behaviour": tt.behavior})

			cookieChecked := false
			ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
				val, ok := addlOpts.Get("use_cookies")
				if !ok || val != tt.expectedCook {
					t.Errorf("use_cookies=%v, want %v", val, tt.expectedCook)
				}
				cookieChecked = true
				return apptest.MediaAttributesReturnFixture(), nil
			})

			ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
				return "<yt:videoId>test_1</yt:videoId>", nil
			})

			_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
			if err != nil {
				t.Fatalf("failed: %v", err)
			}

			if !cookieChecked {
				t.Error("yt-dlp not called")
			}
		})
	}
}

func TestFastIndexingHelpers_Backends(t *testing.T) {
	t.Parallel()
	t.Run("uses API when enabled", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		ta.SetSetting(ta.Ctx, store.KW{store.Opt("youtube_api_key", "test_key")})

		apiUsed := false
		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			if strings.Contains(url, "youtube.googleapis.com/youtube/v3/playlistItems") {
				apiUsed = true
			}
			return "{}", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if !apiUsed {
			t.Error("API not used")
		}
	})

	t.Run("API creates records correctly", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		ta.SetSetting(ta.Ctx, store.KW{store.Opt("youtube_api_key", "test_key")})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
			return apptest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			if strings.Contains(url, "youtube.googleapis.com") {
				response := map[string]any{
					"items": []map[string]any{
						{"contentDetails": map[string]any{"videoId": "test_1"}},
					},
				}
				jsonStr, _ := json.Marshal(response)
				return string(jsonStr), nil
			}
			return "", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("items=%d, want 1", len(items))
		}
	})

	t.Run("falls back to RSS if API fails", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		ta.SetSetting(ta.Ctx, store.KW{store.Opt("youtube_api_key", "test_key")})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
			return apptest.MediaAttributesReturnFixture(), nil
		})

		callCount := 0
		ta.HTTPMock.Get.Stub(func(url string, headers, opts store.KW) (string, error) {
			callCount++
			if callCount == 1 && strings.Contains(url, "youtube.googleapis.com") {
				return "", &fsutil.CommandError{Output: "API failed", Status: 1}
			}
			if strings.Contains(url, "youtube.com/feeds") {
				return "<yt:videoId>test_1</yt:videoId>", nil
			}
			return "", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("items=%d, want 1", len(items))
		}
	})

	t.Run("uses RSS when API disabled", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		ta.SetSetting(ta.Ctx, store.KW{store.Opt("youtube_api_key", nil)})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, outputTemplate string, addlOpts store.KW) (string, error) {
			return apptest.MediaAttributesReturnFixture(), nil
		})

		rssUsed := false
		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			if strings.Contains(url, "youtube.com/feeds/videos.xml") {
				rssUsed = true
			}
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if !rssUsed {
			t.Error("RSS not used")
		}
	})
}
