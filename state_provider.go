package nimsforestviewer

// StateProvider provides the current ViewState for visualization.
type StateProvider interface {
	// GetViewState returns the current visualization state.
	GetViewState() (*ViewState, error)
}

// StaticStateProvider wraps a fixed ViewState.
type StaticStateProvider struct {
	state *ViewState
}

// NewStaticStateProvider creates a StateProvider from a fixed ViewState.
func NewStaticStateProvider(state *ViewState) *StaticStateProvider {
	return &StaticStateProvider{state: state}
}

// GetViewState implements StateProvider.
func (p *StaticStateProvider) GetViewState() (*ViewState, error) {
	return p.state, nil
}

// CallbackStateProvider calls a function to get state.
type CallbackStateProvider struct {
	fn func() (*ViewState, error)
}

// NewCallbackStateProvider creates a StateProvider from a callback function.
func NewCallbackStateProvider(fn func() (*ViewState, error)) *CallbackStateProvider {
	return &CallbackStateProvider{fn: fn}
}

// GetViewState implements StateProvider.
func (p *CallbackStateProvider) GetViewState() (*ViewState, error) {
	return p.fn()
}
