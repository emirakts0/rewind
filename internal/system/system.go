package system

import (
	"fmt"
	"log/slog"

	"rewind/internal/capture"
	"rewind/internal/display"
	"rewind/internal/gpu"
)

type Info struct {
	GPUs         gpu.GPUList
	Displays     display.DisplayList
	HasHybridGPU bool
}

func Detect() (*Info, error) {
	gpus, err := gpu.DetectGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to detect GPUs: %w", err)
	}

	gpu.ValidateEncoders(gpus)

	displays, err := display.DetectDisplays()
	if err != nil {
		return nil, fmt.Errorf("failed to detect displays: %w", err)
	}

	if err := display.AssociateWithGPUs(displays); err != nil {
		slog.Warn("could not associate displays with GPUs", "error", err)
	}

	for _, d := range displays {
		if d.GPUIndex >= 0 && d.GPUIndex < len(gpus) {
			if d.GPUName == "" {
				d.GPUName = gpus[d.GPUIndex].Name
			}
		}
	}

	return &Info{
		GPUs:         gpus,
		Displays:     displays,
		HasHybridGPU: gpus.HasHybridSetup(),
	}, nil
}

func (i *Info) Print() {
	for _, g := range i.GPUs {
		gpuType := "discrete"
		if g.IsIntegrated {
			gpuType = "integrated"
		}
		encoder := "none"
		if enc := g.GetPreferredEncoder(); enc != nil && enc.Available {
			encoder = enc.Name
		}
		slog.Info("detected GPU",
			"index", g.Index,
			"name", g.Name,
			"type", gpuType,
			"encoder", encoder,
		)
	}

	for _, d := range i.Displays {
		slog.Info("detected display",
			"index", d.Index,
			"resolution", fmt.Sprintf("%dx%d", d.Width, d.Height),
			"primary", d.IsPrimary,
			"gpu", d.GPUName,
		)
	}

	if i.HasHybridGPU {
		slog.Info("hybrid GPU system detected, using same-GPU capture+encode")
	}
}

func (i *Info) CreateCaptureConfig(displayIndex int) (*capture.Config, error) {
	d := i.Displays.FindByIndex(displayIndex)
	if d == nil {
		d = i.Displays.FindPrimary()
		if d == nil {
			return nil, fmt.Errorf("no displays found")
		}
	}

	var captureGPU *gpu.GPU
	if d.GPUIndex >= 0 && d.GPUIndex < len(i.GPUs) {
		captureGPU = i.GPUs[d.GPUIndex]
	} else if len(i.GPUs) > 0 {
		captureGPU = i.GPUs[0]
	}

	var encoder *gpu.Encoder
	if captureGPU != nil {
		encoder = captureGPU.GetPreferredEncoder()
	}

	cfg := capture.DefaultConfig()
	cfg.Display = d
	cfg.CaptureGPU = captureGPU
	cfg.Encoder = encoder
	if captureGPU != nil {
		cfg.GPU = captureGPU
	}

	return cfg, nil
}

func (i *Info) SelectBestDisplay() *display.Display {
	if primary := i.Displays.FindPrimary(); primary != nil {
		return primary
	}
	if len(i.Displays) > 0 {
		return i.Displays[0]
	}
	return nil
}
