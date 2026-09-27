package fsutil_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

func TestClamp(t *testing.T) {
	cases := []struct {
		name          string
		num, min, max int
		want          int
	}{
		{"returns the minimum when the number is less than the minimum", 1, 2, 3, 2},
		{"returns the maximum when the number is greater than the maximum", 4, 2, 3, 3},
		{"returns the number when it is between the minimum and maximum", 2, 1, 3, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fsutil.Clamp(tc.num, tc.min, tc.max); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestHumanByteSize(t *testing.T) {
	t.Run("converts byte size to human readable format", func(t *testing.T) {
		cases := []struct {
			input    any
			wantVal  float64
			wantUnit string
		}{
			{float64(1024), 1, "KB"},
			{float64(1024 * 1024), 1, "MB"},
			{float64(1024 * 1024 * 1024), 1, "GB"},
			{float64(1024) * 1024 * 1024 * 1024, 1, "TB"},
			{float64(1024) * 1024 * 1024 * 1024 * 1024, 1, "PB"},
			{float64(1024) * 1024 * 1024 * 1024 * 1024 * 1024, 1, "EB"},
			{float64(1024) * 1024 * 1024 * 1024 * 1024 * 1024 * 1024, 1, "ZB"},
			{float64(1024) * 1024 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024, 1, "YB"},
		}
		for _, tc := range cases {
			val, unit := fsutil.HumanByteSize(tc.input, 2)
			if val != tc.wantVal || unit != tc.wantUnit {
				t.Errorf("input %v: got (%v, %s), want (%v, %s)", tc.input, val, unit, tc.wantVal, tc.wantUnit)
			}
		}
	})

	t.Run("returns the number when it is less than 1024", func(t *testing.T) {
		val, unit := fsutil.HumanByteSize(512, 2)
		if val != 512.0 || unit != "B" {
			t.Errorf("got (%v, %s), want (512, B)", val, unit)
		}
	})

	t.Run("takes a precision", func(t *testing.T) {
		cases := []struct {
			precision int
			want      float64
		}{
			{0, 1},
			{1, 1.2},
			{2, 1.21},
		}
		for _, tc := range cases {
			val, unit := fsutil.HumanByteSize(float64(1234*1024), tc.precision)
			if val != tc.want || unit != "MB" {
				t.Errorf("precision %d: got (%v, %s), want (%v, MB)", tc.precision, val, unit, tc.want)
			}
		}
	})

	t.Run("handles 0's and nil well", func(t *testing.T) {
		for _, input := range []any{0, nil} {
			val, unit := fsutil.HumanByteSize(input, 2)
			if val != 0.0 || unit != "B" {
				t.Errorf("input %v: got (%v, %s), want (0, B)", input, val, unit)
			}
		}
	})
}

func TestAddJitter(t *testing.T) {
	t.Run("returns 0 when the number is less than or equal to 0", func(t *testing.T) {
		for _, n := range []int{0, -1} {
			if got := fsutil.AddJitter(n, 0.5); got != 0 {
				t.Errorf("AddJitter(%d, 0.5) = %d, want 0", n, got)
			}
		}
	})

	t.Run("returns the number with jitter added", func(t *testing.T) {
		cases := []struct {
			pct      float64
			min, max int
		}{
			{0.1, 100, 110},
			{0.5, 100, 150},
			{1.0, 100, 200},
		}
		for _, tc := range cases {
			for i := 0; i < 10; i++ {
				result := fsutil.AddJitter(100, tc.pct)
				if result < tc.min || result > tc.max {
					t.Errorf("jitter %v: got %d, want between %d and %d", tc.pct, result, tc.min, tc.max)
				}
			}
		}
	})
}
