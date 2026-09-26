package coretest

import (
	"context"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// YtDlpRunFunc is the signature of YtDlpRunner.Run without ctx.
type YtDlpRunFunc func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error)

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

func (r ytDlpRunner) Run(_ context.Context, url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
	return r.m.Run.next()(url, action, opts, ot, addl)
}
func (r ytDlpRunner) Version(context.Context) (string, error) { return r.m.Version.next()() }
func (r ytDlpRunner) Update(context.Context) (string, error)  { return r.m.Update.next()() }

// AppriseMock replaces AppriseRunnerMock.
type AppriseMock struct {
	Run     *Mock[func(endpoints []string, opts core.KW) error]
	Version *Mock[func() (string, error)]
}

func NewAppriseMock(t testing.TB) *AppriseMock {
	return &AppriseMock{
		Run:     newMock[func([]string, core.KW) error](t, "AppriseRunnerMock.run"),
		Version: newMock[func() (string, error)](t, "AppriseRunnerMock.version"),
	}
}

type appriseRunner struct{ m *AppriseMock }

func (r appriseRunner) Run(_ context.Context, endpoints []string, opts core.KW) error {
	return r.m.Run.next()(endpoints, opts)
}
func (r appriseRunner) Version(context.Context) (string, error) { return r.m.Version.next()() }

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
	Get *Mock[func(url string, headers, opts core.KW) (string, error)]
}

func NewHTTPMock(t testing.TB) *HTTPMock {
	return &HTTPMock{Get: newMock[func(string, core.KW, core.KW) (string, error)](t, "HTTPClientMock.get")}
}

type httpClient struct{ m *HTTPMock }

func (c httpClient) Get(_ context.Context, url string, headers, opts core.KW) (string, error) {
	return c.m.Get.next()(url, headers, opts)
}
