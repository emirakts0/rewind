package capture

import (
	"fmt"

	"rewind/internal/display"
	"rewind/internal/gpu"
)

type Config struct {
	Display       *display.Display
	GPU           *gpu.GPU
	Encoder       *gpu.Encoder
	CaptureGPU    *gpu.GPU
	FPS           int
	Bitrate       string
	RecordSeconds int
	OutputDir     string
	FFmpegPath    string
	DrawMouse     bool
}

func DefaultConfig() *Config {
	return &Config{
		FPS:           60,
		Bitrate:       "15M",
		RecordSeconds: 30,
		OutputDir:     "./clips",
		FFmpegPath:    "bin/ffmpeg.exe",
		DrawMouse:     true,
	}
}

func (c *Config) Validate() error {
	if c.Display == nil {
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
