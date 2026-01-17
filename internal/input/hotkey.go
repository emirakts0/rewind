package input

import (
	"log/slog"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
)

const (
	ModAlt     = 0x0001
	ModControl = 0x0002
	ModShift   = 0x0004
	ModWin     = 0x0008

	VkF9  = 0x78
	VkF10 = 0x79

	WmHotkey = 0x0312
)

type HotkeyManager struct {
	callbacks map[int]func()
	quit      chan struct{}
}

func NewHotkeyManager() *HotkeyManager {
	return &HotkeyManager{
		callbacks: make(map[int]func()),
		quit:      make(chan struct{}),
	}
}

func (h *HotkeyManager) Register(id int, callback func()) {
	h.callbacks[id] = callback
}

func (h *HotkeyManager) Start() {
	go h.loop()
}

func (h *HotkeyManager) Stop() {
	close(h.quit)
}

func (h *HotkeyManager) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := registerHotKey(0, 1, ModControl, VkF9); err != nil {
		slog.Error("failed to register Ctrl+F9", "error", err)
	} else {
		defer unregisterHotKey(0, 1)
		slog.Info("Registered global hotkey: Ctrl+F9 (Record/Stop)")
	}

	if err := registerHotKey(0, 2, ModControl, VkF10); err != nil {
		slog.Error("failed to register Ctrl+F10", "error", err)
	} else {
		defer unregisterHotKey(0, 2)
		slog.Info("Registered global hotkey: Ctrl+F10 (Save Clip)")
	}

	// Message loop
	var msg msg
	for {
		select {
		case <-h.quit:
			return
		default:
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(ret) == -1 {
				slog.Error("GetMessage failed")
				return
			}
			if int32(ret) == 0 { // WM_QUIT
				return
			}

			if msg.message == WmHotkey {
				id := int(msg.wParam)
				if callback, ok := h.callbacks[id]; ok {
					go callback()
				}
			}

			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func registerHotKey(hwnd uintptr, id int, modifiers int, vk int) error {
	ret, _, err := procRegisterHotKey.Call(
		hwnd,
		uintptr(id),
		uintptr(modifiers),
		uintptr(vk),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func unregisterHotKey(hwnd uintptr, id int) error {
	ret, _, err := procUnregisterHotKey.Call(hwnd, uintptr(id))
	if ret == 0 {
		return err
	}
	return nil
}

type msg struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type point struct {
	x, y int32
}
