package store_test

import (
	"reflect"
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
		if err.Error() != "invalid value for onboarding: expected bool" {
			t.Errorf("expected type error, got %q", err.Error())
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

func TestSettingParamsValidate(t *testing.T) {
	t.Parallel()
	ts := storetest.NewStore(t)

	setting, err := ts.GetSettingsRecord(ts.Ctx)
	if err != nil {
		t.Fatalf("SettingsRecord failed: %v", err)
	}

	tests := []struct {
		name     string
		params   store.SettingParams
		expected map[string][]string
	}{
		{
			"empty params",
			store.SettingParams{},
			map[string][]string{},
		},
		{
			"negative extractor sleep",
			store.SettingParams{ExtractorSleepIntervalSeconds: store.Ptr(-1)},
			map[string][]string{"extractor_sleep_interval_seconds": {"must be greater than or equal to 0"}},
		},
		{
			"zero extractor sleep",
			store.SettingParams{ExtractorSleepIntervalSeconds: store.Ptr(0)},
			map[string][]string{},
		},
		{
			"positive extractor sleep",
			store.SettingParams{ExtractorSleepIntervalSeconds: store.Ptr(1)},
			map[string][]string{},
		},
		{
			"valid codec preferences",
			store.SettingParams{VideoCodecPreference: store.Ptr("h264"), AudioCodecPreference: store.Ptr("aac")},
			map[string][]string{},
		},
		{
			"empty video codec",
			store.SettingParams{VideoCodecPreference: store.Ptr("")},
			map[string][]string{"video_codec_preference": {"can't be blank"}},
		},
		{
			"empty audio codec",
			store.SettingParams{AudioCodecPreference: store.Ptr("")},
			map[string][]string{"audio_codec_preference": {"can't be blank"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.params.Validate(setting)
			if !reflect.DeepEqual(errs, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, errs)
			}
		})
	}
}
