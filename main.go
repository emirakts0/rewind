package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	"rewind/internal/app"
	"rewind/internal/logging"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func getFFmpegPath() string {
	// Try to find ffmpeg relative to the executable
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		// Check in same directory as executable
		ffmpegPath := filepath.Join(exeDir, "ffmpeg.exe")
		if _, err := os.Stat(ffmpegPath); err == nil {
			return ffmpegPath
		}
		// Check in bin subdirectory
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
	// Setup logging
	logPath := logging.GetDefaultLogPath()
	if err := logging.Setup(logPath, true); err != nil {
		log.Printf("Failed to setup logging: %v", err)
	}
	defer logging.Close()

	ffmpegPath := getFFmpegPath()
	log.Printf("Using FFmpeg: %s", ffmpegPath)

	// Start pprof server
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("pprof failed: %v", err)
		}
	}()

	// Create app instance
	rewindApp := app.New(ffmpegPath)

	// Set startup callback
	rewindApp.OnStartup = func(ctx context.Context) {
		if err := rewindApp.Initialize(); err != nil {
			log.Printf("Failed to initialize: %v", err)
		}
	}

	err := wails.Run(&options.App{
		Title:         "Rewind",
		Width:         420,
		Height:        750,
		MinWidth:      420,
		MinHeight:     750,
		MaxWidth:      420,
		MaxHeight:     750,
		DisableResize: true,
		Frameless:     false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 20, A: 255},
		OnStartup:        rewindApp.Startup,
		OnShutdown:       rewindApp.Shutdown,
		Bind: []interface{}{
			rewindApp,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
