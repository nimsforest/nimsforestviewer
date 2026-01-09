package nimsforestviewer

import (
	sprites "github.com/nimsforest/nimsforestsprites"
)

// SpritesStateAdapter adapts ViewState to sprites.State interface.
type SpritesStateAdapter struct {
	viewState *ViewState
}

// NewSpritesStateAdapter creates an adapter for sprites rendering.
func NewSpritesStateAdapter(state *ViewState) *SpritesStateAdapter {
	return &SpritesStateAdapter{viewState: state}
}

// Lands implements sprites.State.
func (a *SpritesStateAdapter) Lands() []sprites.Land {
	if a.viewState == nil {
		return nil
	}

	result := make([]sprites.Land, len(a.viewState.Lands))
	for i, land := range a.viewState.Lands {
		landType := "normal"
		if land.IsManaland {
			landType = "mana"
		}
		result[i] = sprites.Land{
			ID:   land.ID,
			Name: land.Hostname,
			X:    float64(land.GridX),
			Y:    float64(land.GridY),
			Type: landType,
		}
	}
	return result
}

// Processes implements sprites.State.
func (a *SpritesStateAdapter) Processes() []sprites.Process {
	if a.viewState == nil {
		return nil
	}

	var result []sprites.Process
	for _, land := range a.viewState.Lands {
		// Add trees
		for _, proc := range land.Trees {
			result = append(result, sprites.Process{
				ID:       proc.ID,
				LandID:   land.ID,
				Type:     "tree",
				Progress: proc.Progress,
				X:        float64(land.GridX),
				Y:        float64(land.GridY),
			})
		}
		// Add treehouses
		for _, proc := range land.Treehouses {
			result = append(result, sprites.Process{
				ID:       proc.ID,
				LandID:   land.ID,
				Type:     "treehouse",
				Progress: proc.Progress,
				X:        float64(land.GridX),
				Y:        float64(land.GridY),
			})
		}
		// Add nims
		for _, proc := range land.Nims {
			result = append(result, sprites.Process{
				ID:       proc.ID,
				LandID:   land.ID,
				Type:     "nim",
				Progress: proc.Progress,
				X:        float64(land.GridX),
				Y:        float64(land.GridY),
			})
		}
	}
	return result
}

// Ensure SpritesStateAdapter implements sprites.State
var _ sprites.State = (*SpritesStateAdapter)(nil)
