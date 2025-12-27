//go:build windows

package hardware

import (
	"regexp"
	"strings"
)

// DetectGPUs returns a list of all GPUs in the system.
func DetectGPUs() (GPUList, error) {
	gpus, err := detectGPUsFromWMIC()
	if err != nil || len(gpus) == 0 {
		return detectGPUsFromEncoders()
	}
	return gpus, nil
}

// detectGPUsFromWMIC uses Windows WMI to get GPU information.
func detectGPUsFromWMIC() (GPUList, error) {
	cmd := Command("wmic", "path", "win32_videocontroller", "get", "name,adapterram,pnpdeviceid", "/format:csv")
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

		// CSV format: Node,AdapterRAM,Name,PNPDeviceID
		vramStr := strings.TrimSpace(parts[1])
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
		vram := parseVRAM(vramStr)

		gpu := &GPU{
			Index:        idx,
			Name:         name,
			Vendor:       vendor,
			VRAM:         vram,
			IsIntegrated: detectIsIntegrated(vendor, name, vram),
			Encoders:     getEncodersForVendor(vendor),
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

func parseVRAM(vramStr string) uint64 {
	re := regexp.MustCompile(`[0-9]+`)
	match := re.FindString(vramStr)
	if match == "" {
		return 0
	}

	var vram uint64
	for _, c := range match {
		vram = vram*10 + uint64(c-'0')
	}
	return vram
}

func detectIsIntegrated(vendor Vendor, name string, vram uint64) bool {
	nameLower := strings.ToLower(name)

	// Intel: integrated unless Arc
	if vendor == VendorIntel {
		return !strings.Contains(nameLower, "arc")
	}

	// AMD APU detection
	if vendor == VendorAMD {
		if strings.Contains(nameLower, "radeon graphics") ||
			strings.Contains(nameLower, "radeon(tm) graphics") {
			return true
		}
		if strings.Contains(nameLower, "vega") && !strings.Contains(nameLower, "rx vega") {
			return true
		}
		// Very low VRAM = APU
		if vram > 0 && vram <= 512*1024*1024 {
			return true
		}
	}

	return false
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

// detectGPUsFromEncoders is a fallback when WMI fails.
func detectGPUsFromEncoders() (GPUList, error) {
	var gpus GPUList
	encoders := DetectAvailableEncoders()

	hasNVIDIA := false
	hasAMD := false
	hasIntel := false

	for _, enc := range encoders {
		switch {
		case strings.Contains(enc, "nvenc"):
			hasNVIDIA = true
		case strings.Contains(enc, "amf"):
			hasAMD = true
		case strings.Contains(enc, "qsv"):
			hasIntel = true
		}
	}

	idx := 0
	if hasNVIDIA {
		gpus = append(gpus, &GPU{
			Index:        idx,
			Name:         "NVIDIA GPU",
			Vendor:       VendorNVIDIA,
			IsIntegrated: false,
			Encoders:     getEncodersForVendor(VendorNVIDIA),
		})
		idx++
	}
	if hasAMD {
		gpus = append(gpus, &GPU{
			Index:    idx,
			Name:     "AMD GPU",
			Vendor:   VendorAMD,
			Encoders: getEncodersForVendor(VendorAMD),
		})
		idx++
	}
	if hasIntel {
		gpus = append(gpus, &GPU{
			Index:        idx,
			Name:         "Intel GPU",
			Vendor:       VendorIntel,
			IsIntegrated: true,
			Encoders:     getEncodersForVendor(VendorIntel),
		})
	}

	if len(gpus) == 0 {
		gpus = append(gpus, &GPU{
			Index:  0,
			Name:   "CPU (Software Encoding)",
			Vendor: VendorUnknown,
		})
	}

	return gpus, nil
}
