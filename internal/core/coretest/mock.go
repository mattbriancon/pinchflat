package coretest

import (
	"fmt"
	"sync"
	"testing"
)

// Mock is a Mox-style expectation list for one function of type F.
//
//	m.Expect(fn)       // expect(Mock, :fun, fn)          — exactly one call
//	m.ExpectN(3, fn)   // expect(Mock, :fun, 3, fn)
//	m.Stub(fn)         // stub(Mock, :fun, fn)            — any number of calls
//
// Expectations are consumed in order before the stub is used. Calling with
// nothing left fails the test. Unconsumed expectations fail the test at
// cleanup (Mox's verify_on_exit!).
type Mock[F any] struct {
	t       testing.TB
	name    string
	mu      sync.Mutex
	expects []F
	stub    *F
	calls   int
}

func newMock[F any](t testing.TB, name string) *Mock[F] {
	m := &Mock[F]{t: t, name: name}
	t.Cleanup(func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if len(m.expects) > 0 {
			t.Errorf("%s: %d expected call(s) not made", m.name, len(m.expects))
		}
	})
	return m
}

func (m *Mock[F]) Expect(f F) *Mock[F] { return m.ExpectN(1, f) }

func (m *Mock[F]) ExpectN(n int, f F) *Mock[F] {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := 0; i < n; i++ {
		m.expects = append(m.expects, f)
	}
	return m
}

func (m *Mock[F]) Stub(f F) *Mock[F] {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stub = &f
	return m
}

// Calls returns how many times the mock was called.
func (m *Mock[F]) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// next returns the function to handle a call. With no expectation or stub
// it fails the test (Mox raises UnexpectedCallError) and panics to stop the
// code under test.
func (m *Mock[F]) next() F {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if len(m.expects) > 0 {
		f := m.expects[0]
		m.expects = m.expects[1:]
		return f
	}
	if m.stub != nil {
		return *m.stub
	}
	msg := fmt.Sprintf("%s: unexpected call (no expectations or stubs defined)", m.name)
	m.t.Error(msg)
	panic(msg)
}
