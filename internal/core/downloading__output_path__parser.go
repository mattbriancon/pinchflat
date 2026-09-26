package core

// OutputPathParser parses liquid-ish-style strings into rendered strings.

// OutputPathParserFetcher is a function type for custom value fetching in parse templates.
type OutputPathParserFetcher func(identifier string, variables map[string]string) string

// OutputPathParserParse/3
func OutputPathParserParse(s string, variables map[string]string, valueFetchFn OutputPathParserFetcher) (string, error) {
	panic("unported: Pinchflat.Downloading.OutputPath.Parser.parse/3")
}

// OutputPathParserDefaultFetcher/2
func OutputPathParserDefaultFetcher(identifier string, variables map[string]string) string {
	panic("unported: Pinchflat.Downloading.OutputPath.Parser.default_fetcher/2")
}
