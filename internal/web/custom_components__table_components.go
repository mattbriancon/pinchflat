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

// TableProps is table/1's assigns. HeaderAttrs, when set, builds the extra
// attributes for a sortable column's <th> (phx-click="sort_update"
// phx-value-sort_key={col[:sort_key]} becomes whatever htmx attributes the
// caller's live table wires up); nil means no attributes.
type TableProps struct {
	Rows          []any
	TableClass    string
	SortKey       string
	SortDirection string // "asc" or "desc"
	Columns       []TableColumn
	HeaderAttrs   func(sortKey string) templ.Attributes
}

func tableHeaderAttrs(p TableProps, col TableColumn) templ.Attributes {
	if p.HeaderAttrs == nil || col.SortKey == "" {
		return nil
	}
	return p.HeaderAttrs(col.SortKey)
}

func tableSortIcon(direction string) string {
	if direction == "asc" {
		return "hero-chevron-up"
	}
	return "hero-chevron-down"
}

func tableConditionalAttrs(cond bool, attrs templ.Attributes) templ.Attributes {
	if !cond {
		return nil
	}
	return attrs
}
