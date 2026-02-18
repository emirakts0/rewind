package internal

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type TrayManager struct {
	app       *application.App
	systray   *application.SystemTray
	window    *application.WebviewWindow
	rewindApp *App
	menu      *application.Menu

	statusItem    *application.MenuItem
	startStopItem *application.MenuItem
	saveItem      *application.MenuItem
	showHideItem  *application.MenuItem

	appIcon          []byte
	appIconRecording []byte
}

func NewTrayManager(appInstance *application.App, rewindApp *App, window *application.WebviewWindow, icon, iconRecording []byte) *TrayManager {
	return &TrayManager{
		app:              appInstance,
		rewindApp:        rewindApp,
		window:           window,
		appIcon:          icon,
		appIconRecording: iconRecording,
	}
}

func (t *TrayManager) Setup() {
	t.systray = t.app.SystemTray.New()
	t.systray.SetIcon(t.appIcon)

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

		if state.Status == StatusSaving {
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
	isRecording := state.Status == StatusRecording
	isSaving := state.Status == StatusSaving

	slog.Info("updating tray state", "recording", isRecording, "saving", isSaving)

	if isSaving {
		t.systray.SetIcon(t.appIconRecording)
		t.statusItem.SetLabel("● Saving...")
		t.startStopItem.SetEnabled(false)
		t.saveItem.SetEnabled(false)
	} else if isRecording {
		t.systray.SetIcon(t.appIconRecording)
		t.statusItem.SetLabel("● Recording")
		t.startStopItem.SetLabel("Stop Recording")
		t.startStopItem.SetEnabled(true)
		t.saveItem.SetEnabled(true)
	} else {
		t.systray.SetIcon(t.appIcon)
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
