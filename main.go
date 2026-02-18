package main

import (
	"embed"
	_ "embed"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"rewind/internal"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/assets/icon.png
var appIcon []byte

//go:embed build/assets/icon-recording.png
var appIconRecording []byte

func main() {
	logPath := internal.GetDefaultLogPath()
	if err := internal.Setup(logPath, true); err != nil {
		log.Printf("Failed to setup logging: %v", err)
	}
	defer internal.Close()

	ffmpegPath := internal.GetFFmpegPath()
	slog.Info("Using FFmpeg", "path", ffmpegPath)

	// Start pprof server
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("pprof failed: %v", err)
		}
	}()

	rewindApp := internal.New(ffmpegPath)

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

	// Setup tray
	trayManager := internal.NewTrayManager(appInstance, rewindApp, window, appIcon, appIconRecording)
	trayManager.Setup()

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		trayManager.UpdateShowHideLabel()
	})

	rewindApp.SetOnStateChange(func(state internal.State) {
		trayManager.UpdateState()
	})

	// Setup hotkeys
	hkManager := internal.NewHotkeyManager()
	hkManager.SetupDefaultHotkeys(rewindApp)
	hkManager.Start()
	defer hkManager.Stop()

	err := appInstance.Run()
	if err != nil {
		log.Fatal(err)
	}
}
