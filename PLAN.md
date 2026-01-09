# Plan: Unified nimsforestviewer Library

## Overview

Create a unified `nimsforestviewer` Go library that outputs visualizations to both Smart TVs and web browsers. This factors the Phaser game engine out of nimsforest2 while providing a single library for all visualization targets.

## Architecture

```
                    ┌─────────────────────────────────┐
                    │  nimsforest2 (or other caller)  │
                    │  - Provides StateProvider       │
                    └───────────────┬─────────────────┘
                                    │ ViewState
                                    ▼
┌───────────────────────────────────────────────────────────────────────┐
│                         nimsforestviewer                              │
│  ┌─────────────┐                                                      │
│  │   Viewer    │──────────────────────────────────────────────┐       │
│  └─────────────┘                                              │       │
│         │                                                     │       │
│         │ AddTarget()                                         │       │
│         ▼                                                     ▼       │
│  ┌─────────────────┬─────────────────┬─────────────────┐             │
│  │   WebTarget     │  SmartTVTarget  │   VideoTarget   │             │
│  │  (HTTP+Phaser)  │ (static images) │  (live stream)  │             │
│  └────────┬────────┴────────┬────────┴────────┬────────┘             │
│           │                 │                 │                       │
│           │          ┌──────┴──────┐   ┌──────┴──────┐               │
│           │          │ sprites     │   │ sprites     │               │
│           │          │ adapter     │   │ + encoder   │               │
│           │          └──────┬──────┘   └──────┬──────┘               │
└───────────┼─────────────────┼─────────────────┼───────────────────────┘
            │                 │                 │
            ▼                 ▼                 ▼
      ┌──────────┐    ┌─────────────┐   ┌─────────────┐
      │  Browser │    │  Smart TV   │   │  Smart TV   │
      │ (Phaser) │    │  (DLNA)     │   │  (DLNA MP4) │
      └──────────┘    └─────────────┘   └─────────────┘
```

## Package Structure

```
nimsforestviewer/
├── go.mod
├── viewer.go           # Main Viewer struct and API
├── state.go            # ViewState, LandView, ProcessView types
├── state_provider.go   # StateProvider interface
├── target.go           # Target interface
├── target_web.go       # WebTarget (HTTP server + embedded Phaser)
├── target_smarttv.go   # SmartTVTarget (static images via DLNA)
├── target_video.go     # VideoTarget (live video streaming)
├── adapter_sprites.go  # ViewState -> sprites.State adapter
├── json.go             # ViewState -> JSON for web frontend
├── web/                # Phaser frontend (moved from nimsforest2)
│   ├── embed.go
│   ├── package.json
│   ├── app/components/game/
│   │   ├── ForestScene.ts
│   │   ├── PhaserGame.tsx
│   │   └── types.ts
│   └── out/            # Built assets (embedded)
└── examples/
    ├── web/main.go
    ├── smarttv/main.go
    └── multi/main.go
```

## Core API

```go
package nimsforestviewer

// Viewer manages visualization output to multiple targets
type Viewer struct { ... }

func New(opts ...Option) *Viewer
func (v *Viewer) SetStateProvider(p StateProvider)
func (v *Viewer) AddTarget(t Target) error
func (v *Viewer) Start(ctx context.Context) error
func (v *Viewer) Update() error  // Trigger immediate update
func (v *Viewer) Close() error

// StateProvider provides current state
type StateProvider interface {
    GetViewState() (*ViewState, error)
}

// Target represents an output destination
type Target interface {
    Update(ctx context.Context, state *ViewState) error
    Close() error
    Name() string
}

// Targets
func NewWebTarget(addr string, opts ...WebOption) (*WebTarget, error)
func NewSmartTVTarget(tv *smarttv.TV, opts ...TVOption) (*SmartTVTarget, error)
func NewVideoTarget(tv *smarttv.TV, opts ...VideoOption) (*VideoTarget, error)
```

## ViewState (Unified State Model)

```go
type ViewState struct {
    Lands   []LandView
    Summary SummaryView
}

type LandView struct {
    ID            string
    Hostname      string
    GridX, GridY  int
    IsManaland    bool
    Occupancy     float64
    RAMTotal      uint64
    RAMAllocated  uint64
    Trees         []ProcessView
    Treehouses    []ProcessView
    Nims          []ProcessView
}

type ProcessView struct {
    ID           string
    Name         string
    Type         string  // "tree", "treehouse", "nim"
    RAMAllocated uint64
    Progress     float64
}
```

## Rendering Modes

The library supports two fundamentally different rendering approaches:

| Mode | Renderer | Use Case | Features |
|------|----------|----------|----------|
| **Passive** | nimsforestsprites (Go) | Smart TV | Static images, video frames |
| **Interactive** | Phaser 3 (JavaScript) | Web browser | Pan, zoom, click-to-select |

Both modes consume the same `ViewState` but render differently:
- Passive targets (SmartTVTarget, VideoTarget) use nimsforestsprites internally
- Interactive target (WebTarget) serves the Phaser app which renders in-browser

## Dependencies

```
nimsforestviewer
├── imports github.com/nimsforest/nimsforestsprites  (passive rendering)
├── imports github.com/nimsforest/nimsforestsmarttv  (DLNA transport)
└── embeds web/ (Phaser interactive frontend)
```

## Implementation Steps

### Step 1: Initialize Repository
- Create repo `nimsforest/nimsforestviewer` via `gh repo create`
- Initialize Go module
- Add LICENSE (MIT) and basic README

### Step 2: Core Types (`state.go`, `state_provider.go`, `target.go`)
- Define `ViewState`, `LandView`, `ProcessView`, `SummaryView`
- Define `StateProvider` interface
- Define `Target` interface

### Step 3: Viewer Core (`viewer.go`)
- Implement `Viewer` struct with target management
- Implement `Start()` for periodic updates
- Implement `Update()` for immediate updates

### Step 4: Web Target (`target_web.go`)
- Move web/ from nimsforest2 to nimsforestviewer
- Implement HTTP server with `/api/viewmodel` endpoint
- Embed Phaser frontend assets
- Implement `WebTarget.Update()` to store latest state

### Step 5: Sprites Adapter (`adapter_sprites.go`)
- Implement `ViewState` -> `sprites.State` conversion
- Map LandView to sprites.Land
- Map ProcessView to sprites.Process

### Step 6: Smart TV Target (`target_smarttv.go`)
- Create sprites renderer internally
- Use adapter to convert ViewState -> sprites.State
- Render to image, convert to JFIF, send via nimsforestsmarttv

### Step 7: Video Target (`target_video.go`)
- Create sprites renderer
- Pipe frames to ffmpeg encoder
- Stream MP4 to TV via nimsforestsmarttv

### Step 8: JSON Conversion (`json.go`)
- Port JSON conversion logic from nimsforest2/webview/json.go
- `ViewStateToJSON()` for web API

### Step 9: Examples
- `examples/web/` - Web-only viewer
- `examples/smarttv/` - TV-only viewer
- `examples/multi/` - Multiple targets simultaneously

## nimsforest2 Migration

After nimsforestviewer is complete:

1. **Add dependency:**
   ```bash
   go get github.com/nimsforest/nimsforestviewer
   ```

2. **Create adapter** in nimsforest2:
   ```go
   // internal/viewer/adapter.go
   type ViewModelAdapter struct {
       vm *viewmodel.ViewModel
   }

   func (a *ViewModelAdapter) GetViewState() (*nimsforestviewer.ViewState, error) {
       a.vm.Refresh()
       return convertWorldToViewState(a.vm.GetWorld()), nil
   }
   ```

3. **Replace webview usage:**
   ```go
   viewer := nimsforestviewer.New()
   viewer.SetStateProvider(NewViewModelAdapter(vm))
   viewer.AddTarget(nimsforestviewer.NewWebTarget(":8080"))
   viewer.Start(ctx)
   ```

4. **Delete** `internal/webview/` and `web/` from nimsforest2

## Files to Move/Delete

| From | To | Action |
|------|-----|--------|
| `nimsforest2/web/` | `nimsforestviewer/web/` | Move |
| `nimsforest2/internal/webview/json.go` | `nimsforestviewer/json.go` | Port logic |
| `nimsforest2/internal/webview/server.go` | `nimsforestviewer/target_web.go` | Port logic |
| `nimsforest2/internal/webview/` | - | Delete after migration |

## Verification

1. **Web Target:**
   - Run `examples/web/main.go`
   - Open browser to `http://localhost:8080`
   - Verify Phaser game renders with isometric view
   - Verify pan, zoom, click-to-select work

2. **Smart TV Target:**
   - Run `examples/smarttv/main.go`
   - Verify TV discovers and displays static image
   - Verify updates only happen on state change

3. **Video Target:**
   - Run `examples/multi/main.go` with video target
   - Verify smooth video streaming to TV

4. **Multi-target:**
   - Run with both web and TV targets
   - Verify both update simultaneously
