// Example: Smart TV viewer
//
// This example demonstrates using nimsforestviewer to display on a Smart TV.
// It discovers TVs on the network and displays a visualization.
//
// Run with: go run main.go
// Flags:
//   -demo    Simulate state changes every 30 seconds
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	viewer "github.com/nimsforest/nimsforestviewer"
	smarttv "github.com/nimsforest/nimsforestsmarttv"
)

func main() {
	demo := flag.Bool("demo", false, "Demo mode: simulate state changes")
	flag.Parse()

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

	fmt.Println("=== nimsforestviewer - Smart TV Example ===")
	fmt.Println()

	// Discover TVs
	fmt.Println("Discovering Smart TVs...")
	tvs, err := smarttv.Discover(ctx, 5*time.Second)
	if err != nil || len(tvs) == 0 {
		fmt.Println("No TVs found on the network")
		return
	}
	tv := &tvs[0]
	fmt.Printf("Found: %s\n\n", tv.String())

	// Create mock state
	state := createMockState()

	// Create viewer with event-driven updates (no interval - only on explicit Update)
	v := viewer.New()
	v.SetStateProvider(viewer.NewStaticStateProvider(state))

	// Add Smart TV target
	tvTarget, err := viewer.NewSmartTVTarget(tv,
		viewer.WithJFIF(true), // Use JFIF for better TV compatibility
	)
	if err != nil {
		fmt.Printf("Error creating TV target: %v\n", err)
		return
	}
	v.AddTarget(tvTarget)

	// Display initial state
	fmt.Println("Displaying visualization on TV...")
	if err := v.Update(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Image displayed!")

	if *demo {
		fmt.Println("\nDemo mode: updating every 30 seconds")
		fmt.Println("Press Ctrl+C to stop")

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		updateCount := 1
		for {
			select {
			case <-ctx.Done():
				goto cleanup
			case <-ticker.C:
				updateCount++
				randomizeState(state)
				v.SetStateProvider(viewer.NewStaticStateProvider(state))
				if err := v.Update(); err != nil {
					fmt.Printf("Update %d error: %v\n", updateCount, err)
				} else {
					fmt.Printf("Update %d: displayed at %s\n", updateCount, time.Now().Format("15:04:05"))
				}
			}
		}
	} else {
		fmt.Println("\nImage will stay displayed until changed.")
		fmt.Println("Run with -demo flag to simulate state changes.")
		fmt.Println("Press Ctrl+C to stop")
		<-ctx.Done()
	}

cleanup:
	fmt.Println("\nStopping TV playback...")
	tvTarget.Stop(context.Background())
	v.Close()
	fmt.Println("Done")
}

func createMockState() *viewer.ViewState {
	return &viewer.ViewState{
		Lands: []viewer.LandView{
			{
				ID: "land-1", Hostname: "node-alpha",
				GridX: 0, GridY: 0,
				IsManaland: false, Occupancy: 0.6,
				RAMTotal: 16e9, RAMAllocated: 10e9,
				Trees: []viewer.ProcessView{
					{ID: "tree-1", Name: "data-parser", Type: "tree", Progress: 0.8},
				},
				Nims: []viewer.ProcessView{
					{ID: "nim-1", Name: "ai-handler", Type: "nim", Progress: 0.5},
				},
			},
			{
				ID: "land-2", Hostname: "node-beta",
				GridX: 1, GridY: 0,
				IsManaland: true, Occupancy: 0.3,
				RAMTotal: 32e9, RAMAllocated: 10e9,
				Trees: []viewer.ProcessView{
					{ID: "tree-2", Name: "gpu-worker", Type: "tree", Progress: 0.9},
				},
			},
			{
				ID: "land-3", Hostname: "node-gamma",
				GridX: 0, GridY: 1,
				IsManaland: false, Occupancy: 0.4,
				RAMTotal: 8e9, RAMAllocated: 3e9,
				Treehouses: []viewer.ProcessView{
					{ID: "th-1", Name: "lua-script", Type: "treehouse", Progress: 1.0},
				},
			},
			{
				ID: "land-4", Hostname: "node-delta",
				GridX: 1, GridY: 1,
				IsManaland: false, Occupancy: 0.2,
				RAMTotal: 16e9, RAMAllocated: 3e9,
			},
		},
		Summary: viewer.SummaryView{
			TotalLands: 4, TotalManalands: 1,
			TotalTrees: 2, TotalTreehouses: 1, TotalNims: 1,
			TotalRAM: 72e9, AllocatedRAM: 26e9,
		},
	}
}

func randomizeState(state *viewer.ViewState) {
	// Simulate some changes
	for i := range state.Lands {
		for j := range state.Lands[i].Trees {
			state.Lands[i].Trees[j].Progress += 0.1
			if state.Lands[i].Trees[j].Progress > 1.0 {
				state.Lands[i].Trees[j].Progress = 0.0
			}
		}
		for j := range state.Lands[i].Nims {
			state.Lands[i].Nims[j].Progress += 0.15
			if state.Lands[i].Nims[j].Progress > 1.0 {
				state.Lands[i].Nims[j].Progress = 0.0
			}
		}
	}
}
