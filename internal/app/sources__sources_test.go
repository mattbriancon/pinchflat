package app_test

import (
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/cmdrun"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

var invalidSourceAttrs = store.SourceParams{CollectionID: store.Ptr("")}

func sourcesTestPlaylistReturn() string {
	jsonStr, _ := db.EncodeJSON(map[string]any{
		"channel":        nil,
		"channel_id":     nil,
		"playlist_id":    "some_playlist_id_" + strconv.Itoa(rand.Intn(1000000)),
		"playlist_title": "some playlist name",
	})
	return jsonStr
}

func sourcesTestChannelReturn() string {
	channelID := "some_channel_id_" + strconv.Itoa(rand.Intn(1000000))
	jsonStr, _ := db.EncodeJSON(map[string]any{
		"channel":        "some channel name",
		"channel_id":     channelID,
		"playlist_id":    channelID,
		"playlist_title": "some channel name - videos",
	})
	return jsonStr
}

func sourcesTestPlaylistMock(_ string, _ string, _ ytdlp.Args, _ string, _ ytdlp.CallOptions) (string, error) {
	return sourcesTestPlaylistReturn(), nil
}

func sourcesTestChannelMock(_ string, _ string, _ ytdlp.Args, _ string, _ ytdlp.CallOptions) (string, error) {
	return sourcesTestChannelReturn(), nil
}

func TestSources_Schema(t *testing.T) {
	t.Run("source_metadata is deleted when the source is deleted", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{
			Metadata: &store.SourceMetadata{MetadataFilepath: "/metadata.json.gz"},
		})
		src, err := ta.PreloadSourceMetadata(ta.Ctx, src)
		must(t, err)
		metadata := src.Metadata
		if metadata == nil {
			t.Fatal("expected source to have metadata")
		}

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, false))

		if _, err := store.Get[store.SourceMetadata](ta.Ctx, ta.Q(ta.Ctx), metadata.ID); err != store.ErrNotFound {
			t.Errorf("expected metadata to be deleted, got err=%v", err)
		}
	})

	t.Run("can be JSON encoded without error", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})

		if _, err := json.Marshal(src); err != nil {
			t.Errorf("json.Marshal failed: %v", err)
		}
	})
}

func TestSources_OutputPathTemplate(t *testing.T) {
	t.Run("returns the source's override if present", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{
			OutputPathTemplateOverride: store.Ptr("/override/{{ title }}.{{ ext }}"),
		})

		got := ta.App.SourcesOutputPathTemplate(ta.Ctx, src)
		if got != "/override/{{ title }}.{{ ext }}" {
			t.Errorf("expected override template, got %q", got)
		}
	})

	t.Run("returns the media profile's template if no override is present", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("/profile/{{ title }}.{{ ext }}")})
		src := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(mediaProfile.ID)})

		got := ta.App.SourcesOutputPathTemplate(ta.Ctx, src)
		if got != "/profile/{{ title }}.{{ ext }}" {
			t.Errorf("expected profile template, got %q", got)
		}
	})

	t.Run("Treats empty strings as being blank", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("/profile/{{ title }}.{{ ext }}")})
		src := apptest.SourceFixture(t, ta, store.SourceParams{
			MediaProfileID:             store.Ptr(mediaProfile.ID),
			OutputPathTemplateOverride: store.Ptr("  "),
		})

		got := ta.App.SourcesOutputPathTemplate(ta.Ctx, src)
		if got != "/profile/{{ title }}.{{ ext }}" {
			t.Errorf("expected profile template, got %q", got)
		}
	})
}

func TestSources_UseCookies(t *testing.T) {
	for _, c := range []struct {
		name      string
		behaviour store.SourceCookieBehaviour
		action    string
		want      bool
	}{
		{"returns true if the source has been set to use cookies", store.SourceCookieBehaviourAllOperations, "downloading", true},
		{"returns false if the source has not been set to use cookies", store.SourceCookieBehaviourDisabled, "downloading", false},
		{"returns true if the action is indexing and the source is set to :when_needed", store.SourceCookieBehaviourWhenNeeded, "indexing", true},
		{"returns false if the action is downloading and the source is set to :when_needed", store.SourceCookieBehaviourWhenNeeded, "downloading", false},
		{"returns true if the action is error_recovery and the source is set to :when_needed", store.SourceCookieBehaviourWhenNeeded, "error_recovery", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := store.UseCookies(&store.Source{CookieBehaviour: c.behaviour}, c.action); got != c.want {
				t.Errorf("UseCookies = %v, want %v", got, c.want)
			}
		})
	}
}

func TestSources_ListSources(t *testing.T) {
	t.Run("it returns all sources", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})

		got, err := ta.App.ListSources(ta.Ctx)
		must(t, err)
		if !reflect.DeepEqual(got, []*store.Source{src}) {
			t.Errorf("expected %+v, got %+v", src, got)
		}
	})
}

func TestSources_ListSourcesFor(t *testing.T) {
	t.Run("returns all sources for a given media profile", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		src := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(mediaProfile.ID)})

		got, err := ta.App.ListSourcesFor(ta.Ctx, mediaProfile)
		must(t, err)
		if !reflect.DeepEqual(got, []*store.Source{src}) {
			t.Errorf("expected %+v, got %+v", src, got)
		}
	})
}

func TestSources_GetSource(t *testing.T) {
	t.Run("it returns the source with given id", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})

		got, err := ta.App.GetSource(ta.Ctx, src.ID)
		must(t, err)
		if !reflect.DeepEqual(got, src) {
			t.Errorf("expected %+v, got %+v", src, got)
		}
	})
}

func TestSources_CreateSource(t *testing.T) {
	t.Run("automatically sets the UUID", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		if src.UUID == nil || len(*src.UUID) != 36 {
			t.Errorf("expected a 36-char UUID, got %v", src.UUID)
		}
	})

	t.Run("UUID is not writable by the user", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		if src.UUID == nil || len(*src.UUID) != 36 {
			t.Errorf("expected a 36-char UUID, got %v", src.UUID)
		}
	})

	t.Run("creates a source and adds name + ID from runner response for channels", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		if src.CollectionName != "some channel name" {
			t.Errorf("expected collection_name %q, got %q", "some channel name", src.CollectionName)
		}
		if !startsWith(src.CollectionID, "some_channel_id_") {
			t.Errorf("expected collection_id to start with some_channel_id_, got %q", src.CollectionID)
		}
	})

	t.Run("creates a source and adds name + ID for playlists", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestPlaylistMock)

		validAttrs := newSourceAttrs(t, ta)
		validAttrs.OriginalURL = store.Ptr("https://www.youtube.com/playlist?list=abc123")

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		if src.CollectionName != "some playlist name" {
			t.Errorf("expected collection_name %q, got %q", "some playlist name", src.CollectionName)
		}
		if !startsWith(src.CollectionID, "some_playlist_id_") {
			t.Errorf("expected collection_id to start with some_playlist_id_, got %q", src.CollectionID)
		}
	})

	t.Run("adds an error if the runner fails", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ ytdlp.Args, _ string, _ ytdlp.CallOptions) (string, error) {
			return "", &cmdrun.Error{Output: "some error", Status: 1}
		})

		validAttrs := newSourceAttrs(t, ta)

		_, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		errs := validationErrors(t, err)
		if !containsString(errs["original_url"], "could not fetch source details from URL") {
			t.Errorf("expected original_url error, got %+v", errs)
		}
	})

	t.Run("adds an error if the runner succeeds but the result was invalid JSON", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ ytdlp.Args, _ string, _ ytdlp.CallOptions) (string, error) {
			return "store.Not JSON", nil
		})

		validAttrs := newSourceAttrs(t, ta)

		_, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		errs := validationErrors(t, err)
		if !containsString(errs["original_url"], "could not fetch source details from URL") {
			t.Errorf("expected original_url error, got %+v", errs)
		}
	})

	t.Run("you can specify a custom custom_name", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)
		validAttrs.CustomName = store.Ptr("some custom name")

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		if src.CustomName != "some custom name" {
			t.Errorf("expected custom_name %q, got %q", "some custom name", src.CustomName)
		}
	})

	t.Run("friendly name is pulled from collection_name if not specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		if src.CustomName != "some channel name" {
			t.Errorf("expected custom_name %q, got %q", "some channel name", src.CustomName)
		}
	})

	t.Run("creation enforces uniqueness of collection_id scoped to the media_profile and title regex", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.ExpectN(2, func(_, _ string, _ ytdlp.Args, _ string, _ ytdlp.CallOptions) (string, error) {
			jsonStr, _ := db.EncodeJSON(map[string]any{
				"channel":        "some channel name",
				"channel_id":     "some_channel_id_12345678",
				"playlist_id":    "some_channel_id_12345678",
				"playlist_title": "some channel name - videos",
			})
			return jsonStr, nil
		})

		validOnceAttrs := newSourceAttrs(t, ta)
		validOnceAttrs.TitleFilterRegex = store.Ptr("")

		mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, validOnceAttrs, true))
		_, err := ta.App.SourcesCreateSource(ta.Ctx, validOnceAttrs, true)
		if err == nil {
			t.Fatal("expected the second create to fail")
		}
		errs, ok := store.AsValidationErrors(err)
		if !ok {
			t.Fatalf("expected a validation error, got %T: %v", err, err)
		}
		if !reflect.DeepEqual(errs, map[string][]string{"original_url": {"has already been taken"}}) {
			t.Errorf("expected a uniqueness error, got %v", errs)
		}
	})

	t.Run("creation lets you duplicate collection_ids and profiles as long as the regex is different", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.ExpectN(2, func(_, _ string, _ ytdlp.Args, _ string, _ ytdlp.CallOptions) (string, error) {
			jsonStr, _ := db.EncodeJSON(map[string]any{
				"channel":        "some channel name",
				"channel_id":     "some_channel_id_12345678",
				"playlist_id":    "some_channel_id_12345678",
				"playlist_title": "some channel name - videos",
			})
			return jsonStr, nil
		})

		validAttrs := newSourceAttrs(t, ta)
		source1Attrs, source2Attrs := validAttrs, validAttrs
		source1Attrs.TitleFilterRegex = store.Ptr("foo")
		source2Attrs.TitleFilterRegex = store.Ptr("bar")

		mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, source1Attrs, true))
		mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, source2Attrs, true))
	})

	t.Run("creation lets you duplicate collection_ids as long as the media profile is different", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.ExpectN(2, func(_, _ string, _ ytdlp.Args, _ string, _ ytdlp.CallOptions) (string, error) {
			jsonStr, _ := db.EncodeJSON(map[string]any{
				"channel":        "some channel name",
				"channel_id":     "some_channel_id_12345678",
				"playlist_id":    "some_channel_id_12345678",
				"playlist_title": "some channel name - videos",
			})
			return jsonStr, nil
		})

		validAttrs := store.SourceParams{
			OriginalURL:      store.Ptr("https://www.youtube.com/channel/abc123"),
			TitleFilterRegex: store.Ptr("TEST"),
		}
		source1Attrs, source2Attrs := validAttrs, validAttrs
		source1Attrs.MediaProfileID = store.Ptr(apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{}).ID)
		source2Attrs.MediaProfileID = store.Ptr(apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{}).ID)

		mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, source1Attrs, true))
		mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, source2Attrs, true))
	})

	t.Run("collection_type is inferred from source details", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)
		ta.YtDlpMock.Run.Expect(sourcesTestPlaylistMock)

		validAttrs := newSourceAttrs(t, ta)

		source1, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		source2, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)

		if source1.CollectionType != store.SourceCollectionTypeChannel {
			t.Errorf("expected channel, got %v", source1.CollectionType)
		}
		if source2.CollectionType != store.SourceCollectionTypePlaylist {
			t.Errorf("expected playlist, got %v", source2.CollectionType)
		}
	})

	t.Run("creation with invalid data returns error changeset", func(t *testing.T) {
		ta := apptest.NewApp(t)

		_, err := ta.App.SourcesCreateSource(ta.Ctx, invalidSourceAttrs, true)
		if err == nil {
			t.Fatal("expected an error")
		}
		if _, ok := store.AsValidationErrors(err); !ok {
			t.Fatalf("expected a validation error, got %T", err)
		}
	})

	t.Run("creation with invalid data fails fast and does not call the runner", func(t *testing.T) {
		ta := apptest.NewApp(t)

		_, err := ta.App.SourcesCreateSource(ta.Ctx, invalidSourceAttrs, true)
		if err == nil {
			t.Fatal("expected an error")
		}
		if _, ok := store.AsValidationErrors(err); !ok {
			t.Fatalf("expected a validation error, got %T", err)
		}
		if ta.YtDlpMock.Run.Calls() != 0 {
			t.Errorf("expected the runner to not be called, got %d calls", ta.YtDlpMock.Run.Calls())
		}
	})

	t.Run("creation will schedule the indexing task", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": src.ID}})
	})

	t.Run("creation will schedule a fast indexing job if the fast_index option is set", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)
		validAttrs.FastIndex = store.Ptr(true)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName, Args: map[string]any{"id": src.ID}})
	})

	t.Run("creation will not schedule a fast indexing job if the fast_index option is not set", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)
		validAttrs.FastIndex = store.Ptr(false)

		mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true))

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
	})

	t.Run("creation schedules an index test even if the index frequency is 0", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)
		validAttrs.IndexFrequencyMinutes = store.Ptr(0)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": src.ID}})
	})

	t.Run("fast_index forces the index frequency to be a default value", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)
		validAttrs.FastIndex = store.Ptr(true)
		validAttrs.IndexFrequencyMinutes = store.Ptr(0)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		if src.IndexFrequencyMinutes != store.SourceIndexFrequencyWhenFastIndexing() {
			t.Errorf("expected %d, got %d", store.SourceIndexFrequencyWhenFastIndexing(), src.IndexFrequencyMinutes)
		}
	})

	t.Run("disabling fast index will not change the index frequency", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)
		validAttrs.FastIndex = store.Ptr(false)
		validAttrs.IndexFrequencyMinutes = store.Ptr(0)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)
		if src.IndexFrequencyMinutes != 0 {
			t.Errorf("expected 0, got %d", src.IndexFrequencyMinutes)
		}
	})

	t.Run("creating will kickoff a metadata storage worker", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)
		validAttrs.FastIndex = store.Ptr(false)
		validAttrs.IndexFrequencyMinutes = store.Ptr(0)

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true)
		must(t, err)

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName, Args: map[string]any{"id": src.ID}})
	})
}

func TestSources_CreateSourceWhenTestingYtDlpOptions(t *testing.T) {
	for _, c := range []struct {
		name      string
		behaviour store.SourceCookieBehaviour
		want      bool
	}{
		{"sets use_cookies to true if the source has been set to use cookies", store.SourceCookieBehaviourAllOperations, true},
		{"does not set use_cookies if the source uses cookies when needed", store.SourceCookieBehaviourWhenNeeded, false},
		{"does not set use_cookies if the source has not been set to use cookies", store.SourceCookieBehaviourDisabled, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			ta.YtDlpMock.Run.Expect(func(_, _ string, _ ytdlp.Args, _ string, addl ytdlp.CallOptions) (string, error) {
				if addl.UseCookies != c.want {
					t.Errorf("use_cookies = %v, want %v", addl.UseCookies, c.want)
				}
				return sourcesTestPlaylistReturn(), nil
			})

			validAttrs := newSourceAttrs(t, ta)
			validAttrs.CookieBehaviour = store.Ptr(c.behaviour)

			mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true))
		})
	}

	t.Run("skips sleep interval", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ ytdlp.Args, _ string, addl ytdlp.CallOptions) (string, error) {
			if !addl.SkipSleepInterval {
				t.Error("expected skip_sleep_interval to be true")
			}
			return sourcesTestPlaylistReturn(), nil
		})

		validAttrs := newSourceAttrs(t, ta)

		mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, validAttrs, true))
	})
}

func TestSources_CreateSourceWhenTestingOptions(t *testing.T) {
	t.Run("run_post_commit_tasks: false won't enqueue post-commit tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		validAttrs := newSourceAttrs(t, ta)

		mustOK(t)(ta.App.SourcesCreateSource(ta.Ctx, validAttrs, false))

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName})
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName})
	})
}

func TestSources_UpdateSource(t *testing.T) {
	t.Run("updates with valid data updates the source", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		updateAttrs := store.SourceParams{CollectionName: store.Ptr("some updated name")}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, true)
		must(t, err)
		if updated.CollectionName != "some updated name" {
			t.Errorf("expected collection_name %q, got %q", "some updated name", updated.CollectionName)
		}
	})

	t.Run("updates with invalid data fails fast and does not call the runner", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})

		_, err := ta.App.SourcesUpdateSource(ta.Ctx, src, invalidSourceAttrs, true)
		if err == nil {
			t.Fatal("expected an error")
		}
		if _, ok := store.AsValidationErrors(err); !ok {
			t.Fatalf("expected a validation error, got %T: %v", err, err)
		}
		if ta.YtDlpMock.Run.Calls() != 0 {
			t.Errorf("expected the runner to not be called, got %d calls", ta.YtDlpMock.Run.Calls())
		}
	})

	t.Run("updating the original_url will re-fetch the source details for channels", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		updateAttrs := store.SourceParams{OriginalURL: store.Ptr("https://www.youtube.com/channel/abc123")}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, true)
		must(t, err)
		if updated.CollectionName != "some channel name" {
			t.Errorf("expected collection_name %q, got %q", "some channel name", updated.CollectionName)
		}
		if !startsWith(updated.CollectionID, "some_channel_id_") {
			t.Errorf("expected collection_id to start with some_channel_id_, got %q", updated.CollectionID)
		}
	})

	t.Run("updating the original_url will re-fetch the source details for playlists", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestPlaylistMock)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		updateAttrs := store.SourceParams{OriginalURL: store.Ptr("https://www.youtube.com/playlist?list=abc123")}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, true)
		must(t, err)
		if updated.CollectionName != "some playlist name" {
			t.Errorf("expected collection_name %q, got %q", "some playlist name", updated.CollectionName)
		}
		if !startsWith(updated.CollectionID, "some_playlist_id_") {
			t.Errorf("expected collection_id to start with some_playlist_id_, got %q", updated.CollectionID)
		}
	})

	t.Run("not updating the original_url will not re-fetch the source details", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		updateAttrs := store.SourceParams{CustomName: store.Ptr("some updated name")}

		mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, true))
		if ta.YtDlpMock.Run.Calls() != 0 {
			t.Errorf("expected the runner to not be called, got %d calls", ta.YtDlpMock.Run.Calls())
		}
	})

	t.Run("updates with invalid data returns error changeset", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})

		_, err := ta.App.SourcesUpdateSource(ta.Ctx, src, invalidSourceAttrs, true)
		if err == nil {
			t.Fatal("expected an error")
		}
		if _, ok := store.AsValidationErrors(err); !ok {
			t.Fatalf("expected a validation error, got %T: %v", err, err)
		}

		reloaded, err := ta.App.GetSource(ta.Ctx, src.ID)
		must(t, err)
		if !reflect.DeepEqual(reloaded, src) {
			t.Errorf("expected source to be unchanged, got %+v want %+v", reloaded, src)
		}
	})

	t.Run("updating will kickoff a metadata storage worker if the original_url changes", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Expect(sourcesTestPlaylistMock)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		updateAttrs := store.SourceParams{OriginalURL: store.Ptr("https://www.youtube.com/channel/cba321")}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, true)
		must(t, err)

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName, Args: map[string]any{"id": updated.ID}})
	})

	t.Run("updating will not kickoff a metadata storage worker other attrs change", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		updateAttrs := store.SourceParams{CustomName: store.Ptr("some new name")}

		mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, true))

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName})
	})
}

// enqueueCase is a source update and whether it should enqueue the worker
// under test (nothing may be enqueued beforehand).
type enqueueCase struct {
	name        string
	src, update store.SourceParams
	want        bool
}

// testUpdateEnqueues runs each case: it creates the source, lets prep add
// fixtures and return the expected job args, updates the source, then
// asserts on the worker's queue.
func testUpdateEnqueues(t *testing.T, worker string, prep func(*testing.T, *apptest.TestApp, *store.Source) map[string]any, cases []enqueueCase) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			src := apptest.SourceFixture(t, ta, c.src)
			args := prep(t, ta, src)

			ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: worker})
			mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, c.update, true))
			if c.want {
				ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: worker, Args: args})
			} else {
				ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: worker})
			}
		})
	}
}

func srcIDArgs(_ *testing.T, _ *apptest.TestApp, src *store.Source) map[string]any {
	return map[string]any{"id": src.ID}
}

// pendingTask inserts a job for worker with a task attached to src.
func pendingTask(t *testing.T, ta *apptest.TestApp, src *store.Source, worker string) *store.Task {
	t.Helper()
	job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), obanlite.NewJob(worker, map[string]any{"id": src.ID}))
	must(t, err)
	return apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(src.ID), JobID: store.Ptr(job.ID)})
}

func wantTaskGone(t *testing.T, ta *apptest.TestApp, task *store.Task) {
	t.Helper()
	if _, err := store.Get[store.Task](ta.Ctx, ta.Q(ta.Ctx), task.ID); err != store.ErrNotFound {
		t.Errorf("expected task %d to be deleted, got err=%v", task.ID, err)
	}
}

func TestSources_UpdateSourceWhenTestingMediaDownloadTasks(t *testing.T) {
	prep := func(t *testing.T, ta *apptest.TestApp, src *store.Source) map[string]any {
		item := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(src.ID), Clear: store.ClearMediaFilepath})
		return map[string]any{"id": item.ID}
	}
	yes, no := store.Ptr(true), store.Ptr(false)
	testUpdateEnqueues(t, app.MediaDownloadWorkerName, prep, []enqueueCase{
		{"enabling the download_media attribute will schedule a download task", store.SourceParams{DownloadMedia: no}, store.SourceParams{DownloadMedia: yes}, true},
		{"enabling download_media will not schedule a task if the source is disabled", store.SourceParams{DownloadMedia: no, Enabled: no}, store.SourceParams{DownloadMedia: yes}, false},
		{"enabling a source will schedule a download task if download_media is true", store.SourceParams{DownloadMedia: yes, Enabled: no}, store.SourceParams{Enabled: yes}, true},
		{"enabling a source will not schedule a download task if download_media is false", store.SourceParams{DownloadMedia: no, Enabled: no}, store.SourceParams{Enabled: yes}, false},
	})

	for _, c := range []struct {
		name   string
		update store.SourceParams
	}{
		{"disabling the download_media attribute will cancel the download task", store.SourceParams{DownloadMedia: no}},
		{"disabling a source will cancel any pending download tasks", store.SourceParams{Enabled: no}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)

			src := apptest.SourceFixture(t, ta, store.SourceParams{DownloadMedia: yes, Enabled: yes})
			mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(src.ID), Clear: store.ClearMediaFilepath})
			must(t, ta.App.DownloadingHelpersEnqueuePendingDownloadTasks(ta.Ctx, src, nil))

			ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
			mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, c.update, true))
			ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		})
	}
}

func TestSources_UpdateSourceWhenTestingSlowIndexing(t *testing.T) {
	yes, no := store.Ptr(true), store.Ptr(false)
	testUpdateEnqueues(t, app.MediaCollectionIndexingWorkerName, srcIDArgs, []enqueueCase{
		{"updating the index frequency to >0 will re-schedule the indexing task", store.SourceParams{}, store.SourceParams{IndexFrequencyMinutes: store.Ptr(123)}, true},
		{"updating the index frequency to 0 will not re-schedule the indexing task", store.SourceParams{}, store.SourceParams{IndexFrequencyMinutes: store.Ptr(0)}, false},
		{"updating the index frequency will not create a task if the source is disabled", store.SourceParams{Enabled: no}, store.SourceParams{IndexFrequencyMinutes: store.Ptr(123)}, false},
		{"enabling a source will create a task if the index frequency is >0", store.SourceParams{Enabled: no, IndexFrequencyMinutes: store.Ptr(123)}, store.SourceParams{Enabled: yes}, true},
		{"enabling a source will not create a task if the index frequency is 0", store.SourceParams{Enabled: no, IndexFrequencyMinutes: store.Ptr(0)}, store.SourceParams{Enabled: yes}, false},
	})

	t.Run("updating the index frequency to >0 stores the new frequency", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, store.SourceParams{IndexFrequencyMinutes: store.Ptr(123)}, true)
		must(t, err)
		if updated.IndexFrequencyMinutes != 123 {
			t.Errorf("expected 123, got %d", updated.IndexFrequencyMinutes)
		}
	})

	t.Run("updating the index frequency to 0 will delete any pending tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		task1 := pendingTask(t, ta, src, app.FastIndexingWorkerName)
		task2 := pendingTask(t, ta, src, app.MediaCollectionIndexingWorkerName)

		mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, store.SourceParams{IndexFrequencyMinutes: store.Ptr(0)}, true))

		wantTaskGone(t, ta, task1)
		wantTaskGone(t, ta, task2)
	})

	t.Run("not updating the index frequency will not re-schedule the indexing task or delete tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(src.ID)})

		mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, store.SourceParams{CustomName: store.Ptr("some updated name")}, true))

		if _, err := store.Get[store.Task](ta.Ctx, ta.Q(ta.Ctx), task.ID); err != nil {
			t.Errorf("expected task to still exist, got err=%v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": src.ID}})
	})
}

func TestSources_UpdateSourceWhenTestingFastIndexing(t *testing.T) {
	yes, no := store.Ptr(true), store.Ptr(false)
	testUpdateEnqueues(t, app.FastIndexingWorkerName, srcIDArgs, []enqueueCase{
		{"enabling fast_index will schedule a fast indexing task", store.SourceParams{FastIndex: no}, store.SourceParams{FastIndex: yes}, true},
		{"updating fast indexing will not create a task if the source is disabled", store.SourceParams{Enabled: no, FastIndex: no}, store.SourceParams{FastIndex: yes}, false},
		{"enabling a source will create a task if fast_index is true", store.SourceParams{Enabled: no, FastIndex: yes}, store.SourceParams{Enabled: yes}, true},
		{"enabling a source will not create a task if fast_index is false", store.SourceParams{Enabled: no, FastIndex: no}, store.SourceParams{Enabled: yes}, false},
	})

	t.Run("disabling fast_index will cancel the fast indexing task", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: yes})
		pendingTask(t, ta, src, app.FastIndexingWorkerName)

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName, Args: map[string]any{"id": src.ID}})
		mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, store.SourceParams{FastIndex: no}, true))
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
	})

	for _, c := range []struct {
		name string
		fast *bool
		want int
	}{
		{"fast_index forces the index frequency to be a default value", yes, store.SourceIndexFrequencyWhenFastIndexing()},
		{"disabling fast index will not change the index frequency", no, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)

			src := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: c.fast})
			updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, store.SourceParams{IndexFrequencyMinutes: store.Ptr(0)}, true)
			must(t, err)
			if updated.IndexFrequencyMinutes != c.want {
				t.Errorf("expected %d, got %d", c.want, updated.IndexFrequencyMinutes)
			}
		})
	}
}

// Disabling a source deletes its pending tasks, whichever worker owns them.
func TestSources_UpdateSourceDisablingDeletesPendingTasks(t *testing.T) {
	for _, worker := range []string{app.MediaCollectionIndexingWorkerName, app.FastIndexingWorkerName} {
		t.Run(worker, func(t *testing.T) {
			ta := apptest.NewApp(t)

			src := apptest.SourceFixture(t, ta, store.SourceParams{})
			task := pendingTask(t, ta, src, worker)

			mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, store.SourceParams{Enabled: store.Ptr(false)}, true))

			wantTaskGone(t, ta, task)
		})
	}
}

func TestSources_UpdateSourceWhenTestingOptions(t *testing.T) {
	t.Run("run_post_commit_tasks: false won't enqueue post-commit tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{
			FastIndex:             store.Ptr(false),
			DownloadMedia:         store.Ptr(false),
			IndexFrequencyMinutes: store.Ptr(-1),
		})
		updateAttrs := store.SourceParams{
			FastIndex:             store.Ptr(true),
			DownloadMedia:         store.Ptr(true),
			IndexFrequencyMinutes: store.Ptr(100),
		}

		mustOK(t)(ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, false))

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName})
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName})
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
	})
}

func TestSources_DeleteSource(t *testing.T) {
	t.Run("it deletes the source", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, false))
		if _, err := ta.App.GetSource(ta.Ctx, src.ID); err != store.ErrNotFound {
			t.Errorf("expected source to be deleted, got err=%v", err)
		}
	})

	t.Run("deletion also deletes all associated tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(src.ID)})

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, false))
		if _, err := store.Get[store.Task](ta.Ctx, ta.Q(ta.Ctx), task.ID); err != store.ErrNotFound {
			t.Errorf("expected task to be deleted, got err=%v", err)
		}
	})

	t.Run("deletion also deletes all associated media items", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(src.ID)})

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, false))
		if _, err := store.Get[store.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem.ID); err != store.ErrNotFound {
			t.Errorf("expected media item to be deleted, got err=%v", err)
		}
	})

	t.Run("deletion does not delete media files by default", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(src.ID)})

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, false))
		if _, err := os.Stat(*mediaItem.MediaFilepath); err != nil {
			t.Errorf("expected media file to still exist: %v", err)
		}
	})

	t.Run("deletes the source's metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		src, err := ta.PreloadSourceMetadata(ta.Ctx, src)
		must(t, err)

		metadataFilepath, err := ta.App.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, src, map[string]any{})
		must(t, err)
		updateAttrs := store.SourceParams{Metadata: &store.SourceMetadata{MetadataFilepath: metadataFilepath}}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, true)
		must(t, err)
		updated, err = ta.PreloadSourceMetadata(ta.Ctx, updated)
		must(t, err)

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, updated, false))
		if _, err := os.Stat(updated.Metadata.MetadataFilepath); !os.IsNotExist(err) {
			t.Errorf("expected metadata file to be deleted, got err=%v", err)
		}
	})

	t.Run("does not delete the source's non-metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)

		filepath, err := fsutil.GenerateTmpfile(ta.Config.TmpfileDirectory, "nfo")
		must(t, err)
		src := apptest.SourceFixture(t, ta, store.SourceParams{NfoFilepath: store.Ptr(filepath)})

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, false))
		if _, err := os.Stat(filepath); err != nil {
			t.Errorf("expected nfo file to still exist: %v", err)
		}
		os.Remove(filepath)
	})
}

func TestSources_DeleteSourceWhenDeletingFiles(t *testing.T) {
	t.Run("deletes source and media_items", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(_ string, _ any) error { return nil })

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(src.ID)})

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, true))

		if _, err := store.Get[store.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem.ID); err != store.ErrNotFound {
			t.Errorf("expected media item to be deleted, got err=%v", err)
		}
		if _, err := store.Get[store.Source](ta.Ctx, ta.Q(ta.Ctx), src.ID); err != store.ErrNotFound {
			t.Errorf("expected source to be deleted, got err=%v", err)
		}
	})

	t.Run("also deletes media files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(_ string, _ any) error { return nil })

		src := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(src.ID)})

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, true))

		if _, err := os.Stat(*mediaItem.MediaFilepath); !os.IsNotExist(err) {
			t.Errorf("expected media file to be deleted, got err=%v", err)
		}
	})

	t.Run("deletes the source's non-metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(_ string, _ any) error { return nil })

		filepath, err := fsutil.GenerateTmpfile(ta.Config.TmpfileDirectory, "nfo")
		must(t, err)
		src := apptest.SourceFixture(t, ta, store.SourceParams{NfoFilepath: store.Ptr(filepath)})

		mustOK(t)(ta.App.SourcesDeleteSource(ta.Ctx, src, true))
		if _, err := os.Stat(filepath); !os.IsNotExist(err) {
			t.Errorf("expected nfo file to be deleted, got err=%v", err)
		}
	})
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// newSourceAttrs returns valid channel-source params on a fresh media profile.
func newSourceAttrs(t *testing.T, ta *apptest.TestApp) store.SourceParams {
	t.Helper()
	return store.SourceParams{
		MediaProfileID: store.Ptr(apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{}).ID),
		OriginalURL:    store.Ptr("https://www.youtube.com/channel/abc123"),
	}
}

// validationErrors asserts err is a non-nil store validation error and
// returns its field errors.
func validationErrors(t *testing.T, err error) map[string][]string {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	errs, ok := store.AsValidationErrors(err)
	if !ok {
		t.Fatalf("expected a validation error, got %T: %v", err, err)
	}
	return errs
}
