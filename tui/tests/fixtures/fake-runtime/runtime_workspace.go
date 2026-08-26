package fakeruntime

import (
	"context"
	"sync"

	"codea/tui/internal/runtime"
)

type runtimeWorkspaceState struct {
	mu        sync.Mutex
	models    []runtime.Model
	modelErr  error
	compactErr error
	compacted []runtime.SessionID
}

var runtimeWorkspaceStates sync.Map // map[*FakeRuntime]*runtimeWorkspaceState

func workspaceState(f *FakeRuntime) *runtimeWorkspaceState {
	state, _ := runtimeWorkspaceStates.LoadOrStore(f, &runtimeWorkspaceState{})
	return state.(*runtimeWorkspaceState)
}

// SetModels configures the models returned by ListModels in tests.
func (f *FakeRuntime) SetModels(models []runtime.Model) {
	state := workspaceState(f)
	state.mu.Lock()
	defer state.mu.Unlock()
	state.models = append([]runtime.Model(nil), models...)
}

// SetListModelsError configures ListModels failure in tests.
func (f *FakeRuntime) SetListModelsError(err error) {
	state := workspaceState(f)
	state.mu.Lock()
	defer state.mu.Unlock()
	state.modelErr = err
}

// SetCompactError configures CompactSession failure in tests.
func (f *FakeRuntime) SetCompactError(err error) {
	state := workspaceState(f)
	state.mu.Lock()
	defer state.mu.Unlock()
	state.compactErr = err
}

func (f *FakeRuntime) ListModels(context.Context) ([]runtime.Model, error) {
	state := workspaceState(f)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.modelErr != nil {
		return nil, state.modelErr
	}
	return append([]runtime.Model(nil), state.models...), nil
}

func (f *FakeRuntime) CompactSession(_ context.Context, sessionID runtime.SessionID) error {
	state := workspaceState(f)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.compactErr != nil {
		return state.compactErr
	}
	state.compacted = append(state.compacted, sessionID)
	return nil
}

func (f *FakeRuntime) CompactCalls() int {
	state := workspaceState(f)
	state.mu.Lock()
	defer state.mu.Unlock()
	return len(state.compacted)
}

func (f *FakeRuntime) CompactedSessions() []runtime.SessionID {
	state := workspaceState(f)
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]runtime.SessionID(nil), state.compacted...)
}
