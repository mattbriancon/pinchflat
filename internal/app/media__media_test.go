package app_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestMedia_Schema(t *testing.T) {
	t.Run("media_metadata is deleted when media_item is deleted", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{
			"metadata": map[string]string{
				"metadata_filepath":  "/metadata.json.gz",
				"thumbnail_filepath": "/thumbnail.jpg",
			},
		})
		if _, err := ta.PreloadMediaItemMetadata(ta.Ctx, mediaItem); err != nil {
			t.Fatal(err)
		}
		metadata := mediaItem.Metadata

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, nil); err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}

		if _, err := store.Reload[store.MediaMetadata](ta.Ctx, ta.Q(ta.Ctx), metadata); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound reloading metadata, got %v", err)
		}
	})

	t.Run("can be JSON encoded without error", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})
		if _, err := ta.PreloadMediaItemSourceAndProfile(ta.Ctx, mediaItem); err != nil {
			t.Fatal(err)
		}

		// Phoenix.json_library().encode(media_item)
		if _, err := json.Marshal(mediaItem); err != nil {
			t.Errorf("expected {:ok, _}, got error: %v", err)
		}
	})
}

func TestMedia_ListMediaItems(t *testing.T) {
	t.Run("it returns all media_items", func(t *testing.T) {
		// removed skip
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})

		got, err := ta.ListMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{mediaItem}) {
			t.Errorf("got %+v, want [%+v]", got, mediaItem)
		}
	})
}

func TestMedia_ListUpgradeableMediaItems(t *testing.T) {
	setup := func(t *testing.T) (*apptest.TestApp, *store.MediaProfile, *store.Source) {
		t.Helper()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"redownload_delay_days": 4})
		source := apptest.SourceFixture(t, ta, store.Attrs{
			"media_profile_id": mediaProfile.ID,
			"inserted_at":      apptest.NowMinus(10, "days"),
		})
		return ta, mediaProfile, source
	}

	t.Run("returns media eligible for redownload", func(t *testing.T) {
		ta, _, source := setup(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         apptest.NowMinus(6, "days"),
			"media_downloaded_at": apptest.NowMinus(5, "days"),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{mediaItem}) {
			t.Errorf("got %+v, want [%+v]", got, mediaItem)
		}
	})

	t.Run("returns media items that were downloaded in past but still meet redownload delay", func(t *testing.T) {
		ta, _, source := setup(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         apptest.NowMinus(20, "days"),
			"media_downloaded_at": apptest.NowMinus(19, "days"),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{mediaItem}) {
			t.Errorf("got %+v, want [%+v]", got, mediaItem)
		}
	})

	t.Run("does not return media items without a media_downloaded_at", func(t *testing.T) {
		ta, _, source := setup(t)

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         apptest.NowMinus(5, "days"),
			"media_downloaded_at": nil,
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("does not return media items that are set to prevent download", func(t *testing.T) {
		ta, _, source := setup(t)

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         apptest.NowMinus(5, "days"),
			"media_downloaded_at": apptest.Now(),
			"prevent_download":    true,
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("does not return media items that have been culled", func(t *testing.T) {
		ta, _, source := setup(t)

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         apptest.NowMinus(5, "days"),
			"media_downloaded_at": apptest.Now(),
			"culled_at":           apptest.Now(),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("does not return media items before the download delay", func(t *testing.T) {
		ta, _, source := setup(t)

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         apptest.NowMinus(3, "days"),
			"media_downloaded_at": apptest.NowMinus(3, "days"),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("does not return media items that have already been redownloaded", func(t *testing.T) {
		ta, _, source := setup(t)

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":             source.ID,
			"uploaded_at":           apptest.NowMinus(5, "days"),
			"media_downloaded_at":   apptest.Now(),
			"media_redownloaded_at": apptest.Now(),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("does not return media items that were first downloaded well after the uploaded_at", func(t *testing.T) {
		ta, _, source := setup(t)

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"media_downloaded_at": apptest.Now(),
			"uploaded_at":         apptest.NowMinus(20, "days"),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("does not return media items that were recently uploaded", func(t *testing.T) {
		ta, _, source := setup(t)

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"media_downloaded_at": apptest.Now(),
			"uploaded_at":         apptest.NowMinus(2, "days"),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("does not return media items without a redownload delay", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"redownload_delay_days": nil})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         apptest.NowMinus(6, "days"),
			"media_downloaded_at": apptest.NowMinus(5, "days"),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})
}

func TestMedia_ListPendingMediaItemsFor(t *testing.T) {
	t.Run("it returns pending without a filepath for a given source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		otherSource := apptest.SourceFixture(t, ta, store.Attrs{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": otherSource.ID, "media_filepath": nil})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{mediaItem}) {
			t.Errorf("got %+v, want [%+v]", got, mediaItem)
		}
	})

	t.Run("it does not return media_items with media_filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		apptest.MediaItemFixture(t, ta, store.Attrs{
			"source_id":      source.ID,
			"media_filepath": "/video/" + apptest.Now().Format("20060102150405") + ".mp4",
		})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})
}

func TestMedia_ListPendingMediaItemsFor_Shorts(t *testing.T) {
	t.Run("returns shorts and normal media when shorts_behaviour is :include", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "include"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})
		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		short := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "short_form_content": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{normal, short}) {
			t.Errorf("got %+v, want [%+v %+v]", got, normal, short)
		}
	})

	t.Run("returns only shorts when shorts_behaviour is :only", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "only"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		short := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "short_form_content": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{short}) {
			t.Errorf("got %+v, want [%+v]", got, short)
		}
	})

	t.Run("returns only normal media when shorts_behaviour is :exclude", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "exclude"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})
		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "short_form_content": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{normal}) {
			t.Errorf("got %+v, want [%+v]", got, normal)
		}
	})
}

func TestMedia_ListPendingMediaItemsFor_Livestreams(t *testing.T) {
	t.Run("returns livestreams and normal media when livestream_behaviour is :include", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"livestream_behaviour": "include"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})
		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		livestream := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "livestream": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{normal, livestream}) {
			t.Errorf("got %+v, want [%+v %+v]", got, normal, livestream)
		}
	})

	t.Run("returns only livestreams when livestream_behaviour is :only", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"livestream_behaviour": "only"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		livestream := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "livestream": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{livestream}) {
			t.Errorf("got %+v, want [%+v]", got, livestream)
		}
	})

	t.Run("returns only normal media when livestream_behaviour is :exclude", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"livestream_behaviour": "exclude"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})
		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "livestream": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{normal}) {
			t.Errorf("got %+v, want [%+v]", got, normal)
		}
	})
}

func TestMedia_ListPendingMediaItemsFor_AllFormatOptions(t *testing.T) {
	t.Run("returns livestreams, shorts, and normal media when behaviour is :include", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "include", "livestream_behaviour": "include"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})

		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		livestream := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "livestream": true})
		short := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "short_form_content": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{normal, livestream, short}) {
			t.Errorf("got %+v, want [%+v %+v %+v]", got, normal, livestream, short)
		}
	})

	t.Run("returns only livestreams and shorts when behaviour is :only", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "only", "livestream_behaviour": "only"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})

		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		livestream := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "livestream": true})
		short := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "short_form_content": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{livestream, short}) {
			t.Errorf("got %+v, want [%+v %+v]", got, livestream, short)
		}
	})

	t.Run("returns only normal media when behaviour is :exclude", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "exclude", "livestream_behaviour": "exclude"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})

		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "livestream": true})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "short_form_content": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{normal}) {
			t.Errorf("got %+v, want [%+v]", got, normal)
		}
	})

	t.Run(":only and :exclude return the expected results", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "only", "livestream_behaviour": "exclude"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})

		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "livestream": true})
		short := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "short_form_content": true})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{short}) {
			t.Errorf("got %+v, want [%+v]", got, short)
		}
	})
}

func TestMedia_ListPendingMediaItemsFor_CutoffDates(t *testing.T) {
	t.Run("does not return media items with an upload date before the cutoff date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"download_cutoff_date": apptest.NowMinus(1, "day")})

		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "uploaded_at": apptest.NowMinus(2, "days")})
		newMediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "uploaded_at": apptest.Now()})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{newMediaItem}) {
			t.Errorf("got %+v, want [%+v]", got, newMediaItem)
		}
	})

	t.Run("does not apply a cutoff if there is no cutoff date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"download_cutoff_date": nil})

		oldMediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "uploaded_at": apptest.NowMinus(2, "days")})
		newMediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "uploaded_at": apptest.Now()})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{oldMediaItem, newMediaItem}) {
			t.Errorf("got %+v, want [%+v %+v]", got, oldMediaItem, newMediaItem)
		}
	})
}

func TestMedia_ListPendingMediaItemsFor_TitleRegex(t *testing.T) {
	t.Run("returns only media items that match the title regex", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"title_filter_regex": "(?i)^FOO$"})

		matching := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "foo"})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "bar"})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{matching}) {
			t.Errorf("got %+v, want [%+v]", got, matching)
		}
	})

	t.Run("does not apply a regex if none is specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"title_filter_regex": nil})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "foo"})
		two := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "bar"})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{one, two}) {
			t.Errorf("got %+v, want [%+v %+v]", got, one, two)
		}
	})
}

func TestMedia_ListPendingMediaItemsFor_MinAndMaxDurations(t *testing.T) {
	t.Run("returns media items that meet the min and max duration", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"min_duration_seconds": 10, "max_duration_seconds": 20})

		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 5})
		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 15})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 25})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{normal}) {
			t.Errorf("got %+v, want [%+v]", got, normal)
		}
	})

	t.Run("does not apply a min duration if none is specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"min_duration_seconds": nil, "max_duration_seconds": 20})

		short := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 5})
		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 15})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 25})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{short, normal}) {
			t.Errorf("got %+v, want [%+v %+v]", got, short, normal)
		}
	})

	t.Run("does not apply a max duration if none is specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"min_duration_seconds": 10, "max_duration_seconds": nil})

		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 5})
		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 15})
		long := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 25})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{normal, long}) {
			t.Errorf("got %+v, want [%+v %+v]", got, normal, long)
		}
	})

	t.Run("does not apply a min or max duration if none are specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"min_duration_seconds": nil, "max_duration_seconds": nil})

		short := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 5})
		normal := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 15})
		long := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 25})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{short, normal, long}) {
			t.Errorf("got %+v, want [%+v %+v %+v]", got, short, normal, long)
		}
	})
}

func TestMedia_ListPendingMediaItemsFor_DownloadPrevention(t *testing.T) {
	t.Run("returns only media items that are not prevented from downloading", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "prevent_download": true})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "prevent_download": false})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []*store.MediaItem{mediaItem}) {
			t.Errorf("got %+v, want [%+v]", got, mediaItem)
		}
	})
}

func TestMedia_PendingDownload(t *testing.T) {
	t.Run("returns true when the media hasn't been downloaded", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"media_filepath": nil})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns false if the media has been downloaded", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"media_filepath": "/video/downloaded.mp4"})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("expected not pending download")
		}
	})

	t.Run("returns false if the media hasn't been downloaded but the profile doesn't DL shorts", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"shorts_behaviour": "exclude"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "short_form_content": true})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("expected not pending download")
		}
	})

	t.Run("returns false if the media hasn't been downloaded but the profile doesn't DL livestreams", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.Attrs{"livestream_behaviour": "exclude"})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": profile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "livestream": true})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("expected not pending download")
		}
	})

	t.Run("returns true if there is a cutoff date before the media's upload date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"download_cutoff_date": apptest.NowMinus(2, "days")})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "uploaded_at": apptest.NowMinus(1, "day")})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns true if the cutoff date is equal to the upload date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"download_cutoff_date": apptest.NowMinus(2, "days")})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "uploaded_at": apptest.NowMinus(2, "days")})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns false if there is a cutoff date after the media's upload date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"download_cutoff_date": apptest.NowMinus(1, "day")})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "uploaded_at": apptest.NowMinus(2, "days")})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("expected not pending download")
		}
	})

	t.Run("returns true if there is no cutoff date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"download_cutoff_date": nil})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "uploaded_at": apptest.NowMinus(1, "day")})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns true if the content matches the title regex", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"title_filter_regex": "(?i)^FOO$"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "foo"})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns false if the content doesn't match the title regex", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"title_filter_regex": "(?i)^FOO$"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "bar"})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("expected not pending download")
		}
	})

	t.Run("return true if there is no title regex", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"title_filter_regex": nil})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "foo"})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns true if the duration is between the min and max", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"min_duration_seconds": 10, "max_duration_seconds": 20})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 15})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns false if the duration is below the min", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"min_duration_seconds": 10, "max_duration_seconds": 20})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 5})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("expected not pending download")
		}
	})

	t.Run("returns false if the duration is above the max", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"min_duration_seconds": 10, "max_duration_seconds": 20})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 25})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("expected not pending download")
		}
	})

	t.Run("returns true if there is no min or max duration", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"min_duration_seconds": nil, "max_duration_seconds": nil})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "duration_seconds": 15})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns true if the media item is not prevented from downloading", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"media_filepath": nil, "prevent_download": false})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Errorf("expected pending download")
		}
	})

	t.Run("returns false if the media item is prevented from downloading", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"media_filepath": nil, "prevent_download": true})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("expected not pending download")
		}
	})
}

func TestMedia_Search(t *testing.T) {
	setup := func(t *testing.T) (*apptest.TestApp, int64) {
		t.Helper()
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{
			"title":       "The quick brown fox",
			"description": "jumps over the lazy dog",
		})
		return ta, mediaItem.ID
	}

	t.Run("searches based on title", func(t *testing.T) {
		ta, mediaItemID := setup(t)

		got, err := ta.Search(ta.Ctx, "quick", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != mediaItemID {
			t.Errorf("got %+v, want a single result with id %d", got, mediaItemID)
		}
	})

	t.Run("searches based on description", func(t *testing.T) {
		ta, mediaItemID := setup(t)

		got, err := ta.Search(ta.Ctx, "lazy", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != mediaItemID {
			t.Errorf("got %+v, want a single result with id %d", got, mediaItemID)
		}
	})

	t.Run("adds a matching_search_term attribute with the relevant text", func(t *testing.T) {
		ta, _ := setup(t)

		got, err := ta.Search(ta.Ctx, "quick", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %+v, want a single result", got)
		}
		if got[0].MatchingSearchTerm == nil ||
			!strings.Contains(*got[0].MatchingSearchTerm, "The [PF_HIGHLIGHT]quick[/PF_HIGHLIGHT] brown fox") {
			t.Errorf("got matching_search_term %v", got[0].MatchingSearchTerm)
		}
	})

	t.Run("doesn't set matching_search_term to nil if one of the attributes we search on is null", func(t *testing.T) {
		ta, _ := setup(t)
		apptest.MediaItemFixture(t, ta, store.Attrs{"title": "foobar baz", "description": nil})

		got, err := ta.Search(ta.Ctx, "baz", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %+v, want a single result", got)
		}
		if got[0].MatchingSearchTerm == nil {
			t.Errorf("expected matching_search_term to not be nil")
		}
	})

	t.Run("optionally lets you specify a limit", func(t *testing.T) {
		ta, _ := setup(t)
		apptest.MediaItemFixture(t, ta, store.Attrs{"title": "The small gray dog"})

		got, err := ta.Search(ta.Ctx, "dog", store.KW{store.Opt("limit", 1)})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("got %+v, want a single result", got)
		}
	})

	t.Run("returns an empty list when the search term is blank", func(t *testing.T) {
		ta, _ := setup(t)

		got, err := ta.Search(ta.Ctx, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("returns an empty list when the search term is nil", func(t *testing.T) {
		ta, _ := setup(t)

		// Go has no nil string; the "" case above covers Media.search(nil, _opts).
		got, err := ta.Search(ta.Ctx, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want []", got)
		}
	})

	t.Run("doesn't blow up if there's an apostrophe or quotes in the search term", func(t *testing.T) {
		ta, _ := setup(t)

		if got, err := ta.Search(ta.Ctx, "don't expl'ode", nil); err != nil || len(got) != 0 {
			t.Errorf("got %+v, err %v", got, err)
		}
		if got, err := ta.Search(ta.Ctx, `dont expl"o"de`, nil); err != nil || len(got) != 0 {
			t.Errorf("got %+v, err %v", got, err)
		}
		if got, err := ta.Search(ta.Ctx, `dont explo"de`, nil); err != nil || len(got) != 0 {
			t.Errorf("got %+v, err %v", got, err)
		}
	})

	t.Run("doesn't blow up if there is a trailing operand", func(t *testing.T) {
		ta, _ := setup(t)

		if got, err := ta.Search(ta.Ctx, "foo OR", nil); err != nil || len(got) != 0 {
			t.Errorf("got %+v, err %v", got, err)
		}
		if got, err := ta.Search(ta.Ctx, "foo AND", nil); err != nil || len(got) != 0 {
			t.Errorf("got %+v, err %v", got, err)
		}
	})
}

func TestMedia_GetMediaItem(t *testing.T) {
	t.Run("it returns the media_item with given id", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})

		got, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, mediaItem) {
			t.Errorf("got %+v, want %+v", got, mediaItem)
		}
	})
}

func TestMedia_CreateMediaItem(t *testing.T) {
	t.Run("creating with valid data creates a media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		validAttrs := store.Attrs{
			"media_id":       "media-" + strconv.Itoa(int(source.ID)),
			"title":          "Some Product",
			"media_filepath": "/video/file.mp4",
			"source_id":      source.ID,
			"original_url":   "https://www.youtube.com/channel/abc",
			"uploaded_at":    apptest.Now(),
		}

		mediaItem, err := ta.CreateMediaItem(ta.Ctx, validAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}

		if store.Deref(mediaItem.Title) != validAttrs["title"] {
			t.Errorf("title = %v, want %v", store.Deref(mediaItem.Title), validAttrs["title"])
		}
		if mediaItem.MediaID != validAttrs["media_id"] {
			t.Errorf("media_id = %v, want %v", mediaItem.MediaID, validAttrs["media_id"])
		}
		if store.Deref(mediaItem.MediaFilepath) != validAttrs["media_filepath"] {
			t.Errorf("media_filepath = %v, want %v", store.Deref(mediaItem.MediaFilepath), validAttrs["media_filepath"])
		}
	})

	t.Run("automatically sets the UUID", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		validAttrs := store.Attrs{
			"media_id":       "media-1",
			"title":          "Some Product",
			"media_filepath": "/video/file.mp4",
			"source_id":      source.ID,
			"original_url":   "https://www.youtube.com/channel/abc",
			"uploaded_at":    apptest.Now(),
		}

		mediaItem, err := ta.CreateMediaItem(ta.Ctx, validAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if mediaItem.UUID == nil || len(*mediaItem.UUID) != 36 {
			t.Errorf("uuid = %v, want length 36", mediaItem.UUID)
		}
	})

	t.Run("UUID is not writable by the user", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})
		validAttrs := store.Attrs{
			"media_id":       "media-1",
			"title":          "Some Product",
			"media_filepath": "/video/file.mp4",
			"source_id":      source.ID,
			"original_url":   "https://www.youtube.com/channel/abc",
			"uploaded_at":    apptest.Now(),
			"uuid":           "some-uuid",
		}

		mediaItem, err := ta.CreateMediaItem(ta.Ctx, validAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if mediaItem.UUID == nil || len(*mediaItem.UUID) != 36 {
			t.Errorf("uuid = %v, want length 36 (not the user-supplied value)", mediaItem.UUID)
		}
	})

	t.Run("creating with invalid data returns error changeset", func(t *testing.T) {
		ta := apptest.NewApp(t)
		invalidAttrs := store.Attrs{"title": nil, "media_id": nil, "media_filepath": nil}

		_, err := ta.CreateMediaItem(ta.Ctx, invalidAttrs)
		if _, ok := store.AsChangesetError(err); !ok {
			t.Fatalf("expected a *store.ChangesetError, got %v", err)
		}
	})
}

func TestMedia_CreateMediaItemFromBackendAttrs(t *testing.T) {
	t.Run("creates a media item for a given source and attributes", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		var attrsMap map[string]any
		err := store.DecodeJSON([]byte(apptest.MediaAttributesReturnFixture()), &attrsMap)
		if err != nil {
			t.Fatal(err)
		}
		mediaAttrs := app.YtDlpMediaResponseToStruct(attrsMap)

		mediaItem, err := ta.CreateMediaItemFromBackendAttrs(ta.Ctx, source, mediaAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}

		if mediaItem.SourceID != source.ID {
			t.Errorf("source_id = %v, want %v", mediaItem.SourceID, source.ID)
		}
		if store.Deref(mediaItem.Title) != mediaAttrs.Title {
			t.Errorf("title = %v, want %v", store.Deref(mediaItem.Title), mediaAttrs.Title)
		}
		if mediaItem.MediaID != mediaAttrs.MediaID {
			t.Errorf("media_id = %v, want %v", mediaItem.MediaID, mediaAttrs.MediaID)
		}
		if mediaItem.OriginalURL != mediaAttrs.OriginalURL {
			t.Errorf("original_url = %v, want %v", mediaItem.OriginalURL, mediaAttrs.OriginalURL)
		}
		if store.Deref(mediaItem.Description) != mediaAttrs.Description {
			t.Errorf("description = %v, want %v", store.Deref(mediaItem.Description), mediaAttrs.Description)
		}
	})

	t.Run("updates the media item if it already exists", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		var attrsMap map[string]any
		err := store.DecodeJSON([]byte(apptest.MediaAttributesReturnFixture()), &attrsMap)
		if err != nil {
			t.Fatal(err)
		}
		mediaAttrs := app.YtDlpMediaResponseToStruct(attrsMap)
		differentAttrs := *mediaAttrs
		differentAttrs.Title = "Different title"

		mediaItem1, err := ta.CreateMediaItemFromBackendAttrs(ta.Ctx, source, mediaAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		mediaItem2, err := ta.CreateMediaItemFromBackendAttrs(ta.Ctx, source, &differentAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}

		if mediaItem1.ID != mediaItem2.ID {
			t.Errorf("id_1 = %v, id_2 = %v, want equal", mediaItem1.ID, mediaItem2.ID)
		}
		reloaded, err := store.Reload[store.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem2)
		if err != nil {
			t.Fatal(err)
		}
		if store.Deref(reloaded.Title) != differentAttrs.Title {
			t.Errorf("title = %v, want %v", store.Deref(reloaded.Title), differentAttrs.Title)
		}
	})

	t.Run("doesn't update fields like playlist_index", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		var attrsMap map[string]any
		err := store.DecodeJSON([]byte(apptest.MediaAttributesReturnFixture()), &attrsMap)
		if err != nil {
			t.Fatal(err)
		}
		attrsMap["playlist_index"] = 1
		mediaAttrs := app.YtDlpMediaResponseToStruct(attrsMap)

		differentAttrs := *mediaAttrs
		nineNineNineNine := 9999
		differentAttrs.PlaylistIndex = &nineNineNineNine

		if _, err := ta.CreateMediaItemFromBackendAttrs(ta.Ctx, source, mediaAttrs); err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		mediaItem2, err := ta.CreateMediaItemFromBackendAttrs(ta.Ctx, source, &differentAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}

		reloaded, err := store.Reload[store.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem2)
		if err != nil {
			t.Fatal(err)
		}
		if reloaded.PlaylistIndex != store.Deref(mediaAttrs.PlaylistIndex) {
			t.Errorf("playlist_index = %v, want %v (unchanged)", reloaded.PlaylistIndex, store.Deref(mediaAttrs.PlaylistIndex))
		}
	})
}

func TestMedia_UpdateMediaItem(t *testing.T) {
	t.Run("updating with valid data updates the media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		updateAttrs := store.Attrs{
			"media_id":       "updated-media-id",
			"title":          "Updated Product",
			"media_filepath": "/video/updated.mp4",
			"source_id":      source.ID,
		}

		updated, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, updateAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if store.Deref(updated.Title) != updateAttrs["title"] {
			t.Errorf("title = %v, want %v", store.Deref(updated.Title), updateAttrs["title"])
		}
		if updated.MediaID != updateAttrs["media_id"] {
			t.Errorf("media_id = %v, want %v", updated.MediaID, updateAttrs["media_id"])
		}
		if store.Deref(updated.MediaFilepath) != updateAttrs["media_filepath"] {
			t.Errorf("media_filepath = %v, want %v", store.Deref(updated.MediaFilepath), updateAttrs["media_filepath"])
		}
	})

	t.Run("updating strips playlist_index from the provided attrs", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"playlist_index": 5})
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		updateAttrs := store.Attrs{
			"media_id":       "updated-media-id",
			"title":          "Updated Product",
			"media_filepath": "/video/updated.mp4",
			"source_id":      source.ID,
			"playlist_index": 1,
		}

		updated, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, updateAttrs)
		if err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if updated.PlaylistIndex != 5 {
			t.Errorf("playlist_index = %v, want 5 (unchanged)", updated.PlaylistIndex)
		}
	})

	t.Run("updating with invalid data returns error changeset", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})
		invalidAttrs := store.Attrs{"title": nil, "media_id": nil, "media_filepath": nil}

		_, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, invalidAttrs)
		if _, ok := store.AsChangesetError(err); !ok {
			t.Fatalf("expected a *store.ChangesetError, got %v", err)
		}

		got, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, mediaItem) {
			t.Errorf("media_item was modified: got %+v, want %+v", got, mediaItem)
		}
	})
}

func TestMedia_DeleteMediaItem(t *testing.T) {
	t.Run("deletion deletes the media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, nil); err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if _, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound, got %v", err)
		}
	})

	t.Run("it also deletes attached tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})
		task := apptest.TaskFixture(t, ta, store.Attrs{"media_item_id": mediaItem.ID})

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, nil); err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if _, err := store.Reload[store.Task](ta.Ctx, ta.Q(ta.Ctx), task); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound reloading task, got %v", err)
		}
	})

	t.Run("does not delete the media_item's files by default", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, nil); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if _, err := os.Stat(store.Deref(mediaItem.MediaFilepath)); err != nil {
			t.Errorf("expected media_filepath to still exist: %v", err)
		}
	})

	t.Run("does delete the media item's metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})
		if _, err := ta.PreloadMediaItemFull(ta.Ctx, mediaItem); err != nil {
			t.Fatal(err)
		}

		metadataFilepath, err := ta.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, mediaItem, map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		thumbnailFilepath, err := ta.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}

		updated, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, store.Attrs{
			"metadata": map[string]any{
				"metadata_filepath":  metadataFilepath,
				"thumbnail_filepath": store.Deref(thumbnailFilepath),
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, updated, nil); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if updated.Metadata == nil {
			t.Fatal("expected updated media_item to have metadata preloaded")
		}
		if _, err := os.Stat(updated.Metadata.MetadataFilepath); err == nil {
			t.Errorf("expected metadata_filepath to be deleted")
		}
	})
}

func TestMedia_DeleteMediaItem_FileDeletion(t *testing.T) {
	stubUserScript := func(ta *apptest.TestApp) {
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })
	}

	t.Run("deletes the media item's files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, store.KW{store.Opt("delete_files", true)}); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if _, err := os.Stat(store.Deref(mediaItem.MediaFilepath)); err == nil {
			t.Errorf("expected media_filepath to be deleted")
		}
	})

	t.Run("deletes the media item's metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})
		if _, err := ta.PreloadMediaItemFull(ta.Ctx, mediaItem); err != nil {
			t.Fatal(err)
		}

		metadataFilepath, err := ta.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, mediaItem, map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		thumbnailFilepath, err := ta.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}

		updated, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, store.Attrs{
			"metadata": map[string]any{
				"metadata_filepath":  metadataFilepath,
				"thumbnail_filepath": store.Deref(thumbnailFilepath),
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, updated, store.KW{store.Opt("delete_files", true)}); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if _, err := os.Stat(updated.Metadata.MetadataFilepath); err == nil {
			t.Errorf("expected metadata_filepath to be deleted")
		}
	})

	t.Run("deletion deletes the media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, store.KW{store.Opt("delete_files", true)}); err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if _, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound, got %v", err)
		}
	})

	t.Run("deletes the parent folder if it is empty", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})
		rootDirectory := filepath.Dir(store.Deref(mediaItem.MediaFilepath))

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, store.KW{store.Opt("delete_files", true)}); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if _, err := os.Stat(rootDirectory); err == nil {
			t.Errorf("expected root_directory to be removed")
		}
	})

	t.Run("does not delete the parent folder if it is not empty", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})
		rootDirectory := filepath.Dir(store.Deref(mediaItem.MediaFilepath))
		if err := os.WriteFile(filepath.Join(rootDirectory, "test.txt"), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, store.KW{store.Opt("delete_files", true)}); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if _, err := os.Stat(rootDirectory); err != nil {
			t.Errorf("expected root_directory to still exist: %v", err)
		}

		if err := os.Remove(filepath.Join(rootDirectory, "test.txt")); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(rootDirectory); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("calls the user script runner", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})

		ta.UserScriptMock.Run.Expect(func(event string, data any) error {
			mi, ok := data.(*store.MediaItem)
			if !ok || mi.ID != mediaItem.ID {
				t.Errorf("expected data.id == %d, got %+v", mediaItem.ID, data)
			}
			if event != "media_deleted" {
				t.Errorf("event = %v, want media_deleted", event)
			}
			return nil
		})

		if _, err := ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, store.KW{store.Opt("delete_files", true)}); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
	})
}

func TestMedia_DeleteMediaFiles(t *testing.T) {
	stubUserScript := func(ta *apptest.TestApp) {
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })
	}

	t.Run("does not delete the media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})

		if _, err := ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, nil); err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if _, err := store.Reload[store.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem); err != nil {
			t.Errorf("expected reload to succeed, got %v", err)
		}
	})

	t.Run("deletes attached tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})
		task := apptest.TaskFixture(t, ta, store.Attrs{"media_item_id": mediaItem.ID})

		if _, err := ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, nil); err != nil {
			t.Fatalf("expected {:ok, %%store.MediaItem{}}, got error: %v", err)
		}
		if _, err := store.Reload[store.Task](ta.Ctx, ta.Q(ta.Ctx), task); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound reloading task, got %v", err)
		}
	})

	t.Run("deletes the media_item's files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})

		if _, err := os.Stat(store.Deref(mediaItem.MediaFilepath)); err != nil {
			t.Fatalf("expected media_filepath to exist before delete: %v", err)
		}
		if _, err := ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, nil); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if _, err := os.Stat(store.Deref(mediaItem.MediaFilepath)); err == nil {
			t.Errorf("expected media_filepath to be deleted")
		}
	})

	t.Run("does not delete the media item's metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})
		if _, err := ta.PreloadMediaItemFull(ta.Ctx, mediaItem); err != nil {
			t.Fatal(err)
		}

		metadataFilepath, err := ta.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, mediaItem, map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		thumbnailFilepath, err := ta.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatal(err)
		}

		updated, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, store.Attrs{
			"metadata": map[string]any{
				"metadata_filepath":  metadataFilepath,
				"thumbnail_filepath": store.Deref(thumbnailFilepath),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ta.PreloadMediaItemMetadata(ta.Ctx, updated); err != nil {
			t.Fatal(err)
		}
		metadata := updated.Metadata

		if _, err := ta.MediaDeleteMediaFiles(ta.Ctx, updated, nil); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if _, err := store.Reload[store.MediaMetadata](ta.Ctx, ta.Q(ta.Ctx), metadata); err != nil {
			t.Errorf("expected metadata to still exist, got %v", err)
		}
		if _, err := os.Stat(updated.Metadata.MetadataFilepath); err != nil {
			t.Errorf("expected metadata_filepath to still exist: %v", err)
		}

		// cleanup
		ta.MediaDeleteMediaItem(ta.Ctx, updated, store.KW{store.Opt("delete_files", true)})
	})

	t.Run("can take additional attributes update media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})

		updated, err := ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, store.Attrs{"prevent_download": true})
		if err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
		if !updated.PreventDownload {
			t.Errorf("expected prevent_download to be true")
		}
	})

	t.Run("calls the user script runner", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})

		ta.UserScriptMock.Run.Expect(func(event string, data any) error {
			mi, ok := data.(*store.MediaItem)
			if !ok || mi.ID != mediaItem.ID {
				t.Errorf("expected data.id == %d, got %+v", mediaItem.ID, data)
			}
			if event != "media_deleted" {
				t.Errorf("event = %v, want media_deleted", event)
			}
			return nil
		})

		if _, err := ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, nil); err != nil {
			t.Fatalf("expected {:ok, _}, got error: %v", err)
		}
	})
}

func TestMedia_ChangeMediaItem(t *testing.T) {
	t.Run("change_media_item/1 returns a media_item changeset", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})

		cs := ta.ChangeMediaItem(ta.Ctx, mediaItem, nil)
		if cs == nil {
			t.Errorf("expected a *store.Changeset")
		}
	})

	t.Run("validates the title doesn't start with 'youtube video #'", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})

		invalid := ta.ChangeMediaItem(ta.Ctx, mediaItem, store.Attrs{"title": "youtube video #123"})
		if invalid.Valid() {
			t.Errorf("expected the changeset to be invalid")
		}

		valid := ta.ChangeMediaItem(ta.Ctx, mediaItem, store.Attrs{"title": "any other title"})
		if !valid.Valid() {
			t.Errorf("expected the changeset to be valid, errors: %+v", valid.ErrorMap())
		}
	})
}

func TestMedia_ChangeMediaItem_UploadDateIndexChannel(t *testing.T) {
	t.Run("upload_date_index is set to 99 if it's the only video uploaded that day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})

		if mediaItem.UploadDateIndex != 99 {
			t.Errorf("upload_date_index = %v, want 99", mediaItem.UploadDateIndex)
		}
	})

	t.Run("upload_date_index is set to 98 if it's the second video uploaded that day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel"})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		two := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})

		if one.UploadDateIndex != 99 {
			t.Errorf("one.upload_date_index = %v, want 99", one.UploadDateIndex)
		}
		if two.UploadDateIndex != 98 {
			t.Errorf("two.upload_date_index = %v, want 98", two.UploadDateIndex)
		}
	})

	t.Run("upload_date_index doesn't decrement if the video is uploaded on a different day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel"})

		newItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		oldItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.NowMinus(1, "day")})

		if newItem.UploadDateIndex != 99 {
			t.Errorf("new.upload_date_index = %v, want 99", newItem.UploadDateIndex)
		}
		if oldItem.UploadDateIndex != 99 {
			t.Errorf("old.upload_date_index = %v, want 99", oldItem.UploadDateIndex)
		}
	})

	t.Run("recomputes upload_date_index if an upload_date is changed...somehow", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel"})

		newItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		oldItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.NowMinus(1, "day")})

		updated, err := ta.UpdateMediaItem(ta.Ctx, oldItem, store.Attrs{"uploaded_at": apptest.Now()})
		if err != nil {
			t.Fatal(err)
		}

		if newItem.UploadDateIndex != 99 {
			t.Errorf("new.upload_date_index = %v, want 99", newItem.UploadDateIndex)
		}
		if updated.UploadDateIndex != 98 {
			t.Errorf("updated.upload_date_index = %v, want 98", updated.UploadDateIndex)
		}
	})

	t.Run("upload_date_index doesn't decrement if the video is for a different source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		sourceOne := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel"})
		sourceTwo := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel"})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": sourceOne.ID, "uploaded_at": apptest.Now()})
		two := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": sourceTwo.ID, "uploaded_at": apptest.Now()})

		if one.UploadDateIndex != 99 {
			t.Errorf("one.upload_date_index = %v, want 99", one.UploadDateIndex)
		}
		if two.UploadDateIndex != 99 {
			t.Errorf("two.upload_date_index = %v, want 99", two.UploadDateIndex)
		}
	})

	t.Run("upload_date_index doesn't decrement if the a video's upload_date is updated but doesn't change", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel"})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})

		updated, err := ta.UpdateMediaItem(ta.Ctx, one, store.Attrs{"uploaded_at": apptest.Now(), "title": "New title"})
		if err != nil {
			t.Fatal(err)
		}

		if updated.UploadDateIndex != 99 {
			t.Errorf("updated.upload_date_index = %v, want 99", updated.UploadDateIndex)
		}
	})

	t.Run("upload_date_index doesn't increment if the a video's upload_date is changed to the same day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel"})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})

		updated, err := ta.UpdateMediaItem(ta.Ctx, one, store.Attrs{"uploaded_at": apptest.NowPlus(1, "minute"), "title": "New title"})
		if err != nil {
			t.Fatal(err)
		}

		if updated.UploadDateIndex != 99 {
			t.Errorf("updated.upload_date_index = %v, want 99", updated.UploadDateIndex)
		}
	})
}

func TestMedia_ChangeMediaItem_UploadDateIndexPlaylist(t *testing.T) {
	t.Run("upload_date_index is set to 0 if it's the only video uploaded that day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})

		if mediaItem.UploadDateIndex != 0 {
			t.Errorf("upload_date_index = %v, want 0", mediaItem.UploadDateIndex)
		}
	})

	t.Run("upload_date_index is set to 1 if it's the second video uploaded that day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist"})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		two := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})

		if one.UploadDateIndex != 0 {
			t.Errorf("one.upload_date_index = %v, want 0", one.UploadDateIndex)
		}
		if two.UploadDateIndex != 1 {
			t.Errorf("two.upload_date_index = %v, want 1", two.UploadDateIndex)
		}
	})

	t.Run("upload_date_index doesn't increment if the video is uploaded on a different day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist"})

		newItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		oldItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.NowMinus(1, "day")})

		if newItem.UploadDateIndex != 0 {
			t.Errorf("new.upload_date_index = %v, want 0", newItem.UploadDateIndex)
		}
		if oldItem.UploadDateIndex != 0 {
			t.Errorf("old.upload_date_index = %v, want 0", oldItem.UploadDateIndex)
		}
	})

	t.Run("recomputes upload_date_index if an upload_date is changed...somehow", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist"})

		newItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		oldItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.NowMinus(1, "day")})

		updated, err := ta.UpdateMediaItem(ta.Ctx, oldItem, store.Attrs{"uploaded_at": apptest.Now()})
		if err != nil {
			t.Fatal(err)
		}

		if newItem.UploadDateIndex != 0 {
			t.Errorf("new.upload_date_index = %v, want 0", newItem.UploadDateIndex)
		}
		if updated.UploadDateIndex != 1 {
			t.Errorf("updated.upload_date_index = %v, want 1", updated.UploadDateIndex)
		}
	})

	t.Run("upload_date_index doesn't increment if the video is for a different source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		sourceOne := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist"})
		sourceTwo := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist"})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": sourceOne.ID, "uploaded_at": apptest.Now()})
		two := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": sourceTwo.ID, "uploaded_at": apptest.Now()})

		if one.UploadDateIndex != 0 {
			t.Errorf("one.upload_date_index = %v, want 0", one.UploadDateIndex)
		}
		if two.UploadDateIndex != 0 {
			t.Errorf("two.upload_date_index = %v, want 0", two.UploadDateIndex)
		}
	})

	t.Run("upload_date_index doesn't increment if the a video's upload_date is updated but doesn't change", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist"})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})

		updated, err := ta.UpdateMediaItem(ta.Ctx, one, store.Attrs{"uploaded_at": apptest.Now(), "title": "New title"})
		if err != nil {
			t.Fatal(err)
		}

		if updated.UploadDateIndex != 0 {
			t.Errorf("updated.upload_date_index = %v, want 0", updated.UploadDateIndex)
		}
	})

	t.Run("upload_date_index doesn't increment if the a video's upload_date is changed to the same day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist"})

		one := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})
		apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "uploaded_at": apptest.Now()})

		updated, err := ta.UpdateMediaItem(ta.Ctx, one, store.Attrs{"uploaded_at": apptest.NowPlus(1, "minute"), "title": "New title"})
		if err != nil {
			t.Fatal(err)
		}

		if updated.UploadDateIndex != 0 {
			t.Errorf("updated.upload_date_index = %v, want 0", updated.UploadDateIndex)
		}
	})
}

// TestMediaItemJSONMatchesElixirGolden checks store.MediaItem.MarshalJSON
// (the Jason.Encoder defimpl for store.MediaItem, ported from media_item.ex)
// against testdata/elixir/golden/user_script_media_item.json, which is
// `Jason.encode!(media_item, pretty: true)` for media item 1 of
// testdata/elixir/populated.db with source/metadata preloaded. Comparison is
// semantic (decode both sides to map[string]any) since key order and
// whitespace aren't meaningful.
func TestMediaItemJSONMatchesElixirGolden(t *testing.T) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	a := &app.App{Store: &store.Store{DB: d}}
	ctx := context.Background()

	mediaItem, err := store.Get[store.MediaItem](ctx, a.Q(ctx), 1)
	if err != nil {
		t.Fatalf("loading media_item 1: %v", err)
	}
	if _, err := a.PreloadMediaItemSourceAndProfile(ctx, mediaItem); err != nil {
		t.Fatalf("preloading source/media_profile: %v", err)
	}

	gotBytes, err := json.Marshal(mediaItem)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	wantBytes, err := os.ReadFile("../../testdata/elixir/golden/user_script_media_item.json")
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}

	var got, want map[string]any
	if err := json.Unmarshal(gotBytes, &got); err != nil {
		t.Fatalf("decoding got JSON: %v", err)
	}
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatalf("decoding golden JSON: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("store.MediaItem JSON does not match golden.\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestMedia_ComputeAndSaveMediaFilesize(t *testing.T) {
	t.Run("updates the media item with the file size", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{})

		if mediaItem.MediaSizeBytes != nil && *mediaItem.MediaSizeBytes != 0 {
			t.Error("media_size_bytes should initially be nil or 0")
		}

		result, err := ta.App.ComputeAndSaveMediaFilesize(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.MediaSizeBytes == nil || *result.MediaSizeBytes == 0 {
			t.Error("media_size_bytes should be set and non-zero")
		}
	})

	t.Run("returns the error if operation fails", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{
			"media_filepath": "/nonexistent/file.mkv",
		})

		_, err := ta.App.ComputeAndSaveMediaFilesize(ta.Ctx, mediaItem)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}
