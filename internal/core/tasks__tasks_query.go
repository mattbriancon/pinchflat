package core

import (
	sq "github.com/Masterminds/squirrel"
)

// TasksQueryNew/0
func TasksQueryNew() sq.SelectBuilder {
	panic("unported: Pinchflat.Tasks.TasksQuery.new/0")
}

// TasksQueryJoinJob/1
func TasksQueryJoinJob(query sq.SelectBuilder) sq.SelectBuilder {
	panic("unported: Pinchflat.Tasks.TasksQuery.join_job/1")
}

// TasksQueryInState/1
func TasksQueryInState(states []string) sq.Sqlizer {
	panic("unported: Pinchflat.Tasks.TasksQuery.in_state/1")
}

// TasksQueryHasTag/1
func TasksQueryHasTag(tag string) sq.Sqlizer {
	panic("unported: Pinchflat.Tasks.TasksQuery.has_tag/1")
}
