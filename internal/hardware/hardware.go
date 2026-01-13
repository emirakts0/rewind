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

func (s *SystemInfo) GetEncoder(name string) *Encoder {
	for i := range s.Encoders {
		if s.Encoders[i].Name == name {
			return &s.Encoders[i]
		}
	}
	return nil
}

func (s *SystemInfo) GetAvailableEncoders() []Encoder {
	var result []Encoder
	for _, e := range s.Encoders {
		if e.Available {
			result = append(result, e)
		}
	}
	return result
}

func (s *SystemInfo) GetDisplay(index int) *Display {
	return s.Displays.FindByIndex(index)
}

func (s *SystemInfo) GetEncodersForDisplay(displayIndex int) []Encoder {
	display := s.GetDisplay(displayIndex)
	targetGPU := -999
	if display != nil {
		targetGPU = display.GPUIndex
	}

	var compatible []Encoder
	for _, e := range s.GetAvailableEncoders() {
		// Include if it's CPU (-1) or matches the display's GPU
		// Also include if no display is found (fallback to all?) - no, logic above implies only matching or CPU.
		// If display is not found, targetGPU is -999, so only CPU encoders (Index -1) will match.
		if e.GPUIndex == -1 || (targetGPU != -999 && e.GPUIndex == targetGPU) {
			compatible = append(compatible, e)
		}
	}
	return compatible
}

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

	return &SystemInfo{
		GPUs:     gpus,
		Displays: displays,
		Encoders: allEncoders,
	}, nil
}

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
