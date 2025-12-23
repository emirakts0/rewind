package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"rewind/internal/buffer"
	"rewind/internal/capture"
	"rewind/internal/display"
	"rewind/internal/gpu"
	"rewind/internal/output"
	"rewind/internal/system"

	gohook "github.com/robotn/gohook"
)

const (
	DefaultFPS           = 60
	DefaultBitrate       = "15M"
	DefaultRecordSeconds = 30
	DefaultOutputDir     = "./clips"
	FFmpegPath           = "bin/ffmpeg.exe"
)

func main() {
	gpu.FFmpegPath = FFmpegPath
	display.FFmpegPath = FFmpegPath

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	slog.Info("Rewind - Screen Replay System")
	slog.Info("detecting system configuration")

	sysInfo, err := system.Detect()
	if err != nil {
		slog.Error("failed to detect system", "error", err)
		os.Exit(1)
	}

	sysInfo.Print()

	if len(sysInfo.Displays) == 0 {
		slog.Error("no displays found")
		os.Exit(1)
	}

	selectedDisplay := sysInfo.SelectBestDisplay()
	if selectedDisplay == nil {
		slog.Error("could not select a display")
		os.Exit(1)
	}

	cfg, err := sysInfo.CreateCaptureConfig(selectedDisplay.Index)
	if err != nil {
		slog.Error("failed to create capture config", "error", err)
		os.Exit(1)
	}

	cfg.FPS = DefaultFPS
	cfg.Bitrate = DefaultBitrate
	cfg.RecordSeconds = DefaultRecordSeconds
	cfg.OutputDir = DefaultOutputDir
	cfg.FFmpegPath = FFmpegPath

	encoderName := "CPU (libx264)"
	if cfg.Encoder != nil {
		encoderName = cfg.Encoder.Name
	}

	slog.Info("capture configuration",
		"display", fmt.Sprintf("[%d] %dx%d", cfg.Display.Index, cfg.Display.Width, cfg.Display.Height),
		"gpu", cfg.CaptureGPU.Name,
		"encoder", encoderName,
		"fps", cfg.FPS,
		"bitrate", cfg.Bitrate,
		"buffer_seconds", cfg.RecordSeconds,
	)

	os.MkdirAll(cfg.OutputDir, os.ModePerm)

	bufSize := capture.CalculateBufferSize(cfg.Bitrate, cfg.RecordSeconds)
	rb := buffer.NewRing(bufSize)
	saver := output.NewSaver(cfg.FFmpegPath, cfg.OutputDir)

	capturer, err := capture.NewCapturer(cfg)
	if err != nil {
		slog.Error("failed to create capturer", "error", err)
		os.Exit(1)
	}

	capturer.OnData = func(data []byte) {
		rb.Write(data)
	}

	capturer.OnError = func(err error) {
		slog.Warn("capture error", "error", err)
	}

	slog.Info("starting capture")
	if err := capturer.Start(); err != nil {
		slog.Error("failed to start capture", "error", err)
		os.Exit(1)
	}

	slog.Info("recording active", "hotkey", "F10", "buffer_seconds", cfg.RecordSeconds)

	evChan := gohook.Start()
	defer gohook.End()

	lastSave := time.Time{}
	for ev := range evChan {
		if ev.Kind == gohook.KeyDown && ev.Rawcode == 121 {
			if time.Since(lastSave) < 3*time.Second {
				continue
			}
			lastSave = time.Now()

			filename := fmt.Sprintf("clip_%s", lastSave.Format("20060102_150405"))
			slog.Info("saving clip", "filename", filename)

			opts := output.DefaultSaveOptions(filename)
			if err := saver.Save(rb, opts); err != nil {
				slog.Error("failed to save clip", "error", err)
			}
		}
	}
}
