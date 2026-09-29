package store

import (
	"context"
	"fmt"
	"reflect"
)

// GetSettingsRecord returns the (singleton) settings row.
func (s *Store) GetSettingsRecord(ctx context.Context) (*Setting, error) {
	return One[Setting](ctx, s.Q(ctx), From[Setting]().Limit(1))
}

// UpdateSetting updates setting with typed parameters.
// Returns the updated Setting, validation errors as a map, and any system error.
func (s *Store) UpdateSetting(ctx context.Context, setting *Setting, p SettingParams) (*Setting, map[string][]string, error) {
	errs := p.Validate(setting)
	if len(errs) > 0 {
		return nil, errs, nil
	}

	// Build the Attrs map from non-nil params
	attrs := Attrs{}
	if p.Onboarding != nil {
		attrs["onboarding"] = *p.Onboarding
	}
	if p.YtDlpVersion != nil {
		attrs["yt_dlp_version"] = *p.YtDlpVersion
	}
	if p.VideoCodecPreference != nil {
		attrs["video_codec_preference"] = *p.VideoCodecPreference
	}
	if p.AudioCodecPreference != nil {
		attrs["audio_codec_preference"] = *p.AudioCodecPreference
	}
	if p.YoutubeAPIKey != nil {
		attrs["youtube_api_key"] = *p.YoutubeAPIKey
	}
	if p.ExtractorSleepIntervalSeconds != nil {
		attrs["extractor_sleep_interval_seconds"] = *p.ExtractorSleepIntervalSeconds
	}
	if p.DownloadThroughputLimit != nil {
		attrs["download_throughput_limit"] = *p.DownloadThroughputLimit
	}
	if p.RestrictFilenames != nil {
		attrs["restrict_filenames"] = *p.RestrictFilenames
	}

	cs := SettingChangeset(setting, attrs)
	updated, err := Update[Setting](ctx, s.Q(ctx), cs)
	if err != nil {
		if cs, ok := AsChangesetError(err); ok {
			return nil, cs.ErrorMap(), nil
		}
		return nil, nil, err
	}

	return updated, nil, nil
}

// SetSetting sets a single named setting (Settings.set/1).
func (s *Store) SetSetting(ctx context.Context, kw KW) (any, error) {
	if len(kw) != 1 {
		return nil, fmt.Errorf("SetSetting: expected exactly one keyword argument, got %d", len(kw))
	}

	attr := kw[0].Key
	value := kw[0].Value

	setting, err := s.GetSettingsRecord(ctx)
	if err != nil {
		return nil, err
	}

	if !settingsFieldExists(attr) {
		return nil, fmt.Errorf("invalid_key")
	}

	// Build SettingParams from the single attribute
	p := SettingParams{}
	switch attr {
	case "onboarding":
		if b, ok := value.(bool); ok {
			p.Onboarding = &b
		} else {
			return nil, fmt.Errorf("invalid value for %s: expected bool", attr)
		}
	case "yt_dlp_version":
		if s, ok := value.(string); ok {
			p.YtDlpVersion = &s
		} else {
			return nil, fmt.Errorf("invalid value for %s: expected string", attr)
		}
	case "video_codec_preference":
		if s, ok := value.(string); ok {
			p.VideoCodecPreference = &s
		} else {
			return nil, fmt.Errorf("invalid value for %s: expected string", attr)
		}
	case "audio_codec_preference":
		if s, ok := value.(string); ok {
			p.AudioCodecPreference = &s
		} else {
			return nil, fmt.Errorf("invalid value for %s: expected string", attr)
		}
	case "youtube_api_key":
		if s, ok := value.(string); ok {
			p.YoutubeAPIKey = &s
		} else {
			return nil, fmt.Errorf("invalid value for %s: expected string", attr)
		}
	case "extractor_sleep_interval_seconds":
		if i, ok := value.(int); ok {
			p.ExtractorSleepIntervalSeconds = &i
		} else {
			return nil, fmt.Errorf("invalid value for %s: expected int", attr)
		}
	case "download_throughput_limit":
		if s, ok := value.(string); ok {
			p.DownloadThroughputLimit = &s
		} else {
			return nil, fmt.Errorf("invalid value for %s: expected string", attr)
		}
	case "restrict_filenames":
		if b, ok := value.(bool); ok {
			p.RestrictFilenames = &b
		} else {
			return nil, fmt.Errorf("invalid value for %s: expected bool", attr)
		}
	}

	// Try to update the setting
	updated, errs, err := s.UpdateSetting(ctx, setting, p)
	if err != nil {
		return nil, err
	}

	if len(errs) > 0 {
		// Return the first error encountered
		for _, msgs := range errs {
			if len(msgs) > 0 {
				return nil, fmt.Errorf("%s", msgs[0])
			}
		}
	}

	// {:ok, %{^attr => _}} -> {:ok, value}: any struct key succeeds and
	// returns the value passed in; unknown keys are {:error, :invalid_key}.
	_ = updated
	return value, nil
}

// GetSetting returns a single named setting (Settings.get/1).
func (s *Store) GetSetting(ctx context.Context, name string) (any, error) {
	setting, err := s.GetSettingsRecord(ctx)
	if err != nil {
		return nil, err
	}

	if !settingsFieldExists(name) {
		return nil, fmt.Errorf("invalid_key")
	}

	val := settingsGetField(setting, name)
	return val, nil
}

// GetSettingBang returns a single named setting, erroring with a message
// naming the setting when not found.
func (s *Store) GetSettingBang(ctx context.Context, name string) (any, error) {
	val, err := s.GetSetting(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("Setting `%s` not found", name)
	}
	return val, nil
}

// settingsGetField retrieves a field value from a Setting struct by field name
func settingsGetField(s *Setting, fieldName string) any {
	rv := reflect.ValueOf(s).Elem()
	fv := rv.FieldByName(settingsFieldNameToStructField(fieldName))
	if !fv.IsValid() {
		return nil
	}
	// Like Elixir's Map.fetch: the plain value, or nil for NULL columns.
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return nil
		}
		return fv.Elem().Interface()
	}
	return fv.Interface()
}

// settingsFieldExists checks if a field name exists in the Setting struct
func settingsFieldExists(fieldName string) bool {
	// Convert db column name to struct field name and check if it exists
	rv := reflect.ValueOf(&Setting{}).Elem()
	return rv.FieldByName(settingsFieldNameToStructField(fieldName)).IsValid()
}

// settingsFieldNameToStructField converts a db column name (snake_case) to a struct field name (CamelCase)
func settingsFieldNameToStructField(dbName string) string {
	// This is a simple mapping based on the Setting struct
	fieldMap := map[string]string{
		"onboarding":                       "Onboarding",
		"yt_dlp_version":                   "YtDlpVersion",
		"video_codec_preference":           "VideoCodecPreference",
		"audio_codec_preference":           "AudioCodecPreference",
		"youtube_api_key":                  "YoutubeAPIKey",
		"extractor_sleep_interval_seconds": "ExtractorSleepIntervalSeconds",
		"download_throughput_limit":        "DownloadThroughputLimit",
		"restrict_filenames":               "RestrictFilenames",
		"route_token":                      "RouteToken",
		"id":                               "ID",
	}
	if name, ok := fieldMap[dbName]; ok {
		return name
	}
	return ""
}
