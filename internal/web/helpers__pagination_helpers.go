package web

// Methods for working with pagination, usually in the context of LiveViews or LiveComponents.

import (
	"context"
	"math"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// PaginationAttributes holds the pagination state returned by GetPaginationAttributes.
type PaginationAttributes struct {
	Page             int `json:"page"`
	TotalPages       int `json:"total_pages"`
	TotalRecordCount int `json:"total_record_count"`
	Limit            int `json:"limit"`
	Offset           int `json:"offset"`
}

// GetPaginationAttributes returns pagination attributes given a query, page number,
// and number of records per page.
func (s *Server) GetPaginationAttributes(ctx context.Context, query sq.SelectBuilder, page, recordsPerPage int) (*PaginationAttributes, error) {
	// Count using a subquery, similar to Repo.aggregate in Elixir
	countQuery := store.SQ.Select("COUNT(*)").FromSelect(query, "sq")
	totalRecordCount, err := store.Scalar[int](ctx, s.App.Q(ctx), countQuery)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Max(math.Ceil(float64(totalRecordCount)/float64(recordsPerPage)), 1))
	clampedPage := clamp(page, 1, totalPages)

	return &PaginationAttributes{
		Page:             clampedPage,
		TotalPages:       totalPages,
		TotalRecordCount: totalRecordCount,
		Limit:            recordsPerPage,
		Offset:           (clampedPage - 1) * recordsPerPage,
	}, nil
}

// clamp clamps num between minimum and maximum.
func clamp(num, minimum, maximum int) int {
	return min(max(num, minimum), maximum)
}
