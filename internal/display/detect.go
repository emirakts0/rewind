//go:build windows

package display

import (
	"bufio"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
)

// RECT structure
type rect struct {
	Left, Top, Right, Bottom int32
}

// MONITORINFOEXW structure
type monitorInfoExW struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
	SzDevice  [32]uint16
}

const MONITORINFOF_PRIMARY = 0x00000001

// FFmpegPath is the path to FFmpeg executable (set by main)
var FFmpegPath = "bin/ffmpeg.exe"

// DetectDisplays returns a list of all active displays.
// It uses FFmpeg's ddagrab to get accurate display information that matches
// the output_idx parameter used for capture.
func DetectDisplays() (DisplayList, error) {
	// First, try to detect displays using FFmpeg's ddagrab
	// This gives us the exact order that ddagrab will use
	displays, err := detectDisplaysFromDDAGrab()
	if err == nil && len(displays) > 0 {
		return displays, nil
	}

	// Fallback to Windows API if ddagrab detection fails
	return detectDisplaysFromWindowsAPI()
}

// detectDisplaysFromDDAGrab probes each output_idx to find all displays
func detectDisplaysFromDDAGrab() (DisplayList, error) {
	var displays DisplayList

	// Try output_idx from 0 to 15 (should be more than enough)
	for idx := 0; idx < 16; idx++ {
		info, err := probeOutputIndex(idx)
		if err != nil {
			break // Error starting FFmpeg, stop probing
		}
		if info == nil {
			break // No display at this index, stop probing
		}
		info.Index = idx
		displays = append(displays, info)
	}

	if len(displays) == 0 {
		return nil, nil
	}

	// Try to enrich with Windows API data (names, primary status)
	enrichFromWindowsAPI(displays)

	return displays, nil
}

// probeOutputIndex uses FFmpeg to get information about a specific output
func probeOutputIndex(idx int) (*Display, error) {
	cmd := exec.Command(FFmpegPath,
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

	// Parse output for resolution
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
		RefreshRate:  60, // Will be enriched later
		FriendlyName: "Display " + strconv.Itoa(idx+1),
		GPUIndex:     -1,
	}, nil
}

// enrichFromWindowsAPI adds additional info from Windows API
func enrichFromWindowsAPI(displays DisplayList) {
	// Get primary display and try to match by resolution
	type winDisplay struct {
		width, height int
		isPrimary     bool
		deviceName    string
	}

	var winDisplays []winDisplay

	// Enumerate Windows monitors
	callback := syscall.NewCallback(func(hMonitor uintptr, hdc uintptr, lprcClip uintptr, lParam uintptr) uintptr {
		var info monitorInfoExW
		info.CbSize = uint32(unsafe.Sizeof(info))

		ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&info)))
		if ret != 0 {
			wd := winDisplay{
				width:      int(info.RcMonitor.Right - info.RcMonitor.Left),
				height:     int(info.RcMonitor.Bottom - info.RcMonitor.Top),
				isPrimary:  info.DwFlags&MONITORINFOF_PRIMARY != 0,
				deviceName: syscall.UTF16ToString(info.SzDevice[:]),
			}
			winDisplays = append(winDisplays, wd)
		}
		return 1
	})

	procEnumDisplayMonitors.Call(0, 0, callback, 0)

	// Match displays by resolution to get primary status
	for _, d := range displays {
		for _, wd := range winDisplays {
			if d.Width == wd.width && d.Height == wd.height {
				d.IsPrimary = wd.isPrimary
				d.Name = wd.deviceName
				break
			}
		}
	}

	// Get GPU information and determine which GPU handles displays
	assignGPUToDisplays(displays)
}

// assignGPUToDisplays determines which GPU each display is connected to.
// In Optimus/Hybrid systems, usually the iGPU handles all display output
// even when the dGPU does the rendering.
func assignGPUToDisplays(displays DisplayList) {
	gpuList := getGPUList()
	if len(gpuList) == 0 {
		return
	}

	// Check if NVIDIA has any active displays (for Optimus detection)
	nvidiaHasDisplay := checkNVIDIADisplayActive()

	// Categorize GPUs
	var iGPU *gpuInfo // Integrated GPU
	var dGPU *gpuInfo // Discrete GPU (NVIDIA, AMD discrete, Intel Arc)

	for i := range gpuList {
		if isIntegratedGPU(gpuList[i].name) {
			iGPU = &gpuList[i]
		} else {
			// Any non-integrated GPU is discrete
			dGPU = &gpuList[i]
		}
	}

	// Determine which GPU handles display output
	var displayGPU *gpuInfo

	// Case 1: Only one GPU in the system (most common desktop case)
	if len(gpuList) == 1 {
		displayGPU = &gpuList[0]
	} else if iGPU != nil && dGPU != nil {
		// Case 2: Hybrid system (iGPU + dGPU)
		// Check if NVIDIA dGPU has active displays (MUX switch enabled)
		if nvidiaHasDisplay && strings.Contains(strings.ToLower(dGPU.name), "nvidia") {
			displayGPU = dGPU
		} else {
			// Optimus/Hybrid mode: iGPU handles all display output
			displayGPU = iGPU
		}
	} else if dGPU != nil {
		// Case 3: Only discrete GPU (desktop with NVIDIA/AMD/Intel Arc)
		displayGPU = dGPU
	} else if iGPU != nil {
		// Case 4: Only integrated GPU
		displayGPU = iGPU
	} else if len(gpuList) > 0 {
		// Case 5: Fallback
		displayGPU = &gpuList[0]
	}

	if displayGPU != nil {
		for _, d := range displays {
			d.GPUIndex = displayGPU.index
			d.GPUName = displayGPU.name
		}
	}

	// Match refresh rates by resolution from all GPUs
	// This handles cases where Windows reports different refresh rates per GPU
	for _, d := range displays {
		for _, gi := range gpuList {
			if d.Width == gi.width && d.Height == gi.height && gi.refreshRate > 0 {
				d.RefreshRate = gi.refreshRate
				break
			}
		}
	}
}

// checkNVIDIADisplayActive checks if NVIDIA GPU has any active display connections
func checkNVIDIADisplayActive() bool {
	cmd := exec.Command("nvidia-smi", "--query-gpu=display_active", "--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	result := strings.TrimSpace(string(out))
	// "Enabled" means NVIDIA has active displays, "Disabled" means Optimus mode
	return strings.EqualFold(result, "enabled")
}

type gpuInfo struct {
	index       int
	name        string
	width       int
	height      int
	refreshRate int
}

func getGPUList() []gpuInfo {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`$i = 0; Get-CimInstance Win32_VideoController | ForEach-Object { 
			Write-Output "$i|$($_.Name)|$($_.CurrentHorizontalResolution)|$($_.CurrentVerticalResolution)|$($_.CurrentRefreshRate)"
			$i++
		}`)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var result []gpuInfo
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 5 {
			idx, _ := strconv.Atoi(parts[0])
			w, _ := strconv.Atoi(parts[2])
			h, _ := strconv.Atoi(parts[3])
			r, _ := strconv.Atoi(parts[4])
			result = append(result, gpuInfo{
				index:       idx,
				name:        parts[1],
				width:       w,
				height:      h,
				refreshRate: r,
			})
		}
	}
	return result
}

func isIntegratedGPU(name string) bool {
	nameLower := strings.ToLower(name)

	// Intel integrated
	if strings.Contains(nameLower, "intel") {
		if strings.Contains(nameLower, "arc") {
			return false // Intel Arc is discrete
		}
		return true
	}

	// AMD integrated (APU)
	if strings.Contains(nameLower, "radeon graphics") ||
		strings.Contains(nameLower, "radeon(tm) graphics") {
		return true
	}
	if strings.Contains(nameLower, "vega") && !strings.Contains(nameLower, "rx vega") {
		return true
	}

	return false
}

func enrichRefreshRates(displays DisplayList) {
	// Try to get per-display refresh rates
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class DisplayInfo {
    [DllImport("user32.dll")]
    public static extern bool EnumDisplaySettings(string deviceName, int modeNum, ref DEVMODE devMode);
    
    [StructLayout(LayoutKind.Sequential)]
    public struct DEVMODE {
        [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 32)]
        public string dmDeviceName;
        public short dmSpecVersion;
        public short dmDriverVersion;
        public short dmSize;
        public short dmDriverExtra;
        public int dmFields;
        public int dmPositionX;
        public int dmPositionY;
        public int dmDisplayOrientation;
        public int dmDisplayFixedOutput;
        public short dmColor;
        public short dmDuplex;
        public short dmYResolution;
        public short dmTTOption;
        public short dmCollate;
        [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 32)]
        public string dmFormName;
        public short dmLogPixels;
        public int dmBitsPerPel;
        public int dmPelsWidth;
        public int dmPelsHeight;
        public int dmDisplayFlags;
        public int dmDisplayFrequency;
    }
}
"@ -ErrorAction SilentlyContinue

$monitors = Get-CimInstance -Namespace root\wmi -ClassName WmiMonitorBasicDisplayParams -ErrorAction SilentlyContinue
foreach ($m in $monitors) {
    $devMode = New-Object DisplayInfo+DEVMODE
    $devMode.dmSize = [System.Runtime.InteropServices.Marshal]::SizeOf($devMode)
    if ([DisplayInfo]::EnumDisplaySettings($null, -1, [ref]$devMode)) {
        Write-Output "$($devMode.dmPelsWidth)|$($devMode.dmPelsHeight)|$($devMode.dmDisplayFrequency)"
    }
}`)
	out, err := cmd.Output()
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			w, _ := strconv.Atoi(parts[0])
			h, _ := strconv.Atoi(parts[1])
			r, _ := strconv.Atoi(parts[2])

			// Match to display by resolution
			for _, d := range displays {
				if d.Width == w && d.Height == h && r > 0 {
					d.RefreshRate = r
					break
				}
			}
		}
	}
}

// detectDisplaysFromWindowsAPI is the fallback method
func detectDisplaysFromWindowsAPI() (DisplayList, error) {
	var displays DisplayList
	idx := 0

	callback := syscall.NewCallback(func(hMonitor uintptr, hdc uintptr, lprcClip uintptr, lParam uintptr) uintptr {
		var info monitorInfoExW
		info.CbSize = uint32(unsafe.Sizeof(info))

		ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&info)))
		if ret == 0 {
			return 1
		}

		deviceName := syscall.UTF16ToString(info.SzDevice[:])

		display := &Display{
			Index:        idx,
			Name:         deviceName,
			FriendlyName: deviceName,
			IsPrimary:    info.DwFlags&MONITORINFOF_PRIMARY != 0,
			Width:        int(info.RcMonitor.Right - info.RcMonitor.Left),
			Height:       int(info.RcMonitor.Bottom - info.RcMonitor.Top),
			X:            int(info.RcMonitor.Left),
			Y:            int(info.RcMonitor.Top),
			RefreshRate:  60,
			GPUIndex:     -1,
		}

		displays = append(displays, display)
		idx++

		return 1
	})

	procEnumDisplayMonitors.Call(0, 0, callback, 0)

	// Assign GPU info
	if len(displays) > 0 {
		assignGPUToDisplays(displays)
	}

	return displays, nil
}

// AssociateWithGPUs links displays to their corresponding GPUs.
func AssociateWithGPUs(displays DisplayList) error {
	assignGPUToDisplays(displays)
	return nil
}
