package app_test

import (
	"os"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

type indexMockFn = func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error)

// indexMock returns the fixture index response, after letting check inspect
// the call.
func indexMock(check func(opts ytdlp.Args, addl ytdlp.CallOptions)) indexMockFn {
	return func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
		if check != nil {
			check(opts, addl)
		}
		return apptest.SourceAttributesReturnFixture(), nil
	}
}

// indexSource runs the indexing helper and returns its per-item results.
func indexSource(t *testing.T, ta *apptest.TestApp, source *store.Source, force bool) []any {
	t.Helper()
	result, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, force)
	must(t, err)
	return result
}

// mediaItemsOf picks the media items out of an indexing result.
func mediaItemsOf(result []any) []*store.MediaItem {
	var items []*store.MediaItem
	for _, r := range result {
		if mi, ok := r.(*store.MediaItem); ok {
			items = append(items, mi)
		}
	}
	return items
}

func allMediaItems(t *testing.T, ta *apptest.TestApp) []*store.MediaItem {
	t.Helper()
	items, err := store.All[store.MediaItem](ta.Ctx, ta.App.Q(ta.Ctx), store.From[store.MediaItem]("mi"))
	must(t, err)
	return items
}

func enqueuedDownloads(t *testing.T, ta *apptest.TestApp) int {
	t.Helper()
	return len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName}))
}

// setExecuting marks the task's job as executing.
func setExecuting(t *testing.T, ta *apptest.TestApp, task *store.Task) {
	t.Helper()
	mustOK(t)(ta.DB.ExecContext(ta.Ctx, `UPDATE oban_jobs SET state = 'executing' WHERE id = ?`, task.JobID))
}

func TestSlowIndexingHelpers_KickoffIndexingTask(t *testing.T) {
	kickoff := func(t *testing.T, ta *apptest.TestApp, source *store.Source, args map[string]any) *store.Task {
		t.Helper()
		task, err := ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, args)
		must(t, err)
		if task == nil {
			t.Fatal("expected task, got nil")
		}
		return task
	}
	ago := func(minutes int) store.SourceParams {
		return store.SourceParams{IndexFrequencyMinutes: store.Ptr(30), LastIndexedAt: store.Ptr(apptest.NowMinus(minutes, "minutes"))}
	}

	for _, c := range []struct {
		name     string
		src      store.SourceParams
		args     map[string]any
		min, max time.Duration // bounds for how far in the future the job is scheduled
	}{
		{"schedules a job", store.SourceParams{IndexFrequencyMinutes: store.Ptr(1)}, map[string]any{}, -time.Second, time.Second},
		{"schedules a job for the future based on when the source was last indexed", ago(5), map[string]any{}, 24 * time.Minute, 26 * time.Minute},
		{"schedules a job immediately if the source was indexed far in the past", ago(60), map[string]any{}, -time.Second, time.Second},
		{"schedules a job immediately if the source has never been indexed", store.SourceParams{IndexFrequencyMinutes: store.Ptr(30), Clear: store.ClearLastIndexedAt}, map[string]any{}, -time.Second, time.Second},
		{"schedules a job immediately if the user is forcing an index", ago(5), map[string]any{"force": true}, -time.Second, time.Second},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, c.src)

			kickoff(t, ta, source, c.args)

			args := map[string]any{"id": source.ID}
			for k, v := range c.args {
				args[k] = v
			}
			jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: args})
			if len(jobs) != 1 {
				t.Fatalf("expected 1 job, got %d", len(jobs))
			}
			if diff := jobs[0].ScheduledAt.Sub(apptest.Now()); diff < c.min || diff > c.max {
				t.Errorf("expected scheduled between %v and %v from now, got %v", c.min, c.max, diff)
			}
		})
	}

	t.Run("creates and attaches a task", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(1)})

		task := kickoff(t, ta, source, map[string]any{})

		if *task.SourceID != source.ID {
			t.Errorf("expected task.SourceID to be %d, got %d", source.ID, *task.SourceID)
		}
	})

	t.Run("deletes any pending media collection tasks for the source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		task := pendingTask(t, ta, source, app.MediaCollectionIndexingWorkerName)

		kickoff(t, ta, source, map[string]any{})

		wantTaskGone(t, ta, task)
	})

	t.Run("deletes any executing media collection tasks for the source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		task := pendingTask(t, ta, source, app.MediaCollectionIndexingWorkerName)
		setExecuting(t, ta, task)

		kickoff(t, ta, source, map[string]any{})

		wantTaskGone(t, ta, task)
	})
}

func TestSlowIndexingHelpers_DeleteIndexingTasks(t *testing.T) {
	for _, c := range []struct{ name, worker string }{
		{"deletes slow indexing tasks for the source", app.MediaCollectionIndexingWorkerName},
		{"deletes fast indexing tasks for the source", app.FastIndexingWorkerName},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, store.SourceParams{})
			pendingTask(t, ta, source, c.worker)

			if jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: c.worker}); len(jobs) == 0 {
				t.Fatal("expected job to be enqueued")
			}

			must(t, ta.App.SlowIndexingHelpersDeleteIndexingTasks(ta.Ctx, source, false))

			if jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: c.worker}); len(jobs) != 0 {
				t.Fatalf("expected no jobs, got %d", len(jobs))
			}
		})
	}

	for _, c := range []struct {
		name  string
		force bool
	}{
		{"doesn't normally delete currently executing tasks", false},
		{"can optionally delete currently executing tasks", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, store.SourceParams{})
			task := pendingTask(t, ta, source, app.MediaCollectionIndexingWorkerName)
			setExecuting(t, ta, task)

			_, err := ta.App.GetTaskBang(ta.Ctx, task.ID)
			must(t, err)

			must(t, ta.App.SlowIndexingHelpersDeleteIndexingTasks(ta.Ctx, source, c.force))

			_, err = ta.App.GetTaskBang(ta.Ctx, task.ID)
			if c.force && err == nil {
				t.Fatal("expected error reloading task, but got nil")
			}
			if !c.force {
				must(t, err)
			}
		})
	}
}

func TestSlowIndexingHelpers_IndexAndEnqueueDownloadForMediaItems(t *testing.T) {
	t.Run("creates a media_item record for each media ID returned, attached to the given source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		ta.YtDlpMock.Run.Expect(indexMock(nil))

		items := mediaItemsOf(indexSource(t, ta, source, false))

		if len(items) != 3 {
			t.Errorf("expected 3 media items, got %d", len(items))
		}
		for _, mi := range items {
			if mi.SourceID != source.ID {
				t.Errorf("expected source_id to be %d, got %d", source.ID, mi.SourceID)
			}
		}
	})

	t.Run("won't duplicate media_items based on media_id and source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		ta.YtDlpMock.Run.ExpectN(2, indexMock(nil))

		indexSource(t, ta, source, false)
		indexSource(t, ta, source, false)

		if items := allMediaItems(t, ta); len(items) != 3 {
			t.Errorf("expected 3 media items, got %d", len(items))
		}
	})

	t.Run("can duplicate media_ids for different sources", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source1 := apptest.SourceFixture(t, ta, store.SourceParams{})
		source2 := apptest.SourceFixture(t, ta, store.SourceParams{})
		ta.YtDlpMock.Run.ExpectN(2, indexMock(nil))

		n1 := len(mediaItemsOf(indexSource(t, ta, source1, false)))
		n2 := len(mediaItemsOf(indexSource(t, ta, source2, false)))

		if n1 != 3 || n2 != 3 {
			t.Errorf("expected 3 media items for each source, got %d and %d", n1, n2)
		}
	})

	t.Run("returns a list of media_items", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		ta.YtDlpMock.Run.ExpectN(2, indexMock(nil))

		items1 := mediaItemsOf(indexSource(t, ta, source, false))
		items2 := mediaItemsOf(indexSource(t, ta, source, false))

		if len(items1) != len(items2) {
			t.Fatalf("expected same number of items, got %d and %d", len(items1), len(items2))
		}
		for i := range items1 {
			if items1[i].ID != items2[i].ID {
				t.Errorf("expected ids to match at index %d, got %d != %d", i, items1[i].ID, items2[i].ID)
			}
		}
	})

	t.Run("updates the source's last_indexed_at field", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		if source.LastIndexedAt != nil {
			t.Errorf("expected last_indexed_at to be nil initially, got %v", source.LastIndexedAt)
		}
		ta.YtDlpMock.Run.Expect(indexMock(nil))

		indexSource(t, ta, source, false)

		reloadedSource, err := ta.App.GetSource(ta.Ctx, source.ID)
		must(t, err)
		if reloadedSource.LastIndexedAt == nil {
			t.Fatal("expected last_indexed_at to be set")
		}
		if diff := time.Now().UTC().Sub(reloadedSource.LastIndexedAt.Time); diff < 0 || diff > 2*time.Second {
			t.Errorf("expected last_indexed_at to be recent, got diff %v", diff)
		}
	})

	t.Run("enqueues a job for each pending media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.Expect(indexMock(nil))

		indexSource(t, ta, source, false)

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
		if len(jobs) == 0 {
			t.Fatal("expected job to be enqueued for pending media item")
		}
	})

	t.Run("does not attach tasks if the source is set to not download", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{DownloadMedia: store.Ptr(false)})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.Expect(indexMock(nil))

		indexSource(t, ta, source, false)

		tasks, err := ta.App.ListTasksFor(ta.Ctx, mediaItem, nil, nil)
		must(t, err)
		if len(tasks) != 0 {
			t.Errorf("expected no tasks for media item, got %d", len(tasks))
		}
	})

	for _, c := range []struct{ name, response string }{
		{"doesn't blow up if a media item cannot be coerced into a struct", `{"id":"video3","title":"Video 3","live_status":"not_live","description":"desc3","original_url":null,"aspect_ratio":null,"duration":null,"upload_date":null}`},
		// a disallowed title - see store.MediaItem changeset or issue #549
		{"doesn't blow up if the media item cannot be saved", `{"id":"video1","title":"youtube video #123","original_url":"https://example.com/video1","live_status":"not_live","description":"desc1","aspect_ratio":1.67,"duration":12.34,"upload_date":"20210101"}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, store.SourceParams{})
			ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
				return c.response, nil
			})

			result := indexSource(t, ta, source, false)

			if len(result) != 1 {
				t.Fatalf("expected 1 result, got %d", len(result))
			}
			if _, ok := result[0].(store.ValidationErrors); !ok {
				t.Errorf("expected a store.ValidationErrors result, got %T", result[0])
			}
		})
	}

	t.Run("passes the source's download options to the yt-dlp runner", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		ta.YtDlpMock.Run.Expect(indexMock(func(opts ytdlp.Args, _ ytdlp.CallOptions) {
			output, hasOutput := opts.Get("output")
			if !hasOutput {
				t.Error("expected output option to be set")
			} else if s, _ := output.(string); s == "" {
				t.Error("expected output option to be a non-empty path")
			}

			remux, hasRemux := opts.Get("remux_video")
			if !hasRemux || remux != "mp4" {
				t.Errorf("expected remux_video to be mp4, got %v (present=%v)", remux, hasRemux)
			}
		}))

		indexSource(t, ta, source, false)
	})
}

func TestSlowIndexingHelpers_IndexAndEnqueueDownloadForMediaItems_Cookies(t *testing.T) {
	for _, c := range []struct {
		name      string
		behaviour store.SourceCookieBehaviour
		want      bool
	}{
		{"sets use_cookies if the source uses cookies", store.SourceCookieBehaviourAllOperations, true},
		{"sets use_cookies if the source uses cookies when needed", store.SourceCookieBehaviourWhenNeeded, true},
		{"doesn't set use_cookies if the source doesn't use cookies", store.SourceCookieBehaviourDisabled, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, store.SourceParams{CookieBehaviour: store.Ptr(c.behaviour)})
			ta.YtDlpMock.Run.Expect(indexMock(func(_ ytdlp.Args, addl ytdlp.CallOptions) {
				if addl.UseCookies != c.want {
					t.Errorf("expected use_cookies to be %v in addl opts, got %v", c.want, addl.UseCookies)
				}
			}))

			indexSource(t, ta, source, false)
		})
	}
}

func TestSlowIndexingHelpers_IndexAndEnqueueDownloadForMediaItems_FileWatcher(t *testing.T) {
	// setup creates a source whose index run writes contents to the output
	// file (which the file watcher picks up) and returns ret synchronously.
	setup := func(t *testing.T, profile store.MediaProfileParams, src store.SourceParams, contents, ret string) (*apptest.TestApp, *store.Source) {
		ta := apptest.NewApp(t)
		src.MediaProfileID = store.Ptr(apptest.MediaProfileFixture(t, ta, profile).ID)
		source := apptest.SourceFixture(t, ta, src)
		pollInterval := ta.App.Config.FileWatcherPollInterval

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			must(t, os.WriteFile(addl.OutputFilepath, []byte(contents), 0o644))
			time.Sleep(pollInterval * 2)
			return ret, nil
		})
		return ta, source
	}
	fixture := apptest.SourceAttributesReturnFixture()

	t.Run("creates a new media item for everything already in the file", func(t *testing.T) {
		// The synchronous call returns "" (creating no records), so anything
		// created came from the file watcher.
		ta, source := setup(t, store.MediaProfileParams{}, store.SourceParams{}, fixture, "")
		if n := len(allMediaItems(t, ta)); n != 0 {
			t.Fatalf("expected 0 media items before indexing, got %d", n)
		}

		indexSource(t, ta, source, false)

		if n := len(allMediaItems(t, ta)); n != 3 {
			t.Errorf("expected 3 media items after indexing, got %d", n)
		}
	})

	t.Run("enqueues a download for everything already in the file", func(t *testing.T) {
		ta, source := setup(t, store.MediaProfileParams{}, store.SourceParams{}, fixture, "")
		if n := enqueuedDownloads(t, ta); n != 0 {
			t.Fatalf("expected no download jobs enqueued before indexing, got %d", n)
		}

		indexSource(t, ta, source, false)

		if enqueuedDownloads(t, ta) == 0 {
			t.Fatal("expected download job to be enqueued")
		}
	})

	t.Run("does not enqueue downloads if the source is set to not download", func(t *testing.T) {
		ta, source := setup(t, store.MediaProfileParams{}, store.SourceParams{DownloadMedia: store.Ptr(false)}, fixture, "")

		indexSource(t, ta, source, false)

		if n := enqueuedDownloads(t, ta); n != 0 {
			t.Errorf("expected no download jobs enqueued, got %d", n)
		}
	})

	t.Run("does not enqueue downloads for media that doesn't match the profile's format options", func(t *testing.T) {
		contents := `{"id":"video2","title":"Video 2","original_url":"https://example.com/shorts/video2","live_status":"is_live","description":"desc2","aspect_ratio":1.67,"duration":345.67,"upload_date":"20210101"}`
		ta, source := setup(t, store.MediaProfileParams{ShortsBehaviour: store.Ptr(store.MediaProfileShortsBehaviourExclude)}, store.SourceParams{}, contents, "")

		indexSource(t, ta, source, false)

		if n := enqueuedDownloads(t, ta); n != 0 {
			t.Errorf("expected no download jobs enqueued, got %d", n)
		}
	})

	t.Run("does not enqueue multiple download jobs for the same media items", func(t *testing.T) {
		// The runner also returns the final result (like real usage), so the
		// media items and jobs are attempted a second time.
		ta, source := setup(t, store.MediaProfileParams{}, store.SourceParams{}, fixture, fixture)

		indexSource(t, ta, source, false)

		if items := allMediaItems(t, ta); len(items) != 3 {
			t.Errorf("expected 3 media items, got %d", len(items))
		}
		if n := enqueuedDownloads(t, ta); n != 3 {
			t.Errorf("expected 3 download jobs enqueued, got %d", n)
		}
	})

	t.Run("does not blow up if the file returns invalid json", func(t *testing.T) {
		ta, source := setup(t, store.MediaProfileParams{}, store.SourceParams{}, "INVALID", "")

		if result := indexSource(t, ta, source, false); len(result) != 0 {
			t.Errorf("expected no results, got %d", len(result))
		}
	})
}

func TestSlowIndexingHelpers_IndexAndEnqueueDownloadForMediaItems_DownloadArchive(t *testing.T) {
	channel, playlist := store.Ptr(store.SourceCollectionTypeChannel), store.Ptr(store.SourceCollectionTypePlaylist)
	archiveOpts := []string{"break_on_existing*", "download_archive*"}

	for _, c := range []struct {
		name        string
		src         store.SourceParams
		force       bool
		wantArchive bool
	}{
		{"a download archive is used if the source is a channel that has been indexed before", store.SourceParams{CollectionType: channel, LastIndexedAt: store.Ptr(apptest.Now())}, false, true},
		{"a download archive is not used if the source is not a channel", store.SourceParams{CollectionType: playlist, LastIndexedAt: store.Ptr(apptest.Now())}, false, false},
		{"a download archive is not used if the source has never been indexed before", store.SourceParams{CollectionType: channel, Clear: store.ClearLastIndexedAt}, false, false},
		{"a download archive is not used if the index has been forced to run", store.SourceParams{CollectionType: channel}, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, c.src)
			ta.YtDlpMock.Run.Expect(indexMock(func(opts ytdlp.Args, _ ytdlp.CallOptions) {
				if c.wantArchive {
					wantOpts(t, opts, archiveOpts, nil)
				} else {
					wantOpts(t, opts, nil, archiveOpts)
				}
			}))

			indexSource(t, ta, source, c.force)
		})
	}

	t.Run("the download archive is formatted correctly and contains the right video", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{CollectionType: channel, LastIndexedAt: store.Ptr(apptest.Now())})

		var mediaItems []*store.MediaItem
		for n := 1; n <= 21; n++ {
			mediaItems = append(mediaItems, apptest.MediaItemFixture(t, ta, store.MediaItemParams{
				SourceID:   store.Ptr(source.ID),
				UploadedAt: store.Ptr(apptest.NowMinus(n, "days")),
			}))
		}
		lastMediaItem := mediaItems[len(mediaItems)-1]

		ta.YtDlpMock.Run.Expect(indexMock(func(opts ytdlp.Args, _ ytdlp.CallOptions) {
			archiveFile, _ := opts.Get("download_archive")
			contents, err := os.ReadFile(archiveFile.(string))
			must(t, err)
			expected := "youtube " + lastMediaItem.MediaID
			if string(contents) != expected {
				t.Errorf("expected archive contents %q, got %q", expected, string(contents))
			}
		}))

		indexSource(t, ta, source, false)
	})
}

// Found by running the Go binary against a real database: with an output
// path override, nothing else preloads media_profile, and building quality
// options dereferenced a nil profile.
func TestSlowIndexingHelpers_IndexAndEnqueuePreloadsMediaProfile(t *testing.T) {
	t.Run("works for a freshly loaded source with an output path override", func(t *testing.T) {
		ta := apptest.NewApp(t)
		fixture := apptest.SourceFixture(t, ta, store.SourceParams{OutputPathTemplateOverride: store.Ptr("/{{ title }}.{{ ext }}")})
		source, err := ta.App.GetSource(ta.Ctx, fixture.ID)
		must(t, err)
		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if _, ok := opts.Get("format_sort"); !ok {
				t.Error("expected the media profile's quality options")
			}
			return "", nil
		})

		indexSource(t, ta, source, false)
	})
}
