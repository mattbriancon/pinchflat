package core

import (
	"context"
)

// UserScriptsCommandRunner runs custom user commands using the System.cmd function.

const userScriptsEventTypesCount = 4 // number of valid event types

var userScriptsEventTypes = []string{
	"app_init",
	"media_pre_download",
	"media_downloaded",
	"media_deleted",
}

// UserScriptsCommandRunnerRun/2
func (a *App) UserScriptsCommandRunnerRun(ctx context.Context, eventType string, encodeableData any) (any, int, error) {
	panic("unported: Pinchflat.Lifecycle.UserScripts.CommandRunner.run/2")
}
