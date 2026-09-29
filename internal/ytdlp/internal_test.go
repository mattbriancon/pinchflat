package ytdlp

import "testing"

func TestAddJitter(t *testing.T) {
	t.Run("returns 0 when the number is less than or equal to 0", func(t *testing.T) {
		for _, n := range []int{0, -1} {
			if got := addJitter(n, 0.5); got != 0 {
				t.Errorf("addJitter(%d, 0.5) = %d, want 0", n, got)
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
				result := addJitter(100, tc.pct)
				if result < tc.min || result > tc.max {
					t.Errorf("jitter %v: got %d, want between %d and %d", tc.pct, result, tc.min, tc.max)
				}
			}
		}
	})
}

func TestKebabCase(t *testing.T) {
	for in, want := range map[string]string{"hello world": "hello-world", "hello_world": "hello-world"} {
		if got := kebabCase(in); got != want {
			t.Errorf("kebabCase(%q) = %q, want %q", in, got, want)
		}
	}
}
