package capture

import (
	"fmt"
	"log/slog"

	"rewind/internal/hardware"
)

type Config struct {
	DisplayIndex int
	EncoderName  string

	FPS           int
	Bitrate       string
	RecordSeconds int
	OutputDir     string
	FFmpegPath    string
	DrawMouse     bool

	MicrophoneDevice  string
	SystemAudioDevice string

	display *hardware.Display
	encoder *hardware.Encoder
	gpu     *hardware.GPU
}

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
		c.EncoderName = "libx264"
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
