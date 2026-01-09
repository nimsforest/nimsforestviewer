package nimsforestviewer

import (
	"context"
	"fmt"
	"image"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	sprites "github.com/nimsforest/nimsforestsprites"
	smarttv "github.com/nimsforest/nimsforestsmarttv"
)

// VideoTarget streams continuous video to Smart TVs.
// Uses nimsforestsprites for rendering and ffmpeg for encoding.
type VideoTarget struct {
	tv             *smarttv.TV
	tvRenderer     *smarttv.Renderer
	sprites        *sprites.Renderer
	spriteOpts     sprites.Options
	fps            int
	duration       time.Duration
	httpServer     *http.Server
	videoFile      string
	localIP        string
	port           int
	mu             sync.Mutex
	cancel         context.CancelFunc
	state          *ViewState
	stateProvider  StateProvider
}

// VideoOption configures a VideoTarget.
type VideoOption func(*VideoTarget)

// WithVideoFPS sets the video frame rate.
func WithVideoFPS(fps int) VideoOption {
	return func(t *VideoTarget) {
		t.fps = fps
	}
}

// WithVideoDuration sets the video duration.
func WithVideoDuration(d time.Duration) VideoOption {
	return func(t *VideoTarget) {
		t.duration = d
	}
}

// WithVideoSpriteOptions sets the sprite renderer options for video.
func WithVideoSpriteOptions(opts sprites.Options) VideoOption {
	return func(t *VideoTarget) {
		t.spriteOpts = opts
	}
}

// NewVideoTarget creates a target that streams video to a Smart TV.
func NewVideoTarget(tv *smarttv.TV, opts ...VideoOption) (*VideoTarget, error) {
	target := &VideoTarget{
		tv:  tv,
		fps: 10,
		duration: 60 * time.Second,
		port: 8889,
		spriteOpts: sprites.Options{
			Width:     1920,
			Height:    1080,
			FrameRate: 30,
			UseGPU:    false,
		},
	}

	for _, opt := range opts {
		opt(target)
	}

	// Create smarttv renderer
	renderer, err := smarttv.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("create smarttv renderer: %w", err)
	}
	target.tvRenderer = renderer

	// Create sprite renderer
	spriteRenderer, err := sprites.New(target.spriteOpts)
	if err != nil {
		renderer.Close()
		return nil, fmt.Errorf("create sprite renderer: %w", err)
	}
	target.sprites = spriteRenderer

	// Get local IP
	target.localIP = getLocalIP()

	return target, nil
}

// Name implements Target.
func (t *VideoTarget) Name() string {
	if t.tv != nil {
		return fmt.Sprintf("VideoTarget(%s)", t.tv.Name)
	}
	return "VideoTarget"
}

// SetStateProvider sets the state provider for continuous frame generation.
func (t *VideoTarget) SetStateProvider(p StateProvider) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stateProvider = p
}

// Update implements Target.
// For VideoTarget, this is a no-op during streaming - use Start() to begin streaming.
func (t *VideoTarget) Update(ctx context.Context, state *ViewState) error {
	t.mu.Lock()
	t.state = state
	t.mu.Unlock()
	return nil
}

// Start begins video streaming to the TV.
// This pre-renders a video file and streams it.
func (t *VideoTarget) Start(ctx context.Context) error {
	t.mu.Lock()
	state := t.state
	t.mu.Unlock()

	if state == nil {
		return fmt.Errorf("no state set - call Update first")
	}

	// Generate video file
	videoFile, err := t.generateVideo(ctx, state)
	if err != nil {
		return fmt.Errorf("generate video: %w", err)
	}
	t.videoFile = videoFile

	// Start HTTP server
	if err := t.startHTTPServer(ctx); err != nil {
		return fmt.Errorf("start HTTP server: %w", err)
	}

	// Send video URL to TV
	videoURL := fmt.Sprintf("http://%s:%d/stream.mp4", t.localIP, t.port)
	if err := t.tvRenderer.StreamVideo(ctx, t.tv, videoURL, "nimsforest"); err != nil {
		return fmt.Errorf("stream to TV: %w", err)
	}

	return nil
}

func (t *VideoTarget) generateVideo(ctx context.Context, state *ViewState) (string, error) {
	totalFrames := int(t.duration.Seconds()) * t.fps
	videoFile := fmt.Sprintf("/tmp/nimsforest_viewer_%d.mp4", time.Now().UnixNano())

	// Start ffmpeg encoder
	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", t.spriteOpts.Width, t.spriteOpts.Height),
		"-r", fmt.Sprintf("%d", t.fps),
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-profile:v", "baseline",
		"-level", "3.0",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		videoFile,
	)

	ffmpegIn, err := ffmpeg.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("create pipe: %w", err)
	}
	ffmpeg.Stderr = io.Discard

	if err := ffmpeg.Start(); err != nil {
		return "", fmt.Errorf("start ffmpeg: %w", err)
	}

	// Convert ViewState to sprites.State
	adapter := NewSpritesStateAdapter(state)

	// Render frames
	for i := 0; i < totalFrames; i++ {
		select {
		case <-ctx.Done():
			ffmpegIn.Close()
			ffmpeg.Wait()
			return "", ctx.Err()
		default:
		}

		frame := t.sprites.Render(adapter)
		if frame == nil {
			continue
		}

		rgba := ensureRGBA(frame)
		if _, err := ffmpegIn.Write(rgba.Pix); err != nil {
			break
		}
	}

	ffmpegIn.Close()
	if err := ffmpeg.Wait(); err != nil {
		return "", fmt.Errorf("ffmpeg encode: %w", err)
	}

	return videoFile, nil
}

func (t *VideoTarget) startHTTPServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream.mp4", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		http.ServeFile(w, r, t.videoFile)
	})

	t.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", t.port),
		Handler: mux,
	}

	go func() {
		t.httpServer.ListenAndServe()
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Close implements Target.
func (t *VideoTarget) Close() error {
	if t.httpServer != nil {
		t.httpServer.Shutdown(context.Background())
	}
	if t.sprites != nil {
		t.sprites.Close()
	}
	if t.tvRenderer != nil {
		t.tvRenderer.Close()
	}
	if t.videoFile != "" {
		os.Remove(t.videoFile)
	}
	return nil
}

// Stop stops video playback on the TV.
func (t *VideoTarget) Stop(ctx context.Context) error {
	return t.tvRenderer.Stop(ctx, t.tv)
}

// getLocalIP returns the local IP address.
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// ensureRGBA converts any image to RGBA.
func ensureRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba
}
