package hardware

import (
	"log/slog"
	"rewind/internal/utils"
	"strings"
)

type Encoder struct {
	Name      string
	Codec     string
	Available bool
	GPUIndex  int // which GPU this encoder belongs to
}

func DetectEncoders() []string {
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

// TODO: rename, DetectEncoders bunu yapalım. üsttekini düşün.
func DetectSystemEncoders(gpus GPUList) []Encoder {
	available := DetectEncoders()
	availableMap := make(map[string]bool)
	for _, enc := range available {
		availableMap[enc] = true
	}

	var allEncoders []Encoder

	for _, gpu := range gpus {
		potential := getEncodersForVendor(gpu.Vendor)
		for _, enc := range potential {
			enc.GPUIndex = gpu.Index
			enc.Available = availableMap[enc.Name]
			if enc.Available {
				allEncoders = append(allEncoders, enc)
			}
		}
	}

	allEncoders = append(allEncoders, Encoder{
		Name:      "libx264",
		Codec:     "h264",
		Available: true,
		GPUIndex:  -1, // CPU
	})

	return allEncoders
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

func CPUEncoderArgs() []string {
	return []string{
		"-vf", "hwdownload,format=bgra,format=nv12",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
	}
}

// TODO
func FindBestEncoder(encoders []Encoder) *Encoder {
	for i := range encoders {
		if encoders[i].Available {
			return &encoders[i]
		}
	}
	return nil
}
