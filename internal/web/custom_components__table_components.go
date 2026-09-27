package web

// Port of lib/pinchflat_web/components/custom_components/table_components.ex
// (the renderable half is custom_components__table_components.templ).

import "github.com/a-h/templ"

// TableColumn is the :col slot on table/1.
type TableColumn struct {
	Label   string
	Class   string
	SortKey string
	// Render builds this column's cell for a row (render_slot(col, row)).
	Render func(row any) templ.Component
}

// TableProps is table/1's assigns. HeaderURL, when set, builds the link
// target for a sortable column's header (a plain page reload with the new
// sort_key/sort_direction in the query string); nil means no columns sort.
type TableProps struct {
	Rows          []any
	TableClass    string
	SortKey       string
	SortDirection string // "asc" or "desc"
	Columns       []TableColumn
	HeaderURL     func(sortKey string) string
}

func tableHeaderURL(p TableProps, col TableColumn) string {
	if p.HeaderURL == nil || col.SortKey == "" {
		return ""
	}
	return p.HeaderURL(col.SortKey)
}

func tableSortIcon(direction string) string {
	if direction == "asc" {
		return "hero-chevron-up"
	}
	return "hero-chevron-down"
}
