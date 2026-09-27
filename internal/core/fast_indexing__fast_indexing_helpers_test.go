package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestFastIndexingHelpers_KickoffIndexingTask(t *testing.T) {
	t.Run("deletes any existing fast indexing tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		// Create an existing task
		existingJob := coretest.JobFixture(t, ta)
		existingTask := coretest.TaskFixture(t, ta, core.Attrs{
			"source_id": source.ID,
			"job_id":    existingJob.ID,
		})

		// Call kickoff, which should delete the existing task and create a new one
		newTask, err := ta.FastIndexingHelpersKickoffIndexingTask(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to kickoff indexing task: %v", err)
		}

		if newTask == nil || newTask.ID == existingTask.ID {
			t.Error("Expected a new task to be created, different from the existing one")
		}
	})

	t.Run("kicks off a new fast indexing task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		task, err := ta.FastIndexingHelpersKickoffIndexingTask(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to kickoff indexing task: %v", err)
		}

		if task == nil || task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("Expected task.SourceID=%d, got %v", source.ID, task.SourceID)
		}

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(workers) != 1 {
			t.Errorf("Expected 1 worker, got %d", len(workers))
		}
		args := workers[0].ArgsMap()
		idVal, ok := args["id"]
		if !ok {
			t.Errorf("Expected id in worker args, got %v", args)
		}
		// Handle both json.Number and int64 types
		var workerID int64
		switch v := idVal.(type) {
		case json.Number:
			n, _ := v.Int64()
			workerID = n
		case int64:
			workerID = v
		case float64:
			workerID = int64(v)
		}
		if workerID != source.ID {
			t.Errorf("Expected worker.Args[id]=%d, got %v", source.ID, idVal)
		}
	})
}

func TestFastIndexingHelpers_IndexAndKickoffDownloads(t *testing.T) {
	t.Run("enqueues a worker for each new media_id in the source's RSS feed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 media item, got %d", len(items))
		}

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(workers) != 1 {
			t.Errorf("Expected 1 worker, got %d", len(workers))
		}
		args := workers[0].ArgsMap()
		idVal, ok := args["id"]
		if !ok {
			t.Errorf("Expected id in worker args, got %v", args)
		}
		var workerID int64
		switch v := idVal.(type) {
		case json.Number:
			n, _ := v.Int64()
			workerID = n
		case int64:
			workerID = v
		case float64:
			workerID = int64(v)
		}
		if workerID != items[0].ID {
			t.Errorf("Expected worker.Args[id]=%d, got %v", items[0].ID, idVal)
		}
		if workers[0].Priority != 0 {
			t.Errorf("Expected priority=0, got %d", workers[0].Priority)
		}
	})

	t.Run("does not enqueue a new worker for the source's media IDs we already know about", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		// Create media item with media_id "test_1" (what RSS feed will return)
		// When we check for existing media items by RSS ID, we should find this one
		existingItem := coretest.MediaItemFixture(t, ta, core.Attrs{
			"source_id": source.ID,
			"media_id":  "test_1",
		})
		if existingItem.MediaID != "test_1" {
			t.Fatalf("Fixture media item has wrong media_id: %s", existingItem.MediaID)
		}

		// yt-dlp should NOT be called since we find the existing media item by RSS ID
		// So don't set up any yt-dlp expectation

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		// RSS feed returns "test_1", and we already have media_id="test_1"
		// So no new items should be created and no yt-dlp call should happen
		if len(items) != 0 {
			t.Errorf("Expected 0 new media items (already have test_1), got %d", len(items))
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
	})

	t.Run("kicks off a download task for all pending media but at a lower priority", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		pendingItem := coretest.MediaItemFixture(t, ta, core.Attrs{
			"source_id":      source.ID,
			"media_filepath": nil,
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 media item, got %d", len(items))
		}

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(workers) != 2 {
			t.Errorf("Expected 2 workers (new + pending), got %d", len(workers))
		}

		// First worker should be the pending item with priority 1
		args := workers[0].ArgsMap()
		idVal, ok := args["id"]
		if !ok {
			t.Errorf("Expected id in worker args, got %v", args)
		}
		var workerID int64
		switch v := idVal.(type) {
		case json.Number:
			n, _ := v.Int64()
			workerID = n
		case int64:
			workerID = v
		case float64:
			workerID = int64(v)
		}
		if workerID != pendingItem.ID {
			t.Errorf("Expected first worker to be for pending item %d, got %v", pendingItem.ID, idVal)
		}
		if workers[0].Priority != 1 {
			t.Errorf("Expected pending item priority=1, got %d", workers[0].Priority)
		}
	})

	t.Run("returns the found media items", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 media item, got %d", len(items))
		}
		if items[0].MediaID != "video1" {
			t.Errorf("Expected media_id=video1, got %s", items[0].MediaID)
		}
	})

	t.Run("does not enqueue a download job if the source does not allow it", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"download_media": false})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 media item, got %d", len(items))
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
	})

	t.Run("creates a download task record", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 media item, got %d", len(items))
		}

		// Get jobs for this media item with the media download worker
		allJobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(allJobs) < 1 {
			t.Fatal("Expected at least one download job")
		}

		// Check if there's a job for this media item
		found := false
		for _, job := range allJobs {
			args := job.ArgsMap()
			if idVal, ok := args["id"]; ok {
				var jobID int64
				switch v := idVal.(type) {
				case json.Number:
					n, _ := v.Int64()
					jobID = n
				case int64:
					jobID = v
				case float64:
					jobID = int64(v)
				}
				if jobID == items[0].ID {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("Expected a download job for media item %d", items[0].ID)
		}
	})

	t.Run("passes the source's download options to the yt-dlp runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		optionsChecked := false
		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			// Check if output option is present
			if val, ok := opts.Get("output"); !ok || val == nil {
				t.Error("Expected output option in opts")
			}
			// Check if remux_video option is present
			found := false
			for _, opt := range opts {
				if opt.Key == "remux_video" {
					found = true
					break
				}
			}
			if !found {
				t.Error("Expected remux_video option in opts")
			}
			optionsChecked = true
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if !optionsChecked {
			t.Error("YtDlpRunner.Run was not called")
		}
	})

	t.Run("does not enqueue a download job if the media item does not match the format rules", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"shorts_behaviour": "exclude"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			// Return media attributes with aspect ratio that indicates a short
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

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 media item, got %d", len(items))
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
	})

	t.Run("does not blow up if a media item cannot be created", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return "{}", nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 0 {
			t.Errorf("Expected 0 media items when creation fails, got %d", len(items))
		}
	})

	t.Run("does not blow up if a media item causes a yt-dlp error", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return "", &core.CommandError{Output: "error message", Status: 1}
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 0 {
			t.Errorf("Expected 0 media items when yt-dlp error occurs, got %d", len(items))
		}
	})
}

func TestFastIndexingHelpers_IndexAndKickoffDownloads_WhenTestingCookies(t *testing.T) {
	t.Run("sets use_cookies if the source uses cookies", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"cookie_behaviour": "all_operations"})

		cookieChecked := false
		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			// Check if use_cookies is in addlOpts
			if val, ok := addlOpts.Get("use_cookies"); !ok || val != true {
				t.Errorf("Expected use_cookies=true in addlOpts, got %v", val)
			}
			cookieChecked = true
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if !cookieChecked {
			t.Error("YtDlpRunner.Run was not called")
		}
	})

	t.Run("does not set use_cookies if the source uses cookies when needed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"cookie_behaviour": "when_needed"})

		cookieChecked := false
		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			// Check if use_cookies is in addlOpts
			if val, ok := addlOpts.Get("use_cookies"); !ok || val != false {
				t.Errorf("Expected use_cookies=false in addlOpts, got %v", val)
			}
			cookieChecked = true
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if !cookieChecked {
			t.Error("YtDlpRunner.Run was not called")
		}
	})

	t.Run("does not set use_cookies if the source does not use cookies", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"cookie_behaviour": "disabled"})

		cookieChecked := false
		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			// Check if use_cookies is in addlOpts
			if val, ok := addlOpts.Get("use_cookies"); !ok || val != false {
				t.Errorf("Expected use_cookies=false in addlOpts, got %v", val)
			}
			cookieChecked = true
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if !cookieChecked {
			t.Error("YtDlpRunner.Run was not called")
		}
	})
}

func TestFastIndexingHelpers_IndexAndKickoffDownloads_WhenTestingBackends(t *testing.T) {
	t.Run("uses the YouTube API if it is enabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "test_key")})

		apiUsed := false
		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if strings.Contains(url, "https://youtube.googleapis.com/youtube/v3/playlistItems") {
				apiUsed = true
			}
			return "{}", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if !apiUsed {
			t.Error("Expected YouTube API to be used")
		}
	})

	t.Run("the YouTube API creates records as expected", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "test_key")})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return coretest.MediaAttributesReturnFixture(), nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if strings.Contains(url, "youtube.googleapis.com") {
				response := map[string]any{
					"items": []map[string]any{
						{
							"contentDetails": map[string]any{
								"videoId": "test_1",
							},
						},
					},
				}
				jsonStr, _ := json.Marshal(response)
				return string(jsonStr), nil
			}
			return "", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 media item, got %d", len(items))
		}
	})

	t.Run("RSS is used as a backup if the API fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "test_key")})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return coretest.MediaAttributesReturnFixture(), nil
		})

		// Use Stub so it can handle multiple calls
		callCount := 0
		ta.HTTPMock.Get.Stub(func(url string, headers, opts core.KW) (string, error) {
			callCount++
			if callCount == 1 && strings.Contains(url, "youtube.googleapis.com") {
				// First call to API fails
				return "", &core.CommandError{Output: "API failed", Status: 1}
			}
			if strings.Contains(url, "youtube.com/feeds") {
				return "<yt:videoId>test_1</yt:videoId>", nil
			}
			return "", nil
		})

		items, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 media item (RSS backup), got %d", len(items))
		}
	})

	t.Run("RSS is used if the API is not enabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", nil)})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return coretest.MediaAttributesReturnFixture(), nil
		})

		rssUsed := false
		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if strings.Contains(url, "https://www.youtube.com/feeds/videos.xml") {
				rssUsed = true
			}
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		_, err := ta.FastIndexingHelpersIndexAndKickoffDownloads(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to index and kickoff downloads: %v", err)
		}

		if !rssUsed {
			t.Error("Expected RSS feed to be used")
		}
	})
}
