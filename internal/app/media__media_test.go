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
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

func TestMedia_Schema(t *testing.T) {
	t.Run("media_metadata is deleted when media_item is deleted", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{
			Metadata: &store.MediaMetadataParams{
				MetadataFilepath:  store.Ptr("/metadata.json.gz"),
				ThumbnailFilepath: store.Ptr("/thumbnail.jpg"),
			},
		})
		mustOK(t)(ta.PreloadMediaItemMetadata(ta.Ctx, mediaItem))
		metadata := mediaItem.Metadata

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, false))

		if _, err := store.Reload[store.MediaMetadata](ta.Ctx, ta.Q(ta.Ctx), metadata); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound reloading metadata, got %v", err)
		}
	})

	t.Run("can be JSON encoded without error", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mustOK(t)(ta.PreloadMediaItemSourceAndProfile(ta.Ctx, mediaItem))

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

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})

		got, err := ta.ListMediaItems(ta.Ctx)
		must(t, err)
		if !reflect.DeepEqual(got, []*store.MediaItem{mediaItem}) {
			t.Errorf("got %+v, want [%+v]", got, mediaItem)
		}
	})
}

// wantItems asserts got is exactly want, in order (nil and empty are equal).
func wantItems(t testing.TB, got []*store.MediaItem, want ...*store.MediaItem) {
	t.Helper()
	if len(got)+len(want) > 0 && !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestMedia_ListUpgradeableMediaItems(t *testing.T) {
	ago := func(days int) *time.Time { return store.Ptr(apptest.NowMinus(days, "days")) }
	now := func() *time.Time { return store.Ptr(apptest.Now()) }

	cases := []struct {
		name string
		item store.MediaItemParams
		want bool
	}{
		{"returns media eligible for redownload", store.MediaItemParams{UploadedAt: ago(6), MediaDownloadedAt: ago(5)}, true},
		{"returns media items that were downloaded in past but still meet redownload delay", store.MediaItemParams{UploadedAt: ago(20), MediaDownloadedAt: ago(19)}, true},
		{"does not return media items without a media_downloaded_at", store.MediaItemParams{UploadedAt: ago(5)}, false},
		{"does not return media items that are set to prevent download", store.MediaItemParams{UploadedAt: ago(5), MediaDownloadedAt: now(), PreventDownload: store.Ptr(true)}, false},
		{"does not return media items that have been culled", store.MediaItemParams{UploadedAt: ago(5), MediaDownloadedAt: now(), CulledAt: now()}, false},
		{"does not return media items before the download delay", store.MediaItemParams{UploadedAt: ago(3), MediaDownloadedAt: ago(3)}, false},
		{"does not return media items that have already been redownloaded", store.MediaItemParams{UploadedAt: ago(5), MediaDownloadedAt: now(), MediaRedownloadedAt: now()}, false},
		{"does not return media items that were first downloaded well after the uploaded_at", store.MediaItemParams{MediaDownloadedAt: now(), UploadedAt: ago(20)}, false},
		{"does not return media items that were recently uploaded", store.MediaItemParams{MediaDownloadedAt: now(), UploadedAt: ago(2)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{RedownloadDelayDays: store.Ptr(4)})
			source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(profile.ID)})
			c.item.SourceID = store.Ptr(source.ID)
			mediaItem := apptest.MediaItemFixture(t, ta, c.item)

			got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
			must(t, err)
			if c.want {
				wantItems(t, got, mediaItem)
			} else {
				wantItems(t, got)
			}
		})
	}

	t.Run("does not return media items without a redownload delay", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(mediaProfile.ID)})

		apptest.MediaItemFixture(t, ta, store.MediaItemParams{
			SourceID:          store.Ptr(source.ID),
			UploadedAt:        ago(6),
			MediaDownloadedAt: ago(5),
		})

		got, err := ta.ListUpgradeableMediaItems(ta.Ctx)
		must(t, err)
		wantItems(t, got)
	})
}

func TestMedia_ListPendingMediaItemsFor(t *testing.T) {
	t.Run("it returns pending without a filepath for a given source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		otherSource := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(otherSource.ID), Clear: store.ClearMediaFilepath})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		must(t, err)
		wantItems(t, got, mediaItem)
	})

	t.Run("it does not return media_items with media_filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		apptest.MediaItemFixture(t, ta, store.MediaItemParams{
			SourceID:      store.Ptr(source.ID),
			MediaFilepath: store.Ptr("/video/" + apptest.Now().Format("20060102150405") + ".mp4"),
		})

		got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
		must(t, err)
		wantItems(t, got)
	})
}

// filterCase describes a profile/source configuration plus the media items
// created under it; used by both the ListPendingMediaItemsFor and
// PendingDownload tables.
type filterCase struct {
	name    string
	profile store.MediaProfileParams
	source  store.SourceParams
	items   []store.MediaItemParams
	want    []int // indexes into items, in order
}

func (c filterCase) setup(t *testing.T) (*apptest.TestApp, *store.Source, []*store.MediaItem) {
	ta := apptest.NewApp(t)
	profile := apptest.MediaProfileFixture(t, ta, c.profile)
	c.source.MediaProfileID = store.Ptr(profile.ID)
	source := apptest.SourceFixture(t, ta, c.source)
	var items []*store.MediaItem
	for _, p := range c.items {
		p.SourceID = store.Ptr(source.ID)
		p.Clear |= store.ClearMediaFilepath
		items = append(items, apptest.MediaItemFixture(t, ta, p))
	}
	return ta, source, items
}

func TestMedia_ListPendingMediaItemsFor_Filters(t *testing.T) {
	var (
		normal = store.MediaItemParams{}
		live   = store.MediaItemParams{Livestream: store.Ptr(true)}
		short  = store.MediaItemParams{ShortFormContent: store.Ptr(true)}
		inc    = store.MediaProfileParams{ShortsBehaviour: store.Ptr(store.MediaProfileShortsBehaviourInclude), LivestreamBehaviour: store.Ptr(store.MediaProfileLivestreamBehaviourInclude)}
		only   = store.MediaProfileParams{ShortsBehaviour: store.Ptr(store.MediaProfileShortsBehaviourOnly), LivestreamBehaviour: store.Ptr(store.MediaProfileLivestreamBehaviourOnly)}
		excl   = store.MediaProfileParams{ShortsBehaviour: store.Ptr(store.MediaProfileShortsBehaviourExclude), LivestreamBehaviour: store.Ptr(store.MediaProfileLivestreamBehaviourExclude)}
		shorts = func(b store.MediaProfileShortsBehaviour) store.MediaProfileParams {
			return store.MediaProfileParams{ShortsBehaviour: store.Ptr(b)}
		}
		lives = func(b store.MediaProfileLivestreamBehaviour) store.MediaProfileParams {
			return store.MediaProfileParams{LivestreamBehaviour: store.Ptr(b)}
		}
		uploaded = func(days int) store.MediaItemParams {
			return store.MediaItemParams{UploadedAt: store.Ptr(apptest.NowMinus(days, "days"))}
		}
		titled   = func(s string) store.MediaItemParams { return store.MediaItemParams{Title: store.Ptr(s)} }
		duration = func(n int) store.MediaItemParams { return store.MediaItemParams{DurationSeconds: store.Ptr(n)} }
		regex    = store.SourceParams{TitleFilterRegex: store.Ptr("(?i)^FOO$")}
		minMax   = store.SourceParams{MinDurationSeconds: store.Ptr(10), MaxDurationSeconds: store.Ptr(20)}
		durs     = []store.MediaItemParams{duration(5), duration(15), duration(25)}
	)

	cases := []filterCase{
		{"returns shorts and normal media when shorts_behaviour is :include", shorts(store.MediaProfileShortsBehaviourInclude), store.SourceParams{}, []store.MediaItemParams{normal, short}, []int{0, 1}},
		{"returns only shorts when shorts_behaviour is :only", shorts(store.MediaProfileShortsBehaviourOnly), store.SourceParams{}, []store.MediaItemParams{normal, short}, []int{1}},
		{"returns only normal media when shorts_behaviour is :exclude", shorts(store.MediaProfileShortsBehaviourExclude), store.SourceParams{}, []store.MediaItemParams{normal, short}, []int{0}},
		{"returns livestreams and normal media when livestream_behaviour is :include", lives(store.MediaProfileLivestreamBehaviourInclude), store.SourceParams{}, []store.MediaItemParams{normal, live}, []int{0, 1}},
		{"returns only livestreams when livestream_behaviour is :only", lives(store.MediaProfileLivestreamBehaviourOnly), store.SourceParams{}, []store.MediaItemParams{normal, live}, []int{1}},
		{"returns only normal media when livestream_behaviour is :exclude", lives(store.MediaProfileLivestreamBehaviourExclude), store.SourceParams{}, []store.MediaItemParams{normal, live}, []int{0}},
		{"returns livestreams, shorts, and normal media when behaviour is :include", inc, store.SourceParams{}, []store.MediaItemParams{normal, live, short}, []int{0, 1, 2}},
		{"returns only livestreams and shorts when behaviour is :only", only, store.SourceParams{}, []store.MediaItemParams{normal, live, short}, []int{1, 2}},
		{"returns only normal media when behaviour is :exclude", excl, store.SourceParams{}, []store.MediaItemParams{normal, live, short}, []int{0}},
		{":only and :exclude return the expected results", store.MediaProfileParams{ShortsBehaviour: only.ShortsBehaviour, LivestreamBehaviour: excl.LivestreamBehaviour}, store.SourceParams{}, []store.MediaItemParams{normal, live, short}, []int{2}},
		{"does not return media items with an upload date before the cutoff date", store.MediaProfileParams{}, store.SourceParams{DownloadCutoffDate: store.Ptr(apptest.NowMinus(1, "day"))}, []store.MediaItemParams{uploaded(2), uploaded(0)}, []int{1}},
		{"does not apply a cutoff if there is no cutoff date", store.MediaProfileParams{}, store.SourceParams{Clear: store.ClearDownloadCutoffDate}, []store.MediaItemParams{uploaded(2), uploaded(0)}, []int{0, 1}},
		{"returns only media items that match the title regex", store.MediaProfileParams{}, regex, []store.MediaItemParams{titled("foo"), titled("bar")}, []int{0}},
		{"does not apply a regex if none is specified", store.MediaProfileParams{}, store.SourceParams{TitleFilterRegex: store.Ptr("")}, []store.MediaItemParams{titled("foo"), titled("bar")}, []int{0, 1}},
		{"returns media items that meet the min and max duration", store.MediaProfileParams{}, minMax, durs, []int{1}},
		{"does not apply a min duration if none is specified", store.MediaProfileParams{}, store.SourceParams{MaxDurationSeconds: store.Ptr(20), Clear: store.ClearMinDurationSeconds}, durs, []int{0, 1}},
		{"does not apply a max duration if none is specified", store.MediaProfileParams{}, store.SourceParams{MinDurationSeconds: store.Ptr(10), Clear: store.ClearMaxDurationSeconds}, durs, []int{1, 2}},
		{"does not apply a min or max duration if none are specified", store.MediaProfileParams{}, store.SourceParams{Clear: store.ClearMinDurationSeconds | store.ClearMaxDurationSeconds}, durs, []int{0, 1, 2}},
		{"returns only media items that are not prevented from downloading", store.MediaProfileParams{}, store.SourceParams{}, []store.MediaItemParams{{PreventDownload: store.Ptr(true)}, {PreventDownload: store.Ptr(false)}}, []int{1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ta, source, items := c.setup(t)
			got, err := ta.ListPendingMediaItemsFor(ta.Ctx, source)
			must(t, err)
			var want []*store.MediaItem
			for _, i := range c.want {
				want = append(want, items[i])
			}
			wantItems(t, got, want...)
		})
	}
}

func TestMedia_PendingDownload(t *testing.T) {
	// The single media item's index 0 in want means "pending".
	var (
		regex  = store.SourceParams{TitleFilterRegex: store.Ptr("(?i)^FOO$")}
		minMax = store.SourceParams{MinDurationSeconds: store.Ptr(10), MaxDurationSeconds: store.Ptr(20)}
		cutoff = func(days int) store.SourceParams {
			return store.SourceParams{DownloadCutoffDate: store.Ptr(apptest.NowMinus(days, "days"))}
		}
		one = func(p store.MediaItemParams) []store.MediaItemParams { return []store.MediaItemParams{p} }
		up  = func(days int) store.MediaItemParams {
			return store.MediaItemParams{UploadedAt: store.Ptr(apptest.NowMinus(days, "days"))}
		}
		yes = []int{0}
		no  = []int(nil)
		mp  store.MediaProfileParams
		sp  store.SourceParams
	)

	cases := []filterCase{
		{"returns true when the media hasn't been downloaded", mp, sp, one(store.MediaItemParams{}), yes},
		{"returns false if the media hasn't been downloaded but the profile doesn't DL shorts", store.MediaProfileParams{ShortsBehaviour: store.Ptr(store.MediaProfileShortsBehaviourExclude)}, sp, one(store.MediaItemParams{ShortFormContent: store.Ptr(true)}), no},
		{"returns false if the media hasn't been downloaded but the profile doesn't DL livestreams", store.MediaProfileParams{LivestreamBehaviour: store.Ptr(store.MediaProfileLivestreamBehaviourExclude)}, sp, one(store.MediaItemParams{Livestream: store.Ptr(true)}), no},
		{"returns true if there is a cutoff date before the media's upload date", mp, cutoff(2), one(up(1)), yes},
		{"returns true if the cutoff date is equal to the upload date", mp, cutoff(2), one(up(2)), yes},
		{"returns false if there is a cutoff date after the media's upload date", mp, cutoff(1), one(up(2)), no},
		{"returns true if there is no cutoff date", mp, store.SourceParams{Clear: store.ClearDownloadCutoffDate}, one(up(1)), yes},
		{"returns true if the content matches the title regex", mp, regex, one(store.MediaItemParams{Title: store.Ptr("foo")}), yes},
		{"returns false if the content doesn't match the title regex", mp, regex, one(store.MediaItemParams{Title: store.Ptr("bar")}), no},
		{"return true if there is no title regex", mp, store.SourceParams{TitleFilterRegex: store.Ptr("")}, one(store.MediaItemParams{Title: store.Ptr("foo")}), yes},
		{"returns true if the duration is between the min and max", mp, minMax, one(store.MediaItemParams{DurationSeconds: store.Ptr(15)}), yes},
		{"returns false if the duration is below the min", mp, minMax, one(store.MediaItemParams{DurationSeconds: store.Ptr(5)}), no},
		{"returns false if the duration is above the max", mp, minMax, one(store.MediaItemParams{DurationSeconds: store.Ptr(25)}), no},
		{"returns true if there is no min or max duration", mp, store.SourceParams{Clear: store.ClearMinDurationSeconds | store.ClearMaxDurationSeconds}, one(store.MediaItemParams{DurationSeconds: store.Ptr(15)}), yes},
		{"returns true if the media item is not prevented from downloading", mp, sp, one(store.MediaItemParams{PreventDownload: store.Ptr(false)}), yes},
		{"returns false if the media item is prevented from downloading", mp, sp, one(store.MediaItemParams{PreventDownload: store.Ptr(true)}), no},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ta, _, items := c.setup(t)
			got, err := ta.PendingDownload(ta.Ctx, items[0])
			must(t, err)
			if got != (len(c.want) == 1) {
				t.Errorf("PendingDownload = %v, want %v", got, len(c.want) == 1)
			}
		})
	}

	t.Run("returns false if the media has been downloaded", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{MediaFilepath: store.Ptr("/video/downloaded.mp4")})

		got, err := ta.PendingDownload(ta.Ctx, mediaItem)
		must(t, err)
		if got {
			t.Errorf("expected not pending download")
		}
	})
}

// itemWithMetadataFiles returns a media item with real metadata and thumbnail
// files stored and attached to its metadata record.
func itemWithMetadataFiles(t *testing.T, ta *apptest.TestApp) *store.MediaItem {
	t.Helper()
	ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
		return "", nil
	})
	mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})
	mustOK(t)(ta.PreloadMediaItemFull(ta.Ctx, mediaItem))

	metadataFilepath, err := ta.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, mediaItem, map[string]any{})
	must(t, err)
	thumbnailFilepath, err := ta.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
	must(t, err)

	updated, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, store.MediaItemParams{
		Metadata: &store.MediaMetadataParams{
			MetadataFilepath:  &metadataFilepath,
			ThumbnailFilepath: thumbnailFilepath,
		},
	})
	must(t, err)
	return updated
}

func TestMedia_Search(t *testing.T) {
	setup := func(t *testing.T) (*apptest.TestApp, int64) {
		t.Helper()
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{
			Title:       store.Ptr("The quick brown fox"),
			Description: store.Ptr("jumps over the lazy dog"),
		})
		return ta, mediaItem.ID
	}

	for _, c := range []struct{ name, term string }{
		{"searches based on title", "quick"},
		{"searches based on description", "lazy"},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta, mediaItemID := setup(t)

			got, err := ta.Search(ta.Ctx, c.term, 0)
			must(t, err)
			if len(got) != 1 || got[0].ID != mediaItemID {
				t.Errorf("got %+v, want a single result with id %d", got, mediaItemID)
			}
		})
	}

	t.Run("adds a matching_search_term attribute with the relevant text", func(t *testing.T) {
		ta, _ := setup(t)

		got, err := ta.Search(ta.Ctx, "quick", 0)
		must(t, err)
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
		apptest.MediaItemFixture(t, ta, store.MediaItemParams{Title: store.Ptr("foobar baz"), Clear: store.ClearDescription})

		got, err := ta.Search(ta.Ctx, "baz", 0)
		must(t, err)
		if len(got) != 1 {
			t.Fatalf("got %+v, want a single result", got)
		}
		if got[0].MatchingSearchTerm == nil {
			t.Errorf("expected matching_search_term to not be nil")
		}
	})

	t.Run("optionally lets you specify a limit", func(t *testing.T) {
		ta, _ := setup(t)
		apptest.MediaItemFixture(t, ta, store.MediaItemParams{Title: store.Ptr("The small gray dog")})

		got, err := ta.Search(ta.Ctx, "dog", 1)
		must(t, err)
		if len(got) != 1 {
			t.Errorf("got %+v, want a single result", got)
		}
	})

	// blank term (Go has no nil string), quotes/apostrophes and trailing
	// operands must not blow up.
	for _, term := range []string{"", "don't expl'ode", `dont expl"o"de`, `dont explo"de`, "foo OR", "foo AND"} {
		t.Run("returns an empty list without error for "+strconv.Quote(term), func(t *testing.T) {
			ta, _ := setup(t)

			got, err := ta.Search(ta.Ctx, term, 0)
			must(t, err)
			if len(got) != 0 {
				t.Errorf("got %+v, want []", got)
			}
		})
	}
}

func TestMedia_GetMediaItem(t *testing.T) {
	t.Run("it returns the media_item with given id", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})

		got, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		must(t, err)
		if !reflect.DeepEqual(got, mediaItem) {
			t.Errorf("got %+v, want %+v", got, mediaItem)
		}
	})
}

func TestMedia_CreateMediaItem(t *testing.T) {
	validParams := func(source *store.Source, mediaID string) store.MediaItemParams {
		return store.MediaItemParams{
			MediaID:       store.Ptr(mediaID),
			Title:         store.Ptr("Some Product"),
			MediaFilepath: store.Ptr("/video/file.mp4"),
			SourceID:      store.Ptr(source.ID),
			OriginalURL:   store.Ptr("https://www.youtube.com/channel/abc"),
			UploadedAt:    store.Ptr(apptest.Now()),
		}
	}

	t.Run("creating with valid data creates a media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		params := validParams(source, "media-"+strconv.Itoa(int(source.ID)))

		mediaItem, err := ta.CreateMediaItem(ta.Ctx, params)
		must(t, err)

		if store.Deref(mediaItem.Title) != *params.Title {
			t.Errorf("title = %v, want %v", store.Deref(mediaItem.Title), *params.Title)
		}
		if mediaItem.MediaID != *params.MediaID {
			t.Errorf("media_id = %v, want %v", mediaItem.MediaID, *params.MediaID)
		}
		if store.Deref(mediaItem.MediaFilepath) != *params.MediaFilepath {
			t.Errorf("media_filepath = %v, want %v", store.Deref(mediaItem.MediaFilepath), *params.MediaFilepath)
		}
	})

	t.Run("automatically sets the UUID", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		mediaItem, err := ta.CreateMediaItem(ta.Ctx, validParams(source, "media-1"))
		must(t, err)
		if mediaItem.UUID == nil || len(*mediaItem.UUID) != 36 {
			t.Errorf("uuid = %v, want length 36", mediaItem.UUID)
		}
	})

	t.Run("creating with invalid data returns error changeset", func(t *testing.T) {
		ta := apptest.NewApp(t)
		invalid := store.MediaItemParams{MediaID: store.Ptr(""), Clear: store.ClearTitle | store.ClearMediaFilepath}

		_, err := ta.CreateMediaItem(ta.Ctx, invalid)
		if _, ok := store.AsValidationErrors(err); !ok {
			t.Fatalf("expected a store.ValidationErrors, got %v", err)
		}
	})

	t.Run("a duplicate media_id for the source is reported on media_id", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		mustOK(t)(ta.CreateMediaItem(ta.Ctx, validParams(source, "media-1")))
		_, err := ta.CreateMediaItem(ta.Ctx, validParams(source, "media-1"))
		errs, ok := store.AsValidationErrors(err)
		if !ok {
			t.Fatalf("expected a store.ValidationErrors, got %v", err)
		}
		if got, want := errs, map[string][]string{"media_id": {"has already been taken"}}; !reflect.DeepEqual(got, want) {
			t.Errorf("errors = %v, want %v", got, want)
		}
	})
}

func TestMedia_UpsertMediaItemFromYtDlp(t *testing.T) {
	t.Run("creates a media item for a given source and attributes", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		attrsMap := mediaAttrsMap(t)
		mediaAttrs := app.YtDlpMediaResponseToStruct(attrsMap)

		mediaItem, err := ta.UpsertMediaItemFromYtDlp(ta.Ctx, source, mediaAttrs)
		must(t, err)

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
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		attrsMap := mediaAttrsMap(t)
		mediaAttrs := app.YtDlpMediaResponseToStruct(attrsMap)
		differentAttrs := *mediaAttrs
		differentAttrs.Title = "Different title"

		mediaItem1, err := ta.UpsertMediaItemFromYtDlp(ta.Ctx, source, mediaAttrs)
		must(t, err)
		mediaItem2, err := ta.UpsertMediaItemFromYtDlp(ta.Ctx, source, &differentAttrs)
		must(t, err)

		if mediaItem1.ID != mediaItem2.ID {
			t.Errorf("id_1 = %v, id_2 = %v, want equal", mediaItem1.ID, mediaItem2.ID)
		}
		reloaded, err := store.Reload[store.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem2)
		must(t, err)
		if store.Deref(reloaded.Title) != differentAttrs.Title {
			t.Errorf("title = %v, want %v", store.Deref(reloaded.Title), differentAttrs.Title)
		}
	})

	t.Run("doesn't update fields like playlist_index", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		attrsMap := mediaAttrsMap(t)
		attrsMap["playlist_index"] = 1
		mediaAttrs := app.YtDlpMediaResponseToStruct(attrsMap)

		differentAttrs := *mediaAttrs
		nineNineNineNine := 9999
		differentAttrs.PlaylistIndex = &nineNineNineNine

		mustOK(t)(ta.UpsertMediaItemFromYtDlp(ta.Ctx, source, mediaAttrs))
		mediaItem2, err := ta.UpsertMediaItemFromYtDlp(ta.Ctx, source, &differentAttrs)
		must(t, err)

		reloaded, err := store.Reload[store.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem2)
		must(t, err)
		if reloaded.PlaylistIndex != store.Deref(mediaAttrs.PlaylistIndex) {
			t.Errorf("playlist_index = %v, want %v (unchanged)", reloaded.PlaylistIndex, store.Deref(mediaAttrs.PlaylistIndex))
		}
	})
}

func TestMedia_UpdateMediaItem(t *testing.T) {
	t.Run("updating with valid data updates the media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		params := store.MediaItemParams{
			MediaID:       store.Ptr("updated-media-id"),
			Title:         store.Ptr("Updated Product"),
			MediaFilepath: store.Ptr("/video/updated.mp4"),
			SourceID:      store.Ptr(source.ID),
		}

		updated, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, params)
		must(t, err)
		if store.Deref(updated.Title) != *params.Title {
			t.Errorf("title = %v, want %v", store.Deref(updated.Title), *params.Title)
		}
		if updated.MediaID != *params.MediaID {
			t.Errorf("media_id = %v, want %v", updated.MediaID, *params.MediaID)
		}
		if store.Deref(updated.MediaFilepath) != *params.MediaFilepath {
			t.Errorf("media_filepath = %v, want %v", store.Deref(updated.MediaFilepath), *params.MediaFilepath)
		}
	})

	t.Run("updating strips playlist_index from the provided params", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{PlaylistIndex: store.Ptr(5)})

		updated, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, store.MediaItemParams{
			Title:         store.Ptr("Updated Product"),
			PlaylistIndex: store.Ptr(1),
		})
		must(t, err)
		if updated.PlaylistIndex != 5 {
			t.Errorf("playlist_index = %v, want 5 (unchanged)", updated.PlaylistIndex)
		}
	})

	t.Run("updating with invalid data returns error changeset", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		invalid := store.MediaItemParams{MediaID: store.Ptr(""), Clear: store.ClearTitle | store.ClearMediaFilepath}

		_, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, invalid)
		if _, ok := store.AsValidationErrors(err); !ok {
			t.Fatalf("expected a store.ValidationErrors, got %v", err)
		}

		got, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		must(t, err)
		if !reflect.DeepEqual(got, mediaItem) {
			t.Errorf("media_item was modified: got %+v, want %+v", got, mediaItem)
		}
	})
}

func TestMedia_DeleteMediaItem(t *testing.T) {
	t.Run("deletion deletes the media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, false))
		if _, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound, got %v", err)
		}
	})

	t.Run("it also deletes attached tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{MediaItemID: store.Ptr(mediaItem.ID)})

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, false))
		if _, err := store.Reload[store.Task](ta.Ctx, ta.Q(ta.Ctx), task); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound reloading task, got %v", err)
		}
	})

	t.Run("does not delete the media_item's files by default", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, false))
		if _, err := os.Stat(store.Deref(mediaItem.MediaFilepath)); err != nil {
			t.Errorf("expected media_filepath to still exist: %v", err)
		}
	})

	t.Run("does delete the media item's metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		updated := itemWithMetadataFiles(t, ta)

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, updated, false))
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
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, true))
		if _, err := os.Stat(store.Deref(mediaItem.MediaFilepath)); err == nil {
			t.Errorf("expected media_filepath to be deleted")
		}
	})

	t.Run("deletes the media item's metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		updated := itemWithMetadataFiles(t, ta)

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, updated, true))
		if _, err := os.Stat(updated.Metadata.MetadataFilepath); err == nil {
			t.Errorf("expected metadata_filepath to be deleted")
		}
	})

	t.Run("deletion deletes the media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, true))
		if _, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound, got %v", err)
		}
	})

	t.Run("deletes the parent folder if it is empty", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})
		rootDirectory := filepath.Dir(store.Deref(mediaItem.MediaFilepath))

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, true))
		if _, err := os.Stat(rootDirectory); err == nil {
			t.Errorf("expected root_directory to be removed")
		}
	})

	t.Run("does not delete the parent folder if it is not empty", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})
		rootDirectory := filepath.Dir(store.Deref(mediaItem.MediaFilepath))
		must(t, os.WriteFile(filepath.Join(rootDirectory, "test.txt"), []byte(""), 0o644))

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, true))
		if _, err := os.Stat(rootDirectory); err != nil {
			t.Errorf("expected root_directory to still exist: %v", err)
		}

		must(t, os.Remove(filepath.Join(rootDirectory, "test.txt")))
		must(t, os.Remove(rootDirectory))
	})

	t.Run("calls the user script runner", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

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

		mustOK(t)(ta.MediaDeleteMediaItem(ta.Ctx, mediaItem, true))
	})
}

func TestMedia_DeleteMediaFiles(t *testing.T) {
	stubUserScript := func(ta *apptest.TestApp) {
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })
	}

	t.Run("does not delete the media_item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})

		mustOK(t)(ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, store.MediaItemParams{}))
		if _, err := store.Reload[store.MediaItem](ta.Ctx, ta.Q(ta.Ctx), mediaItem); err != nil {
			t.Errorf("expected reload to succeed, got %v", err)
		}
	})

	t.Run("deletes attached tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{MediaItemID: store.Ptr(mediaItem.ID)})

		mustOK(t)(ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, store.MediaItemParams{}))
		if _, err := store.Reload[store.Task](ta.Ctx, ta.Q(ta.Ctx), task); err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound reloading task, got %v", err)
		}
	})

	t.Run("deletes the media_item's files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		mustOK(t)(os.Stat(store.Deref(mediaItem.MediaFilepath)))
		mustOK(t)(ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, store.MediaItemParams{}))
		if _, err := os.Stat(store.Deref(mediaItem.MediaFilepath)); err == nil {
			t.Errorf("expected media_filepath to be deleted")
		}
	})

	t.Run("does not delete the media item's metadata files", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		updated := itemWithMetadataFiles(t, ta)
		mustOK(t)(ta.PreloadMediaItemMetadata(ta.Ctx, updated))
		metadata := updated.Metadata

		mustOK(t)(ta.MediaDeleteMediaFiles(ta.Ctx, updated, store.MediaItemParams{}))
		if _, err := store.Reload[store.MediaMetadata](ta.Ctx, ta.Q(ta.Ctx), metadata); err != nil {
			t.Errorf("expected metadata to still exist, got %v", err)
		}
		if _, err := os.Stat(updated.Metadata.MetadataFilepath); err != nil {
			t.Errorf("expected metadata_filepath to still exist: %v", err)
		}

		// cleanup
		ta.MediaDeleteMediaItem(ta.Ctx, updated, true)
	})

	t.Run("can take additional attributes update media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		stubUserScript(ta)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		updated, err := ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, store.MediaItemParams{PreventDownload: store.Ptr(true)})
		must(t, err)
		if !updated.PreventDownload {
			t.Errorf("expected prevent_download to be true")
		}
	})

	t.Run("calls the user script runner", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

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

		mustOK(t)(ta.MediaDeleteMediaFiles(ta.Ctx, mediaItem, store.MediaItemParams{}))
	})
}

func TestMedia_UpdateMediaItem_TitleRestriction(t *testing.T) {
	t.Run("rejects a title that starts with 'youtube video #'", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})

		_, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, store.MediaItemParams{Title: store.Ptr("youtube video #123")})
		errs, ok := store.AsValidationErrors(err)
		if !ok {
			t.Fatalf("expected validation errors, got %v", err)
		}
		if got, want := errs, map[string][]string{"title": {"has invalid format"}}; !reflect.DeepEqual(got, want) {
			t.Errorf("errors = %v, want %v", got, want)
		}

		if _, err := ta.UpdateMediaItem(ta.Ctx, mediaItem, store.MediaItemParams{Title: store.Ptr("any other title")}); err != nil {
			t.Errorf("expected a valid update, got %v", err)
		}
	})
}

// testUploadDateIndex checks upload_date_index numbering for a collection
// type: channels count down from 99, playlists count up from 0.
func testUploadDateIndex(t *testing.T, collType store.SourceCollectionType, first, second int) {
	newSource := func(t *testing.T, ta *apptest.TestApp) *store.Source {
		return apptest.SourceFixture(t, ta, store.SourceParams{CollectionType: store.Ptr(collType)})
	}
	item := func(t *testing.T, ta *apptest.TestApp, source *store.Source, uploadedAt time.Time) *store.MediaItem {
		return apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), UploadedAt: store.Ptr(uploadedAt)})
	}
	wantIndex := func(t *testing.T, name string, mi *store.MediaItem, want int) {
		t.Helper()
		if mi.UploadDateIndex != want {
			t.Errorf("%s.upload_date_index = %v, want %v", name, mi.UploadDateIndex, want)
		}
	}
	update := func(t *testing.T, ta *apptest.TestApp, mi *store.MediaItem, p store.MediaItemParams) *store.MediaItem {
		updated, err := ta.UpdateMediaItem(ta.Ctx, mi, p)
		must(t, err)
		return updated
	}
	yesterday := func() time.Time { return apptest.NowMinus(1, "day") }

	t.Run("upload_date_index is the first index if it's the only video uploaded that day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		wantIndex(t, "item", item(t, ta, newSource(t, ta), apptest.Now()), first)
	})

	t.Run("upload_date_index is the second index if it's the second video uploaded that day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := newSource(t, ta)
		one := item(t, ta, source, apptest.Now())
		two := item(t, ta, source, apptest.Now())
		wantIndex(t, "one", one, first)
		wantIndex(t, "two", two, second)
	})

	t.Run("upload_date_index doesn't move if the video is uploaded on a different day", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := newSource(t, ta)
		wantIndex(t, "new", item(t, ta, source, apptest.Now()), first)
		wantIndex(t, "old", item(t, ta, source, yesterday()), first)
	})

	t.Run("recomputes upload_date_index if an upload_date is changed...somehow", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := newSource(t, ta)
		newItem := item(t, ta, source, apptest.Now())
		oldItem := item(t, ta, source, yesterday())

		updated := update(t, ta, oldItem, store.MediaItemParams{UploadedAt: store.Ptr(apptest.Now())})
		wantIndex(t, "new", newItem, first)
		wantIndex(t, "updated", updated, second)
	})

	t.Run("upload_date_index doesn't move if the video is for a different source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		wantIndex(t, "one", item(t, ta, newSource(t, ta), apptest.Now()), first)
		wantIndex(t, "two", item(t, ta, newSource(t, ta), apptest.Now()), first)
	})

	for _, c := range []struct {
		name string
		at   func() time.Time
	}{
		{"is updated but doesn't change", apptest.Now},
		{"is changed to the same day", func() time.Time { return apptest.NowPlus(1, "minute") }},
	} {
		t.Run("upload_date_index doesn't move if the a video's upload_date "+c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := newSource(t, ta)
			one := item(t, ta, source, apptest.Now())
			item(t, ta, source, apptest.Now())

			updated := update(t, ta, one, store.MediaItemParams{UploadedAt: store.Ptr(c.at()), Title: store.Ptr("New title")})
			wantIndex(t, "updated", updated, first)
		})
	}
}

func TestMedia_UpdateMediaItem_UploadDateIndexChannel(t *testing.T) {
	testUploadDateIndex(t, store.SourceCollectionTypeChannel, 99, 98)
}

func TestMedia_UpdateMediaItem_UploadDateIndexPlaylist(t *testing.T) {
	testUploadDateIndex(t, store.SourceCollectionTypePlaylist, 0, 1)
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
	must(t, err)
	mustOK(t)(a.PreloadMediaItemSourceAndProfile(ctx, mediaItem))

	gotBytes, err := json.Marshal(mediaItem)
	must(t, err)

	wantBytes, err := os.ReadFile("../../testdata/elixir/golden/user_script_media_item.json")
	must(t, err)

	var got, want map[string]any
	must(t, json.Unmarshal(gotBytes, &got))
	must(t, json.Unmarshal(wantBytes, &want))

	if !reflect.DeepEqual(got, want) {
		t.Errorf("store.MediaItem JSON does not match golden.\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestMedia_ComputeAndSaveMediaFilesize(t *testing.T) {
	t.Run("updates the media item with the file size", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		if mediaItem.MediaSizeBytes != nil && *mediaItem.MediaSizeBytes != 0 {
			t.Error("media_size_bytes should initially be nil or 0")
		}

		result, err := ta.App.ComputeAndSaveMediaFilesize(ta.Ctx, mediaItem)
		must(t, err)

		if result.MediaSizeBytes == nil || *result.MediaSizeBytes == 0 {
			t.Error("media_size_bytes should be set and non-zero")
		}
	})

	t.Run("returns the error if operation fails", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{
			MediaFilepath: store.Ptr("/nonexistent/file.mkv"),
		})

		_, err := ta.App.ComputeAndSaveMediaFilesize(ta.Ctx, mediaItem)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func mediaAttrsMap(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	must(t, store.DecodeJSON([]byte(apptest.MediaAttributesReturnFixture()), &m))
	return m
}
