//go:build windows

package hardware

import (
	"bufio"
	"fmt"
	"log/slog"
	"regexp"
	"rewind/internal/utils"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type Display struct {
	Index        int
	Name         string
	FriendlyName string
	IsPrimary    bool
	Width        int
	Height       int
	X, Y         int
	GPUIndex     int
}

func (d *Display) String() string {
	primary := ""
	if d.IsPrimary {
		primary = " [primary]"
	}
	return fmt.Sprintf("[%d] %dx%d%s", d.Index, d.Width, d.Height, primary)
}

type DisplayList []*Display

func (l DisplayList) FindByIndex(index int) *Display {
	for _, d := range l {
		if d.Index == index {
			return d
		}
	}
	return nil
}

func (l DisplayList) FindPrimary() *Display {
	for _, d := range l {
		if d.IsPrimary {
			return d
		}
	}
	if len(l) > 0 {
		return l[0]
	}
	return nil
}

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type monitorInfoExW struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
	SzDevice  [32]uint16
}

const monitorInfoFPrimary = 0x00000001

func DetectDisplays() (DisplayList, error) {
	displays, err := detectDisplaysFromDDAGrab()
	if err != nil && len(displays) <= 0 {
		return nil, fmt.Errorf("ddagrab display detection failed: %w", err)
	}

	enrichPrimaryStatus(displays)

	for _, d := range displays {
		d.GPUIndex = GetMonitorGPUIndex(d.Index)
	}

	for _, d := range displays {
		slog.Info("detected display",
			"index", d.Index,
			"resolution", fmt.Sprintf("%dx%d", d.Width, d.Height),
			"primary", d.IsPrimary,
			"name", d.Name,
		)
	}

	return displays, nil
}

// detectDisplaysFromDDAGrab probes each output_idx using FFmpeg
func detectDisplaysFromDDAGrab() (DisplayList, error) {
	var displays DisplayList

	for idx := 0; idx < 16; idx++ {
		info, err := probeOutputIndex(idx)
		if err != nil {
			break
		}
		if info == nil {
			break
		}
		info.Index = idx
		slog.Debug("ddagrab probe", "output_idx", idx, "resolution", fmt.Sprintf("%dx%d", info.Width, info.Height))
		displays = append(displays, info)
	}

	if len(displays) == 0 {
		return nil, nil
	}

	return displays, nil
}

func probeOutputIndex(idx int) (*Display, error) {
	cmd := utils.Command(FFmpegPath,
		"-hide_banner",
		"-f", "lavfi",
		"-i", "ddagrab=output_idx="+strconv.Itoa(idx)+":framerate=1",
		"-frames:v", "1",
		"-f", "null",
		"NUL",
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var width, height int
	scanner := bufio.NewScanner(stderr)
	resRegex := regexp.MustCompile(`(\d+)x(\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Video:") && strings.Contains(line, "d3d11") {
			matches := resRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				width, _ = strconv.Atoi(matches[1])
				height, _ = strconv.Atoi(matches[2])
				break
			}
		}
	}

	cmd.Wait()

	if width == 0 || height == 0 {
		return nil, nil
	}

	return &Display{
		Index:        idx,
		Width:        width,
		Height:       height,
		FriendlyName: "Display " + strconv.Itoa(idx+1),
	}, nil
}

func enrichPrimaryStatus(displays DisplayList) {
	type winDisplay struct {
		width, height int
		isPrimary     bool
		deviceName    string
	}

	var winDisplays []winDisplay

	callback := syscall.NewCallback(func(hMonitor uintptr, hdc uintptr, lprcClip uintptr, lParam uintptr) uintptr {
		var info monitorInfoExW
		info.CbSize = uint32(unsafe.Sizeof(info))

		ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&info)))
		if ret != 0 {
			wd := winDisplay{
				width:      int(info.RcMonitor.Right - info.RcMonitor.Left),
				height:     int(info.RcMonitor.Bottom - info.RcMonitor.Top),
				isPrimary:  info.DwFlags&monitorInfoFPrimary != 0,
				deviceName: syscall.UTF16ToString(info.SzDevice[:]),
			}
			winDisplays = append(winDisplays, wd)
		}
		return 1
	})

	procEnumDisplayMonitors.Call(0, 0, callback, 0)

	for _, d := range displays {
		for _, wd := range winDisplays {
			if d.Width == wd.width && d.Height == wd.height {
				d.IsPrimary = wd.isPrimary
				d.Name = wd.deviceName
				break
			}
		}
	}
}
