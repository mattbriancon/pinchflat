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
