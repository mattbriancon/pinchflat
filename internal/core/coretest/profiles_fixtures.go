package coretest

import (
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// MediaProfileFixture creates a MediaProfile with sensible defaults, merging in attrs.
func MediaProfileFixture(t testing.TB, ta *TestApp, attrs core.Attrs) *core.MediaProfile {
	t.Helper()
	defaults := core.Attrs{
		"name":                 "Media Profile #" + strconv.Itoa(randInt(1000000)+1),
		"output_path_template": "{{title}}.{{ext}}",
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	mediaProfile, err := ta.ProfilesCreateMediaProfile(ta.Ctx, defaults)
	if err != nil {
		t.Fatalf("MediaProfileFixture: %v", err)
	}
	return mediaProfile
}
