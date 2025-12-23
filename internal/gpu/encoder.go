package gpu

import (
	"os/exec"
	"strings"
)

// FFmpegPath is the path to the FFmpeg executable.
// This should be set by the application at startup.
var FFmpegPath = "bin/ffmpeg.exe"

// DetectAvailableEncoders returns a list of available hardware encoders.
func DetectAvailableEncoders() []string {
	cmd := exec.Command(FFmpegPath, "-hide_banner", "-encoders")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	output := string(out)
	var encoders []string

	// Hardware encoders we care about
	hwEncoders := []string{
		"h264_nvenc", "hevc_nvenc", // NVIDIA
		"h264_amf", "hevc_amf", // AMD
		"h264_qsv", "hevc_qsv", // Intel
	}

	for _, enc := range hwEncoders {
		if strings.Contains(output, enc) {
			encoders = append(encoders, enc)
		}
	}

	return encoders
}

// ValidateEncoders checks which encoders are actually available in FFmpeg
// and updates the GPU's encoder availability flags.
func ValidateEncoders(gpus GPUList) {
	available := DetectAvailableEncoders()
	availableMap := make(map[string]bool)
	for _, enc := range available {
		availableMap[enc] = true
	}

	for _, gpu := range gpus {
		for i := range gpu.Encoders {
			gpu.Encoders[i].Available = availableMap[gpu.Encoders[i].Name]
		}
	}
}

// GetEncoderArgs returns FFmpeg arguments for a specific encoder.
func GetEncoderArgs(encoder *Encoder, captureVendor Vendor) []string {
	if encoder == nil {
		// CPU fallback
		return []string{
			"-vf", "hwdownload,format=bgra,format=nv12",
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-tune", "zerolatency",
		}
	}

	switch encoder.Name {
	case "h264_amf", "hevc_amf":
		return getAMFEncoderArgs(encoder)
	case "h264_nvenc", "hevc_nvenc":
		return getNVENCEncoderArgs(encoder, captureVendor)
	case "h264_qsv", "hevc_qsv":
		return getQSVEncoderArgs(encoder)
	}

	return nil
}

func getAMFEncoderArgs(encoder *Encoder) []string {
	// AMD AMF - optimal for ddagrab (same GPU, no transfer)
	return []string{
		"-vf", "scale_d3d11=format=nv12",
		"-c:v", encoder.Name,
		"-usage", "lowlatency",
		"-rc", "cbr",
		"-quality", "speed",
	}
}

func getNVENCEncoderArgs(encoder *Encoder, captureVendor Vendor) []string {
	// NVIDIA NVENC
	// If capture is from a different GPU, we need hwdownload first
	if captureVendor != VendorNVIDIA {
		return []string{
			"-vf", "hwdownload,format=bgra,hwupload_cuda,scale_cuda=format=nv12",
			"-c:v", encoder.Name,
			"-preset", "p1",
			"-rc", "cbr",
			"-delay", "0",
			"-zerolatency", "1",
		}
	}

	// Same GPU capture and encode
	return []string{
		"-vf", "hwmap=derive_device=cuda,scale_cuda=format=nv12",
		"-c:v", encoder.Name,
		"-preset", "p1",
		"-rc", "cbr",
		"-delay", "0",
		"-zerolatency", "1",
	}
}

func getQSVEncoderArgs(encoder *Encoder) []string {
	// Intel QuickSync
	return []string{
		"-vf", "hwmap=derive_device=qsv,format=qsv,scale_qsv=format=nv12",
		"-c:v", encoder.Name,
		"-preset", "veryfast",
		"-look_ahead", "0",
	}
}

// CPUEncoderArgs returns fallback CPU encoder arguments.
func CPUEncoderArgs() []string {
	return []string{
		"-vf", "hwdownload,format=bgra,format=nv12",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
	}
}
