package core_test

import (
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

var invalidSourceAttrs = core.Attrs{"name": nil, "collection_id": nil}

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

func sourcesTestPlaylistMock(_ string, _ string, _ core.KW, _ string, _ core.KW) (string, error) {
	return sourcesTestPlaylistReturn(), nil
}

func sourcesTestChannelMock(_ string, _ string, _ core.KW, _ string, _ core.KW) (string, error) {
	return sourcesTestChannelReturn(), nil
}

func TestSources_Schema(t *testing.T) {
	t.Run("source_metadata is deleted when the source is deleted", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{
			"metadata": core.Attrs{"metadata_filepath": "/metadata.json.gz"},
		})
		src, err := ta.PreloadSourceMetadata(ta.Ctx, src)
		if err != nil {
			t.Fatalf("PreloadSourceMetadata: %v", err)
		}
		metadata := src.Metadata
		if metadata == nil {
			t.Fatal("expected source to have metadata")
		}

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}

		if _, err := core.Get[core.SourceMetadata](ta.Ctx, ta.Q(ta.Ctx), metadata.ID); err != core.ErrNotFound {
			t.Errorf("expected metadata to be deleted, got err=%v", err)
		}
	})

	t.Run("can be JSON encoded without error", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})

		if _, err := json.Marshal(src); err != nil {
			t.Errorf("json.Marshal failed: %v", err)
		}
	})
}

func TestSources_OutputPathTemplate(t *testing.T) {
	t.Run("returns the source's override if present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{
			"output_path_template_override": "/override/{{ title }}.{{ ext }}",
		})

		got := ta.App.SourcesOutputPathTemplate(ta.Ctx, src)
		if got != "/override/{{ title }}.{{ ext }}" {
			t.Errorf("expected override template, got %q", got)
		}
	})

	t.Run("returns the media profile's template if no override is present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"output_path_template": "/profile/{{ title }}.{{ ext }}"})
		src := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})

		got := ta.App.SourcesOutputPathTemplate(ta.Ctx, src)
		if got != "/profile/{{ title }}.{{ ext }}" {
			t.Errorf("expected profile template, got %q", got)
		}
	})

	t.Run("Treats empty strings as being blank", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"output_path_template": "/profile/{{ title }}.{{ ext }}"})
		src := coretest.SourceFixture(t, ta, core.Attrs{
			"media_profile_id":              mediaProfile.ID,
			"output_path_template_override": "  ",
		})

		got := ta.App.SourcesOutputPathTemplate(ta.Ctx, src)
		if got != "/profile/{{ title }}.{{ ext }}" {
			t.Errorf("expected profile template, got %q", got)
		}
	})
}

func TestSources_UseCookies(t *testing.T) {
	t.Run("returns true if the source has been set to use cookies", func(t *testing.T) {
		src := &core.Source{CookieBehaviour: core.SourceCookieBehaviourAllOperations}
		if !core.SourcesUseCookies(src, "downloading") {
			t.Error("expected true")
		}
	})

	t.Run("returns false if the source has not been set to use cookies", func(t *testing.T) {
		src := &core.Source{CookieBehaviour: core.SourceCookieBehaviourDisabled}
		if core.SourcesUseCookies(src, "downloading") {
			t.Error("expected false")
		}
	})

	t.Run("returns true if the action is indexing and the source is set to :when_needed", func(t *testing.T) {
		src := &core.Source{CookieBehaviour: core.SourceCookieBehaviourWhenNeeded}
		if !core.SourcesUseCookies(src, "indexing") {
			t.Error("expected true")
		}
	})

	t.Run("returns false if the action is downloading and the source is set to :when_needed", func(t *testing.T) {
		src := &core.Source{CookieBehaviour: core.SourceCookieBehaviourWhenNeeded}
		if core.SourcesUseCookies(src, "downloading") {
			t.Error("expected false")
		}
	})

	t.Run("returns true if the action is error_recovery and the source is set to :when_needed", func(t *testing.T) {
		src := &core.Source{CookieBehaviour: core.SourceCookieBehaviourWhenNeeded}
		if !core.SourcesUseCookies(src, "error_recovery") {
			t.Error("expected true")
		}
	})
}

func TestSources_ListSources(t *testing.T) {
	t.Run("it returns all sources", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})

		got, err := ta.App.SourcesListSources(ta.Ctx)
		if err != nil {
			t.Fatalf("SourcesListSources: %v", err)
		}
		if !reflect.DeepEqual(got, []*core.Source{src}) {
			t.Errorf("expected %+v, got %+v", src, got)
		}
	})
}

func TestSources_ListSourcesFor(t *testing.T) {
	t.Run("returns all sources for a given media profile", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		src := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})

		got, err := ta.App.SourcesListSourcesFor(ta.Ctx, mediaProfile)
		if err != nil {
			t.Fatalf("SourcesListSourcesFor: %v", err)
		}
		if !reflect.DeepEqual(got, []*core.Source{src}) {
			t.Errorf("expected %+v, got %+v", src, got)
		}
	})
}

func TestSources_GetSource(t *testing.T) {
	t.Run("it returns the source with given id", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})

		got, err := ta.App.SourcesGetSource(ta.Ctx, src.ID)
		if err != nil {
			t.Fatalf("SourcesGetSource: %v", err)
		}
		if !reflect.DeepEqual(got, src) {
			t.Errorf("expected %+v, got %+v", src, got)
		}
	})
}

func TestSources_CreateSource(t *testing.T) {
	t.Run("automatically sets the UUID", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
		if src.UUID == nil || len(*src.UUID) != 36 {
			t.Errorf("expected a 36-char UUID, got %v", src.UUID)
		}
	})

	t.Run("UUID is not writable by the user", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
			"uuid":             "some_uuid",
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
		if src.UUID == nil || len(*src.UUID) != 36 {
			t.Errorf("expected a 36-char UUID, got %v", src.UUID)
		}
	})

	t.Run("creates a source and adds name + ID from runner response for channels", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
		if src.CollectionName != "some channel name" {
			t.Errorf("expected collection_name %q, got %q", "some channel name", src.CollectionName)
		}
		if !startsWith(src.CollectionID, "some_channel_id_") {
			t.Errorf("expected collection_id to start with some_channel_id_, got %q", src.CollectionID)
		}
	})

	t.Run("creates a source and adds name + ID for playlists", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestPlaylistMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/playlist?list=abc123",
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
		if src.CollectionName != "some playlist name" {
			t.Errorf("expected collection_name %q, got %q", "some playlist name", src.CollectionName)
		}
		if !startsWith(src.CollectionID, "some_playlist_id_") {
			t.Errorf("expected collection_id to start with some_playlist_id_, got %q", src.CollectionID)
		}
	})

	t.Run("adds an error if the runner fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ core.KW, _ string, _ core.KW) (string, error) {
			return "", &core.CommandError{Output: "some error", Status: 1}
		})

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		_, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err == nil {
			t.Fatal("expected an error")
		}
		cs, ok := core.AsChangesetError(err)
		if !ok {
			t.Fatalf("expected a changeset error, got %T", err)
		}
		if !containsString(cs.ErrorMap()["original_url"], "could not fetch source details from URL") {
			t.Errorf("expected original_url error, got %+v", cs.ErrorMap())
		}
	})

	t.Run("adds an error if the runner succeeds but the result was invalid JSON", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ core.KW, _ string, _ core.KW) (string, error) {
			return "Not JSON", nil
		})

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		_, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err == nil {
			t.Fatal("expected an error")
		}
		cs, ok := core.AsChangesetError(err)
		if !ok {
			t.Fatalf("expected a changeset error, got %T", err)
		}
		if !containsString(cs.ErrorMap()["original_url"], "could not fetch source details from URL") {
			t.Errorf("expected original_url error, got %+v", cs.ErrorMap())
		}
	})

	t.Run("you can specify a custom custom_name", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
			"custom_name":      "some custom name",
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
		if src.CustomName != "some custom name" {
			t.Errorf("expected custom_name %q, got %q", "some custom name", src.CustomName)
		}
	})

	t.Run("friendly name is pulled from collection_name if not specified", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
		if src.CustomName != "some channel name" {
			t.Errorf("expected custom_name %q, got %q", "some channel name", src.CustomName)
		}
	})

	t.Run("creation enforces uniqueness of collection_id scoped to the media_profile and title regex", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.ExpectN(2, func(_, _ string, _ core.KW, _ string, _ core.KW) (string, error) {
			jsonStr, _ := db.EncodeJSON(map[string]any{
				"channel":        "some channel name",
				"channel_id":     "some_channel_id_12345678",
				"playlist_id":    "some_channel_id_12345678",
				"playlist_title": "some channel name - videos",
			})
			return jsonStr, nil
		})

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validOnceAttrs := core.Attrs{
			"media_profile_id":   mediaProfile.ID,
			"original_url":       "https://www.youtube.com/channel/abc123",
			"title_filter_regex": nil,
		}

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, validOnceAttrs, core.KW{}); err != nil {
			t.Fatalf("first SourcesCreateSource: %v", err)
		}
		_, err := ta.App.SourcesCreateSource(ta.Ctx, validOnceAttrs, core.KW{})
		if err == nil {
			t.Fatal("expected the second create to fail")
		}
		if _, ok := core.AsChangesetError(err); !ok {
			t.Fatalf("expected a changeset error, got %T: %v", err, err)
		}
	})

	t.Run("creation lets you duplicate collection_ids and profiles as long as the regex is different", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.ExpectN(2, func(_, _ string, _ core.KW, _ string, _ core.KW) (string, error) {
			jsonStr, _ := db.EncodeJSON(map[string]any{
				"channel":        "some channel name",
				"channel_id":     "some_channel_id_12345678",
				"playlist_id":    "some_channel_id_12345678",
				"playlist_title": "some channel name - videos",
			})
			return jsonStr, nil
		})

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"name":             "some name",
			"original_url":     "https://www.youtube.com/channel/abc123",
		}
		source1Attrs := core.Attrs{}
		source2Attrs := core.Attrs{}
		for k, v := range validAttrs {
			source1Attrs[k] = v
			source2Attrs[k] = v
		}
		source1Attrs["title_filter_regex"] = "foo"
		source2Attrs["title_filter_regex"] = "bar"

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, source1Attrs, core.KW{}); err != nil {
			t.Fatalf("first SourcesCreateSource: %v", err)
		}
		if _, err := ta.App.SourcesCreateSource(ta.Ctx, source2Attrs, core.KW{}); err != nil {
			t.Fatalf("second SourcesCreateSource: %v", err)
		}
	})

	t.Run("creation lets you duplicate collection_ids as long as the media profile is different", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.ExpectN(2, func(_, _ string, _ core.KW, _ string, _ core.KW) (string, error) {
			jsonStr, _ := db.EncodeJSON(map[string]any{
				"channel":        "some channel name",
				"channel_id":     "some_channel_id_12345678",
				"playlist_id":    "some_channel_id_12345678",
				"playlist_title": "some channel name - videos",
			})
			return jsonStr, nil
		})

		validAttrs := core.Attrs{
			"name":               "some name",
			"original_url":       "https://www.youtube.com/channel/abc123",
			"title_filter_regex": "TEST",
		}
		source1Attrs := core.Attrs{}
		source2Attrs := core.Attrs{}
		for k, v := range validAttrs {
			source1Attrs[k] = v
			source2Attrs[k] = v
		}
		source1Attrs["media_profile_id"] = coretest.MediaProfileFixture(t, ta, core.Attrs{}).ID
		source2Attrs["media_profile_id"] = coretest.MediaProfileFixture(t, ta, core.Attrs{}).ID

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, source1Attrs, core.KW{}); err != nil {
			t.Fatalf("first SourcesCreateSource: %v", err)
		}
		if _, err := ta.App.SourcesCreateSource(ta.Ctx, source2Attrs, core.KW{}); err != nil {
			t.Fatalf("second SourcesCreateSource: %v", err)
		}
	})

	t.Run("collection_type is inferred from source details", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)
		ta.YtDlpMock.Run.Expect(sourcesTestPlaylistMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		source1, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("first SourcesCreateSource: %v", err)
		}
		source2, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("second SourcesCreateSource: %v", err)
		}

		if source1.CollectionType != core.SourceCollectionTypeChannel {
			t.Errorf("expected channel, got %v", source1.CollectionType)
		}
		if source2.CollectionType != core.SourceCollectionTypePlaylist {
			t.Errorf("expected playlist, got %v", source2.CollectionType)
		}
	})

	t.Run("creation with invalid data returns error changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		_, err := ta.App.SourcesCreateSource(ta.Ctx, invalidSourceAttrs, core.KW{})
		if err == nil {
			t.Fatal("expected an error")
		}
		if _, ok := core.AsChangesetError(err); !ok {
			t.Fatalf("expected a changeset error, got %T", err)
		}
	})

	t.Run("creation with invalid data fails fast and does not call the runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		_, err := ta.App.SourcesCreateSource(ta.Ctx, invalidSourceAttrs, core.KW{})
		if err == nil {
			t.Fatal("expected an error")
		}
		if _, ok := core.AsChangesetError(err); !ok {
			t.Fatalf("expected a changeset error, got %T", err)
		}
		if ta.YtDlpMock.Run.Calls() != 0 {
			t.Errorf("expected the runner to not be called, got %d calls", ta.YtDlpMock.Run.Calls())
		}
	})

	t.Run("creation will schedule the indexing task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})

	t.Run("creation will schedule a fast indexing job if the fast_index option is set", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
			"fast_index":       true,
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.FastIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})

	t.Run("creation will not schedule a fast indexing job if the fast_index option is not set", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
			"fast_index":       false,
		}

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
	})

	t.Run("creation schedules an index test even if the index frequency is 0", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id":        mediaProfile.ID,
			"original_url":            "https://www.youtube.com/channel/abc123",
			"index_frequency_minutes": 0,
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})

	t.Run("fast_index forces the index frequency to be a default value", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id":        mediaProfile.ID,
			"original_url":            "https://www.youtube.com/channel/abc123",
			"fast_index":              true,
			"index_frequency_minutes": 0,
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
		if src.IndexFrequencyMinutes != core.SourceIndexFrequencyWhenFastIndexing() {
			t.Errorf("expected %d, got %d", core.SourceIndexFrequencyWhenFastIndexing(), src.IndexFrequencyMinutes)
		}
	})

	t.Run("disabling fast index will not change the index frequency", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id":        mediaProfile.ID,
			"original_url":            "https://www.youtube.com/channel/abc123",
			"fast_index":              false,
			"index_frequency_minutes": 0,
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
		if src.IndexFrequencyMinutes != 0 {
			t.Errorf("expected 0, got %d", src.IndexFrequencyMinutes)
		}
	})

	t.Run("creating will kickoff a metadata storage worker", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id":        mediaProfile.ID,
			"original_url":            "https://www.youtube.com/channel/abc123",
			"fast_index":              false,
			"index_frequency_minutes": 0,
		}

		src, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.SourceMetadataStorageWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})
}

func TestSources_CreateSourceWhenTestingYtDlpOptions(t *testing.T) {
	t.Run("sets use_cookies to true if the source has been set to use cookies", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ core.KW, _ string, addl core.KW) (string, error) {
			if !addl.Bool("use_cookies") {
				t.Error("expected use_cookies to be true")
			}
			return sourcesTestPlaylistReturn(), nil
		})

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
			"cookie_behaviour": "all_operations",
		}

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
	})

	t.Run("does not set use_cookies if the source uses cookies when needed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ core.KW, _ string, addl core.KW) (string, error) {
			if addl.Bool("use_cookies") {
				t.Error("expected use_cookies to be false")
			}
			return sourcesTestPlaylistReturn(), nil
		})

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
			"cookie_behaviour": "when_needed",
		}

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
	})

	t.Run("does not set use_cookies if the source has not been set to use cookies", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ core.KW, _ string, addl core.KW) (string, error) {
			if addl.Bool("use_cookies") {
				t.Error("expected use_cookies to be false")
			}
			return sourcesTestPlaylistReturn(), nil
		})

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
			"cookie_behaviour": "disabled",
		}

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
	})

	t.Run("skips sleep interval", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(func(_, _ string, _ core.KW, _ string, addl core.KW) (string, error) {
			if !addl.Bool("skip_sleep_interval") {
				t.Error("expected skip_sleep_interval to be true")
			}
			return sourcesTestPlaylistReturn(), nil
		})

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}
	})
}

func TestSources_CreateSourceWhenTestingOptions(t *testing.T) {
	t.Run("run_post_commit_tasks: false won't enqueue post-commit tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		validAttrs := core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"original_url":     "https://www.youtube.com/channel/abc123",
		}

		if _, err := ta.App.SourcesCreateSource(ta.Ctx, validAttrs, core.KW{core.Opt("run_post_commit_tasks", false)}); err != nil {
			t.Fatalf("SourcesCreateSource: %v", err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.SourceMetadataStorageWorkerName})
	})
}

func TestSources_UpdateSource(t *testing.T) {
	t.Run("updates with valid data updates the source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"collection_name": "some updated name"}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		if updated.CollectionName != "some updated name" {
			t.Errorf("expected collection_name %q, got %q", "some updated name", updated.CollectionName)
		}
	})

	t.Run("updates with invalid data fails fast and does not call the runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})

		_, err := ta.App.SourcesUpdateSource(ta.Ctx, src, invalidSourceAttrs, core.KW{})
		if err == nil {
			t.Fatal("expected an error")
		}
		if _, ok := core.AsChangesetError(err); !ok {
			t.Fatalf("expected a changeset error, got %T: %v", err, err)
		}
		if ta.YtDlpMock.Run.Calls() != 0 {
			t.Errorf("expected the runner to not be called, got %d calls", ta.YtDlpMock.Run.Calls())
		}
	})

	t.Run("updating the original_url will re-fetch the source details for channels", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestChannelMock)

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"original_url": "https://www.youtube.com/channel/abc123"}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		if updated.CollectionName != "some channel name" {
			t.Errorf("expected collection_name %q, got %q", "some channel name", updated.CollectionName)
		}
		if !startsWith(updated.CollectionID, "some_channel_id_") {
			t.Errorf("expected collection_id to start with some_channel_id_, got %q", updated.CollectionID)
		}
	})

	t.Run("updating the original_url will re-fetch the source details for playlists", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestPlaylistMock)

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"original_url": "https://www.youtube.com/playlist?list=abc123"}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		if updated.CollectionName != "some playlist name" {
			t.Errorf("expected collection_name %q, got %q", "some playlist name", updated.CollectionName)
		}
		if !startsWith(updated.CollectionID, "some_playlist_id_") {
			t.Errorf("expected collection_id to start with some_playlist_id_, got %q", updated.CollectionID)
		}
	})

	t.Run("not updating the original_url will not re-fetch the source details", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"custom_name": "some updated name"}

		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		if ta.YtDlpMock.Run.Calls() != 0 {
			t.Errorf("expected the runner to not be called, got %d calls", ta.YtDlpMock.Run.Calls())
		}
	})

	t.Run("updates with invalid data returns error changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})

		_, err := ta.App.SourcesUpdateSource(ta.Ctx, src, invalidSourceAttrs, core.KW{})
		if err == nil {
			t.Fatal("expected an error")
		}
		if _, ok := core.AsChangesetError(err); !ok {
			t.Fatalf("expected a changeset error, got %T: %v", err, err)
		}

		reloaded, err := ta.App.SourcesGetSource(ta.Ctx, src.ID)
		if err != nil {
			t.Fatalf("SourcesGetSource: %v", err)
		}
		if !reflect.DeepEqual(reloaded, src) {
			t.Errorf("expected source to be unchanged, got %+v want %+v", reloaded, src)
		}
	})

	t.Run("updating will kickoff a metadata storage worker if the original_url changes", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.YtDlpMock.Run.Expect(sourcesTestPlaylistMock)

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"original_url": "https://www.youtube.com/channel/cba321"}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.SourceMetadataStorageWorkerName,
			Args:   map[string]any{"id": updated.ID},
		})
	})

	t.Run("updating will not kickoff a metadata storage worker other attrs change", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"custom_name": "some new name"}

		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.SourceMetadataStorageWorkerName})
	})
}

func TestSources_UpdateSourceWhenTestingMediaDownloadTasks(t *testing.T) {
	t.Run("enabling the download_media attribute will schedule a download task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"download_media": false})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": src.ID, "media_filepath": nil})
		updateAttrs := core.Attrs{"download_media": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaDownloadWorkerName,
			Args:   map[string]any{"id": mediaItem.ID},
		})
	})

	t.Run("disabling the download_media attribute will cancel the download task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"download_media": true, "enabled": true})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": src.ID, "media_filepath": nil})
		updateAttrs := core.Attrs{"download_media": false}
		if err := ta.App.DownloadingHelpersEnqueuePendingDownloadTasks(ta.Ctx, src, core.KW{}); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaDownloadWorkerName,
			Args:   map[string]any{"id": mediaItem.ID},
		})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
	})

	t.Run("enabling download_media will not schedule a task if the source is disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"download_media": false, "enabled": false})
		coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": src.ID, "media_filepath": nil})
		updateAttrs := core.Attrs{"download_media": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
	})

	t.Run("disabling a source will cancel any pending download tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"download_media": true, "enabled": true})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": src.ID, "media_filepath": nil})
		updateAttrs := core.Attrs{"enabled": false}
		if err := ta.App.DownloadingHelpersEnqueuePendingDownloadTasks(ta.Ctx, src, core.KW{}); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaDownloadWorkerName,
			Args:   map[string]any{"id": mediaItem.ID},
		})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
	})

	t.Run("enabling a source will schedule a download task if download_media is true", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"download_media": true, "enabled": false})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": src.ID, "media_filepath": nil})
		updateAttrs := core.Attrs{"enabled": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaDownloadWorkerName,
			Args:   map[string]any{"id": mediaItem.ID},
		})
	})

	t.Run("enabling a source will not schedule a download task if download_media is false", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"download_media": false, "enabled": false})
		coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": src.ID, "media_filepath": nil})
		updateAttrs := core.Attrs{"enabled": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
	})
}

func TestSources_UpdateSourceWhenTestingSlowIndexing(t *testing.T) {
	t.Run("updating the index frequency to >0 will re-schedule the indexing task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"index_frequency_minutes": 123}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		if updated.IndexFrequencyMinutes != 123 {
			t.Errorf("expected 123, got %d", updated.IndexFrequencyMinutes)
		}
		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": updated.ID},
		})
	})

	t.Run("updating the index frequency to 0 will not re-schedule the indexing task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"index_frequency_minutes": 0}

		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})

	t.Run("updating the index frequency to 0 will delete any pending tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"index_frequency_minutes": 0}

		job1, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": src.ID}))
		if err != nil {
			t.Fatalf("Oban.Insert: %v", err)
		}
		task1 := coretest.TaskFixture(t, ta, core.Attrs{"source_id": src.ID, "job_id": job1.ID})
		job2, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), obanlite.NewJob(core.MediaCollectionIndexingWorkerName, map[string]any{"id": src.ID}))
		if err != nil {
			t.Fatalf("Oban.Insert: %v", err)
		}
		task2 := coretest.TaskFixture(t, ta, core.Attrs{"source_id": src.ID, "job_id": job2.ID})

		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}

		if _, err := core.Get[core.Task](ta.Ctx, ta.Q(ta.Ctx), task1.ID); err != core.ErrNotFound {
			t.Errorf("expected task1 to be deleted, got err=%v", err)
		}
		if _, err := core.Get[core.Task](ta.Ctx, ta.Q(ta.Ctx), task2.ID); err != core.ErrNotFound {
			t.Errorf("expected task2 to be deleted, got err=%v", err)
		}
	})

	t.Run("not updating the index frequency will not re-schedule the indexing task or delete tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": src.ID})
		updateAttrs := core.Attrs{"custom_name": "some updated name"}

		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}

		if _, err := core.Get[core.Task](ta.Ctx, ta.Q(ta.Ctx), task.ID); err != nil {
			t.Errorf("expected task to still exist, got err=%v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})

	t.Run("disabling a source will delete any pending tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"enabled": false}

		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), obanlite.NewJob(core.MediaCollectionIndexingWorkerName, map[string]any{"id": src.ID}))
		if err != nil {
			t.Fatalf("Oban.Insert: %v", err)
		}
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": src.ID, "job_id": job.ID})

		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}

		if _, err := core.Get[core.Task](ta.Ctx, ta.Q(ta.Ctx), task.ID); err != core.ErrNotFound {
			t.Errorf("expected task to be deleted, got err=%v", err)
		}
	})

	t.Run("updating the index frequency will not create a task if the source is disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"enabled": false})
		updateAttrs := core.Attrs{"index_frequency_minutes": 123}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
	})

	t.Run("enabling a source will create a task if the index frequency is >0", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"enabled": false, "index_frequency_minutes": 123})
		updateAttrs := core.Attrs{"enabled": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})

	t.Run("enabling a source will not create a task if the index frequency is 0", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"enabled": false, "index_frequency_minutes": 0})
		updateAttrs := core.Attrs{"enabled": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
	})
}

func TestSources_UpdateSourceWhenTestingFastIndexing(t *testing.T) {
	t.Run("enabling fast_index will schedule a fast indexing task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": false})
		updateAttrs := core.Attrs{"fast_index": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.FastIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})

	t.Run("disabling fast_index will cancel the fast indexing task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})
		updateAttrs := core.Attrs{"fast_index": false}
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": src.ID}))
		if err != nil {
			t.Fatalf("Oban.Insert: %v", err)
		}
		coretest.TaskFixture(t, ta, core.Attrs{"source_id": src.ID, "job_id": job.ID})

		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.FastIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
	})

	t.Run("fast_index forces the index frequency to be a default value", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})
		updateAttrs := core.Attrs{"index_frequency_minutes": 0}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		if updated.IndexFrequencyMinutes != core.SourceIndexFrequencyWhenFastIndexing() {
			t.Errorf("expected %d, got %d", core.SourceIndexFrequencyWhenFastIndexing(), updated.IndexFrequencyMinutes)
		}
	})

	t.Run("disabling fast index will not change the index frequency", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": false})
		updateAttrs := core.Attrs{"index_frequency_minutes": 0}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		if updated.IndexFrequencyMinutes != 0 {
			t.Errorf("expected 0, got %d", updated.IndexFrequencyMinutes)
		}
	})

	t.Run("disabling a source will delete any pending tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		updateAttrs := core.Attrs{"enabled": false}

		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": src.ID}))
		if err != nil {
			t.Fatalf("Oban.Insert: %v", err)
		}
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": src.ID, "job_id": job.ID})

		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}

		if _, err := core.Get[core.Task](ta.Ctx, ta.Q(ta.Ctx), task.ID); err != core.ErrNotFound {
			t.Errorf("expected task to be deleted, got err=%v", err)
		}
	})

	t.Run("updating fast indexing will not create a task if the source is disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"enabled": false, "fast_index": false})
		updateAttrs := core.Attrs{"fast_index": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
	})

	t.Run("enabling a source will create a task if fast_index is true", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"enabled": false, "fast_index": true})
		updateAttrs := core.Attrs{"enabled": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.FastIndexingWorkerName,
			Args:   map[string]any{"id": src.ID},
		})
	})

	t.Run("enabling a source will not create a task if fast_index is false", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"enabled": false, "fast_index": false})
		updateAttrs := core.Attrs{"enabled": true}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
	})
}

func TestSources_UpdateSourceWhenTestingOptions(t *testing.T) {
	t.Run("run_post_commit_tasks: false won't enqueue post-commit tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{
			"fast_index":              false,
			"download_media":          false,
			"index_frequency_minutes": -1,
		})
		updateAttrs := core.Attrs{
			"fast_index":              true,
			"download_media":          true,
			"index_frequency_minutes": 100,
		}

		if _, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{core.Opt("run_post_commit_tasks", false)}); err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.SourceMetadataStorageWorkerName})
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
	})
}

func TestSources_DeleteSource(t *testing.T) {
	t.Run("it deletes the source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}
		if _, err := ta.App.SourcesGetSource(ta.Ctx, src.ID); err != core.ErrNotFound {
			t.Errorf("expected source to be deleted, got err=%v", err)
		}
	})

	t.Run("it returns a source changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{}, "")
		if cs == nil {
			t.Fatal("expected a changeset")
		}
	})

	t.Run("deletion also deletes all associated tasks", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		task := coretest.TaskFixture(t, ta, core.Attrs{"source_id": src.ID})

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}
		if _, err := core.Get[core.Task](ta.Ctx, ta.Q(ta.Ctx), task.ID); err != core.ErrNotFound {
			t.Errorf("expected task to be deleted, got err=%v", err)
		}
	})

	t.Run("deletion also deletes all associated media items", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": src.ID})

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}
		if _, err := core.Get[core.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem.ID); err != core.ErrNotFound {
			t.Errorf("expected media item to be deleted, got err=%v", err)
		}
	})

	t.Run("deletion does not delete media files by default", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": src.ID})

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}
		if _, err := os.Stat(*mediaItem.MediaFilepath); err != nil {
			t.Errorf("expected media file to still exist: %v", err)
		}
	})

	t.Run("deletes the source's metadata files", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		src, err := ta.PreloadSourceMetadata(ta.Ctx, src)
		if err != nil {
			t.Fatalf("PreloadSourceMetadata: %v", err)
		}

		metadataFilepath, err := ta.App.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, src, map[string]any{})
		if err != nil {
			t.Fatalf("MetadataFileHelpersCompressAndStoreMetadataFor: %v", err)
		}
		updateAttrs := core.Attrs{"metadata": core.Attrs{"metadata_filepath": metadataFilepath}}

		updated, err := ta.App.SourcesUpdateSource(ta.Ctx, src, updateAttrs, core.KW{})
		if err != nil {
			t.Fatalf("SourcesUpdateSource: %v", err)
		}
		updated, err = ta.PreloadSourceMetadata(ta.Ctx, updated)
		if err != nil {
			t.Fatalf("PreloadSourceMetadata: %v", err)
		}

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, updated, core.KW{}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}
		if _, err := os.Stat(updated.Metadata.MetadataFilepath); !os.IsNotExist(err) {
			t.Errorf("expected metadata file to be deleted, got err=%v", err)
		}
	})

	t.Run("does not delete the source's non-metadata files", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		filepath, err := ta.App.FilesystemUtilsGenerateMetadataTmpfile(ta.Ctx, "nfo")
		if err != nil {
			t.Fatalf("FilesystemUtilsGenerateMetadataTmpfile: %v", err)
		}
		src := coretest.SourceFixture(t, ta, core.Attrs{"nfo_filepath": filepath})

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}
		if _, err := os.Stat(filepath); err != nil {
			t.Errorf("expected nfo file to still exist: %v", err)
		}
		os.Remove(filepath)
	})
}

func TestSources_DeleteSourceWhenDeletingFiles(t *testing.T) {
	t.Run("deletes source and media_items", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.UserScriptMock.Run.Stub(func(_ string, _ any) error { return nil })

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": src.ID})

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{core.Opt("delete_files", true)}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}

		if _, err := core.Get[core.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem.ID); err != core.ErrNotFound {
			t.Errorf("expected media item to be deleted, got err=%v", err)
		}
		if _, err := core.Get[core.Source](ta.Ctx, ta.Q(ta.Ctx), src.ID); err != core.ErrNotFound {
			t.Errorf("expected source to be deleted, got err=%v", err)
		}
	})

	t.Run("also deletes media files", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.UserScriptMock.Run.Stub(func(_ string, _ any) error { return nil })

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": src.ID})

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{core.Opt("delete_files", true)}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}

		if _, err := os.Stat(*mediaItem.MediaFilepath); !os.IsNotExist(err) {
			t.Errorf("expected media file to be deleted, got err=%v", err)
		}
	})

	t.Run("deletes the source's non-metadata files", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()
		ta.UserScriptMock.Run.Stub(func(_ string, _ any) error { return nil })

		filepath, err := ta.App.FilesystemUtilsGenerateMetadataTmpfile(ta.Ctx, "nfo")
		if err != nil {
			t.Fatalf("FilesystemUtilsGenerateMetadataTmpfile: %v", err)
		}
		src := coretest.SourceFixture(t, ta, core.Attrs{"nfo_filepath": filepath})

		if _, err := ta.App.SourcesDeleteSource(ta.Ctx, src, core.KW{core.Opt("delete_files", true)}); err != nil {
			t.Fatalf("SourcesDeleteSource: %v", err)
		}
		if _, err := os.Stat(filepath); !os.IsNotExist(err) {
			t.Errorf("expected nfo file to be deleted, got err=%v", err)
		}
	})
}

func TestSources_ChangeSource(t *testing.T) {
	t.Run("it returns a changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{}, "")
		if cs == nil {
			t.Fatal("expected a changeset")
		}
	})
}

func TestSources_ChangeSourceWhenTestingRegexValidation(t *testing.T) {
	t.Run("succeeds when a valid regex is provided", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"title_filter_regex": "(?i)^How to Bike$"}, "")
		if len(cs.Errors) != 0 {
			t.Errorf("expected no errors, got %+v", cs.Errors)
		}
	})

	t.Run("succeeds when a regex is set back to nil", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{"title_filter_regex": "(?i)^How to Bike$"})
		cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"title_filter_regex": nil}, "")
		if len(cs.Errors) != 0 {
			t.Errorf("expected no errors, got %+v", cs.Errors)
		}
	})

	t.Run("fails when an invalid regex is provided", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"title_filter_regex": "*FOO"}, "")
		if !containsString(cs.ErrorMap()["title_filter_regex"], "is invalid") {
			t.Errorf("expected an 'is invalid' error, got %+v", cs.ErrorMap())
		}
	})
}

func TestSources_ChangeSourceWhenTestingMinMaxDurationValidations(t *testing.T) {
	t.Run("succeeds if min and max are nil", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"min_duration_seconds": nil, "max_duration_seconds": nil}, "")
		if len(cs.Errors) != 0 {
			t.Errorf("expected no errors, got %+v", cs.Errors)
		}
	})

	t.Run("succeeds if either min or max is nil", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		cs1 := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"min_duration_seconds": nil, "max_duration_seconds": 100}, "")
		if len(cs1.Errors) != 0 {
			t.Errorf("expected no errors, got %+v", cs1.Errors)
		}
		cs2 := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"min_duration_seconds": 100, "max_duration_seconds": nil}, "")
		if len(cs2.Errors) != 0 {
			t.Errorf("expected no errors, got %+v", cs2.Errors)
		}
	})

	t.Run("succeeds if min is less than max", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"min_duration_seconds": 100, "max_duration_seconds": 200}, "")
		if len(cs.Errors) != 0 {
			t.Errorf("expected no errors, got %+v", cs.Errors)
		}
	})

	t.Run("fails if min is greater than or equal to max", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		cs1 := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"min_duration_seconds": 200, "max_duration_seconds": 100}, "")
		if len(cs1.Errors) == 0 {
			t.Errorf("expected an error")
		}
		cs2 := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"min_duration_seconds": 100, "max_duration_seconds": 100}, "")
		if len(cs2.Errors) == 0 {
			t.Errorf("expected an error")
		}
	})
}

func TestSources_ChangeSourceWhenTestingOriginalURLValidation(t *testing.T) {
	t.Run("succeeds when an original URL is valid", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		validURLs := []string{
			"https://www.youtube.com/channel/UCkRfArvrzheW2E7b6SVT7vQ",
			"https://www.youtube.com/channel/UCkRfArvrzheW2E7b6SVT7vQ/videos",
			"https://www.youtube.com/@youtubecreators/featured",
			"https://www.youtube.com/@youtubecreators",
			"https://www.youtube.com/c/YouTubeCreators",
			"https://www.youtube.com/user/YouTubeCreators",
			"https://www.youtube.com/YouTubeCreators",
			"https://www.youtube.com/playlist?list=PLpjK416fmKwRtq-9-O_NbZlkW0k6zu2Wn",
			"https://www.youtube.com/playlist?list=UUkRfArvrzheW2E7b6SVT7vQ",
		}

		for _, url := range validURLs {
			cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"original_url": url}, "")
			if len(cs.Errors) != 0 {
				t.Errorf("expected no errors for %q, got %+v", url, cs.Errors)
			}
		}
	})

	t.Run("fails when an original URL points to a video", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		invalidURLs := []string{
			"https://www.youtube.com/watch?v=72maj9FLQZI",
			"https://youtu.be/72maj9FLQZI",
			"https://www.youtube.com/watch?v=1FwGFhMAmBo&list=PLpjK416fmKwRtq-9-O_NbZlkW0k6zu2Wn",
			"https://www.youtube.com/shorts/Dq0eH-ZhQTU",
			"https://www.youtube.com/embed/X64LHlfx4qg",
		}

		for _, url := range invalidURLs {
			cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"original_url": url}, "")
			if len(cs.Errors) == 0 {
				t.Errorf("expected an error for %q", url)
			}
		}
	})

	t.Run("passes when a non-youtube link is provided", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		src := coretest.SourceFixture(t, ta, core.Attrs{})
		validURLs := []string{
			"https://www.example.com",
			"https://www.example.com/playlist",
			"https://www.example.com/channel",
			"https://www.example.com/user",
			"https://www.example.com/watch?v=72maj9FLQZI",
			"https://www.example.com/embed/X64LHlfx4qg",
		}

		for _, url := range validURLs {
			cs := ta.App.SourcesChangeSource(ta.Ctx, src, core.Attrs{"original_url": url}, "")
			if len(cs.Errors) != 0 {
				t.Errorf("expected no errors for %q, got %+v", url, cs.Errors)
			}
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
