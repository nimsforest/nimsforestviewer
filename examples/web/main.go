// Example: Web-only viewer
//
// This example demonstrates using nimsforestviewer to serve a web visualization.
// Run with: go run main.go
// Then open http://localhost:8080 in your browser.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	viewer "github.com/nimsforest/nimsforestviewer"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Create mock state
	state := createMockState()

	// Create viewer
	v := viewer.New(viewer.WithInterval(5 * time.Second))
	v.SetStateProvider(viewer.NewStaticStateProvider(state))

	// Add web target
	webTarget, err := viewer.NewWebTarget(":8080")
	if err != nil {
		fmt.Printf("Error creating web target: %v\n", err)
		return
	}
	v.AddTarget(webTarget)

	// Start viewer
	fmt.Println("Starting nimsforestviewer...")
	fmt.Println("Open http://localhost:8080 in your browser")
	fmt.Println("Press Ctrl+C to stop")

	if err := v.Start(ctx); err != nil {
		fmt.Printf("Error starting viewer: %v\n", err)
		return
	}

	// Wait for shutdown
	<-ctx.Done()

	// Cleanup
	v.Close()
	fmt.Println("Done")
}

func createMockState() *viewer.ViewState {
	return &viewer.ViewState{
		Lands: []viewer.LandView{
			{
				ID:           "land-1",
				Hostname:     "node-alpha",
				GridX:        0,
				GridY:        0,
				IsManaland:   false,
				Occupancy:    0.6,
				RAMTotal:     16 * 1024 * 1024 * 1024,
				RAMAllocated: 10 * 1024 * 1024 * 1024,
				Trees: []viewer.ProcessView{
					{ID: "tree-1", Name: "data-parser", Type: "tree", Progress: 0.8},
				},
				Nims: []viewer.ProcessView{
					{ID: "nim-1", Name: "ai-handler", Type: "nim", Progress: 0.5},
				},
			},
			{
				ID:           "land-2",
				Hostname:     "node-beta",
				GridX:        1,
				GridY:        0,
				IsManaland:   true,
				Occupancy:    0.3,
				RAMTotal:     32 * 1024 * 1024 * 1024,
				RAMAllocated: 10 * 1024 * 1024 * 1024,
				Trees: []viewer.ProcessView{
					{ID: "tree-2", Name: "gpu-worker", Type: "tree", Progress: 0.9},
				},
			},
			{
				ID:           "land-3",
				Hostname:     "node-gamma",
				GridX:        0,
				GridY:        1,
				IsManaland:   false,
				Occupancy:    0.4,
				RAMTotal:     8 * 1024 * 1024 * 1024,
				RAMAllocated: 3 * 1024 * 1024 * 1024,
				Treehouses: []viewer.ProcessView{
					{ID: "th-1", Name: "lua-script", Type: "treehouse", Progress: 1.0},
				},
			},
		},
		Summary: viewer.SummaryView{
			TotalLands:      3,
			TotalManalands:  1,
			TotalTrees:      2,
			TotalTreehouses: 1,
			TotalNims:       1,
			TotalRAM:        56 * 1024 * 1024 * 1024,
			AllocatedRAM:    23 * 1024 * 1024 * 1024,
		},
	}
}
