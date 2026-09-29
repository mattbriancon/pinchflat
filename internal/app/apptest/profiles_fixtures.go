package apptest

import (
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// MediaProfileFixture creates a MediaProfile with sensible defaults for the
// name and output path template, then applies p over them.
func MediaProfileFixture(t testing.TB, ta *TestApp, p store.MediaProfileParams) *store.MediaProfile {
	t.Helper()
	if p.Name == nil {
		p.Name = store.Ptr("Media Profile #" + strconv.Itoa(randInt(1000000)+1))
	}
	if p.OutputPathTemplate == nil {
		p.OutputPathTemplate = store.Ptr("{{title}}.{{ext}}")
	}

	mediaProfile, errs, err := ta.CreateMediaProfile(ta.Ctx, p)
	if err != nil || len(errs) > 0 {
		t.Fatalf("MediaProfileFixture: %v %v", errs, err)
	}
	return mediaProfile
}
