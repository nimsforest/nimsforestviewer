package nimsforestviewer

import "context"

// Target represents a visualization output destination.
type Target interface {
	// Update sends new state to the target.
	Update(ctx context.Context, state *ViewState) error

	// Close cleans up the target.
	Close() error

	// Name returns a descriptive name for logging.
	Name() string
}
