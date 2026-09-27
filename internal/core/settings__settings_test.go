package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestSettingsRecord(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)

	setting, err := ta.SettingsRecord(ta.Ctx)
	if err != nil {
		t.Fatalf("SettingsRecord failed: %v", err)
	}
	if setting == nil {
		t.Fatalf("expected Setting")
	}
}

func TestSettingsUpdateSetting(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)

	_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", false)})
	if err != nil {
		t.Fatalf("setup: set onboarding to false failed: %v", err)
	}

	setting, err := ta.SettingsRecord(ta.Ctx)
	if err != nil {
		t.Fatalf("SettingsRecord failed: %v", err)
	}

	val, err := ta.SettingsGet(ta.Ctx, "onboarding")
	if err != nil {
		t.Fatalf("SettingsGet failed: %v", err)
	}
	if val != false {
		t.Errorf("expected onboarding=false")
	}

	updated, err := ta.SettingsUpdateSetting(ta.Ctx, setting, core.Attrs{"onboarding": true})
	if err != nil {
		t.Fatalf("SettingsUpdateSetting failed: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected Setting")
	}

	val, err = ta.SettingsGet(ta.Ctx, "onboarding")
	if err != nil {
		t.Fatalf("SettingsGet failed: %v", err)
	}
	if val != true {
		t.Errorf("expected onboarding=true")
	}
}

func TestSettingsSet(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("updates setting", func(t *testing.T) {
		t.Parallel()
		val, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", true)})
		if err != nil {
			t.Fatalf("SettingsSet failed: %v", err)
		}
		if val != true {
			t.Errorf("expected true")
		}

		val, err = ta.SettingsGet(ta.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGet failed: %v", err)
		}
		if val != true {
			t.Errorf("expected onboarding=true")
		}
	})

	t.Run("errors on invalid key", func(t *testing.T) {
		t.Parallel()
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("foo", "bar")})
		if err == nil {
			t.Fatalf("expected error")
		}
		if err.Error() != "invalid_key" {
			t.Errorf("expected 'invalid_key', got %q", err.Error())
		}
	})

	t.Run("errors on invalid value", func(t *testing.T) {
		t.Parallel()
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", "bar")})
		if err == nil {
			t.Fatalf("expected error")
		}
		if _, ok := core.AsChangesetError(err); !ok {
			t.Errorf("expected ChangesetError, got %T", err)
		}
	})
}

func TestSettingsGet(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("returns value", func(t *testing.T) {
		t.Parallel()
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", false)})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		val, err := ta.SettingsGet(ta.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGet failed: %v", err)
		}
		if val != false {
			t.Errorf("expected false")
		}
	})

	t.Run("errors on invalid key", func(t *testing.T) {
		t.Parallel()
		_, err := ta.SettingsGet(ta.Ctx, "foo")
		if err == nil {
			t.Fatalf("expected error")
		}
		if err.Error() != "invalid_key" {
			t.Errorf("expected 'invalid_key', got %q", err.Error())
		}
	})
}

func TestSettingsGetBang(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("returns value", func(t *testing.T) {
		t.Parallel()
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", false)})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		val, err := ta.SettingsGetBang(ta.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGetBang failed: %v", err)
		}
		if val != false {
			t.Errorf("expected false")
		}
	})

	t.Run("raises on invalid key", func(t *testing.T) {
		t.Parallel()
		_, err := ta.SettingsGetBang(ta.Ctx, "foo")
		if err == nil {
			t.Fatalf("expected error")
		}
		if err.Error() != "Setting `foo` not found" {
			t.Errorf("expected 'Setting `foo` not found', got %q", err.Error())
		}
	})
}

func TestSettingsChangeSetting(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("returns changeset", func(t *testing.T) {
		t.Parallel()
		setting, err := ta.SettingsRecord(ta.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		cs := ta.SettingsChangeSetting(ta.Ctx, setting, core.Attrs{"onboarding": true})
		if cs == nil {
			t.Fatalf("expected Changeset")
		}
	})

	t.Run("validates extractor sleep interval", func(t *testing.T) {
		t.Parallel()
		setting, err := ta.SettingsRecord(ta.Ctx)
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
			cs := ta.SettingsChangeSetting(ta.Ctx, setting, core.Attrs{"extractor_sleep_interval_seconds": tt.value})
			if cs.Valid() != tt.isValid {
				t.Errorf("value %d: expected valid=%v, got %v", tt.value, tt.isValid, cs.Valid())
			}
		}
	})

	t.Run("allows resetting sleep interval", func(t *testing.T) {
		t.Parallel()
		setting, err := ta.SettingsRecord(ta.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		updated, err := ta.SettingsUpdateSetting(ta.Ctx, setting, core.Attrs{"extractor_sleep_interval_seconds": 1})
		if err != nil {
			t.Fatalf("SettingsUpdateSetting failed: %v", err)
		}

		cs := ta.SettingsChangeSetting(ta.Ctx, updated, core.Attrs{"extractor_sleep_interval_seconds": 0})
		if !cs.Valid() {
			t.Errorf("resetting to 0 should be valid, got errors: %v", cs.Errors)
		}
	})
}
