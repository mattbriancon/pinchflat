package store_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/store/storetest"
)

func TestSettingsRecord(t *testing.T) {
	t.Parallel()
	ts := storetest.NewStore(t)

	setting, err := ts.GetSettingsRecord(ts.Ctx)
	if err != nil {
		t.Fatalf("SettingsRecord failed: %v", err)
	}
	if setting == nil {
		t.Fatalf("expected store.Setting")
	}
}

func TestSettingsUpdateSetting(t *testing.T) {
	t.Parallel()
	ts := storetest.NewStore(t)

	_, err := ts.SetSetting(ts.Ctx, store.KW{store.Opt("onboarding", false)})
	if err != nil {
		t.Fatalf("setup: set onboarding to false failed: %v", err)
	}

	setting, err := ts.GetSettingsRecord(ts.Ctx)
	if err != nil {
		t.Fatalf("SettingsRecord failed: %v", err)
	}

	val, err := ts.GetSetting(ts.Ctx, "onboarding")
	if err != nil {
		t.Fatalf("SettingsGet failed: %v", err)
	}
	if val != false {
		t.Errorf("expected onboarding=false")
	}

	updated, err := ts.UpdateSetting(ts.Ctx, setting, store.Attrs{"onboarding": true})
	if err != nil {
		t.Fatalf("SettingsUpdateSetting failed: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected store.Setting")
	}

	val, err = ts.GetSetting(ts.Ctx, "onboarding")
	if err != nil {
		t.Fatalf("SettingsGet failed: %v", err)
	}
	if val != true {
		t.Errorf("expected onboarding=true")
	}
}

func TestSettingsSet(t *testing.T) {
	ts := storetest.NewStore(t)

	t.Run("updates setting", func(t *testing.T) {
		t.Parallel()
		val, err := ts.SetSetting(ts.Ctx, store.KW{store.Opt("onboarding", true)})
		if err != nil {
			t.Fatalf("SettingsSet failed: %v", err)
		}
		if val != true {
			t.Errorf("expected true")
		}

		val, err = ts.GetSetting(ts.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGet failed: %v", err)
		}
		if val != true {
			t.Errorf("expected onboarding=true")
		}
	})

	t.Run("errors on invalid key", func(t *testing.T) {
		t.Parallel()
		_, err := ts.SetSetting(ts.Ctx, store.KW{store.Opt("foo", "bar")})
		if err == nil {
			t.Fatalf("expected error")
		}
		if err.Error() != "invalid_key" {
			t.Errorf("expected 'invalid_key', got %q", err.Error())
		}
	})

	t.Run("errors on invalid value", func(t *testing.T) {
		t.Parallel()
		_, err := ts.SetSetting(ts.Ctx, store.KW{store.Opt("onboarding", "bar")})
		if err == nil {
			t.Fatalf("expected error")
		}
		if _, ok := store.AsChangesetError(err); !ok {
			t.Errorf("expected store.ChangesetError, got %T", err)
		}
	})
}

func TestSettingsGet(t *testing.T) {
	ts := storetest.NewStore(t)

	t.Run("returns value", func(t *testing.T) {
		t.Parallel()
		_, err := ts.SetSetting(ts.Ctx, store.KW{store.Opt("onboarding", false)})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		val, err := ts.GetSetting(ts.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGet failed: %v", err)
		}
		if val != false {
			t.Errorf("expected false")
		}
	})

	t.Run("errors on invalid key", func(t *testing.T) {
		t.Parallel()
		_, err := ts.GetSetting(ts.Ctx, "foo")
		if err == nil {
			t.Fatalf("expected error")
		}
		if err.Error() != "invalid_key" {
			t.Errorf("expected 'invalid_key', got %q", err.Error())
		}
	})
}

func TestSettingsGetBang(t *testing.T) {
	ts := storetest.NewStore(t)

	t.Run("returns value", func(t *testing.T) {
		t.Parallel()
		_, err := ts.SetSetting(ts.Ctx, store.KW{store.Opt("onboarding", false)})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		val, err := ts.GetSettingBang(ts.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGetBang failed: %v", err)
		}
		if val != false {
			t.Errorf("expected false")
		}
	})

	t.Run("raises on invalid key", func(t *testing.T) {
		t.Parallel()
		_, err := ts.GetSettingBang(ts.Ctx, "foo")
		if err == nil {
			t.Fatalf("expected error")
		}
		if err.Error() != "Setting `foo` not found" {
			t.Errorf("expected 'Setting `foo` not found', got %q", err.Error())
		}
	})
}

func TestSettingsChangeSetting(t *testing.T) {
	ts := storetest.NewStore(t)

	t.Run("returns changeset", func(t *testing.T) {
		t.Parallel()
		setting, err := ts.GetSettingsRecord(ts.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		cs := ts.ChangeSetting(ts.Ctx, setting, store.Attrs{"onboarding": true})
		if cs == nil {
			t.Fatalf("expected store.Changeset")
		}
	})

	t.Run("validates extractor sleep interval", func(t *testing.T) {
		t.Parallel()
		setting, err := ts.GetSettingsRecord(ts.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		tests := []struct {
			value   int
			isValid bool
		}{
			{1, true},
			{0, true},
			{-1, false},
		}

		for _, tt := range tests {
			cs := ts.ChangeSetting(ts.Ctx, setting, store.Attrs{"extractor_sleep_interval_seconds": tt.value})
			if cs.Valid() != tt.isValid {
				t.Errorf("value %d: expected valid=%v, got %v", tt.value, tt.isValid, cs.Valid())
			}
		}
	})

	t.Run("allows resetting sleep interval", func(t *testing.T) {
		t.Parallel()
		setting, err := ts.GetSettingsRecord(ts.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		updated, err := ts.UpdateSetting(ts.Ctx, setting, store.Attrs{"extractor_sleep_interval_seconds": 1})
		if err != nil {
			t.Fatalf("SettingsUpdateSetting failed: %v", err)
		}

		cs := ts.ChangeSetting(ts.Ctx, updated, store.Attrs{"extractor_sleep_interval_seconds": 0})
		if !cs.Valid() {
			t.Errorf("resetting to 0 should be valid, got errors: %v", cs.Errors)
		}
	})
}

func TestSettingParamsValidateMatchesChangeset(t *testing.T) {
	t.Parallel()
	ts := storetest.NewStore(t)

	setting, err := ts.GetSettingsRecord(ts.Ctx)
	if err != nil {
		t.Fatalf("SettingsRecord failed: %v", err)
	}

	tests := []struct {
		name  string
		attrs store.Attrs
	}{
		{"empty attrs", store.Attrs{}},
		{"negative extractor sleep", store.Attrs{"extractor_sleep_interval_seconds": -1}},
		{"zero extractor sleep", store.Attrs{"extractor_sleep_interval_seconds": 0}},
		{"positive extractor sleep", store.Attrs{"extractor_sleep_interval_seconds": 1}},
		{"valid codec preferences", store.Attrs{"video_codec_preference": "h264", "audio_codec_preference": "aac"}},
		{"empty video codec", store.Attrs{"video_codec_preference": ""}},
		{"empty audio codec", store.Attrs{"audio_codec_preference": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := store.SettingChangeset(setting, tt.attrs)
			changesetErrors := cs.ErrorMap()

			// Convert attrs to SettingParams for comparison
			params := store.SettingParams{}
			if v, ok := tt.attrs["extractor_sleep_interval_seconds"]; ok {
				i := v.(int)
				params.ExtractorSleepIntervalSeconds = &i
			}
			if v, ok := tt.attrs["video_codec_preference"]; ok {
				s := v.(string)
				params.VideoCodecPreference = &s
			}
			if v, ok := tt.attrs["audio_codec_preference"]; ok {
				s := v.(string)
				params.AudioCodecPreference = &s
			}

			paramsErrors := params.Validate(setting)

			// Compare error sets
			if !errorMapsEqual(changesetErrors, paramsErrors) {
				t.Errorf("error mismatch\nchangeset: %v\nparams:    %v", changesetErrors, paramsErrors)
			}
		})
	}
}

func errorMapsEqual(m1, m2 map[string][]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok || !slicesEqual(v1, v2) {
			return false
		}
	}
	return true
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
