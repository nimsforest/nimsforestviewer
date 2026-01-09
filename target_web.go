package nimsforestviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// WebTarget serves the visualization via HTTP for web browsers.
// It provides a JSON API at /api/viewmodel and can serve static assets.
type WebTarget struct {
	addr     string
	server   *http.Server
	state    *ViewState
	mu       sync.RWMutex
	webDir   string // Optional directory with static web assets
	started  bool
}

// WebOption configures a WebTarget.
type WebOption func(*WebTarget)

// WithWebDir sets the directory containing static web assets.
func WithWebDir(dir string) WebOption {
	return func(t *WebTarget) {
		t.webDir = dir
	}
}

// NewWebTarget creates a target that serves the visualization via HTTP.
func NewWebTarget(addr string, opts ...WebOption) (*WebTarget, error) {
	target := &WebTarget{
		addr: addr,
	}

	for _, opt := range opts {
		opt(target)
	}

	return target, nil
}

// Name implements Target.
func (t *WebTarget) Name() string {
	return fmt.Sprintf("WebTarget(%s)", t.addr)
}

// Update implements Target.
func (t *WebTarget) Update(ctx context.Context, state *ViewState) error {
	t.mu.Lock()
	t.state = state
	wasStarted := t.started
	t.mu.Unlock()

	// Auto-start server on first update
	if !wasStarted {
		return t.start()
	}
	return nil
}

// Handler returns the HTTP handler for embedding in existing servers.
func (t *WebTarget) Handler() http.Handler {
	mux := http.NewServeMux()

	// API endpoint
	mux.HandleFunc("/api/viewmodel", t.handleViewmodel)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Static files
	if t.webDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(t.webDir)))
	} else {
		// Serve a simple status page if no web assets
		mux.HandleFunc("/", t.handleIndex)
	}

	return mux
}

func (t *WebTarget) handleViewmodel(w http.ResponseWriter, r *http.Request) {
	t.mu.RLock()
	state := t.state
	t.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if state == nil {
		json.NewEncoder(w).Encode(WorldJSON{})
		return
	}

	worldJSON := ViewStateToJSON(state)
	json.NewEncoder(w).Encode(worldJSON)
}

func (t *WebTarget) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	t.mu.RLock()
	state := t.state
	t.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html")

	landCount := 0
	if state != nil {
		landCount = len(state.Lands)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>nimsforestviewer</title>
    <style>
        body { font-family: system-ui; background: #1a1a2e; color: #eee; padding: 2rem; }
        h1 { color: #4ade80; }
        .info { background: #16213e; padding: 1rem; border-radius: 8px; margin: 1rem 0; }
        a { color: #60a5fa; }
    </style>
</head>
<body>
    <h1>nimsforestviewer</h1>
    <div class="info">
        <p><strong>Status:</strong> Running</p>
        <p><strong>Lands:</strong> %d</p>
        <p><strong>API:</strong> <a href="/api/viewmodel">/api/viewmodel</a></p>
    </div>
    <p>For the full interactive visualization, configure WebTarget with a web assets directory.</p>
</body>
</html>`, landCount)

	w.Write([]byte(html))
}

func (t *WebTarget) start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return nil
	}

	t.server = &http.Server{
		Addr:    t.addr,
		Handler: t.Handler(),
	}

	go func() {
		t.server.ListenAndServe()
	}()

	t.started = true
	return nil
}

// Close implements Target.
func (t *WebTarget) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.server != nil {
		return t.server.Shutdown(context.Background())
	}
	return nil
}

// URL returns the URL where the web target is serving.
func (t *WebTarget) URL() string {
	return "http://localhost" + t.addr
}
