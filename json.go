package nimsforestviewer

import (
	"encoding/json"
	"math"
)

// WorldJSON is the JSON representation of ViewState for the web frontend.
type WorldJSON struct {
	Lands   []LandJSON  `json:"lands"`
	Summary SummaryJSON `json:"summary"`
}

// LandJSON is the JSON representation of a Land tile.
type LandJSON struct {
	ID           string        `json:"id"`
	Hostname     string        `json:"hostname"`
	RAMTotal     uint64        `json:"ram_total"`
	RAMAllocated uint64        `json:"ram_allocated"`
	CPUCores     int           `json:"cpu_cores,omitempty"`
	CPUFreqGHz   float64       `json:"cpu_freq_ghz,omitempty"`
	GPUVram      uint64        `json:"gpu_vram,omitempty"`
	GPUTflops    float64       `json:"gpu_tflops,omitempty"`
	Occupancy    float64       `json:"occupancy"`
	IsManaland   bool          `json:"is_manaland"`
	GridX        int           `json:"grid_x"`
	GridY        int           `json:"grid_y"`
	Trees        []ProcessJSON `json:"trees"`
	Treehouses   []ProcessJSON `json:"treehouses"`
	Nims         []ProcessJSON `json:"nims"`
}

// ProcessJSON is the JSON representation of a process.
type ProcessJSON struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RAMAllocated uint64   `json:"ram_allocated"`
	Type         string   `json:"type"`
	Progress     float64  `json:"progress,omitempty"`
	Subjects     []string `json:"subjects,omitempty"`
	ScriptPath   string   `json:"script_path,omitempty"`
	AIEnabled    bool     `json:"ai_enabled,omitempty"`
	Model        string   `json:"model,omitempty"`
}

// SummaryJSON is the JSON representation of the world summary.
type SummaryJSON struct {
	LandCount      int     `json:"land_count"`
	ManalandCount  int     `json:"manaland_count"`
	TreeCount      int     `json:"tree_count"`
	TreehouseCount int     `json:"treehouse_count"`
	NimCount       int     `json:"nim_count"`
	TotalRAM       uint64  `json:"total_ram"`
	RAMAllocated   uint64  `json:"ram_allocated"`
	Occupancy      float64 `json:"occupancy"`
}

// ViewStateToJSON converts a ViewState to WorldJSON for the web frontend.
func ViewStateToJSON(state *ViewState) WorldJSON {
	if state == nil {
		return WorldJSON{}
	}

	// Calculate grid positions if not already set
	gridSize := int(math.Ceil(math.Sqrt(float64(len(state.Lands)))))
	if gridSize < 1 {
		gridSize = 1
	}

	landsJSON := make([]LandJSON, len(state.Lands))
	for i, land := range state.Lands {
		// Use existing grid positions if set, otherwise calculate
		gridX, gridY := land.GridX, land.GridY
		if gridX == 0 && gridY == 0 && i > 0 {
			gridX = i % gridSize
			gridY = i / gridSize
		}

		landsJSON[i] = LandJSON{
			ID:           land.ID,
			Hostname:     land.Hostname,
			RAMTotal:     land.RAMTotal,
			RAMAllocated: land.RAMAllocated,
			Occupancy:    land.Occupancy,
			IsManaland:   land.IsManaland,
			GridX:        gridX,
			GridY:        gridY,
			Trees:        processViewsToJSON(land.Trees, "tree"),
			Treehouses:   processViewsToJSON(land.Treehouses, "treehouse"),
			Nims:         processViewsToJSON(land.Nims, "nim"),
		}
	}

	return WorldJSON{
		Lands: landsJSON,
		Summary: SummaryJSON{
			LandCount:      state.Summary.TotalLands,
			ManalandCount:  state.Summary.TotalManalands,
			TreeCount:      state.Summary.TotalTrees,
			TreehouseCount: state.Summary.TotalTreehouses,
			NimCount:       state.Summary.TotalNims,
			TotalRAM:       state.Summary.TotalRAM,
			RAMAllocated:   state.Summary.AllocatedRAM,
			Occupancy:      calculateOccupancy(state.Summary.AllocatedRAM, state.Summary.TotalRAM),
		},
	}
}

func processViewsToJSON(processes []ProcessView, procType string) []ProcessJSON {
	result := make([]ProcessJSON, len(processes))
	for i, p := range processes {
		result[i] = ProcessJSON{
			ID:           p.ID,
			Name:         p.Name,
			RAMAllocated: p.RAMAllocated,
			Type:         procType,
			Progress:     p.Progress,
		}
	}
	return result
}

func calculateOccupancy(allocated, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(allocated) / float64(total)
}

// ViewStateToJSONBytes converts a ViewState to JSON bytes.
func ViewStateToJSONBytes(state *ViewState) ([]byte, error) {
	worldJSON := ViewStateToJSON(state)
	return json.Marshal(worldJSON)
}
