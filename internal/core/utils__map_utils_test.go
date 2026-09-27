package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

func TestMapUtils_FromNestedList(t *testing.T) {
	t.Run("creates a map from a nested 2-element tuple list", func(t *testing.T) {
		list := []interface{}{
			[]interface{}{"key1", "value1"},
			[]interface{}{"key2", "value2"},
		}

		result := core.MapUtilsFromNestedList(list)
		expected := map[any]any{
			"key1": "value1",
			"key2": "value2",
		}

		if len(result) != len(expected) {
			t.Errorf("length mismatch: got %d, want %d", len(result), len(expected))
		}
		for k, v := range expected {
			if result[k] != v {
				t.Errorf("key %v: got %v, want %v", k, result[k], v)
			}
		}
	})

	t.Run("creates a map from a nested 2-element list of lists", func(t *testing.T) {
		list := []interface{}{
			[]interface{}{"key1", "value1"},
			[]interface{}{"key2", "value2"},
		}

		result := core.MapUtilsFromNestedList(list)
		expected := map[any]any{
			"key1": "value1",
			"key2": "value2",
		}

		if len(result) != len(expected) {
			t.Errorf("length mismatch: got %d, want %d", len(result), len(expected))
		}
		for k, v := range expected {
			if result[k] != v {
				t.Errorf("key %v: got %v, want %v", k, result[k], v)
			}
		}
	})
}
