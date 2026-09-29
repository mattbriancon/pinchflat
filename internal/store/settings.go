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

// UpdateSetting updates setting with attrs (deprecated; use UpdateSettingWithParams).
func (s *Store) UpdateSetting(ctx context.Context, setting *Setting, attrs Attrs) (*Setting, error) {
	cs := SettingChangeset(setting, attrs)
	return Update[Setting](ctx, s.Q(ctx), cs)
}

// UpdateSettingWithParams updates setting with typed parameters.
// Returns the updated Setting, validation errors as a map, and any system error.
func (s *Store) UpdateSettingWithParams(ctx context.Context, setting *Setting, p SettingParams) (*Setting, map[string][]string, error) {
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

	updated, err := s.UpdateSetting(ctx, setting, attrs)
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

	// Try to update the setting
	updated, err := s.UpdateSetting(ctx, setting, Attrs{attr: value})
	if err != nil {
		return nil, err
	}

	// {:ok, %{^attr => _}} -> {:ok, value}: any struct key succeeds and
	// returns the value passed in; unknown keys are {:error, :invalid_key}.
	_ = updated
	if !settingsFieldExists(attr) {
		return nil, fmt.Errorf("invalid_key")
	}
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

// ChangeSetting builds a changeset for setting from attrs.
func (s *Store) ChangeSetting(ctx context.Context, setting *Setting, attrs Attrs) *Changeset {
	return SettingChangeset(setting, attrs)
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
