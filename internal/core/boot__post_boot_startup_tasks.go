package core

import (
	"context"
)

// PostBootStartupTasks runs startup tasks after the app has fully booted.

type PostBootStartupTasks struct {
	// GenServer state will be added during implementation
}

// PostBootStartupTasksStartLink/1
func PostBootStartupTasksStartLink(ctx context.Context, opts KW) (any, error) {
	panic("unported: Pinchflat.Boot.PostBootStartupTasks.start_link/1")
}

// PostBootStartupTasksInit/1
func (p *PostBootStartupTasks) PostBootStartupTasksInit(ctx context.Context, state map[string]any) (map[string]any, error) {
	panic("unported: Pinchflat.Boot.PostBootStartupTasks.init/1")
}
