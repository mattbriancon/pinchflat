package core

// FunctionUtilsWrapOk(value)
// Wraps the provided value in an ok tuple (or just returns it as the success value).
// Returns [2]any{true, value} to represent {:ok, value} in Elixir.
func FunctionUtilsWrapOk(value any) any {
	return [2]any{true, value}
}
