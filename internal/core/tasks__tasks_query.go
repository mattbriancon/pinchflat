package core

// Port of lib/pinchflat/tasks/tasks_query.ex (aliases: tasks AS t, oban_jobs AS j).

import sq "github.com/Masterminds/squirrel"

// new/0
func TasksQueryNew() sq.SelectBuilder { return From[Task]("t") }

// join_job/1: LEFT JOIN oban_jobs AS j. Elixir also preloads the job; in Go
// call PreloadTaskJob on the results.
func TasksQueryJoinJob(query sq.SelectBuilder) sq.SelectBuilder {
	return query.LeftJoin("oban_jobs AS j ON j.id = t.job_id")
}

// in_state/1 (needs join_job)
func TasksQueryInState(states []string) sq.Sqlizer { return sq.Eq{"j.state": states} }

// has_tag/1 (needs join_job): `^tag in j.tags` on a JSON array column.
func TasksQueryHasTag(tag string) sq.Sqlizer {
	return sq.Expr("? IN (SELECT value FROM json_each(j.tags))", tag)
}
