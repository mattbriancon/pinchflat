package apptest

import (
	"context"
	"net/http"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// YtDlpRunFunc is the signature of YtDlpRunner.Run without ctx.
type YtDlpRunFunc func(url, action string, args ytdlp.Args, outputTemplate string, opts ytdlp.CallOptions) (string, error)

// YtDlpMock replaces YtDlpRunnerMock.
type YtDlpMock struct {
	Run     *Mock[YtDlpRunFunc]
	Version *Mock[func() (string, error)]
	Update  *Mock[func() (string, error)]
}

func NewYtDlpMock(t testing.TB) *YtDlpMock {
	return &YtDlpMock{
		Run:     newMock[YtDlpRunFunc](t, "YtDlpRunnerMock.run"),
		Version: newMock[func() (string, error)](t, "YtDlpRunnerMock.version"),
		Update:  newMock[func() (string, error)](t, "YtDlpRunnerMock.update"),
	}
}

type ytDlpRunner struct{ m *YtDlpMock }

func (r ytDlpRunner) Run(_ context.Context, url, action string, args ytdlp.Args, ot string, opts ytdlp.CallOptions) (string, error) {
	return r.m.Run.next()(url, action, args, ot, opts)
}
func (r ytDlpRunner) Version(context.Context) (string, error) { return r.m.Version.next()() }
func (r ytDlpRunner) Update(context.Context) (string, error)  { return r.m.Update.next()() }

// UserScriptMock replaces UserScriptRunnerMock.
type UserScriptMock struct {
	Run *Mock[func(event string, data any) error]
}

func NewUserScriptMock(t testing.TB) *UserScriptMock {
	return &UserScriptMock{Run: newMock[func(string, any) error](t, "UserScriptRunnerMock.run")}
}

type userScriptRunner struct{ m *UserScriptMock }

func (r userScriptRunner) Run(_ context.Context, event string, data any) error {
	return r.m.Run.next()(event, data)
}

// HTTPMock replaces HTTPClientMock.
type HTTPMock struct {
	Get *Mock[func(url string, headers http.Header) (string, error)]
}

func NewHTTPMock(t testing.TB) *HTTPMock {
	return &HTTPMock{Get: newMock[func(string, http.Header) (string, error)](t, "HTTPClientMock.get")}
}

type httpClient struct{ m *HTTPMock }

func (c httpClient) Get(_ context.Context, url string, headers http.Header) (string, error) {
	return c.m.Get.next()(url, headers)
}
