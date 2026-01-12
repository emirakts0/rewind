package hardware

import (
	"fmt"
	"log/slog"
)

// Detect performs full system hardware detection
func Detect() (*SystemInfo, error) {
	gpus, err := DetectGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to detect GPUs: %w", err)
	}

	ValidateEncoders(gpus)

	displays, err := DetectDisplays()
	if err != nil {
		return nil, fmt.Errorf("failed to detect displays: %w", err)
	}

	// Collect all encoders from all GPUs
	var allEncoders []Encoder
	for _, gpu := range gpus {
		for _, enc := range gpu.Encoders {
			enc.GPUIndex = gpu.Index
			allEncoders = append(allEncoders, enc)
		}
	}

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
