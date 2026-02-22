package internal

import (
	"log/slog"
	"syscall"
	"time"
	"unsafe"
)

const (
	wsExToolWindow uintptr = 0x00000080
	wsExAppWindow  uintptr = 0x00040000

	spiGetWorkArea = 0x0030
)

var gwlExStylePtr = ^uintptr(19)

var procSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")

type rect struct {
	Left, Top, Right, Bottom int32
}

type WorkArea struct {
	X      int
	Y      int
	Width  int
	Height int
}

func getWorkArea() (WorkArea, error) {
	var r rect
	ret, _, err := procSystemParametersInfoW.Call(
		spiGetWorkArea,
		0,
		uintptr(unsafe.Pointer(&r)),
		0,
	)
	if ret == 0 {
		return WorkArea{}, err
	}

	return WorkArea{
		X:      int(r.Left),
		Y:      int(r.Top),
		Width:  int(r.Right - r.Left),
		Height: int(r.Bottom - r.Top),
	}, nil
}

func hideWindowFromTaskbar(title string) {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		slog.Warn("failed to convert window title", "error", err)
		return
	}

	for i := 0; i < 20; i++ {
		time.Sleep(250 * time.Millisecond)

		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
		if hwnd == 0 {
			continue
		}

		exStyle, _, _ := procGetWindowLongW.Call(hwnd, gwlExStylePtr)

		newStyle := (exStyle &^ wsExAppWindow) | wsExToolWindow
		procSetWindowLongW.Call(hwnd, gwlExStylePtr, newStyle)

		slog.Debug("notification window hidden from taskbar")
		return
	}

	slog.Warn("could not find notification window to hide from taskbar")
}
