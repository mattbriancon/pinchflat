package ytdlp_test

import (
	"reflect"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

func TestArgsStrings(t *testing.T) {
	args := ytdlp.Args{}.
		Flag("no_warnings").
		Opt("output", "a b").
		Opt("playlist_items", 0).
		Opt("--already-dashed", "x").
		Flag("no_warnings")

	want := []string{"--no-warnings", "--output", "a b", "--playlist-items", "0", "--already-dashed", "x", "--no-warnings"}
	if got := args.Strings(); !reflect.DeepEqual(got, want) {
		t.Errorf("Strings() = %q, want %q", got, want)
	}
}
