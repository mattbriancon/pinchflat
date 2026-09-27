package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestUserScriptsCommandRunner_Run(t *testing.T) {
	writeLifecycleScript := func(ta *coretest.TestApp, content string) string {
		lifecycleDir := filepath.Join(ta.App.Config.ExtrasDirectory, "user-scripts")
		os.MkdirAll(lifecycleDir, 0o755)
		lifecycleFile := filepath.Join(lifecycleDir, "lifecycle")
		os.WriteFile(lifecycleFile, []byte(content), 0o755)
		return lifecycleFile
	}

	t.Run("runs provided file", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		testFile := filepath.Join(ta.App.Config.TmpfileDirectory, "test_file")
		script := "#!/bin/bash\ntouch " + testFile + "\n"
		writeLifecycleScript(ta, script)

		if _, err := os.Stat(testFile); err == nil {
			t.Fatal("test file should not exist yet")
		}

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("RunWithResult failed: %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable to be present")
		}

		if _, err := os.Stat(testFile); err != nil {
			t.Error("test file should have been created")
		}
	})

	t.Run("passes event name", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		eventFile := filepath.Join(ta.App.Config.TmpfileDirectory, "event_name")
		script := "#!/bin/bash\necho $1 > " + eventFile + "\n"
		writeLifecycleScript(ta, script)

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("RunWithResult failed: %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable")
		}

		content, _ := os.ReadFile(eventFile)
		if strings.TrimSpace(string(content)) != "media_downloaded" {
			t.Errorf("expected 'media_downloaded', got %q", strings.TrimSpace(string(content)))
		}
	})

	t.Run("passes encoded data", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		dataFile := filepath.Join(ta.App.Config.TmpfileDirectory, "encoded_data")
		script := "#!/bin/bash\necho $2 > " + dataFile + "\n"
		writeLifecycleScript(ta, script)

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{"foo": "bar"})
		if err != nil {
			t.Fatalf("RunWithResult failed: %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable")
		}

		content, _ := os.ReadFile(dataFile)
		if strings.TrimSpace(string(content)) != `{"foo":"bar"}` {
			t.Errorf("expected JSON, got %q", strings.TrimSpace(string(content)))
		}
	})

	t.Run("handles missing file", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("RunWithResult failed: %v", err)
		}
		if !result.NoExecutable {
			t.Error("expected NoExecutable=true when file missing")
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		writeLifecycleScript(ta, "")

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("RunWithResult failed: %v", err)
		}
		if !result.NoExecutable {
			t.Error("expected NoExecutable=true for empty file")
		}
	})

	t.Run("returns exit code", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		script := "#!/bin/bash\nexit 1\n"
		writeLifecycleScript(ta, script)

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("RunWithResult failed: %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable")
		}
		if result.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", result.ExitCode)
		}
	})

	t.Run("returns output", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		script := "#!/bin/bash\necho 'hello'\n"
		writeLifecycleScript(ta, script)

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("RunWithResult failed: %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable")
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if strings.TrimSpace(result.Output) != "hello" {
			t.Errorf("expected 'hello', got %q", strings.TrimSpace(result.Output))
		}
	})

	t.Run("errors on invalid event type", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		_, err := runner.RunWithResult(ta.Ctx, "invalid_event", map[string]interface{}{})
		if err == nil {
			t.Error("expected error for invalid event type")
		}
		if !strings.Contains(err.Error(), "Invalid event type") {
			t.Errorf("expected 'Invalid event type' in error, got %v", err)
		}
	})

	t.Run("errors on non-JSON-encodable data", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		writeLifecycleScript(ta, "#!/bin/bash")

		_, err := runner.RunWithResult(ta.Ctx, "media_downloaded", make(chan int))
		if err == nil {
			t.Error("expected error when encoding non-JSON-encodable value")
		}
	})
}
