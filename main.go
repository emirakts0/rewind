package main

import (
	"embed"
	_ "embed"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"

	"rewind/internal/app"
	"rewind/internal/utils"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/assets/icon.png
var appIcon []byte

//go:embed build/assets/icon-recording.png
var appIconRecording []byte

type TrayManager struct {
	app       *application.App
	systray   *application.SystemTray
	window    *application.WebviewWindow
	rewindApp *app.App
	menu      *application.Menu

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

	t.systray.OnClick(func() {
		if t.window.IsVisible() {
			t.window.Hide()
		} else {
			t.window.Show()
			t.window.Focus()
		}
		t.UpdateShowHideLabel()
	})

	t.systray.OnRightClick(func() {
		t.UpdateShowHideLabel()
		t.systray.OpenMenu()
	})
}

func (t *TrayManager) createMenu() {
	t.menu = t.app.NewMenu()

	t.statusItem = t.menu.Add("● Ready")
	t.statusItem.SetEnabled(false)

	t.menu.AddSeparator()

	t.startStopItem = t.menu.Add("Start Recording")
	t.startStopItem.OnClick(func(ctx *application.Context) {
		state := t.rewindApp.GetRecordingState()

		if state.Status == app.StatusSaving {
			slog.Warn("Cannot start/stop while saving")
			return
		}

		if t.rewindApp.IsRecording() {
			if err := t.rewindApp.StopRecording(); err != nil {
				slog.Error("Failed to stop recording", "error", err)
			}
		} else {
			if err := t.rewindApp.StartRecording(); err != nil {
				slog.Error("Failed to start recording", "error", err)
			}
		}
		t.UpdateState()
	})

	t.saveItem = t.menu.Add("Save Clip")
	t.saveItem.SetEnabled(false)
	t.saveItem.OnClick(func(ctx *application.Context) {
		if _, err := t.rewindApp.SaveCurrentClip(); err != nil {
			slog.Error("Failed to save clip", "error", err)
		}
	})

	t.menu.AddSeparator()

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

	reloadItem := t.menu.Add("Reload UI")
	reloadItem.OnClick(func(ctx *application.Context) {
		t.window.Reload()
		slog.Info("UI reloaded")
	})

	t.menu.AddSeparator()

	quitItem := t.menu.Add("Quit Rewind")
	quitItem.OnClick(func(ctx *application.Context) {
		if t.rewindApp.IsRecording() {
			t.rewindApp.StopRecording()
		}
		t.app.Quit()
	})

	t.systray.SetMenu(t.menu)
}

func (t *TrayManager) UpdateState() {
	state := t.rewindApp.GetRecordingState()
	isRecording := state.Status == app.StatusRecording
	isSaving := state.Status == app.StatusSaving

	slog.Info("updating tray state", "recording", isRecording, "saving", isSaving)

	if isSaving {
		t.systray.SetIcon(appIconRecording)
		t.statusItem.SetLabel("● Saving...")
		t.startStopItem.SetEnabled(false)
		t.saveItem.SetEnabled(false)
	} else if isRecording {
		t.systray.SetIcon(appIconRecording)
		t.statusItem.SetLabel("● Recording")
		t.startStopItem.SetLabel("Stop Recording")
		t.startStopItem.SetEnabled(true)
		t.saveItem.SetEnabled(true)
	} else {
		t.systray.SetIcon(appIcon)
		t.statusItem.SetLabel("● Ready")
		t.startStopItem.SetLabel("Start Recording")
		t.startStopItem.SetEnabled(true)
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
	logPath := utils.GetDefaultLogPath()
	if err := utils.Setup(logPath, true); err != nil {
		log.Printf("Failed to setup logging: %v", err)
	}
	defer utils.Close()

	ffmpegPath := utils.GetFFmpegPath()
	slog.Info("Using FFmpeg", "path", ffmpegPath)

	// Start pprof server
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("pprof failed: %v", err)
		}
	}()

	rewindApp := app.New(ffmpegPath)

	var mainWindow *application.WebviewWindow

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
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.emirakts.rewind.single.instance",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				slog.Info("Second instance launched, bringing window to front", "args", data.Args)
				if mainWindow != nil {
					mainWindow.Show()
					mainWindow.Focus()
				}
			},
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
	mainWindow = window

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	trayManager := NewTrayManager(appInstance, rewindApp, window)
	trayManager.Setup()

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		trayManager.UpdateShowHideLabel()
	})

	// Set callback for state changes to update tray
	rewindApp.SetOnStateChange(func(state app.State) {
		trayManager.UpdateState()
	})

	hkManager := utils.NewHotkeyManager()

	hkManager.Register(1, func() {
		state := rewindApp.GetRecordingState()

		if state.Status == app.StatusSaving {
			slog.Warn("Cannot start/stop while saving")
			return
		}

		if rewindApp.IsRecording() {
			if err := rewindApp.StopRecording(); err != nil {
				slog.Error("Failed to stop recording via hotkey", "error", err)
			}
		} else {
			if err := rewindApp.StartRecording(); err != nil {
				slog.Error("Failed to start recording via hotkey", "error", err)
			}
		}
	})

	hkManager.Register(2, func() {
		if _, err := rewindApp.SaveCurrentClip(); err != nil {
			slog.Error("Failed to save clip via hotkey", "error", err)
		}
	})

	hkManager.Start()
	defer hkManager.Stop()

	err := appInstance.Run()
	if err != nil {
		log.Fatal(err)
	}
}
