package core_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestNotificationsCommandRunner_Run(t *testing.T) {
	t.Run("returns :ok when the command succeeds", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.NotificationsCommandRunner{App: ta.App}

		output, err := runner.RunFull(ta.Ctx, []string{"server_1"}, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if output == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("includes the servers as the first argument", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.NotificationsCommandRunner{App: ta.App}

		output, err := runner.RunFull(ta.Ctx, []string{"server_1", "server_2"}, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !strings.Contains(output, "server_1 server_2") {
			t.Errorf("expected output to contain 'server_1 server_2', got %q", output)
		}
	})

	t.Run("lets you pass a single server as a string", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.NotificationsCommandRunner{App: ta.App}

		output, err := runner.RunFull(ta.Ctx, []string{"server_1"}, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !strings.Contains(output, "server_1") {
			t.Errorf("expected output to contain 'server_1', got %q", output)
		}
	})

	t.Run("passes all arguments to the command", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.NotificationsCommandRunner{App: ta.App}

		output, err := runner.RunFull(ta.Ctx, []string{"server_1"}, core.KW{core.Opt("dry-run", "")})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !strings.Contains(output, "--dry-run") {
			t.Errorf("expected output to contain '--dry-run', got %q", output)
		}
	})

	t.Run("returns the output when the command fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.App.Config.AppriseExecutable = "/bin/false"
		runner := &core.NotificationsCommandRunner{App: ta.App}

		output, err := runner.RunFull(ta.Ctx, []string{"server_1"}, nil)
		if err == nil {
			t.Error("expected an error, got nil")
		}
		if output != "" {
			t.Errorf("expected empty output on error, got %q", output)
		}
	})

	t.Run("returns a relevant error if no servers are provided", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.NotificationsCommandRunner{App: ta.App}

		_, err := runner.RunFull(ta.Ctx, []string{}, nil)
		if err == nil || err.Error() != "no_servers" {
			t.Errorf("expected error 'no_servers', got %v", err)
		}
	})
}

func TestNotificationsCommandRunner_Version(t *testing.T) {
	t.Run("adds the version arg", func(t *testing.T) {
		ta := coretest.NewApp(t)
		runner := &core.NotificationsCommandRunner{App: ta.App}

		output, err := runner.Version(ta.Ctx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		// The repeater.sh mock should echo the arguments, so we expect "--version" in the output
		if !strings.Contains(output, "--version") {
			t.Errorf("expected output to contain '--version', got %q", output)
		}
	})
}
