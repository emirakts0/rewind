package internal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"syscall"
	"unsafe"
)

// --- DLL Loading ---

var (
	mod                  *syscall.LazyDLL
	procGetMonitors      *syscall.LazyProc
	procListAudioDevices *syscall.LazyProc
	procInit             *syscall.LazyProc
	procSave             *syscall.LazyProc
	procStop             *syscall.LazyProc
	procClear            *syscall.LazyProc
	procFreeString       *syscall.LazyProc
	procGetStatus        *syscall.LazyProc
	procSetLogCallback   *syscall.LazyProc

	// Callback for runtime errors/warnings from Rust
	runtimeErrorCallback func(level string, message string)
)

func init() {
	Load("rewind_api.dll")
	SetupRustLogging()
}

func Load(path string) {
	mod = syscall.NewLazyDLL(path)
	procGetMonitors = mod.NewProc("rewind_get_monitors")
	procListAudioDevices = mod.NewProc("rewind_list_audio_devices")
	procInit = mod.NewProc("rewind_init")
	procSave = mod.NewProc("rewind_save")
	procStop = mod.NewProc("rewind_stop")
	procClear = mod.NewProc("rewind_clear")
	procFreeString = mod.NewProc("rewind_free_string")
	procGetStatus = mod.NewProc("rewind_get_status")
	procSetLogCallback = mod.NewProc("rewind_set_log_callback")
}

// SetRuntimeErrorCallback sets a callback for runtime errors/warnings from Rust
func SetRuntimeErrorCallback(callback func(level string, message string)) {
	runtimeErrorCallback = callback
}

// rustLogCallback is called from Rust code to log messages
func rustLogCallback(level int32, message uintptr) uintptr {
	if message == 0 {
		return 0
	}

	msg := ptrToString(message)

	switch level {
	case 0: // Error
		slog.Error(msg, "source", "rust")
		if runtimeErrorCallback != nil {
			runtimeErrorCallback("error", msg)
		}
	case 1: // Warn
		slog.Warn(msg, "source", "rust")
		if runtimeErrorCallback != nil {
			runtimeErrorCallback("warning", msg)
		}
	case 2: // Info
		slog.Info(msg, "source", "rust")
	case 3: // Debug
		slog.Debug(msg, "source", "rust")
	case 4: // Trace
		slog.Debug(msg, "source", "rust")
	default:
		slog.Info(msg, "source", "rust")
	}

	return 0
}

// SetupRustLogging configures Rust to use Go's logging system
func SetupRustLogging() {
	callback := syscall.NewCallback(rustLogCallback)
	procSetLogCallback.Call(callback)
}

// --- Structs ---

type MonitorInfo struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Width       uint32 `json:"width"`
	Height      uint32 `json:"height"`
	RefreshRate uint32 `json:"refresh_rate"`
}

type AudioDeviceInfo struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	IsInput bool   `json:"is_input"`
}

type AudioConfig struct {
	SampleRate         uint32  `json:"sample_rate"`
	Channels           uint16  `json:"channels"`
	MicEnabled         bool    `json:"mic_enabled"`
	MicDeviceIndex     *int    `json:"mic_device_index"`
	MicVolume          float32 `json:"mic_volume"`
	SpeakerEnabled     bool    `json:"speaker_enabled"`
	SpeakerDeviceIndex *int    `json:"speaker_device_index"`
	SpeakerVolume      float32 `json:"speaker_volume"`
}

type ReplayRecordingConfig struct {
	Width               uint32      `json:"width"`
	Height              uint32      `json:"height"`
	Fps                 uint32      `json:"fps"`
	VideoBitrate        uint32      `json:"video_bitrate"`
	Audio               AudioConfig `json:"audio"`
	BufferDurationSecs  uint64      `json:"buffer_duration_secs"`
	SegmentDurationSecs uint64      `json:"segment_duration_secs"`
	ShowCursor          bool        `json:"show_cursor"`
	ShowBorder          bool        `json:"show_border"`
	FfmpegPath          string      `json:"ffmpeg_path"`
	TempPath            string      `json:"temp_path"`
}

type ReplayStatus struct {
	Duration     float64 `json:"duration_secs"`
	SegmentCount int     `json:"segment_count"`
	IsActive     bool    `json:"is_active"`
	DiskUsage    uint64  `json:"disk_usage_bytes"`
	MemoryUsage  uint64  `json:"memory_usage_bytes"`
}

// --- Helpers ---

func ptrToString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	var b []byte
	p := unsafe.Pointer(ptr)
	for {
		val := *(*byte)(p)
		if val == 0 {
			break
		}
		b = append(b, val)
		p = unsafe.Pointer(uintptr(p) + 1)
	}
	return string(b)
}

func stringToPtr(s string) uintptr {
	if !strings.HasSuffix(s, "\x00") {
		s += "\x00"
	}
	b := []byte(s)
	return uintptr(unsafe.Pointer(&b[0]))
}

// --- API Functions ---

type MonitorsResult struct {
	Success bool             `json:"success"`
	Data    *json.RawMessage `json:"data"`
	Error   *string          `json:"error"`
}

func GetMonitors() ([]MonitorInfo, error) {
	ptr, _, _ := procGetMonitors.Call()
	if ptr == 0 {
		return nil, fmt.Errorf("failed to get monitors: no response from native library")
	}
	defer procFreeString.Call(ptr)

	jsonStr := ptrToString(ptr)
	var result MonitorsResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse monitors result: %v", err)
	}

	if !result.Success {
		if result.Error != nil {
			return nil, fmt.Errorf(*result.Error)
		}
		return nil, fmt.Errorf("failed to get monitors: unknown error")
	}

	if result.Data == nil {
		return []MonitorInfo{}, nil
	}

	var monitors []MonitorInfo
	if err := json.Unmarshal(*result.Data, &monitors); err != nil {
		return nil, fmt.Errorf("failed to parse monitors data: %v", err)
	}
	return monitors, nil
}

type AudioDevicesResult struct {
	Success bool             `json:"success"`
	Data    *json.RawMessage `json:"data"`
	Error   *string          `json:"error"`
}

func ListAudioDevices() ([]AudioDeviceInfo, error) {
	ptr, _, _ := procListAudioDevices.Call()
	if ptr == 0 {
		return nil, fmt.Errorf("failed to list audio devices: no response from native library")
	}
	defer procFreeString.Call(ptr)

	jsonStr := ptrToString(ptr)
	var result AudioDevicesResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse audio devices result: %v", err)
	}

	if !result.Success {
		if result.Error != nil {
			return nil, fmt.Errorf(*result.Error)
		}
		return nil, fmt.Errorf("failed to list audio devices: unknown error")
	}

	if result.Data == nil {
		return []AudioDeviceInfo{}, nil
	}

	var devices []AudioDeviceInfo
	if err := json.Unmarshal(*result.Data, &devices); err != nil {
		return nil, fmt.Errorf("failed to parse audio devices data: %v", err)
	}
	return devices, nil
}

type InitResult struct {
	Success bool    `json:"success"`
	Handle  uintptr `json:"handle"`
	Error   *string `json:"error"`
}

type Handle uintptr

func InitReplayBuffer(monitorIndex int, config ReplayRecordingConfig) (Handle, error) {
	configJson, err := json.Marshal(config)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal config: %w", err)
	}

	ptr := stringToPtr(string(configJson))

	resultPtr, _, _ := procInit.Call(
		uintptr(uint32(monitorIndex)),
		ptr,
	)

	if resultPtr == 0 {
		return 0, fmt.Errorf("failed to initialize replay buffer: no response from native library")
	}
	defer procFreeString.Call(resultPtr)

	jsonStr := ptrToString(resultPtr)
	var result InitResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return 0, fmt.Errorf("failed to parse init result: %v", err)
	}

	if !result.Success {
		if result.Error != nil {
			return 0, fmt.Errorf("%s", *result.Error)
		}
		return 0, fmt.Errorf("initialization failed: unknown error")
	}

	return Handle(result.Handle), nil
}

type SaveResult struct {
	Success bool    `json:"success"`
	Error   *string `json:"error"`
}

func (h Handle) Save(path string) error {
	pathPtr := stringToPtr(path)
	ptr, _, _ := procSave.Call(uintptr(h), pathPtr)
	if ptr == 0 {
		return fmt.Errorf("save failed: no response from native library")
	}
	defer procFreeString.Call(ptr)

	jsonStr := ptrToString(ptr)
	var result SaveResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return fmt.Errorf("failed to parse save result: %v", err)
	}

	if !result.Success {
		if result.Error != nil {
			return fmt.Errorf("%s", *result.Error)
		}
		return fmt.Errorf("save failed: unknown error")
	}

	return nil
}

func (h Handle) Stop() {
	procStop.Call(uintptr(h))
}

type ClearResult struct {
	Success bool    `json:"success"`
	Error   *string `json:"error"`
}

func (h Handle) Clear() error {
	ptr, _, _ := procClear.Call(uintptr(h))
	if ptr == 0 {
		return fmt.Errorf("failed to clear segments: no response from native library")
	}
	defer procFreeString.Call(ptr)

	jsonStr := ptrToString(ptr)
	var result ClearResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return fmt.Errorf("failed to parse clear result: %v", err)
	}

	if !result.Success {
		if result.Error != nil {
			return fmt.Errorf(*result.Error)
		}
		return fmt.Errorf("failed to clear segments: unknown error")
	}

	return nil
}

type StatusResult struct {
	Success bool          `json:"success"`
	Data    *ReplayStatus `json:"data"`
	Error   *string       `json:"error"`
}

func (h Handle) GetStatus() (*ReplayStatus, error) {
	ptr, _, _ := procGetStatus.Call(uintptr(h))
	if ptr == 0 {
		return nil, fmt.Errorf("failed to get status: no response from native library")
	}
	defer procFreeString.Call(ptr)

	jsonStr := ptrToString(ptr)
	var result StatusResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse status result: %v", err)
	}

	if !result.Success {
		if result.Error != nil {
			return nil, fmt.Errorf(*result.Error)
		}
		return nil, fmt.Errorf("failed to get status: unknown error")
	}

	if result.Data == nil {
		return nil, fmt.Errorf("status data is null")
	}

	return result.Data, nil
}
