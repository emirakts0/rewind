package hardware

import (
	"log/slog"
	"rewind/internal/utils"
	"strings"
)

// DetectAvailableEncoders returns a list of available hardware encoders.
func DetectAvailableEncoders() []string {
	slog.Debug("detecting encoders", "ffmpegPath", FFmpegPath)

	cmd := utils.Command(FFmpegPath, "-hide_banner", "-encoders")
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("ffmpeg encoder detection failed", "error", err, "output", string(out))
		return nil
	}

	output := string(out)
	var encoders []string

	hwEncoders := []string{
		"h264_nvenc", "hevc_nvenc", // NVIDIA
		"h264_amf", "hevc_amf", // AMD
		"h264_qsv", "hevc_qsv", // Intel
	}

	for _, enc := range hwEncoders {
		if strings.Contains(output, enc) {
			encoders = append(encoders, enc)
			slog.Debug("found encoder", "name", enc)
		}
	}

	slog.Info("detected encoders", "count", len(encoders), "encoders", encoders)
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
		return CPUEncoderArgs()
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
	return []string{
		"-vf", "scale_d3d11=format=nv12",
		"-c:v", encoder.Name,
		"-usage", "lowlatency",
		"-rc", "cbr",
		"-quality", "speed",
	}
}

func getNVENCEncoderArgs(encoder *Encoder, captureVendor Vendor) []string {
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

// TODO
func FindBestEncoder(gpus GPUList) *Encoder {
	for _, gpu := range gpus {
		for i := range gpu.Encoders {
			enc := &gpu.Encoders[i]
			if enc.Available {
				return enc
			}
		}
	}
	return nil
}
