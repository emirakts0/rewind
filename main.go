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
	"rewind/internal/utils"

	"rewind/internal/app"
	"rewind/internal/input"
	"rewind/internal/logging"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.org/x/sys/windows/registry"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/assets/icon.png
var appIcon []byte

//go:embed build/assets/icon-recording.png
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

func ensureStartup() {
	exePath, err := os.Executable()
	if err != nil {
		slog.Error("Failed to get executable path for startup registration", "error", err)
		return
	}

	// Only register if running as the production binary name to avoid registering temp dev builds
	if filepath.Base(exePath) != "rewind.exe" {
		slog.Debug("Skipping startup registration for non-production binary", "name", filepath.Base(exePath))
		return
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	if err != nil {
		slog.Error("Failed to open startup registry key", "error", err)
		return
	}
	defer key.Close()

	if err := key.SetStringValue("Rewind", exePath); err != nil {
		slog.Error("Failed to set startup registry value", "error", err)
	} else {
		slog.Info("Ensured application is in startup registry", "path", exePath)
	}
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
	slog.Info("updating tray state", "recording", isRecording)

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
	t.systray.SetMenu(t.menu)
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
	// Ensure only one instance is running
	mutex, err := utils.AcquireSingleInstance("Global\\RewindApp")
	if err != nil {
		log.Fatalf("Cannot start: %v", err)
		return
	}
	defer mutex.Release()

	logPath := logging.GetDefaultLogPath()
	if err = logging.Setup(logPath, true); err != nil {
		log.Printf("Failed to setup logging: %v", err)
	}
	defer logging.Close()

	ffmpegPath := getFFmpegPath()
	slog.Info("Using FFmpeg", "path", ffmpegPath)

	ensureStartup()

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

	// Global Hotkeys (works system-wide)
	hkManager := input.NewHotkeyManager()

	// Start/Stop: Ctrl+F9
	hkManager.Register(1, func() {
		if rewindApp.IsRecording() {
			rewindApp.Stop()
		} else {
			rewindApp.Start()
		}
	})

	// Save Clip: Ctrl+F10
	hkManager.Register(2, func() {
		if rewindApp.IsRecording() {
			rewindApp.SaveClip()
		}
	})

	hkManager.Start()
	defer hkManager.Stop()

	// Set callback for state changes to update tray
	rewindApp.SetOnStateChange(func(state app.State) {
		trayManager.UpdateState()
	})

	err = appInstance.Run()
	if err != nil {
		log.Fatal(err)
	}
}
