package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestFastIndexingWorker_KickoffWithTask(t *testing.T) {
	t.Run("starts the worker", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(workers) != 0 {
			t.Errorf("Expected 0 workers initially, got %d", len(workers))
		}

		_, err := ta.App.FastIndexingWorkerKickoffWithTask(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("Failed to kickoff: %v", err)
		}

		workers = ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(workers) != 1 {
			t.Errorf("Expected 1 worker, got %d", len(workers))
		}
	})

	t.Run("attaches a task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		task, err := ta.App.FastIndexingWorkerKickoffWithTask(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("Failed to kickoff: %v", err)
		}

		if *task.SourceID != source.ID {
			t.Errorf("Expected task.SourceID=%d, got %d", source.ID, *task.SourceID)
		}
	})
}

func TestFastIndexingWorker_Perform(t *testing.T) {
	t.Run("calls out to Youtube RSS if enabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}
	})

	t.Run("reschedules itself if fast indexing is enabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}

		// Should reschedule itself
		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(jobs) == 0 {
			t.Error("Expected a scheduled job after perform")
		}
	})

	t.Run("does not reschedule if that would create a duplicate job", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		ta.HTTPMock.Get.Stub(func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform first time: %v", err)
		}

		// Perform again
		job2, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job2); err != nil {
			t.Errorf("Failed to perform second time: %v", err)
		}

		// Should only have 1 job (duplicate prevention)
		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(jobs) != 1 {
			t.Errorf("Expected 1 job (duplicate prevention), got %d", len(jobs))
		}
	})

	t.Run("does not call out to Youtube RSS if disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": false})

		ta.HTTPMock.Get.ExpectN(0, func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}
	})

	t.Run("does not reschedule itself if fast indexing is disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": false})

		err := ta.Oban.PerformJob(ta.Ctx, core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 0 {
			t.Errorf("expected no rescheduled jobs, got %d", len(jobs))
		}
	})

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		ta := coretest.NewApp(t)

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": 0})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Expected no error for missing source, got %v", err)
		}
	})
}

func TestFastIndexingWorker_Perform_WhenTestingNotifications(t *testing.T) {
	setupNotifications := func(ta *coretest.TestApp) {
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("apprise_server", "server_1")})
	}

	t.Run("sends a notification if new media was found", func(t *testing.T) {
		ta := coretest.NewApp(t)
		setupNotifications(ta)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		metadata, err := coretest.RenderMetadata("media_metadata")
		if err != nil {
			t.Fatalf("Failed to render metadata: %v", err)
		}

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return metadata, nil
		})

		ta.AppriseMock.Run.Expect(func(servers []string, opts core.KW) error {
			if len(servers) != 1 || servers[0] != "server_1" {
				t.Errorf("expected servers to be [\"server_1\"], got %v", servers)
			}
			if title, _ := opts.Get("title"); title == nil {
				t.Error("expected title to be set")
			} else if _, ok := title.(string); !ok {
				t.Error("expected title to be a string")
			}
			if body, _ := opts.Get("body"); body == nil {
				t.Error("expected body to be set")
			} else if _, ok := body.(string); !ok {
				t.Error("expected body to be a string")
			}
			return nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}
	})

	t.Run("doesn't send a notification if new media is not found", func(t *testing.T) {
		ta := coretest.NewApp(t)
		setupNotifications(ta)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		ta.AppriseMock.Run.ExpectN(0, func(endpoints []string, opts core.KW) error {
			return nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}
	})

	t.Run("doesn't send a notification if the source doesn't download media", func(t *testing.T) {
		ta := coretest.NewApp(t)
		setupNotifications(ta)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true, "download_media": false})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		metadata, err := coretest.RenderMetadata("media_metadata")
		if err != nil {
			t.Fatalf("Failed to render metadata: %v", err)
		}

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return metadata, nil
		})

		ta.AppriseMock.Run.ExpectN(0, func(endpoints []string, opts core.KW) error {
			return nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}
	})

	t.Run("doesn't send a notification if the media isn't pending download", func(t *testing.T) {
		ta := coretest.NewApp(t)
		setupNotifications(ta)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true, "title_filter_regex": "foobar"})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId>", nil
		})

		metadata, err := coretest.RenderMetadata("media_metadata")
		if err != nil {
			t.Fatalf("Failed to render metadata: %v", err)
		}

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return metadata, nil
		})

		ta.AppriseMock.Run.ExpectN(0, func(endpoints []string, opts core.KW) error {
			return nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}
	})
}
