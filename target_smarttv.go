package nimsforestviewer

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"time"

	sprites "github.com/nimsforest/nimsforestsprites"
	smarttv "github.com/nimsforest/nimsforestsmarttv"
)

// SmartTVTarget displays static images on Smart TVs via DLNA.
// Uses nimsforestsprites for passive rendering and nimsforestsmarttv for transport.
type SmartTVTarget struct {
	tv             *smarttv.TV
	renderer       *smarttv.Renderer
	sprites        *sprites.Renderer
	useJFIF        bool // Convert to JFIF format for better TV compatibility
	spriteOpts     sprites.Options
	lastImageBytes []byte // Cache to avoid redundant updates
}

// TVOption configures a SmartTVTarget.
type TVOption func(*SmartTVTarget)

// WithJFIF enables JFIF conversion for better TV compatibility.
// Requires ffmpeg and imagemagick to be installed.
func WithJFIF(enable bool) TVOption {
	return func(t *SmartTVTarget) {
		t.useJFIF = enable
	}
}

// WithSpriteOptions sets the sprite renderer options.
func WithSpriteOptions(opts sprites.Options) TVOption {
	return func(t *SmartTVTarget) {
		t.spriteOpts = opts
	}
}

// NewSmartTVTarget creates a target that displays images on a Smart TV.
func NewSmartTVTarget(tv *smarttv.TV, opts ...TVOption) (*SmartTVTarget, error) {
	target := &SmartTVTarget{
		tv:      tv,
		useJFIF: true, // Default to JFIF for better compatibility
		spriteOpts: sprites.Options{
			Width:     1920,
			Height:    1080,
			FrameRate: 30,
			UseGPU:    false, // Use software rendering for headless
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
	target.renderer = renderer

	// Create sprite renderer
	spriteRenderer, err := sprites.New(target.spriteOpts)
	if err != nil {
		renderer.Close()
		return nil, fmt.Errorf("create sprite renderer: %w", err)
	}
	target.sprites = spriteRenderer

	return target, nil
}

// Name implements Target.
func (t *SmartTVTarget) Name() string {
	if t.tv != nil {
		return fmt.Sprintf("SmartTV(%s)", t.tv.Name)
	}
	return "SmartTV"
}

// Update implements Target.
func (t *SmartTVTarget) Update(ctx context.Context, state *ViewState) error {
	// Convert ViewState to sprites.State
	adapter := NewSpritesStateAdapter(state)

	// Render frame
	frame := t.sprites.Render(adapter)
	if frame == nil {
		return fmt.Errorf("failed to render frame")
	}

	// Convert to JPEG
	var jpegData []byte
	var err error
	if t.useJFIF {
		jpegData, err = convertToJFIF(frame)
	} else {
		jpegData, err = encodeJPEG(frame)
	}
	if err != nil {
		return fmt.Errorf("convert to JPEG: %w", err)
	}

	// Skip if image hasn't changed
	if bytes.Equal(jpegData, t.lastImageBytes) {
		return nil
	}
	t.lastImageBytes = jpegData

	// Display on TV
	if err := t.renderer.DisplayImageJPEG(ctx, t.tv, jpegData); err != nil {
		return fmt.Errorf("display on TV: %w", err)
	}

	return nil
}

// Close implements Target.
func (t *SmartTVTarget) Close() error {
	if t.sprites != nil {
		t.sprites.Close()
	}
	if t.renderer != nil {
		t.renderer.Close()
	}
	return nil
}

// Stop stops playback on the TV.
func (t *SmartTVTarget) Stop(ctx context.Context) error {
	return t.renderer.Stop(ctx, t.tv)
}

// convertToJFIF converts an image to JFIF-compliant JPEG using ffmpeg + magick.
// This produces JPEG files that are compatible with more TVs (especially JVC).
func convertToJFIF(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}

	tmpFile := fmt.Sprintf("/tmp/viewer_%d.jpg", time.Now().UnixNano())
	jfifFile := fmt.Sprintf("/tmp/viewer_%d_jfif.jpg", time.Now().UnixNano())
	defer os.Remove(tmpFile)
	defer os.Remove(jfifFile)

	cmd := exec.Command("ffmpeg",
		"-y", "-loglevel", "error",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", width, height),
		"-i", "pipe:0",
		"-vframes", "1",
		"-pix_fmt", "yuvj420p",
		"-q:v", "2",
		tmpFile,
	)
	cmd.Stdin = bytes.NewReader(rgba.Pix)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w", err)
	}

	cmd2 := exec.Command("magick", tmpFile, jfifFile)
	if err := cmd2.Run(); err != nil {
		// Fallback to ffmpeg output if magick not available
		return os.ReadFile(tmpFile)
	}

	return os.ReadFile(jfifFile)
}

// encodeJPEG encodes an image as standard JPEG (may not work on all TVs).
func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
