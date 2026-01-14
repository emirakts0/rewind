package main

import (
	"embed"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	"rewind/internal/app"
	"rewind/internal/logging"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func getFFmpegPath() string {
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		ffmpegPath := filepath.Join(exeDir, "ffmpeg.exe")
		if _, err := os.Stat(ffmpegPath); err == nil {
			return ffmpegPath
		}

		ffmpegPath = filepath.Join(exeDir, "bin", "ffmpeg.exe")
		if _, err := os.Stat(ffmpegPath); err == nil {
			return ffmpegPath
		}
	}

	// Fallback: check working directory
	if _, err := os.Stat("bin/ffmpeg.exe"); err == nil {
		return "bin/ffmpeg.exe"
	}
	if _, err := os.Stat("ffmpeg.exe"); err == nil {
		return "ffmpeg.exe"
	}

	// Last resort: hope it's in PATH
	return "ffmpeg"
}

func main() {
	logPath := logging.GetDefaultLogPath()
	if err := logging.Setup(logPath, true); err != nil {
		log.Printf("Failed to setup logging: %v", err)
	}
	defer logging.Close()

	ffmpegPath := getFFmpegPath()
	slog.Info("Using FFmpeg", "path", ffmpegPath)

	// Start pprof server
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("pprof failed: %v", err)
		}
	}()

	rewindApp := app.New(ffmpegPath)

	appInstance := application.New(application.Options{
		Name:        "Rewind",
		Description: "Screen recording application with instant replay",
		Services: []application.Service{
			application.NewService(rewindApp),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Store the app instance for events
	rewindApp.SetApp(appInstance)

	appInstance.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Rewind",
		Width:            420,
		Height:           750,
		MinWidth:         420,
		MinHeight:        750,
		MaxWidth:         420,
		MaxHeight:        750,
		DisableResize:    true,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(15, 15, 20, 255),
		URL:              "/",
	})

	err := appInstance.Run()
	if err != nil {
		log.Fatal(err)
	}
}
