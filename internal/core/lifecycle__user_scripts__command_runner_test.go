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
	t.Run("runs the provided lifecycle file if present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		// Create the user-scripts directory and lifecycle file
		lifecycleDir := filepath.Join(ta.App.Config.ExtrasDirectory, "user-scripts")
		if err := os.MkdirAll(lifecycleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		lifecycleFile := filepath.Join(lifecycleDir, "lifecycle")

		// Create a test file that will be touched by the script
		tmpDir := ta.App.Config.TmpfileDirectory
		testFilename := filepath.Join(tmpDir, "test_file")

		// Write a script that touches the test file
		scriptContent := "#!/bin/bash\ntouch " + testFilename + "\n"
		if err := os.WriteFile(lifecycleFile, []byte(scriptContent), 0o755); err != nil {
			t.Fatal(err)
		}

		// Verify the file doesn't exist yet
		if _, err := os.Stat(testFilename); err == nil {
			t.Fatal("test file should not exist yet")
		}

		// Run the script
		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable to be present")
		}

		// Verify the file was created
		if _, err := os.Stat(testFilename); err != nil {
			t.Fatalf("expected test file to be created, got error: %v", err)
		}
	})

	t.Run("passes the event name to the script", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		lifecycleDir := filepath.Join(ta.App.Config.ExtrasDirectory, "user-scripts")
		if err := os.MkdirAll(lifecycleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		lifecycleFile := filepath.Join(lifecycleDir, "lifecycle")
		tmpDir := ta.App.Config.TmpfileDirectory
		eventNameFile := filepath.Join(tmpDir, "event_name")

		scriptContent := "#!/bin/bash\necho $1 > " + eventNameFile + "\n"
		if err := os.WriteFile(lifecycleFile, []byte(scriptContent), 0o755); err != nil {
			t.Fatal(err)
		}

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable to be present")
		}

		content, err := os.ReadFile(eventNameFile)
		if err != nil {
			t.Fatalf("expected event name file to be created, got error: %v", err)
		}
		if strings.TrimSpace(string(content)) != "media_downloaded" {
			t.Errorf("expected 'media_downloaded', got %q", strings.TrimSpace(string(content)))
		}
	})

	t.Run("passes the encoded data to the script", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		lifecycleDir := filepath.Join(ta.App.Config.ExtrasDirectory, "user-scripts")
		if err := os.MkdirAll(lifecycleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		lifecycleFile := filepath.Join(lifecycleDir, "lifecycle")
		tmpDir := ta.App.Config.TmpfileDirectory
		encodedDataFile := filepath.Join(tmpDir, "encoded_data")

		scriptContent := "#!/bin/bash\necho $2 > " + encodedDataFile + "\n"
		if err := os.WriteFile(lifecycleFile, []byte(scriptContent), 0o755); err != nil {
			t.Fatal(err)
		}

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{"foo": "bar"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable to be present")
		}

		content, err := os.ReadFile(encodedDataFile)
		if err != nil {
			t.Fatalf("expected encoded data file to be created, got error: %v", err)
		}
		if strings.TrimSpace(string(content)) != `{"foo":"bar"}` {
			t.Errorf("expected '{\"foo\":\"bar\"}', got %q", strings.TrimSpace(string(content)))
		}
	})

	t.Run("does nothing if the lifecycle file is not present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !result.NoExecutable {
			t.Error("expected NoExecutable to be true")
		}
	})

	t.Run("does nothing if the lifecycle file is empty", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		lifecycleDir := filepath.Join(ta.App.Config.ExtrasDirectory, "user-scripts")
		if err := os.MkdirAll(lifecycleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		lifecycleFile := filepath.Join(lifecycleDir, "lifecycle")

		if err := os.WriteFile(lifecycleFile, []byte(""), 0o755); err != nil {
			t.Fatal(err)
		}

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !result.NoExecutable {
			t.Error("expected NoExecutable to be true")
		}
	})

	t.Run("returns :ok if the command exits with a non-zero status", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		lifecycleDir := filepath.Join(ta.App.Config.ExtrasDirectory, "user-scripts")
		if err := os.MkdirAll(lifecycleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		lifecycleFile := filepath.Join(lifecycleDir, "lifecycle")

		scriptContent := "#!/bin/bash\nexit 1\n"
		if err := os.WriteFile(lifecycleFile, []byte(scriptContent), 0o755); err != nil {
			t.Fatal(err)
		}

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable to be present")
		}
		if result.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", result.ExitCode)
		}
	})

	t.Run("returns the output of the command", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		lifecycleDir := filepath.Join(ta.App.Config.ExtrasDirectory, "user-scripts")
		if err := os.MkdirAll(lifecycleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		lifecycleFile := filepath.Join(lifecycleDir, "lifecycle")

		scriptContent := "#!/bin/bash\necho 'hello'\n"
		if err := os.WriteFile(lifecycleFile, []byte(scriptContent), 0o755); err != nil {
			t.Fatal(err)
		}

		result, err := runner.RunWithResult(ta.Ctx, "media_downloaded", map[string]interface{}{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.NoExecutable {
			t.Error("expected executable to be present")
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if strings.TrimSpace(result.Output) != "hello" {
			t.Errorf("expected 'hello', got %q", strings.TrimSpace(result.Output))
		}
	})

	t.Run("gets upset if you pass an invalid event type", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		_, err := runner.RunWithResult(ta.Ctx, "invalid_event", map[string]interface{}{})
		if err == nil {
			t.Error("expected an error for invalid event type")
		}
		if !strings.Contains(err.Error(), "Invalid event type") {
			t.Errorf("expected error to contain 'Invalid event type', got %v", err)
		}
	})

	t.Run("gets upset if the record cannot be decoded", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.UserScriptsCommandRunner{App: ta.App}

		lifecycleDir := filepath.Join(ta.App.Config.ExtrasDirectory, "user-scripts")
		if err := os.MkdirAll(lifecycleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		lifecycleFile := filepath.Join(lifecycleDir, "lifecycle")
		if err := os.WriteFile(lifecycleFile, []byte("#!/bin/bash"), 0o755); err != nil {
			t.Fatal(err)
		}

		// Try to pass a non-JSON-encodable value (a store.Changeset, but we'll use a channel which can't be JSON encoded)
		_, err := runner.RunWithResult(ta.Ctx, "media_downloaded", make(chan int))
		if err == nil {
			t.Error("expected an error when trying to encode a non-JSON-encodable value")
		}
	})
}
