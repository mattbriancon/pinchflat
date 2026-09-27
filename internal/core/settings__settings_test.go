package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestSettingsRecord(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("returns the only setting", func(t *testing.T) {
		setting, err := ta.SettingsRecord(ta.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}
		if setting == nil {
			t.Fatalf("expected Setting, got nil")
		}
	})
}

func TestSettingsUpdateSetting(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("updates the setting", func(t *testing.T) {
		// Setup
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", false)})
		if err != nil {
			t.Fatalf("setup: set onboarding to false failed: %v", err)
		}

		setting, err := ta.SettingsRecord(ta.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		// Check initial value
		val, err := ta.SettingsGet(ta.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGet failed: %v", err)
		}
		if val != false {
			t.Errorf("expected onboarding=false, got %v", val)
		}

		// Update
		updated, err := ta.SettingsUpdateSetting(ta.Ctx, setting, core.Attrs{"onboarding": true})
		if err != nil {
			t.Fatalf("SettingsUpdateSetting failed: %v", err)
		}
		if updated == nil {
			t.Fatalf("expected Setting, got nil")
		}

		// Check updated value
		val, err = ta.SettingsGet(ta.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGet failed: %v", err)
		}
		if val != true {
			t.Errorf("expected onboarding=true, got %v", val)
		}
	})
}

func TestSettingsSet(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("updates the setting", func(t *testing.T) {
		val, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", true)})
		if err != nil {
			t.Fatalf("SettingsSet failed: %v", err)
		}
		if val != true {
			t.Errorf("expected true, got %v", val)
		}

		val, err = ta.SettingsGet(ta.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGet failed: %v", err)
		}
		if val != true {
			t.Errorf("expected onboarding=true, got %v", val)
		}
	})

	t.Run("returns an error if the setting key doesn't exist", func(t *testing.T) {
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("foo", "bar")})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "invalid_key" {
			t.Errorf("expected 'invalid_key', got %q", err.Error())
		}
	})

	t.Run("returns an error if the setting value is invalid", func(t *testing.T) {
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", "bar")})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		// Check if it's a changeset error
		if _, ok := core.AsChangesetError(err); !ok {
			t.Errorf("expected ChangesetError, got %v", err)
		}
	})
}

func TestSettingsGet(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("returns the setting value", func(t *testing.T) {
		// Setup
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", false)})
		if err != nil {
			t.Fatalf("setup: set onboarding to false failed: %v", err)
		}

		val, err := ta.SettingsGet(ta.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGet failed: %v", err)
		}
		if val != false {
			t.Errorf("expected false, got %v", val)
		}
	})

	t.Run("returns an error if the setting key doesn't exist", func(t *testing.T) {
		_, err := ta.SettingsGet(ta.Ctx, "foo")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "invalid_key" {
			t.Errorf("expected 'invalid_key', got %q", err.Error())
		}
	})
}

func TestSettingsGetBang(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("returns the setting value", func(t *testing.T) {
		// Setup
		_, err := ta.SettingsSet(ta.Ctx, core.KW{core.Opt("onboarding", false)})
		if err != nil {
			t.Fatalf("setup: set onboarding to false failed: %v", err)
		}

		val, err := ta.SettingsGetBang(ta.Ctx, "onboarding")
		if err != nil {
			t.Fatalf("SettingsGetBang failed: %v", err)
		}
		if val != false {
			t.Errorf("expected false, got %v", val)
		}
	})

	t.Run("raises an error if the setting key doesn't exist", func(t *testing.T) {
		_, err := ta.SettingsGetBang(ta.Ctx, "foo")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "Setting `foo` not found" {
			t.Errorf("expected 'Setting `foo` not found', got %q", err.Error())
		}
	})
}

func TestSettingsChangeSetting(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("returns a changeset", func(t *testing.T) {
		setting, err := ta.SettingsRecord(ta.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		cs := ta.SettingsChangeSetting(ta.Ctx, setting, core.Attrs{"onboarding": true})
		if cs == nil {
			t.Fatalf("expected Changeset, got nil")
		}
	})

	t.Run("ensures the extractor sleep interval is positive", func(t *testing.T) {
		setting, err := ta.SettingsRecord(ta.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		// Test with 1
		cs := ta.SettingsChangeSetting(ta.Ctx, setting, core.Attrs{"extractor_sleep_interval_seconds": 1})
		if !cs.Valid() {
			t.Errorf("expected valid changeset for value 1, got errors: %v", cs.Errors)
		}

		// Test with 0
		cs = ta.SettingsChangeSetting(ta.Ctx, setting, core.Attrs{"extractor_sleep_interval_seconds": 0})
		if !cs.Valid() {
			t.Errorf("expected valid changeset for value 0, got errors: %v", cs.Errors)
		}

		// Test with -1
		cs = ta.SettingsChangeSetting(ta.Ctx, setting, core.Attrs{"extractor_sleep_interval_seconds": -1})
		if cs.Valid() {
			t.Errorf("expected invalid changeset for value -1, got no errors")
		}
	})

	t.Run("allows you to reset the extractor sleep interval", func(t *testing.T) {
		setting, err := ta.SettingsRecord(ta.Ctx)
		if err != nil {
			t.Fatalf("SettingsRecord failed: %v", err)
		}

		// Update to 1
		updated, err := ta.SettingsUpdateSetting(ta.Ctx, setting, core.Attrs{"extractor_sleep_interval_seconds": 1})
		if err != nil {
			t.Fatalf("SettingsUpdateSetting failed: %v", err)
		}

		// Now try to set it to 0
		cs := ta.SettingsChangeSetting(ta.Ctx, updated, core.Attrs{"extractor_sleep_interval_seconds": 0})
		if !cs.Valid() {
			t.Errorf("expected valid changeset for value 0, got errors: %v", cs.Errors)
		}
	})
}
