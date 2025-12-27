//go:build windows

package hardware

import (
	"os/exec"
	"syscall"
)

// Command creates a command that won't show a console window on Windows
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd
}
