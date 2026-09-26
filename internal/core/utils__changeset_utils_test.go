package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// MockSchema represents a simple schema for testing changesets.
type MockSchema struct {
	ID    int64  `db:"id"`
	Title string `db:"title"`
}

func (MockSchema) TableName() string { return "mock_schemas" }

func TestChangesetUtils_DynamicDefault(t *testing.T) {
	t.Run("sets the default value if the field is nil", func(t *testing.T) {
		data := &MockSchema{}
		cs := core.Cast(data, core.Attrs{}, []string{"title"})
		cs = core.ChangesetUtilsDynamicDefault(cs, "title", func(_ *core.Changeset) any {
			return "default"
		})

		if change := cs.GetChange("title"); change != "default" {
			t.Errorf("expected 'default', got %v", change)
		}
	})

	t.Run("does not set the default value if the field is not nil", func(t *testing.T) {
		data := &MockSchema{}
		cs := core.Cast(data, core.Attrs{"title": "custom"}, []string{"title"})
		cs = core.ChangesetUtilsDynamicDefault(cs, "title", func(_ *core.Changeset) any {
			return "default"
		})

		if change := cs.GetChange("title"); change != "custom" {
			t.Errorf("expected 'custom', got %v", change)
		}
	})
}
