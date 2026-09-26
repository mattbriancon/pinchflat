package core

import (
	"context"
)

// (a *App) CliUtilsWrapCmd(ctx, command, args, passthrough_opts, opts)
// Wraps a command in a shell script that will terminate the command if stdin is closed.
// Useful for stopping commands if the job runner is cancelled.
func (a *App) CliUtilsWrapCmd(ctx context.Context, command string, args []string, passthroughOpts KW, opts KW) (string, int, error) {
	panic("unported: Pinchflat.Utils.CliUtils.wrap_cmd/4")
}

// CliUtilsParseOptions(command_opts)
// Parses a list of command options into a list of strings suitable for passing to System.cmd.
func CliUtilsParseOptions(commandOpts interface{}) []string {
	panic("unported: Pinchflat.Utils.CliUtils.parse_options/1")
}
