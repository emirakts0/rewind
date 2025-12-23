//go:build windows

package gpu

import (
	"os/exec"
	"regexp"
	"strings"
)

// DetectGPUs returns a list of all GPUs in the system.
// Uses FFmpeg to detect available hardware encoders and infers GPU presence.
func DetectGPUs() (GPUList, error) {
	return detectGPUsFromDXDiag()
}

// detectGPUsFromDXDiag uses Windows built-in tools to get GPU info.
func detectGPUsFromDXDiag() (GPUList, error) {
	// Use wmic to get GPU information
	cmd := exec.Command("wmic", "path", "win32_videocontroller", "get", "name,adapterram,pnpdeviceid", "/format:csv")
	out, err := cmd.Output()
	if err != nil {
		// Fallback to encoder-based detection
		return detectGPUsFromEncoders()
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
		pnpID := strings.TrimSpace(parts[3])

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
			IsIntegrated: isIntegratedFromPNP(pnpID, vendor, vram, name),
			Encoders:     getEncodersForVendor(vendor),
		}

		gpus = append(gpus, gpu)
		idx++
	}

	if len(gpus) == 0 {
		return detectGPUsFromEncoders()
	}

	return gpus, nil
}

func detectVendorFromName(name string) Vendor {
	nameLower := strings.ToLower(name)
	switch {
	case strings.Contains(nameLower, "nvidia") || strings.Contains(nameLower, "geforce") || strings.Contains(nameLower, "rtx") || strings.Contains(nameLower, "gtx"):
		return VendorNVIDIA
	case strings.Contains(nameLower, "amd") || strings.Contains(nameLower, "radeon") || strings.Contains(nameLower, "rx "):
		return VendorAMD
	case strings.Contains(nameLower, "intel") || strings.Contains(nameLower, "iris") || strings.Contains(nameLower, "uhd"):
		return VendorIntel
	}
	return VendorUnknown
}

func parseVRAM(vramStr string) uint64 {
	// Remove any non-numeric characters
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

func isIntegratedFromPNP(pnpID string, vendor Vendor, vram uint64, name string) bool {
	nameLower := strings.ToLower(name)

	// Intel GPUs are almost always integrated (except Arc)
	if vendor == VendorIntel {
		if strings.Contains(nameLower, "arc") {
			return false
		}
		return true
	}

	// AMD APUs (integrated graphics)
	if vendor == VendorAMD {
		// "Radeon Graphics" or "Radeon(TM) Graphics" without RX model = integrated APU
		if (strings.Contains(nameLower, "radeon graphics") || strings.Contains(nameLower, "radeon(tm) graphics")) &&
			!strings.Contains(nameLower, "rx") {
			return true
		}
		// Vega integrated
		if strings.Contains(nameLower, "vega") && !strings.Contains(nameLower, "rx vega") {
			return true
		}
		// Very low VRAM usually indicates integrated (APU shares system RAM)
		// Integrated AMD GPUs often report 512MB or less dedicated VRAM
		if vram > 0 && vram <= 512*1024*1024 {
			return true
		}
	}

	// NVIDIA doesn't have integrated GPUs in consumer laptops
	// (they use Optimus with Intel/AMD iGPU)

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

// detectGPUsFromEncoders is a fallback when other methods fail.
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

	// If nothing detected, add a placeholder for CPU encoding
	if len(gpus) == 0 {
		gpus = append(gpus, &GPU{
			Index:  0,
			Name:   "CPU (Software Encoding)",
			Vendor: VendorUnknown,
		})
	}

	return gpus, nil
}
