//go:build windows

package hardware

import (
	"fmt"
	"log/slog"
	"strings"
)

type Vendor string

const (
	VendorNVIDIA  Vendor = "nvidia"
	VendorAMD     Vendor = "amd"
	VendorIntel   Vendor = "intel"
	VendorUnknown Vendor = "unknown"
)

type GPU struct {
	Index  int
	Name   string
	Vendor Vendor
}

func (g *GPU) String() string {
	return fmt.Sprintf("[%d] %s (%s)", g.Index, g.Name, g.Vendor)
}

type GPUList []*GPU

func (l GPUList) FindByIndex(index int) *GPU {
	for _, g := range l {
		if g.Index == index {
			return g
		}
	}
	return nil
}
func DetectGPUs() (GPUList, error) {
	gpus, err := detectGPUsFromDXGI()
	if err != nil || len(gpus) == 0 {
		return nil, fmt.Errorf("GPU detection failed: %w", err)
	}
	return gpus, nil
}

// detectGPUsFromDXGI uses DXGI (DirectX Graphics Infrastructure) to detect GPUs.
func detectGPUsFromDXGI() (GPUList, error) {
	gpuNames := EnumerateGPUsDXGI()
	if len(gpuNames) == 0 {
		return nil, fmt.Errorf("no GPUs found via DXGI")
	}

	var gpus GPUList
	idx := 0
	for _, name := range gpuNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Skip Microsoft Basic Display
		if strings.Contains(strings.ToLower(name), "microsoft") ||
			strings.Contains(strings.ToLower(name), "basic") {
			slog.Info("Skipping basic display adapter", "name", name)
			continue
		}

		vendor := detectVendorFromName(name)

		gpu := &GPU{
			Index:  idx,
			Name:   name,
			Vendor: vendor,
		}

		gpus = append(gpus, gpu)
		idx++
	}

	return gpus, nil
}

func detectVendorFromName(name string) Vendor {
	nameLower := strings.ToLower(name)
	switch {
	case strings.Contains(nameLower, "nvidia") || strings.Contains(nameLower, "geforce") ||
		strings.Contains(nameLower, "rtx") || strings.Contains(nameLower, "gtx"):
		return VendorNVIDIA
	case strings.Contains(nameLower, "amd") || strings.Contains(nameLower, "radeon") ||
		strings.Contains(nameLower, "rx "):
		return VendorAMD
	case strings.Contains(nameLower, "intel") || strings.Contains(nameLower, "iris") ||
		strings.Contains(nameLower, "uhd"):
		return VendorIntel
	}
	return VendorUnknown
}
