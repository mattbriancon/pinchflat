package core

import (
	"context"
	"fmt"
	"reflect"
)

// SettingsRecord/0
func (a *App) SettingsRecord(ctx context.Context) (*Setting, error) {
	return One[Setting](ctx, a.Q(ctx), From[Setting]().Limit(1))
}

// SettingsUpdateSetting/2
func (a *App) SettingsUpdateSetting(ctx context.Context, setting *Setting, attrs Attrs) (*Setting, error) {
	cs := SettingChangeset(setting, attrs)
	return Update[Setting](ctx, a.Q(ctx), cs)
}

// SettingsSet/1
func (a *App) SettingsSet(ctx context.Context, kw KW) (any, error) {
	if len(kw) != 1 {
		return nil, fmt.Errorf("SettingsSet: expected exactly one keyword argument, got %d", len(kw))
	}

	attr := kw[0].Key
	value := kw[0].Value

	setting, err := a.SettingsRecord(ctx)
	if err != nil {
		return nil, err
	}

	// Try to update the setting
	updated, err := a.SettingsUpdateSetting(ctx, setting, Attrs{attr: value})
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

// SettingsGet/1
func (a *App) SettingsGet(ctx context.Context, name string) (any, error) {
	setting, err := a.SettingsRecord(ctx)
	if err != nil {
		return nil, err
	}

	if !settingsFieldExists(name) {
		return nil, fmt.Errorf("invalid_key")
	}

	val := settingsGetField(setting, name)
	return val, nil
}

// SettingsGetBang/1
func (a *App) SettingsGetBang(ctx context.Context, name string) (any, error) {
	val, err := a.SettingsGet(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("Setting `%s` not found", name)
	}
	return val, nil
}

// SettingsChangeSetting/2
func (a *App) SettingsChangeSetting(ctx context.Context, setting *Setting, attrs Attrs) *Changeset {
	return SettingChangeset(setting, attrs)
}

// settingsGetField retrieves a field value from a Setting struct by field name
func settingsGetField(s *Setting, fieldName string) any {
	rv := reflect.ValueOf(s).Elem()
	fv := rv.FieldByName(settingsFieldNameToStructField(fieldName))
	if !fv.IsValid() {
		return nil
	}
	if fv.Kind() == reflect.Pointer && fv.IsNil() {
		return nil
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
		"pro_enabled":                      "ProEnabled",
		"yt_dlp_version":                   "YtDlpVersion",
		"apprise_server":                   "AppriseServer",
		"apprise_version":                  "AppriseVersion",
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
