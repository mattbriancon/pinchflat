package app

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestSourcesTaskActions(t *testing.T) {
	// change builds the diff of an update that set the given fields to the
	// values in after, starting from before.
	change := func(before, after store.Source, fields ...string) store.SourceChanges {
		changed := map[string]bool{}
		for _, f := range fields {
			changed[f] = true
		}
		return store.SourceChanges{Changed: changed, Before: &before, After: &after}
	}

	tests := []struct {
		name              string
		c                 store.SourceChanges
		media, slow, fast taskAction
	}{
		{"nothing changed", change(store.Source{Enabled: true}, store.Source{Enabled: true}), taskNone, taskNone, taskNone},
		{"download_media on, enabled", change(store.Source{Enabled: true}, store.Source{Enabled: true, DownloadMedia: true}, "download_media"), taskEnqueue, taskNone, taskNone},
		{"download_media on, disabled", change(store.Source{}, store.Source{DownloadMedia: true}, "download_media"), taskNone, taskNone, taskNone},
		{"download_media off", change(store.Source{Enabled: true, DownloadMedia: true}, store.Source{Enabled: true}, "download_media"), taskDequeue, taskNone, taskNone},
		{"enabled on, download_media", change(store.Source{DownloadMedia: true}, store.Source{Enabled: true, DownloadMedia: true}, "enabled"), taskEnqueue, taskNone, taskNone},
		{"enabled on, no download_media", change(store.Source{}, store.Source{Enabled: true}, "enabled"), taskNone, taskNone, taskNone},
		{"enabled off", change(store.Source{Enabled: true, DownloadMedia: true, FastIndex: true, IndexFrequencyMinutes: 5}, store.Source{DownloadMedia: true, FastIndex: true, IndexFrequencyMinutes: 5}, "enabled"), taskDequeue, taskDequeue, taskDequeue},
		{"frequency to >0, enabled", change(store.Source{Enabled: true}, store.Source{Enabled: true, IndexFrequencyMinutes: 60}, "index_frequency_minutes"), taskNone, taskEnqueue, taskNone},
		{"frequency to >0, disabled", change(store.Source{}, store.Source{IndexFrequencyMinutes: 60}, "index_frequency_minutes"), taskNone, taskDequeue, taskNone},
		{"frequency to 0", change(store.Source{Enabled: true, IndexFrequencyMinutes: 60}, store.Source{Enabled: true}, "index_frequency_minutes"), taskNone, taskDequeue, taskNone},
		{"enabled on, frequency >0", change(store.Source{IndexFrequencyMinutes: 60}, store.Source{Enabled: true, IndexFrequencyMinutes: 60}, "enabled"), taskNone, taskEnqueue, taskNone},
		{"enabled on, frequency 0", change(store.Source{}, store.Source{Enabled: true}, "enabled"), taskNone, taskNone, taskNone},
		{"fast_index on, enabled", change(store.Source{Enabled: true}, store.Source{Enabled: true, FastIndex: true}, "fast_index"), taskNone, taskNone, taskEnqueue},
		{"fast_index on, disabled", change(store.Source{}, store.Source{FastIndex: true}, "fast_index"), taskNone, taskNone, taskNone},
		{"fast_index off", change(store.Source{Enabled: true, FastIndex: true}, store.Source{Enabled: true}, "fast_index"), taskNone, taskNone, taskDequeue},
		{"enabled on, fast_index", change(store.Source{FastIndex: true}, store.Source{Enabled: true, FastIndex: true}, "enabled"), taskNone, taskNone, taskEnqueue},
		{"unrelated field", change(store.Source{Enabled: true, DownloadMedia: true}, store.Source{Enabled: true, DownloadMedia: true, CustomName: "x"}, "custom_name"), taskNone, taskNone, taskNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sourcesMediaAction(tc.c); got != tc.media {
				t.Errorf("media: got %v, want %v", got, tc.media)
			}
			if got := sourcesSlowIndexingAction(tc.c); got != tc.slow {
				t.Errorf("slow indexing: got %v, want %v", got, tc.slow)
			}
			if got := sourcesFastIndexingAction(tc.c); got != tc.fast {
				t.Errorf("fast indexing: got %v, want %v", got, tc.fast)
			}
		})
	}
}
