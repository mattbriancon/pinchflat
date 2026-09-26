package web_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/web"
)

func TestGetSortDirection(t *testing.T) {
	t.Run("returns the correct sort direction when the new sort attribute is the same as the old sort attribute", func(t *testing.T) {
		result := web.GetSortDirection("name", "name", "desc")
		if result != "asc" {
			t.Errorf("expected asc, got %s", result)
		}
	})

	t.Run("returns the correct sort direction when the new sort attribute is the same as the old sort attribute in the other direction", func(t *testing.T) {
		result := web.GetSortDirection("name", "name", "asc")
		if result != "desc" {
			t.Errorf("expected desc, got %s", result)
		}
	})

	t.Run("returns the correct sort direction when the new sort attribute is different from the old sort attribute", func(t *testing.T) {
		result := web.GetSortDirection("name", "date", "asc")
		if result != "asc" {
			t.Errorf("expected asc, got %s", result)
		}
	})
}
