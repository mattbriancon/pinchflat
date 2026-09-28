package web_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestGetPaginationAttributes(t *testing.T) {
	t.Run("returns the correct pagination attributes", func(t *testing.T) {
		c := webtest.New(t)
		coretest.SourceFixture(t, c.TestApp, store.Attrs{})

		query := store.From[store.Source]("s")
		attrs, err := c.Server.GetPaginationAttributes(c.Ctx, query, 1, 10)
		if err != nil {
			t.Fatalf("GetPaginationAttributes failed: %v", err)
		}

		if attrs.Page != 1 {
			t.Errorf("expected page 1, got %d", attrs.Page)
		}
		if attrs.TotalPages != 1 {
			t.Errorf("expected total_pages 1, got %d", attrs.TotalPages)
		}
		if attrs.TotalRecordCount != 1 {
			t.Errorf("expected total_record_count 1, got %d", attrs.TotalRecordCount)
		}
		if attrs.Limit != 10 {
			t.Errorf("expected limit 10, got %d", attrs.Limit)
		}
		if attrs.Offset != 0 {
			t.Errorf("expected offset 0, got %d", attrs.Offset)
		}
	})

	t.Run("returns the correct pagination attributes when there are multiple pages", func(t *testing.T) {
		c := webtest.New(t)
		coretest.SourceFixture(t, c.TestApp, store.Attrs{})
		coretest.SourceFixture(t, c.TestApp, store.Attrs{})

		query := store.From[store.Source]("s")
		attrs, err := c.Server.GetPaginationAttributes(c.Ctx, query, 1, 1)
		if err != nil {
			t.Fatalf("GetPaginationAttributes failed: %v", err)
		}

		if attrs.Page != 1 {
			t.Errorf("expected page 1, got %d", attrs.Page)
		}
		if attrs.TotalPages != 2 {
			t.Errorf("expected total_pages 2, got %d", attrs.TotalPages)
		}
		if attrs.TotalRecordCount != 2 {
			t.Errorf("expected total_record_count 2, got %d", attrs.TotalRecordCount)
		}
		if attrs.Limit != 1 {
			t.Errorf("expected limit 1, got %d", attrs.Limit)
		}
		if attrs.Offset != 0 {
			t.Errorf("expected offset 0, got %d", attrs.Offset)
		}
	})

	t.Run("returns the correct attributes when on a page other than the first", func(t *testing.T) {
		c := webtest.New(t)
		coretest.SourceFixture(t, c.TestApp, store.Attrs{})
		coretest.SourceFixture(t, c.TestApp, store.Attrs{})

		query := store.From[store.Source]("s")
		attrs, err := c.Server.GetPaginationAttributes(c.Ctx, query, 2, 1)
		if err != nil {
			t.Fatalf("GetPaginationAttributes failed: %v", err)
		}

		if attrs.Page != 2 {
			t.Errorf("expected page 2, got %d", attrs.Page)
		}
		if attrs.TotalPages != 2 {
			t.Errorf("expected total_pages 2, got %d", attrs.TotalPages)
		}
		if attrs.TotalRecordCount != 2 {
			t.Errorf("expected total_record_count 2, got %d", attrs.TotalRecordCount)
		}
		if attrs.Limit != 1 {
			t.Errorf("expected limit 1, got %d", attrs.Limit)
		}
		if attrs.Offset != 1 {
			t.Errorf("expected offset 1, got %d", attrs.Offset)
		}
	})
}
