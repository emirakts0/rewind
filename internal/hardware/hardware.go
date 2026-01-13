package hardware

import (
	"fmt"
	"log/slog"
)

// FFmpegPath is the path to the FFmpeg executable.
var FFmpegPath = "bin/ffmpeg.exe"

type SystemInfo struct {
	GPUs     GPUList
	Displays DisplayList
	Encoders []Encoder // all available encoders from all GPUs
}

// GetEncoder finds an encoder by name
func (s *SystemInfo) GetEncoder(name string) *Encoder {
	for i := range s.Encoders {
		if s.Encoders[i].Name == name {
			return &s.Encoders[i]
		}
	}
	return nil
}

// GetAvailableEncoders returns all available (working) encoders
func (s *SystemInfo) GetAvailableEncoders() []Encoder {
	var result []Encoder
	for _, e := range s.Encoders {
		if e.Available {
			result = append(result, e)
		}
	}
	return result
}

// GetDisplay finds a display by index
func (s *SystemInfo) GetDisplay(index int) *Display {
	return s.Displays.FindByIndex(index)
}

// Detect performs full system hardware detection
func Detect() (*SystemInfo, error) {
	gpus, err := DetectGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to detect GPUs: %w", err)
	}

	displays, err := DetectDisplays()
	if err != nil {
		return nil, fmt.Errorf("failed to detect displays: %w", err)
	}

	allEncoders := DetectSystemEncoders(gpus)

	// Add CPU encoder as fallback
	allEncoders = append(allEncoders, Encoder{
		Name:      "libx264",
		Codec:     "h264",
		Available: true,
		GPUIndex:  -1, // CPU
	})

	return &SystemInfo{
		GPUs:     gpus,
		Displays: displays,
		Encoders: allEncoders,
	}, nil
}

// Print logs all detected hardware information
func (s *SystemInfo) Print() {
	for _, g := range s.GPUs {
		slog.Info("detected GPU",
			"index", g.Index,
			"name", g.Name,
		)
	}

	for _, d := range s.Displays {
		slog.Info("detected display",
			"index", d.Index,
			"resolution", fmt.Sprintf("%dx%d", d.Width, d.Height),
			"primary", d.IsPrimary,
		)
	}

	slog.Info("available encoders")
	for _, e := range s.GetAvailableEncoders() {
		gpuName := "CPU"
		if e.GPUIndex >= 0 {
			if gpu := s.GPUs.FindByIndex(e.GPUIndex); gpu != nil {
				gpuName = gpu.Name
			}
		}
		slog.Info("  encoder",
			"name", e.Name,
			"codec", e.Codec,
			"gpu", gpuName,
		)
	}
}
