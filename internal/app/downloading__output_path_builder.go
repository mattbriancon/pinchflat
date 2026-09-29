package app

// OutputPathBuilder builds yt-dlp-friendly output paths for downloaded media.

// OutputPathBuilderBuild/2
func OutputPathBuilderBuild(templateString string, additionalTemplateOptions map[string]string) (string, error) {
	// Merge custom_yt_dlp_option_map with additional options
	combinedOptions := make(map[string]string)
	for k, v := range outputPathBuilderCustomYtDlpOptionMap() {
		combinedOptions[k] = v
	}
	for k, v := range additionalTemplateOptions {
		combinedOptions[k] = v
	}

	// Recursively expand variables (one level deep)
	expandedOptions := make(map[string]string)
	for key, value := range combinedOptions {
		expanded, err := OutputPathParserParse(value, combinedOptions, outputPathBuilderIdentifierFn)
		if err != nil {
			return "", err
		}
		expandedOptions[key] = expanded
	}

	return OutputPathParserParse(templateString, expandedOptions, outputPathBuilderIdentifierFn)
}

// outputPathBuilderIdentifierFn is the custom fetcher for template variables.
// If a variable is found in the map, use its value. Otherwise, wrap the identifier
// in yt-dlp style syntax: %(identifier)S
func outputPathBuilderIdentifierFn(identifier string, variables map[string]string) string {
	if value, ok := variables[identifier]; ok {
		return value
	}
	return "%(" + identifier + ")S"
}

// outputPathBuilderCustomYtDlpOptionMap returns the predefined custom yt-dlp options
func outputPathBuilderCustomYtDlpOptionMap() map[string]string {
	return map[string]string{
		// Individual parts of the upload date
		"upload_year":                               "%(upload_date>%Y)S",
		"upload_month":                              "%(upload_date>%m)S",
		"upload_day":                                "%(upload_date>%d)S",
		"upload_yyyy_mm_dd":                         "%(upload_date>%Y-%m-%d)S",
		"season_from_date":                          "%(upload_date>%Y)S",
		"season_episode_from_date":                  "s%(upload_date>%Y)Se%(upload_date>%m%d)S",
		"season_episode_index_from_date":            "s%(upload_date>%Y)Se%(upload_date>%m%d)S{{ media_upload_date_index }}",
		"artist_name":                               "%(artist,creator,uploader,uploader_id)S",
		"static_season__episode_by_index":           "Season 1/s01e{{ media_playlist_index }}",
		"static_season__episode_by_date":            "Season 1/s01e%(upload_date>%y%m%d)S",
		"season_by_year__episode_by_date":           "Season %(upload_date>%Y)S/s%(upload_date>%Y)Se%(upload_date>%m%d)S",
		"season_by_year__episode_by_date_and_index": "Season %(upload_date>%Y)S/s%(upload_date>%Y)Se%(upload_date>%m%d)S{{ media_upload_date_index }}",
	}
}
