package utils

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32        = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

// SingleInstanceMutex holds a Windows mutex to prevent multiple instances
type SingleInstanceMutex struct {
	handle windows.Handle
}

// AcquireSingleInstance creates a named mutex to ensure only one instance runs
func AcquireSingleInstance(name string) (*SingleInstanceMutex, error) {
	mutexName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("failed to convert mutex name: %w", err)
	}

	ret, _, err := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(mutexName)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("failed to create mutex: %w", err)
	}

	handle := windows.Handle(ret)

	if errors.Is(err, syscall.ERROR_ALREADY_EXISTS) {
		windows.CloseHandle(handle)
		return nil, fmt.Errorf("another instance is already running")
	}

	return &SingleInstanceMutex{handle: handle}, nil
}

func (m *SingleInstanceMutex) Release() {
	if m.handle != 0 {
		windows.CloseHandle(m.handle)
		m.handle = 0
	}
}
