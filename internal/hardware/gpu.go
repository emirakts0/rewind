//go:build windows

package hardware

import (
	"fmt"
	"rewind/internal/utils"
	"strings"
)

// DetectGPUs returns a list of all GPUs in the system.
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
			continue
		}

		vendor := detectVendorFromName(name)

		gpu := &GPU{
			Index:    idx,
			Name:     name,
			Vendor:   vendor,
			Encoders: getEncodersForVendor(vendor),
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

func getEncodersForVendor(vendor Vendor) []Encoder {
	switch vendor {
	case VendorNVIDIA:
		return []Encoder{
			{Name: "h264_nvenc", Codec: "h264"},
			{Name: "hevc_nvenc", Codec: "hevc"},
		}
	case VendorAMD:
		return []Encoder{
			{Name: "h264_amf", Codec: "h264"},
			{Name: "hevc_amf", Codec: "hevc"},
		}
	case VendorIntel:
		return []Encoder{
			{Name: "h264_qsv", Codec: "h264"},
			{Name: "hevc_qsv", Codec: "hevc"},
		}
	}
	return nil
}
