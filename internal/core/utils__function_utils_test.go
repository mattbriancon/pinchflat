package core_test

import (
	"reflect"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

func TestFunctionUtils_WrapOk(t *testing.T) {
	t.Run("wraps the provided term in an :ok tuple", func(t *testing.T) {
		result := core.FunctionUtilsWrapOk("hello")
		expected := [2]any{true, "hello"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})
}
