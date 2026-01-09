// Example: Multi-target viewer
//
// This example demonstrates using nimsforestviewer with multiple targets
// simultaneously - both a web server (JSON API) and a Smart TV.
//
// Run with: go run main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	viewer "github.com/nimsforest/nimsforestviewer"
	smarttv "github.com/nimsforest/nimsforestsmarttv"
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

	fmt.Println("=== nimsforestviewer - Multi-Target Example ===")
	fmt.Println()

	// Create viewer
	v := viewer.New(viewer.WithInterval(30 * time.Second))

	// Create mock state provider that updates over time
	stateProvider := &mockStateProvider{state: createMockState()}
	v.SetStateProvider(stateProvider)

	// Add Web target (always available)
	webTarget, err := viewer.NewWebTarget(":8080")
	if err != nil {
		fmt.Printf("Error creating web target: %v\n", err)
		return
	}
	v.AddTarget(webTarget)
	fmt.Println("Web target: http://localhost:8080")
	fmt.Println("  - API: http://localhost:8080/api/viewmodel")

	// Try to add Smart TV target
	fmt.Println("\nDiscovering Smart TVs...")
	tvs, err := smarttv.Discover(ctx, 5*time.Second)
	if err == nil && len(tvs) > 0 {
		tv := &tvs[0]
		fmt.Printf("Found TV: %s\n", tv.String())

		tvTarget, err := viewer.NewSmartTVTarget(tv, viewer.WithJFIF(true))
		if err != nil {
			fmt.Printf("Warning: could not create TV target: %v\n", err)
		} else {
			v.AddTarget(tvTarget)
			fmt.Println("Smart TV target added!")
		}
	} else {
		fmt.Println("No TVs found - running with web target only")
	}

	// Start viewer with periodic updates
	fmt.Println("\nStarting viewer...")
	fmt.Println("Updates every 30 seconds")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	if err := v.Start(ctx); err != nil {
		fmt.Printf("Error starting viewer: %v\n", err)
		return
	}

	// Simulate state changes in background
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stateProvider.randomize()
				fmt.Printf("[%s] State updated\n", time.Now().Format("15:04:05"))
			}
		}
	}()

	// Wait for shutdown
	<-ctx.Done()

	// Cleanup
	v.Close()
	fmt.Println("Done")
}

type mockStateProvider struct {
	state *viewer.ViewState
}

func (p *mockStateProvider) GetViewState() (*viewer.ViewState, error) {
	return p.state, nil
}

func (p *mockStateProvider) randomize() {
	for i := range p.state.Lands {
		for j := range p.state.Lands[i].Trees {
			p.state.Lands[i].Trees[j].Progress += 0.1
			if p.state.Lands[i].Trees[j].Progress > 1.0 {
				p.state.Lands[i].Trees[j].Progress = 0.0
			}
		}
		for j := range p.state.Lands[i].Nims {
			p.state.Lands[i].Nims[j].Progress += 0.15
			if p.state.Lands[i].Nims[j].Progress > 1.0 {
				p.state.Lands[i].Nims[j].Progress = 0.0
			}
		}
		// Update occupancy slightly
		p.state.Lands[i].Occupancy += 0.05
		if p.state.Lands[i].Occupancy > 1.0 {
			p.state.Lands[i].Occupancy = 0.2
		}
	}
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
