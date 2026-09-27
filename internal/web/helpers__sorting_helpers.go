package web

// Port of lib/pinchflat_web/helpers/sorting_helpers.ex.
// Methods for working with sorting, usually in the context of LiveViews or LiveComponents.

// GetSortDirection returns the sort direction given the old sort attribute,
// new sort attribute, and old sort direction.
// If the new attribute is the same as the old one, it toggles the direction (desc -> asc, asc -> desc).
// If the new attribute is different, it returns asc.
// Returns "asc" or "desc".
func GetSortDirection(oldSortAttr, newSortAttr, oldSortDirection string) string {
	if newSortAttr == oldSortAttr {
		if oldSortDirection == "desc" {
			return "asc"
		}
		return "desc"
	}
	return "asc"
}
