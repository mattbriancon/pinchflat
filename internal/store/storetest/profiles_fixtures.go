package storetest

import (
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// MediaProfileFixture creates a MediaProfile with sensible defaults, merging in attrs.
func MediaProfileFixture(t testing.TB, ts *TestStore, attrs store.Attrs) *store.MediaProfile {
	t.Helper()
	defaults := store.Attrs{
		"name":                 "Media Profile #" + strconv.Itoa(randInt(1000000)+1),
		"output_path_template": "{{title}}.{{ext}}",
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	mediaProfile, err := ts.CreateMediaProfile(ts.Ctx, defaults)
	if err != nil {
		t.Fatalf("MediaProfileFixture: %v", err)
	}
	return mediaProfile
}
