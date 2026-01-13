//go:build windows

package hardware

import (
	"fmt"
	"log/slog"
	"rewind/internal/utils"
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
	gpus, err := detectGPUsFromWMIC()
	if err != nil || len(gpus) == 0 {
		return nil, fmt.Errorf("WMI GPU detection failed: %w", err)
	}
	return gpus, nil
}

// detectGPUsFromWMIC uses Windows WMI to get GPU information.
func detectGPUsFromWMIC() (GPUList, error) {
	cmd := utils.Command("wmic", "path", "win32_videocontroller", "get", "name,adapterram,pnpdeviceid", "/format:csv")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var gpus GPUList
	lines := strings.Split(string(out), "\n")

	idx := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node,") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		name := strings.TrimSpace(parts[2])

		if name == "" || name == "Name" {
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
