// Package nimsforestviewer provides a unified visualization viewer for Smart TVs and web browsers.
package nimsforestviewer

// ViewState represents the complete visualization state.
type ViewState struct {
	Lands   []LandView
	Summary SummaryView
}

// LandView represents a single land/node in the visualization.
type LandView struct {
	ID           string
	Hostname     string
	GridX, GridY int
	IsManaland   bool
	Occupancy    float64
	RAMTotal     uint64
	RAMAllocated uint64
	Trees        []ProcessView
	Treehouses   []ProcessView
	Nims         []ProcessView
}

// AllProcesses returns all processes on this land.
func (l *LandView) AllProcesses() []ProcessView {
	result := make([]ProcessView, 0, len(l.Trees)+len(l.Treehouses)+len(l.Nims))
	result = append(result, l.Trees...)
	result = append(result, l.Treehouses...)
	result = append(result, l.Nims...)
	return result
}

// ProcessView represents a process running on a land.
type ProcessView struct {
	ID           string
	Name         string
	Type         string // "tree", "treehouse", "nim"
	RAMAllocated uint64
	Progress     float64
}

// SummaryView contains aggregate statistics.
type SummaryView struct {
	TotalLands      int
	TotalManalands  int
	TotalTrees      int
	TotalTreehouses int
	TotalNims       int
	TotalRAM        uint64
	AllocatedRAM    uint64
}
