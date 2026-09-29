package web

import "testing"

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
			if got := clamp(tc.num, tc.min, tc.max); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestHumanByteSize(t *testing.T) {
	cases := []struct {
		name      string
		input     int64
		precision int
		wantVal   float64
		wantUnit  string
	}{
		{"kilobytes", 1024, 2, 1, "KB"},
		{"megabytes", 1024 * 1024, 2, 1, "MB"},
		{"gigabytes", 1024 * 1024 * 1024, 2, 1, "GB"},
		{"terabytes", 1 << 40, 2, 1, "TB"},
		{"petabytes", 1 << 50, 2, 1, "PB"},
		{"exabytes", 1 << 60, 2, 1, "EB"},
		{"less than 1024 stays in bytes", 512, 2, 512, "B"},
		{"zero", 0, 2, 0, "B"},
		{"precision 0", 1234 * 1024, 0, 1, "MB"},
		{"precision 1", 1234 * 1024, 1, 1.2, "MB"},
		{"precision 2", 1234 * 1024, 2, 1.21, "MB"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, unit := humanByteSize(tc.input, tc.precision)
			if val != tc.wantVal || unit != tc.wantUnit {
				t.Errorf("got (%v, %s), want (%v, %s)", val, unit, tc.wantVal, tc.wantUnit)
			}
		})
	}
}
