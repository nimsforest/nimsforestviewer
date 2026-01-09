package nimsforestviewer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Viewer manages visualization output to multiple targets.
type Viewer struct {
	mu       sync.RWMutex
	provider StateProvider
	targets  []Target
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

// Option configures the Viewer.
type Option func(*Viewer)

// WithInterval sets the update interval for periodic updates.
func WithInterval(d time.Duration) Option {
	return func(v *Viewer) {
		v.interval = d
	}
}

// New creates a new Viewer with the given options.
func New(opts ...Option) *Viewer {
	v := &Viewer{
		interval: time.Second, // Default 1 second
		done:     make(chan struct{}),
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// SetStateProvider sets the source of ViewState.
func (v *Viewer) SetStateProvider(p StateProvider) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.provider = p
}

// AddTarget adds an output target.
func (v *Viewer) AddTarget(t Target) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.targets = append(v.targets, t)
	return nil
}

// RemoveTarget removes a target by reference.
func (v *Viewer) RemoveTarget(t Target) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for i, target := range v.targets {
		if target == t {
			v.targets = append(v.targets[:i], v.targets[i+1:]...)
			return
		}
	}
}

// Start begins periodic updates to all targets.
func (v *Viewer) Start(ctx context.Context) error {
	v.mu.Lock()
	if v.cancel != nil {
		v.mu.Unlock()
		return fmt.Errorf("viewer already started")
	}

	ctx, v.cancel = context.WithCancel(ctx)
	v.mu.Unlock()

	// Initial update
	if err := v.Update(); err != nil {
		return err
	}

	go v.run(ctx)
	return nil
}

func (v *Viewer) run(ctx context.Context) {
	ticker := time.NewTicker(v.interval)
	defer ticker.Stop()
	defer close(v.done)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = v.Update() // Ignore errors in background loop
		}
	}
}

// Stop stops periodic updates.
func (v *Viewer) Stop() {
	v.mu.Lock()
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
	v.mu.Unlock()

	// Wait for run goroutine to finish
	<-v.done
}

// Update triggers an immediate update to all targets.
func (v *Viewer) Update() error {
	v.mu.RLock()
	provider := v.provider
	targets := make([]Target, len(v.targets))
	copy(targets, v.targets)
	v.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("no state provider set")
	}

	state, err := provider.GetViewState()
	if err != nil {
		return fmt.Errorf("failed to get view state: %w", err)
	}

	ctx := context.Background()
	var lastErr error
	for _, target := range targets {
		if err := target.Update(ctx, state); err != nil {
			lastErr = fmt.Errorf("target %s: %w", target.Name(), err)
		}
	}
	return lastErr
}

// Close stops the viewer and closes all targets.
func (v *Viewer) Close() error {
	v.mu.Lock()
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
	targets := v.targets
	v.targets = nil
	v.mu.Unlock()

	var lastErr error
	for _, target := range targets {
		if err := target.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
