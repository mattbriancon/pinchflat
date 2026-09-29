package web

import (
	"context"
	"math"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/store"
)

const historyTableLimit = 5

// historyTableFetch fetches one media_state's page of history for the home
// page (mount/3 + handle_params in the old LiveView).
func historyTableFetch(ctx context.Context, app *app.App, mediaState string, page int) (records []*store.MediaItem, clampedPage, totalPages, totalRecordCount int, err error) {
	baseQuery := generateHistoryTableBaseQuery(mediaState)
	totalRecordCount, err = store.Scalar[int](ctx, app.Q(ctx),
		store.SQ.Select("COUNT(*)").FromSelect(baseQuery.B, "mi"))
	if err != nil {
		return nil, 0, 0, 0, err
	}
	totalPages = int(math.Max(math.Ceil(float64(totalRecordCount)/float64(historyTableLimit)), 1))
	clampedPage = clamp(page, 1, totalPages)

	offset := (clampedPage - 1) * historyTableLimit
	records, err = store.All[store.MediaItem](ctx, app.Q(ctx),
		baseQuery.B.Limit(uint64(historyTableLimit)).Offset(uint64(offset)))
	if err != nil {
		return nil, 0, 0, 0, err
	}

	// Preload source for each record
	for _, record := range records {
		source, _ := app.GetSource(ctx, record.SourceID)
		record.Source = source
	}

	return records, clampedPage, totalPages, totalRecordCount, nil
}

// generateHistoryTableBaseQuery creates the base query for the history table.
func generateHistoryTableBaseQuery(mediaState string) *store.MediaQ {
	q := store.MediaQueryNew().RequireAssoc("media_profile")
	if mediaState == "pending" {
		q = q.Where(store.MediaQueryPending())
	} else {
		q = q.Where(store.MediaQueryDownloaded())
	}
	return q.Map(func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.OrderBy("mi.id DESC")
	})
}
