package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// metaRun describes a run of the source metadata storage worker.
type metaRun struct {
	profile  store.MediaProfileParams
	src      store.SourceParams
	dir      string                              // directory below the media dir of the details' filename; "Season 1" if empty
	metadata string                              // get_source_metadata response; "{}" if empty
	initial  func(t *testing.T, s *store.Source) // checks the source before the worker runs
	check    func(action string, opts ytdlp.Args, addl ytdlp.CallOptions)
}

// perform creates the source, stubs yt-dlp, runs the worker and returns the
// reloaded source with its metadata preloaded.
func (r metaRun) perform(t *testing.T) (*apptest.TestApp, *store.Source) {
	t.Helper()
	ta := apptest.NewApp(t)
	r.src.MediaProfileID = store.Ptr(apptest.MediaProfileFixture(t, ta, r.profile).ID)
	source := apptest.SourceFixture(t, ta, r.src)
	if r.initial != nil {
		r.initial(t, source)
	}
	dir, metadata := r.dir, r.metadata
	if dir == "" {
		dir = "Season 1"
	}
	if metadata == "" {
		metadata = "{}"
	}

	ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
		if r.check != nil {
			r.check(action, opts, addl)
		}
		switch action {
		case "get_source_details":
			return apptest.SourceDetailsReturnFixture(map[string]any{"filename": filepath.Join(ta.Config.MediaDirectory, dir, "bar.mp4")}), nil
		case "get_source_metadata":
			return metadata, nil
		}
		return "", nil
	})

	_ = ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

	reloaded, err := ta.GetSource(ta.Ctx, source.ID)
	must(t, err)
	reloaded, err = ta.PreloadSourceMetadata(ta.Ctx, reloaded)
	must(t, err)
	return ta, reloaded
}

// fixtureMetadata renders a metadata fixture, with its paths pointed at a
// writable directory.
func fixtureMetadata(t *testing.T, name string) string {
	t.Helper()
	s, err := apptest.RenderMetadataWithFixedPaths(name)
	must(t, err)
	return s
}

func wantSet(t *testing.T, name string, p *string) {
	t.Helper()
	if p == nil || *p == "" {
		t.Errorf("Expected %s to be set", name)
	}
}

func wantFileExists(t *testing.T, name string, p *string) {
	t.Helper()
	if p == nil {
		t.Errorf("Expected %s to be set", name)
	} else if _, err := os.Stat(*p); err != nil {
		t.Errorf("Expected %s file to exist at %s", name, *p)
	}
}

func wantNilPath(t *testing.T, name string, p *string) {
	t.Helper()
	if p != nil {
		t.Errorf("Expected %s to be nil, got %v", name, *p)
	}
}

func TestSourceMetadataStorageWorker_KickoffWithTask(t *testing.T) {
	t.Parallel()

	t.Run("enqueues a new worker for the source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		_, err := ta.SourceMetadataStorageWorkerKickoffWithTask(ta.Ctx, source)
		must(t, err)

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName, Args: map[string]any{"id": source.ID}})
		if len(enqueued) != 1 {
			t.Errorf("Expected 1 job enqueued, got %d", len(enqueued))
		}
	})

	t.Run("creates a new task for the source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		task, err := ta.SourceMetadataStorageWorkerKickoffWithTask(ta.Ctx, source)
		must(t, err)

		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("Expected task.source_id to be %d, got %v", source.ID, task.SourceID)
		}
	})
}

func TestSourceMetadataStorageWorker_Perform(t *testing.T) {
	t.Parallel()

	t.Run("won't call itself in an infinite loop", func(t *testing.T) {
		t.Parallel()
		ta, _ := metaRun{}.perform(t)

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName})
		if len(enqueued) != 0 {
			t.Errorf("Expected 0 jobs enqueued, got %d", len(enqueued))
		}
	})

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		must(t, ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": int64(0)}))
	})
}

// TestSourceMetadataStorageWorker_PerformSourceState runs the worker for a
// setup and checks the resulting source.
func TestSourceMetadataStorageWorker_PerformSourceState(t *testing.T) {
	t.Parallel()

	images := store.MediaProfileParams{DownloadSourceImages: store.Ptr(true)}
	nfo := func(download bool) store.MediaProfileParams {
		return store.MediaProfileParams{DownloadNfo: store.Ptr(download)}
	}
	cleanup := func(ta *apptest.TestApp, s *store.Source) { ta.SourcesDeleteSource(ta.Ctx, s, false) }

	for _, c := range []struct {
		name  string
		run   func(t *testing.T) metaRun
		check func(t *testing.T, ta *apptest.TestApp, s *store.Source)
	}{
		{"the source description is saved",
			func(t *testing.T) metaRun {
				return metaRun{
					src:      store.SourceParams{Description: store.Ptr("")},
					metadata: fixtureMetadata(t, "channel_source_metadata"),
					initial: func(t *testing.T, s *store.Source) {
						if s.Description != nil {
							t.Errorf("Expected source description to be nil, got %v", s.Description)
						}
					},
				}
			},
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				if s.Description == nil || *s.Description != "This is a test file for Pinchflat" {
					t.Errorf("Expected source description to be 'This is a test file for Pinchflat', got %v", s.Description)
				}
			}},
		{"sets metadata location for source",
			func(t *testing.T) metaRun { return metaRun{} },
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				if s.Metadata == nil || s.Metadata.MetadataFilepath == "" {
					t.Fatalf("Expected metadata filepath to be set, got %v", s.Metadata)
				}
				os.Remove(s.Metadata.MetadataFilepath)
			}},
		{"sets metadata image location for source",
			func(t *testing.T) metaRun { return metaRun{metadata: fixtureMetadata(t, "channel_source_metadata")} },
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantSet(t, "fanart_filepath", s.Metadata.FanartFilepath)
				wantSet(t, "poster_filepath", s.Metadata.PosterFilepath)
				wantSet(t, "banner_filepath", s.Metadata.BannerFilepath)
				cleanup(ta, s)
			}},
		{"stores metadata images for source",
			func(t *testing.T) metaRun { return metaRun{metadata: fixtureMetadata(t, "channel_source_metadata")} },
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantFileExists(t, "fanart", s.Metadata.FanartFilepath)
				wantFileExists(t, "poster", s.Metadata.PosterFilepath)
				wantFileExists(t, "banner", s.Metadata.BannerFilepath)
				cleanup(ta, s)
			}},
		{"downloads and stores source images",
			func(t *testing.T) metaRun {
				return metaRun{profile: images, metadata: fixtureMetadata(t, "channel_source_metadata")}
			},
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantSet(t, "fanart_filepath", s.FanartFilepath)
				wantSet(t, "poster_filepath", s.PosterFilepath)
				wantSet(t, "banner_filepath", s.BannerFilepath)
				wantFileExists(t, "fanart", s.FanartFilepath)
				wantFileExists(t, "poster", s.PosterFilepath)
				wantFileExists(t, "banner", s.BannerFilepath)
				cleanup(ta, s)
			}},
		{"does not store source images if the profile is not set to",
			func(t *testing.T) metaRun {
				return metaRun{profile: store.MediaProfileParams{DownloadSourceImages: store.Ptr(false)}, metadata: fixtureMetadata(t, "channel_source_metadata")}
			},
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantNilPath(t, "fanart_filepath", s.FanartFilepath)
				wantNilPath(t, "poster_filepath", s.PosterFilepath)
				wantNilPath(t, "banner_filepath", s.BannerFilepath)
			}},
		{"does not store source images if the series directory cannot be determined",
			func(t *testing.T) metaRun {
				return metaRun{profile: images, dir: "foo", metadata: fixtureMetadata(t, "channel_source_metadata")}
			},
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantNilPath(t, "fanart_filepath", s.FanartFilepath)
				wantNilPath(t, "poster_filepath", s.PosterFilepath)
				wantNilPath(t, "banner_filepath", s.BannerFilepath)
			}},
		{"sets the series directory based on the returned media filepath",
			func(t *testing.T) metaRun { return metaRun{src: store.SourceParams{SeriesDirectory: store.Ptr("")}} },
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				if s.SeriesDirectory == nil {
					t.Errorf("Expected series directory to be set")
				}
			}},
		{"does not set the series directory if it cannot be determined",
			func(t *testing.T) metaRun {
				return metaRun{src: store.SourceParams{SeriesDirectory: store.Ptr("")}, dir: "foo"}
			},
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantNilPath(t, "series directory", s.SeriesDirectory)
			}},
		{"stores the NFO if specified",
			func(t *testing.T) metaRun {
				return metaRun{profile: nfo(true), src: store.SourceParams{NfoFilepath: store.Ptr("")}}
			},
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantSet(t, "nfo filepath", s.NfoFilepath)
				if expected := filepath.Join(*s.SeriesDirectory, "tvshow.nfo"); s.NfoFilepath != nil && *s.NfoFilepath != expected {
					t.Errorf("Expected nfo filepath to be %s, got %s", expected, *s.NfoFilepath)
				}
				if s.NfoFilepath != nil {
					os.Remove(*s.NfoFilepath)
				}
			}},
		{"does not store the NFO if not specified",
			func(t *testing.T) metaRun {
				return metaRun{profile: nfo(false), src: store.SourceParams{NfoFilepath: store.Ptr("")}}
			},
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantNilPath(t, "nfo filepath", s.NfoFilepath)
			}},
		{"does not store the NFO if the series directory cannot be determined",
			func(t *testing.T) metaRun {
				return metaRun{profile: nfo(true), src: store.SourceParams{NfoFilepath: store.Ptr("")}, dir: "foo"}
			},
			func(t *testing.T, ta *apptest.TestApp, s *store.Source) {
				wantNilPath(t, "nfo filepath", s.NfoFilepath)
			}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ta, source := c.run(t).perform(t)
			c.check(t, ta, source)
		})
	}

	t.Run("fetches and stores returned metadata for source", func(t *testing.T) {
		t.Parallel()
		ta, source := metaRun{metadata: `{"title": "test"}`}.perform(t)

		metadata, err := ta.MetadataFileHelpersReadCompressedMetadata(ta.Ctx, source.Metadata.MetadataFilepath)
		must(t, err)
		if metadata == nil || metadata["title"] != "test" {
			t.Errorf("Expected metadata title to be 'test', got %v", metadata)
		}
	})
}

func TestSourceMetadataStorageWorker_PerformSourceImageDownloading(t *testing.T) {
	t.Parallel()

	t.Run("calls appropriate yt-dlp opts by collection type", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name              string
			collectionType    store.SourceCollectionType
			expectedPlaylist  any
			expectedThumbnail string
		}{
			{"channel", store.SourceCollectionTypeChannel, 0, "write_all_thumbnails"},
			{"playlist", store.SourceCollectionTypePlaylist, 1, "write_thumbnail"},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				metaRun{
					profile:  store.MediaProfileParams{DownloadSourceImages: store.Ptr(true)},
					src:      store.SourceParams{CollectionType: store.Ptr(tt.collectionType)},
					metadata: fixtureMetadata(t, "channel_source_metadata"),
					check: func(action string, opts ytdlp.Args, _ ytdlp.CallOptions) {
						if action != "get_source_metadata" {
							return
						}
						if !opts.Contains(ytdlp.Arg{Key: "playlist_items", Value: tt.expectedPlaylist}) {
							t.Errorf("Expected playlist_items=%v for %s", tt.expectedPlaylist, tt.name)
						}
						if !opts.Contains(ytdlp.Arg{Key: tt.expectedThumbnail, Flag: true}) {
							t.Errorf("Expected %s for %s", tt.expectedThumbnail, tt.name)
						}
					},
				}.perform(t)
			})
		}
	})
}

func TestSourceMetadataStorageWorker_PerformCookies(t *testing.T) {
	t.Parallel()

	for _, withImages := range []bool{true, false} {
		name := "sets use_cookies based on source behavior"
		if !withImages {
			name += " during series directory determination"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, tt := range []struct {
				name      string
				behaviour store.SourceCookieBehaviour
				want      bool
			}{
				{"all_operations", store.SourceCookieBehaviourAllOperations, true},
				{"when_needed", store.SourceCookieBehaviourWhenNeeded, false},
				{"disabled", store.SourceCookieBehaviourDisabled, false},
			} {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					run := metaRun{src: store.SourceParams{SeriesDirectory: store.Ptr(""), CookieBehaviour: store.Ptr(tt.behaviour)}}
					if withImages {
						run = metaRun{
							profile:  store.MediaProfileParams{DownloadSourceImages: store.Ptr(true)},
							src:      store.SourceParams{CookieBehaviour: store.Ptr(tt.behaviour)},
							metadata: fixtureMetadata(t, "channel_source_metadata"),
						}
					}
					calls := 0
					run.check = func(_ string, _ ytdlp.Args, addl ytdlp.CallOptions) {
						calls++
						if addl.UseCookies != tt.want {
							t.Errorf("Expected use_cookies=%v in addl, got %v", tt.want, addl.UseCookies)
						}
					}

					run.perform(t)

					if calls < 2 {
						t.Errorf("Expected at least 2 yt-dlp calls, got %d", calls)
					}
				})
			}
		})
	}
}
