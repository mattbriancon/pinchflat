package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

func TestNumberUtils_Clamp(t *testing.T) {
	t.Run("returns the minimum when the number is less than the minimum", func(t *testing.T) {
		result := core.NumberUtilsClamp(1, 2, 3)
		if result != 2 {
			t.Errorf("got %d, want 2", result)
		}
	})

	t.Run("returns the maximum when the number is greater than the maximum", func(t *testing.T) {
		result := core.NumberUtilsClamp(4, 2, 3)
		if result != 3 {
			t.Errorf("got %d, want 3", result)
		}
	})

	t.Run("returns the number when it is between the minimum and maximum", func(t *testing.T) {
		result := core.NumberUtilsClamp(2, 1, 3)
		if result != 2 {
			t.Errorf("got %d, want 2", result)
		}
	})
}

func TestNumberUtils_HumanByteSize(t *testing.T) {
	t.Run("converts byte size to human readable format", func(t *testing.T) {
		cases := []struct {
			input    any
			expected [2]interface{}
		}{
			{float64(1024), [2]interface{}{float64(1), "KB"}},
			{float64(1024 * 1024), [2]interface{}{float64(1), "MB"}},
			{float64(1024 * 1024 * 1024), [2]interface{}{float64(1), "GB"}},
			{float64(1024) * 1024 * 1024 * 1024, [2]interface{}{float64(1), "TB"}},
			{float64(1024) * 1024 * 1024 * 1024 * 1024, [2]interface{}{float64(1), "PB"}},
			{float64(1024) * 1024 * 1024 * 1024 * 1024 * 1024, [2]interface{}{float64(1), "EB"}},
			{float64(1024) * 1024 * 1024 * 1024 * 1024 * 1024 * 1024, [2]interface{}{float64(1), "ZB"}},
			{float64(1024) * 1024 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024, [2]interface{}{float64(1), "YB"}},
		}

		for _, tc := range cases {
			val, suffix := core.NumberUtilsHumanByteSize(tc.input, core.KW{})
			if val != tc.expected[0] || suffix != tc.expected[1] {
				t.Errorf("input %v: got (%v, %s), want (%v, %s)", tc.input, val, suffix, tc.expected[0], tc.expected[1])
			}
		}
	})

	t.Run("returns the number when it is less than 1024", func(t *testing.T) {
		val, suffix := core.NumberUtilsHumanByteSize(512, core.KW{})
		if val != 512.0 || suffix != "B" {
			t.Errorf("got (%v, %s), want (512, B)", val, suffix)
		}
	})

	t.Run("optionally takes a precision", func(t *testing.T) {
		cases := []struct {
			input     any
			precision int
			expected  [2]interface{}
		}{
			{float64(1234 * 1024), 0, [2]interface{}{float64(1), "MB"}},
			{float64(1234 * 1024), 1, [2]interface{}{1.2, "MB"}},
			{float64(1234 * 1024), 2, [2]interface{}{1.21, "MB"}},
		}

		for _, tc := range cases {
			val, suffix := core.NumberUtilsHumanByteSize(tc.input, core.KW{core.Opt("precision", tc.precision)})
			if val != tc.expected[0] || suffix != tc.expected[1] {
				t.Errorf("input %v, precision %d: got (%v, %s), want (%v, %s)", tc.input, tc.precision, val, suffix, tc.expected[0], tc.expected[1])
			}
		}
	})

	t.Run("handles 0's well", func(t *testing.T) {
		val, suffix := core.NumberUtilsHumanByteSize(0, core.KW{})
		if val != 0.0 || suffix != "B" {
			t.Errorf("got (%v, %s), want (0, B)", val, suffix)
		}
	})

	t.Run("handles nil well", func(t *testing.T) {
		val, suffix := core.NumberUtilsHumanByteSize(nil, core.KW{})
		if val != 0.0 || suffix != "B" {
			t.Errorf("got (%v, %s), want (0, B)", val, suffix)
		}
	})
}

func TestNumberUtils_AddJitter(t *testing.T) {
	t.Run("returns 0 when the number is less than or equal to 0", func(t *testing.T) {
		if core.NumberUtilsAddJitter(0, 0.5) != 0 {
			t.Error("expected 0 for input 0")
		}
		if core.NumberUtilsAddJitter(-1, 0.5) != 0 {
			t.Error("expected 0 for input -1")
		}
	})

	t.Run("returns the number with jitter added", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			result := core.NumberUtilsAddJitter(100, 0.5)
			if result < 100 || result > 150 {
				t.Errorf("got %d, want between 100 and 150", result)
			}
		}
	})

	t.Run("optionally takes a jitter percentage", func(t *testing.T) {
		// Test 0.1 jitter (range 90-110)
		for i := 0; i < 10; i++ {
			result := core.NumberUtilsAddJitter(100, 0.1)
			if result < 100 || result > 110 {
				t.Errorf("jitter 0.1: got %d, want between 100 and 110", result)
			}
		}

		// Test 0.5 jitter (range 50-150)
		for i := 0; i < 10; i++ {
			result := core.NumberUtilsAddJitter(100, 0.5)
			if result < 100 || result > 150 {
				t.Errorf("jitter 0.5: got %d, want between 100 and 150", result)
			}
		}

		// Test 1 jitter (range 0-200)
		for i := 0; i < 10; i++ {
			result := core.NumberUtilsAddJitter(100, 1.0)
			if result < 100 || result > 200 {
				t.Errorf("jitter 1.0: got %d, want between 100 and 200", result)
			}
		}
	})
}
