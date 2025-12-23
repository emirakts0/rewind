package capture

import (
	"fmt"
	"strconv"
	"strings"

	"rewind/internal/gpu"
)

type FFmpegCommandBuilder struct {
	config *Config
}

func NewFFmpegCommandBuilder(cfg *Config) *FFmpegCommandBuilder {
	return &FFmpegCommandBuilder{config: cfg}
}

func (b *FFmpegCommandBuilder) BuildArgs() []string {
	args := []string{"-hide_banner"}
	args = append(args, b.getHWDeviceArgs()...)
	args = append(args, b.getInputArgs()...)
	args = append(args, b.getEncoderArgs()...)
	args = append(args, b.getOutputArgs()...)
	return args
}

func (b *FFmpegCommandBuilder) getHWDeviceArgs() []string {
	if b.config.CaptureGPU == nil {
		return nil
	}

	switch b.config.CaptureGPU.Vendor {
	case gpu.VendorAMD, gpu.VendorIntel, gpu.VendorNVIDIA:
		return []string{
			"-init_hw_device", "d3d11va=d3d11",
			"-filter_hw_device", "d3d11",
		}
	}
	return nil
}

func (b *FFmpegCommandBuilder) getInputArgs() []string {
	drawMouse := 0
	if b.config.DrawMouse {
		drawMouse = 1
	}

	outputIdx := 0
	if b.config.Display != nil {
		outputIdx = b.config.Display.Index
	}

	return []string{
		"-f", "lavfi",
		"-rtbufsize", "100M",
		"-i", fmt.Sprintf("ddagrab=output_idx=%d:framerate=%d:draw_mouse=%d",
			outputIdx, b.config.FPS, drawMouse),
	}
}

func (b *FFmpegCommandBuilder) getEncoderArgs() []string {
	if b.config.Encoder == nil {
		return gpu.CPUEncoderArgs()
	}

	captureVendor := gpu.VendorUnknown
	if b.config.CaptureGPU != nil {
		captureVendor = b.config.CaptureGPU.Vendor
	}

	return gpu.GetEncoderArgs(b.config.Encoder, captureVendor)
}

func (b *FFmpegCommandBuilder) getOutputArgs() []string {
	return []string{
		"-b:v", b.config.Bitrate,
		"-maxrate", b.config.Bitrate,
		"-bufsize", b.config.Bitrate,
		"-g", strconv.Itoa(b.config.FPS),
		"-f", "mpegts",
		"-",
	}
}

func ParseBitrate(br string) int {
	br = strings.ToLower(br)
	mul := 1
	if strings.Contains(br, "m") {
		mul = 1000000
		br = strings.ReplaceAll(br, "m", "")
	} else if strings.Contains(br, "k") {
		mul = 1000
		br = strings.ReplaceAll(br, "k", "")
	}
	val, _ := strconv.Atoi(br)
	return (val * mul) / 8
}

func CalculateBufferSize(bitrate string, seconds int) int {
	bps := ParseBitrate(bitrate)
	return int(float64(bps*seconds) * 1.5)
}
