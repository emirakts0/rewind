package internal

import (
	"log/slog"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmDeviceChange = 0x0219

	dbtDevNodesChanged      = 0x0007
	dbtDeviceArrival        = 0x8000
	dbtDeviceRemoveComplete = 0x8004

	dbtDevtypDevinterface = 0x00000005

	deviceNotifyWindowHandle        = 0x00000000
	deviceNotifyAllInterfaceClasses = 0x00000004

	wmApp     = 0x8000
	wmAppQuit = wmApp + 1
)

type devBroadcastDevinterface struct {
	Size       uint32
	DeviceType uint32
	Reserved   uint32
	ClassGuid  windows.GUID
	Name       [1]uint16
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   syscall.Handle
	Icon       syscall.Handle
	Cursor     syscall.Handle
	Background syscall.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     syscall.Handle
}

type winMsg struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

var (
	procRegisterDeviceNotification   = user32.NewProc("RegisterDeviceNotificationW")
	procUnregisterDeviceNotification = user32.NewProc("UnregisterDeviceNotification")
	procCreateWindowExW              = user32.NewProc("CreateWindowExW")
	procDestroyWindow                = user32.NewProc("DestroyWindow")
	procDefWindowProcW               = user32.NewProc("DefWindowProcW")
	procRegisterClassExW             = user32.NewProc("RegisterClassExW")
	procUnregisterClassW             = user32.NewProc("UnregisterClassW")
	procGetMessageDM                 = user32.NewProc("GetMessageW")
	procPostMessageW                 = user32.NewProc("PostMessageW")
	procPostQuitMessageDM            = user32.NewProc("PostQuitMessage")
)

type DeviceMonitor struct {
	cb      func()
	hwnd    atomic.Uintptr
	ready   chan struct{}
	stopped atomic.Bool

	debounceTimer *time.Timer
}

func NewDeviceMonitor(cb func()) *DeviceMonitor {
	return &DeviceMonitor{
		cb:    cb,
		ready: make(chan struct{}),
	}
}

func (dm *DeviceMonitor) Start() error {
	slog.Info("Starting device monitor...")
	go dm.messageLoop()
	<-dm.ready
	return nil
}

func (dm *DeviceMonitor) Stop() {
	if !dm.stopped.CompareAndSwap(false, true) {
		return
	}
	if hwnd := dm.hwnd.Load(); hwnd != 0 {
		procPostMessageW.Call(hwnd, wmAppQuit, 0, 0)
	}
	slog.Info("Device monitor stopped")
}

func (dm *DeviceMonitor) messageLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _, _ := syscall.NewLazyDLL("kernel32.dll").
		NewProc("GetModuleHandleW").Call(0)

	className, _ := syscall.UTF16PtrFromString("RewindDevMon")

	var wc wndClassEx
	wc.Size = uint32(unsafe.Sizeof(wc))
	wc.WndProc = syscall.NewCallback(dm.wndProc)
	wc.Instance = syscall.Handle(hInstance)
	wc.ClassName = className

	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		slog.Error("DeviceMonitor: RegisterClassEx failed", "err", err)
		close(dm.ready)
		return
	}
	defer procUnregisterClassW.Call(uintptr(unsafe.Pointer(className)), hInstance)

	winName, _ := syscall.UTF16PtrFromString("RewindDevMonWin")

	hwnd, _, err2 := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(winName)),
		0,
		0, 0, 0, 0,
		0,
		0, hInstance, 0,
	)
	if hwnd == 0 {
		slog.Error("DeviceMonitor: CreateWindowEx failed", "err", err2)
		close(dm.ready)
		return
	}
	defer procDestroyWindow.Call(hwnd)
	dm.hwnd.Store(hwnd)

	notifHandle := dm.registerNotification(hwnd)
	defer func() {
		if notifHandle != 0 {
			procUnregisterDeviceNotification.Call(notifHandle)
		}
	}()

	slog.Info("Device monitor ready")
	close(dm.ready)

	var m winMsg
	for {
		ret, _, _ := procGetMessageDM.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		r := int32(ret)
		if r == 0 {
			break
		}
		if r == -1 {
			slog.Error("DeviceMonitor: GetMessage error")
			break
		}
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (dm *DeviceMonitor) registerNotification(hwnd uintptr) uintptr {
	var filter devBroadcastDevinterface
	filter.Size = uint32(unsafe.Sizeof(filter))
	filter.DeviceType = dbtDevtypDevinterface

	h, _, _ := procRegisterDeviceNotification.Call(
		hwnd,
		uintptr(unsafe.Pointer(&filter)),
		deviceNotifyWindowHandle|deviceNotifyAllInterfaceClasses,
	)
	if h == 0 {
		slog.Warn("DeviceMonitor: RegisterDeviceNotification failed")
	}
	return h
}

// wndProc runs exclusively on the message-loop OS thread.
func (dm *DeviceMonitor) wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmAppQuit:
		procPostQuitMessageDM.Call(0)
		return 0

	case wmDeviceChange:
		switch wParam {
		case dbtDevNodesChanged,
			dbtDeviceArrival,
			dbtDeviceRemoveComplete:
			dm.scheduleCallback()
		}
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func (dm *DeviceMonitor) scheduleCallback() {
	if dm.debounceTimer != nil {
		dm.debounceTimer.Stop()
	}
	cb := dm.cb
	dm.debounceTimer = time.AfterFunc(2*time.Second, func() {
		if cb != nil && !dm.stopped.Load() {
			cb()
		}
	})
}
