package core

type MediaItemsSearchIndex struct {
	// ID maps to rowid in the FTS5 table
	ID          int64   `db:"id"`
	Title       *string `db:"title"`
	Description *string `db:"description"`

	// rank is a virtual field populated by FTS5 queries
	Rank *float64 `db:"rank,virtual"`
}

func (MediaItemsSearchIndex) TableName() string { return "media_items_search_index" }

func NewMediaItemsSearchIndex() *MediaItemsSearchIndex {
	return &MediaItemsSearchIndex{}
}
