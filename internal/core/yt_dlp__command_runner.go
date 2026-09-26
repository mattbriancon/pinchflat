package core

import "context"

// YtDlpCommandRunner is Pinchflat.YtDlp.CommandRunner, the real yt-dlp
// runner. It implements YtDlpRunner (app.go).
type YtDlpCommandRunner struct {
	App *App
}

var _ YtDlpRunner = (*YtDlpCommandRunner)(nil)

// run/5 — {:ok, output} | {:error, output, status} (as *CommandError)
func (r *YtDlpCommandRunner) Run(ctx context.Context, url string, actionName string, commandOpts KW, outputTemplate string, addlOpts KW) (string, error) {
	panic("unported: Pinchflat.YtDlp.CommandRunner.run/5")
}

// version/0
func (r *YtDlpCommandRunner) Version(ctx context.Context) (string, error) {
	panic("unported: Pinchflat.YtDlp.CommandRunner.version/0")
}

// update/0
func (r *YtDlpCommandRunner) Update(ctx context.Context) (string, error) {
	panic("unported: Pinchflat.YtDlp.CommandRunner.update/0")
}
