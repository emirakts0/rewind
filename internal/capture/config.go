package capture

import (
	"fmt"
	"log/slog"

	"rewind/internal/hardware"
)

// Config holds capture configuration
type Config struct {
	// Selected display index
	DisplayIndex int

	// Selected encoder name (e.g., "h264_nvenc", "h264_amf", "libx264")
	// If empty, will use CPU encoding (libx264)
	EncoderName string

	// Capture settings
	FPS           int
	Bitrate       string
	RecordSeconds int
	OutputDir     string
	FFmpegPath    string
	DrawMouse     bool

	// Resolved references (set by Resolve)
	display *hardware.Display
	encoder *hardware.Encoder
	gpu     *hardware.GPU
}

// DefaultConfig returns default capture configuration
func DefaultConfig() *Config {
	return &Config{
		DisplayIndex:  0,
		EncoderName:   "", // empty = auto (CPU fallback)
		FPS:           60,
		Bitrate:       "15M",
		RecordSeconds: 30,
		OutputDir:     "./clips",
		FFmpegPath:    "bin/ffmpeg.exe",
		DrawMouse:     true,
	}
}

// Resolve validates and resolves the configuration against detected hardware
func (c *Config) Resolve(sysInfo *hardware.SystemInfo) error {
	slog.Debug("resolving capture config",
		"displayIndex", c.DisplayIndex,
		"encoderName", c.EncoderName,
	)

	// Resolve display
	c.display = sysInfo.GetDisplay(c.DisplayIndex)
	if c.display == nil {
		return fmt.Errorf("display not found: index %d", c.DisplayIndex)
	}

	slog.Info("display resolved",
		"requestedIndex", c.DisplayIndex,
		"resolvedIndex", c.display.Index,
		"resolution", fmt.Sprintf("%dx%d", c.display.Width, c.display.Height),
		"name", c.display.Name,
	)

	// Resolve encoder
	if c.EncoderName == "" {
		// Default to CPU encoder
		c.encoder = sysInfo.GetEncoder("libx264")
	} else {
		c.encoder = sysInfo.GetEncoder(c.EncoderName)
		if c.encoder == nil {
			return fmt.Errorf("encoder not found: %s", c.EncoderName)
		}
		if !c.encoder.Available {
			return fmt.Errorf("encoder not available: %s", c.EncoderName)
		}
	}

	// Resolve GPU for encoder
	if c.encoder != nil && c.encoder.GPUIndex >= 0 {
		c.gpu = sysInfo.GPUs.FindByIndex(c.encoder.GPUIndex)
	}

	return c.Validate()
}

func (c *Config) Validate() error {
	if c.display == nil {
		return fmt.Errorf("display is required")
	}
	if c.FPS <= 0 || c.FPS > 240 {
		return fmt.Errorf("FPS must be between 1 and 240")
	}
	if c.RecordSeconds <= 0 {
		return fmt.Errorf("record seconds must be positive")
	}
	return nil
}

// Display returns the resolved display
func (c *Config) Display() *hardware.Display {
	return c.display
}

// Encoder returns the resolved encoder
func (c *Config) Encoder() *hardware.Encoder {
	return c.encoder
}

// GPU returns the GPU associated with the encoder (nil for CPU encoding)
func (c *Config) GPU() *hardware.GPU {
	return c.gpu
}

// EncoderDisplayName returns a human-readable encoder name
func (c *Config) EncoderDisplayName() string {
	if c.encoder == nil {
		return "CPU (libx264)"
	}
	if c.gpu != nil {
		return fmt.Sprintf("%s (%s)", c.encoder.Name, c.gpu.Name)
	}
	return c.encoder.Name
}
