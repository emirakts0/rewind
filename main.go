package main

import (
	"embed"
	_ "embed"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	"rewind/internal/app"
	"rewind/internal/logging"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/public/icon.png
var appIcon []byte

//go:embed frontend/public/icon-recording.png
var appIconRecording []byte

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

// TrayManager handles system tray functionality
type TrayManager struct {
	app       *application.App
	systray   *application.SystemTray
	window    *application.WebviewWindow
	rewindApp *app.App
	menu      *application.Menu

	// Menu items that need updating
	statusItem    *application.MenuItem
	startStopItem *application.MenuItem
	saveItem      *application.MenuItem
	showHideItem  *application.MenuItem
}

func NewTrayManager(appInstance *application.App, rewindApp *app.App, window *application.WebviewWindow) *TrayManager {
	return &TrayManager{
		app:       appInstance,
		rewindApp: rewindApp,
		window:    window,
	}
}

func (t *TrayManager) Setup() {
	t.systray = t.app.SystemTray.New()
	t.systray.SetIcon(appIcon)

	t.createMenu()

	// Left click - toggle window
	t.systray.OnClick(func() {
		if t.window.IsVisible() {
			t.window.Hide()
		} else {
			t.window.Show()
			t.window.Focus()
		}
		t.UpdateShowHideLabel()
	})

	// Right click - show menu
	t.systray.OnRightClick(func() {
		t.UpdateShowHideLabel()
		t.systray.OpenMenu()
	})
}

func (t *TrayManager) createMenu() {
	t.menu = t.app.NewMenu()

	// Status item (disabled, shows current state)
	t.statusItem = t.menu.Add("● Ready")
	t.statusItem.SetEnabled(false)

	t.menu.AddSeparator()

	// Start/Stop Recording
	t.startStopItem = t.menu.Add("Start Recording")
	t.startStopItem.OnClick(func(ctx *application.Context) {
		if t.rewindApp.IsRecording() {
			t.rewindApp.Stop()
		} else {
			t.rewindApp.Start()
		}
		t.UpdateState()
	})

	// Save Clip
	t.saveItem = t.menu.Add("Save Clip")
	t.saveItem.SetEnabled(false)
	t.saveItem.OnClick(func(ctx *application.Context) {
		if t.rewindApp.IsRecording() {
			t.rewindApp.SaveClip()
		}
	})

	t.menu.AddSeparator()

	// Show/Hide Window
	t.showHideItem = t.menu.Add("Show Window")
	t.showHideItem.OnClick(func(ctx *application.Context) {
		if t.window.IsVisible() {
			t.window.Hide()
		} else {
			t.window.Show()
			t.window.Focus()
		}
		t.UpdateShowHideLabel()
	})

	// Reload UI (for testing - simulates frontend crash recovery)
	reloadItem := t.menu.Add("Reload UI")
	reloadItem.OnClick(func(ctx *application.Context) {
		t.window.Reload()
		slog.Info("UI reloaded - frontend should fetch state from backend")
	})

	t.menu.AddSeparator()

	// Quit
	quitItem := t.menu.Add("Quit Rewind")
	quitItem.OnClick(func(ctx *application.Context) {
		// Stop recording before quit
		if t.rewindApp.IsRecording() {
			t.rewindApp.Stop()
		}
		t.app.Quit()
	})

	t.systray.SetMenu(t.menu)
}

func (t *TrayManager) UpdateState() {
	isRecording := t.rewindApp.IsRecording()

	if isRecording {
		t.systray.SetIcon(appIconRecording)
		t.statusItem.SetLabel("● Recording")
		t.startStopItem.SetLabel("Stop Recording")
		t.saveItem.SetEnabled(true)
	} else {
		t.systray.SetIcon(appIcon)
		t.statusItem.SetLabel("● Ready")
		t.startStopItem.SetLabel("Start Recording")
		t.saveItem.SetEnabled(false)
	}

	t.menu.Update()
}

func (t *TrayManager) UpdateShowHideLabel() {
	if t.window.IsVisible() {
		t.showHideItem.SetLabel("Hide Window")
	} else {
		t.showHideItem.SetLabel("Show Window")
	}
	t.menu.Update()
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
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(rewindApp),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// Store the app instance for events
	rewindApp.SetApp(appInstance)

	window := appInstance.Window.NewWithOptions(application.WebviewWindowOptions{
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

	// Hide to tray instead of closing
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel() // Prevent actual close
	})

	// Setup system tray
	trayManager := NewTrayManager(appInstance, rewindApp, window)
	trayManager.Setup()

	// Hook to update tray menu when window is hidden via close button
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		trayManager.UpdateShowHideLabel()
	})

	// Global Key Bindings
	// Start/Stop Recording: Ctrl + F9
	appInstance.KeyBinding.Add("Ctrl+F9", func(window application.Window) {
		if rewindApp.IsRecording() {
			rewindApp.Stop()
		} else {
			rewindApp.Start()
		}
		trayManager.UpdateState()
	})

	// Save Clip: Ctrl + F10
	appInstance.KeyBinding.Add("Ctrl+F10", func(window application.Window) {
		if rewindApp.IsRecording() {
			rewindApp.SaveClip()
		}
	})

	// Set callback for state changes to update tray
	rewindApp.SetOnStateChange(func() {
		trayManager.UpdateState()
	})

	err := appInstance.Run()
	if err != nil {
		log.Fatal(err)
	}
}
