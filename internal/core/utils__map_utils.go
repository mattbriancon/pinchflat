package core

// MapUtilsFromNestedList(list)
// Converts a nested list of 2-element tuples or lists into a map.
func MapUtilsFromNestedList(list interface{}) map[any]any {
	result := make(map[any]any)
	if list == nil {
		return result
	}

	// Handle slice of interfaces
	slice, ok := list.([]interface{})
	if !ok {
		return result
	}

	for _, item := range slice {
		var key, value any
		found := false

		// Try to extract key and value from the item
		if pair, ok := item.([]interface{}); ok && len(pair) == 2 {
			key = pair[0]
			value = pair[1]
			found = true
		}

		if found {
			result[key] = value
		}
	}

	return result
}
