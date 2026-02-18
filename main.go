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
	internal.SetDefaultLogging()
	defer internal.CloseLogFile()

	startPprofServer()

	ffmpegPath := internal.GetFFmpegPath()
	slog.Info("Using FFmpeg", "path", ffmpegPath)

	rewindApp := internal.New(ffmpegPath)
	appInstance := createApplication(rewindApp)
	window := createWindow(appInstance)

	setupTray(appInstance, rewindApp, window)
	setupHotkeys(rewindApp)

	if err := appInstance.Run(); err != nil {
		log.Fatal(err)
	}
}

func startPprofServer() {
	go func() {
		slog.Debug("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			slog.Debug("pprof failed", err)
		}
	}()
}

func createApplication(rewindApp *internal.App) *application.App {
	var mainWindow *application.WebviewWindow

	app := application.New(application.Options{
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

	rewindApp.SetApp(app)
	return app
}

func createWindow(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
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

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	return window
}

func setupTray(app *application.App, rewindApp *internal.App, window *application.WebviewWindow) {
	trayManager := internal.NewTrayManager(app, rewindApp, window, appIcon, appIconRecording)
	trayManager.Setup()

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		trayManager.UpdateShowHideLabel()
	})

	rewindApp.SetOnStateChange(func(state internal.State) {
		trayManager.UpdateState()
	})
}

func setupHotkeys(rewindApp *internal.App) {
	hkManager := internal.NewHotkeyManager()
	hkManager.SetupDefaultHotkeys(rewindApp)
	hkManager.Start()
}
