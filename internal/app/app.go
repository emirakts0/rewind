package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"runtime/debug"
	"sync"
	"time"

	"rewind/internal/audio"
	"rewind/internal/buffer"
	"rewind/internal/capture"
	"rewind/internal/hardware"
	"rewind/internal/utils"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Status represents the current application state
type Status string

const (
	StatusIdle      Status = "idle"
	StatusRecording Status = "recording"
	StatusSaving    Status = "saving"
	StatusError     Status = "error"
)

// Config represents user-configurable settings
type Config struct {
	DisplayIndex      int    `json:"displayIndex"`
	EncoderName       string `json:"encoderName"`
	FPS               int    `json:"fps"`
	Bitrate           string `json:"bitrate"`
	RecordSeconds     int    `json:"recordSeconds"`
	OutputDir         string `json:"outputDir"`
	ConvertToMP4      bool   `json:"convertToMP4"`
	MicrophoneDevice  string `json:"microphoneDevice"`
	MicVolume         int    `json:"micVolume"` // 0-200
	SystemAudioDevice string `json:"systemAudioDevice"`
	SysVolume         int    `json:"sysVolume"` // 0-200
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	// Get default clips directory from user's AppData
	outputDir, err := utils.GetClipsDir()
	if err != nil {
		// Fallback to current directory if AppData is not available
		outputDir = "./clips"
	}

	return Config{
		DisplayIndex:      0,
		EncoderName:       "", // auto-select
		FPS:               30,
		Bitrate:           "15M",
		RecordSeconds:     30,
		OutputDir:         outputDir,
		ConvertToMP4:      true,
		MicrophoneDevice:  "",
		MicVolume:         100,
		SystemAudioDevice: "",
		SysVolume:         100,
	}
}

// State holds the current application state
type State struct {
	Status       Status `json:"status"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	BufferUsage  int    `json:"bufferUsage"`  // percentage 0-100
	RecordingFor int    `json:"recordingFor"` // seconds since recording started
}

// App is the main application service for Wails binding
type App struct {
	mu  sync.RWMutex
	ctx context.Context

	// Wails v3 application instance
	app *application.App

	// Configuration
	config     Config
	ffmpegPath string

	// Hardware info (detected once)
	sysInfo *hardware.SystemInfo

	// Runtime state
	state        State
	capturer     *capture.Capturer
	audioManager *audio.CaptureManager
	ringBuffer   *buffer.Buffer
	saver        *capture.Saver
	startTime    time.Time
	lastSaveTime time.Time

	// Event callbacks (legacy - kept for compatibility)
	OnStateChange func(state State)
	OnClipSaved   func(filename string)

	// Tray state change callback
	onTrayStateChange func(State)
}

// New creates a new App instance
func New(ffmpegPath string) *App {
	app := &App{
		config:     DefaultConfig(),
		ffmpegPath: ffmpegPath,
		state:      State{Status: StatusIdle},
	}

	// Load saved config (if exists)
	if err := app.LoadConfig(); err != nil {
		slog.Warn("failed to load config", "error", err)
	}

	return app
}

// SetApp stores the Wails application instance for event emission
func (a *App) SetApp(app *application.App) {
	a.app = app
}

// SetOnStateChange sets a callback for tray state updates
func (a *App) SetOnStateChange(callback func(State)) {
	a.onTrayStateChange = callback
}

// ServiceStartup is called when the Wails v3 app starts (lifecycle hook)
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.ctx = ctx
	slog.Info("Rewind service starting up...")

	// Initialize the app
	if err := a.Initialize(); err != nil {
		slog.Error("Failed to initialize", "error", err)
		return err
	}

	return nil
}

// ServiceShutdown is called when the Wails v3 app is closing (lifecycle hook)
func (a *App) ServiceShutdown() error {
	slog.Info("Rewind service shutting down...")

	// Stop recording if active
	if a.IsRecording() {
		a.Stop()
	}

	// Clear buffer
	if a.ringBuffer != nil {
		a.ringBuffer.Clear()
	}

	return nil
}

// Initialize detects hardware and prepares the app
// Safe to call multiple times - will skip if already initialized
func (a *App) Initialize() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Skip if already initialized
	if a.sysInfo != nil {
		slog.Info("app already initialized, skipping")
		return nil
	}

	hardware.FFmpegPath = a.ffmpegPath

	sysInfo, err := hardware.Detect()
	if err != nil {
		return fmt.Errorf("hardware detection failed: %w", err)
	}

	a.sysInfo = sysInfo

	// Auto-select encoder if not set
	if a.config.EncoderName == "" {
		a.config.EncoderName = hardware.FindBestEncoder(sysInfo.Encoders).Name
	}

	slog.Info("app initialized",
		"displays", len(sysInfo.Displays),
		"encoders", len(sysInfo.GetAvailableEncoders()),
	)

	return nil
}

// GetDisplays returns all available displays
func (a *App) GetDisplays() []DisplayInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.sysInfo == nil {
		return nil
	}

	var displays []DisplayInfo
	for _, d := range a.sysInfo.Displays {
		displays = append(displays, DisplayInfo{
			Index:       d.Index,
			Name:        d.FriendlyName,
			Width:       d.Width,
			Height:      d.Height,
			RefreshRate: d.RefreshRate,
			IsPrimary:   d.IsPrimary,
		})
	}
	return displays
}

// GetEncodersForDisplay returns available encoders for a specific display
func (a *App) GetEncodersForDisplay(displayIndex int) []EncoderInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.sysInfo == nil {
		return nil
	}

	encoders := a.sysInfo.GetEncodersForDisplay(displayIndex)
	var result []EncoderInfo

	for _, e := range encoders {
		gpuName := "CPU"
		if e.GPUIndex >= 0 {
			if gpu := a.sysInfo.GPUs.FindByIndex(e.GPUIndex); gpu != nil {
				gpuName = gpu.Name
			}
		}

		result = append(result, EncoderInfo{
			Name:    e.Name,
			Codec:   e.Codec,
			GPUName: gpuName,
		})
	}

	return result
}

// GetInputDevices returns input (microphone) devices
func (a *App) GetInputDevices() []string {
	devices, err := audio.ListInputDevices()
	if err != nil {
		slog.Error("failed to list input devices", "error", err)
		return nil
	}

	var names []string
	for _, d := range devices {
		names = append(names, d.Name)
	}
	return names
}

// GetOutputDevices returns output (playback) devices for loopback capture
func (a *App) GetOutputDevices() []string {
	devices, err := audio.ListOutputDevices()
	if err != nil {
		slog.Error("failed to list output devices", "error", err)
		return nil
	}

	var names []string
	for _, d := range devices {
		names = append(names, d.Name)
	}
	return names
}

// GetConfig returns the current configuration
func (a *App) GetConfig() Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// SetConfig updates the configuration (only when not recording)
func (a *App) SetConfig(cfg Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status == StatusRecording {
		return fmt.Errorf("cannot change config while recording")
	}

	// Validate
	if cfg.FPS <= 0 || cfg.FPS > 240 {
		return fmt.Errorf("FPS must be between 1 and 240")
	}
	if cfg.RecordSeconds <= 0 {
		return fmt.Errorf("record seconds must be positive")
	}

	// Validate display exists
	if a.sysInfo != nil && a.sysInfo.GetDisplay(cfg.DisplayIndex) == nil {
		return fmt.Errorf("display not found: %d", cfg.DisplayIndex)
	}

	// Validate encoder exists
	if cfg.EncoderName != "" && a.sysInfo != nil {
		if a.sysInfo.GetEncoder(cfg.EncoderName) == nil {
			return fmt.Errorf("encoder not found: %s", cfg.EncoderName)
		}
	}

	a.config = cfg
	slog.Info("config updated", "config", cfg)

	// Save config to file (use helper to avoid mutex deadlock)
	if err := saveConfigToFile(cfg); err != nil {
		slog.Warn("failed to save config", "error", err)
	}

	return nil
}

// GetState returns the current state
func (a *App) GetState() State {
	a.mu.RLock()
	defer a.mu.RUnlock()

	state := a.state
	if a.ringBuffer != nil && a.state.Status == StatusRecording {
		total := a.ringBuffer.Size()
		used := a.ringBuffer.Len()
		if total > 0 {
			state.BufferUsage = (used * 100) / total
		}
		state.RecordingFor = int(time.Since(a.startTime).Seconds())
	}
	return state
}

// Start begins recording
func (a *App) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status == StatusRecording {
		return fmt.Errorf("already recording")
	}

	if a.sysInfo == nil {
		return fmt.Errorf("not initialized")
	}

	// Ensure OutputDir is absolute and create it
	absDir, err := utils.ResolveAbsPath(a.config.OutputDir, "")
	if err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}
	a.config.OutputDir = absDir

	if err := os.MkdirAll(a.config.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build capture config
	captureCfg := capture.DefaultConfig()
	captureCfg.DisplayIndex = a.config.DisplayIndex
	captureCfg.EncoderName = a.config.EncoderName
	captureCfg.FPS = a.config.FPS
	captureCfg.Bitrate = a.config.Bitrate
	captureCfg.RecordSeconds = a.config.RecordSeconds
	captureCfg.OutputDir = a.config.OutputDir
	captureCfg.FFmpegPath = a.ffmpegPath
	captureCfg.MicrophoneDevice = a.config.MicrophoneDevice
	captureCfg.SystemAudioDevice = a.config.SystemAudioDevice

	if err := captureCfg.Resolve(a.sysInfo); err != nil {
		return fmt.Errorf("config resolution failed: %w", err)
	}

	// Create components
	bufSize := capture.CalculateBufferSize(a.config.Bitrate, a.config.RecordSeconds)
	a.ringBuffer = buffer.New(bufSize)
	a.saver = capture.NewSaver(a.ffmpegPath, a.config.OutputDir)

	capturer, err := capture.NewCapturer(captureCfg)
	if err != nil {
		return fmt.Errorf("failed to create capturer: %w", err)
	}

	capturer.OnData = func(data []byte) {
		a.ringBuffer.Write(data)
	}

	capturer.OnError = func(err error) {
		slog.Warn("capture error", "error", err)
	}

	if err := capturer.Start(); err != nil {
		return fmt.Errorf("failed to start capture: %w", err)
	}

	a.capturer = capturer
	a.startTime = time.Now()
	a.setState(StatusRecording, "")

	slog.Info("recording started",
		"display", a.config.DisplayIndex,
		"encoder", a.config.EncoderName,
		"outputDir", a.config.OutputDir,
	)

	if a.config.MicrophoneDevice != "" || a.config.SystemAudioDevice != "" {
		micID, _ := audio.FindDeviceIDByName(a.config.MicrophoneDevice)
		sysID, _ := audio.FindDeviceIDByName(a.config.SystemAudioDevice)

		if micID != "" || sysID != "" {
			am, err := audio.NewCaptureManager()
			if err != nil {
				slog.Error("failed to create audio manager", "error", err)
			} else {
				a.audioManager = am
				if err := am.StartCapture(micID, sysID, a.config.MicVolume, a.config.SysVolume, a.config.RecordSeconds); err != nil {
					slog.Error("failed to start audio capture", "error", err)
					a.audioManager.Close()
					a.audioManager = nil
				}
			}
		}
	}

	return nil
}

// Stop stops recording
func (a *App) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status != StatusRecording {
		return fmt.Errorf("not recording")
	}

	if a.capturer != nil {
		a.capturer.Stop()
		a.capturer = nil
	}

	if a.audioManager != nil {
		a.audioManager.Close()
		a.audioManager = nil
	}

	// Release memory immediately
	a.ringBuffer = nil
	stdruntime.GC()
	debug.FreeOSMemory()

	a.setState(StatusIdle, "")
	slog.Info("recording stopped")
	return nil
}

// SaveClip saves the current buffer as a clip
func (a *App) SaveClip() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status != StatusRecording {
		return "", fmt.Errorf("not recording")
	}

	// Debounce
	if time.Since(a.lastSaveTime) < 3*time.Second {
		return "", fmt.Errorf("please wait before saving another clip")
	}

	if a.ringBuffer == nil || a.saver == nil {
		return "", fmt.Errorf("not initialized")
	}

	filename := fmt.Sprintf("clip_%s", time.Now().Format("20060102_150405"))
	opts := capture.DefaultSaveOptions(filename)
	opts.ConvertToMP4, opts.DeleteTS = a.config.ConvertToMP4, a.config.ConvertToMP4
	opts.DurationSec = a.config.RecordSeconds

	var audioSrc capture.Snapshotter
	if a.audioManager != nil && a.audioManager.IsRunning() {
		audioSrc = a.audioManager.GetBuffer()
	}

	if err := a.saver.SaveWithAudio(a.ringBuffer, audioSrc, opts); err != nil {
		return "", fmt.Errorf("save failed: %w", err)
	}

	a.lastSaveTime = time.Now()

	ext := "/"
	if a.config.ConvertToMP4 {
		ext = ".mp4"
	}

	if a.OnClipSaved != nil {
		go a.OnClipSaved(filename + ext)
	}

	slog.Info("clip saved", "filename", filename)
	return filename + ext, nil
}

// IsRecording returns true if currently recording
func (a *App) IsRecording() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state.Status == StatusRecording
}

func (a *App) SelectDirectory() (string, error) {
	slog.Info("SelectDirectory called")

	if a.app == nil {
		return "", fmt.Errorf("application not initialized")
	}

	selection, err := a.app.Dialog.OpenFile().
		SetTitle("Select Output Directory").
		SetDirectory(a.config.OutputDir).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()

	if err != nil {
		return "", err
	}

	return selection, nil
}

// EstimateMemory calculates the estimated buffer size based on bitrate and duration
func (a *App) EstimateMemory(bitrate string, seconds int, hasMic bool, hasSys bool) string {
	videoSize := capture.CalculateBufferSize(bitrate, seconds)

	audioSize := 0
	activeStreams := 0

	if hasMic {
		activeStreams++
		audioSize += audio.CalculateStreamBufferSize(2)
	}
	if hasSys {
		activeStreams++
		audioSize += audio.CalculateStreamBufferSize(2)
	}

	if activeStreams > 0 {
		audioSize += audio.CalculateMixedBufferSize(seconds)
	}

	totalSize := videoSize + audioSize

	// Convert to MegaBytes (1024^2)
	mb := float64(totalSize) / (1024 * 1024)
	return fmt.Sprintf("~%.0fMB", mb)
}

// Clip represents a saved video file or raw clip folder
type Clip struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"modTime"`
	IsRawFolder bool      `json:"isRawFolder"`
	DurationSec int       `json:"durationSec,omitempty"`
}

// GetClips returns a list of saved clips in the output directory.
func (a *App) GetClips() ([]Clip, error) {
	// Ensure OutputDir is absolute
	outputDir, err := utils.ResolveAbsPath(a.config.OutputDir, "")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve output directory: %w", err)
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		// If dir doesn't exist, return empty
		if os.IsNotExist(err) {
			return []Clip{}, nil
		}
		return nil, err
	}

	var clips []Clip
	for _, f := range files {
		absPath := filepath.Join(outputDir, f.Name())

		if f.IsDir() {
			metadata, err := capture.ReadMetadata(absPath)
			if err != nil {
				continue
			}

			var folderSize int64
			filepath.Walk(absPath, func(_ string, info os.FileInfo, _ error) error {
				if info != nil && !info.IsDir() {
					folderSize += info.Size()
				}
				return nil
			})

			info, _ := f.Info()
			modTime := time.Now()
			if info != nil {
				modTime = info.ModTime()
			}
			if !metadata.CreatedAt.IsZero() {
				modTime = metadata.CreatedAt
			}

			clips = append(clips, Clip{
				Name:        f.Name(),
				Path:        absPath,
				Size:        folderSize,
				ModTime:     modTime,
				IsRawFolder: true,
				DurationSec: metadata.DurationSec,
			})
			continue
		}

		ext := filepath.Ext(f.Name())
		if ext != ".mp4" && ext != ".ts" {
			continue
		}

		info, err := f.Info()
		if err != nil {
			continue
		}

		clips = append(clips, Clip{
			Name:        f.Name(),
			Path:        absPath,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			IsRawFolder: false,
		})
	}

	return clips, nil
}

// OpenClip opens a clip in the default system player
func (a *App) OpenClip(path string) error {
	slog.Info("opening clip", "path", path)

	// Resolve path and validate it exists
	absPath, err := utils.ResolveAndValidatePath(path, a.config.OutputDir)
	if err != nil {
		return fmt.Errorf("clip not found: %w", err)
	}

	cmd := exec.Command("explorer", absPath)
	return cmd.Start()
}

// ConvertToMP4 converts a raw clip folder or .ts file to .mp4
func (a *App) ConvertToMP4(inputPath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.saver == nil {
		a.saver = capture.NewSaver(a.ffmpegPath, a.config.OutputDir)
	}

	// Check if input is a directory (raw folder) or a file
	info, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to stat input: %w", err)
	}

	if info.IsDir() {
		// Raw folder conversion
		if err := a.saver.ConvertRawFolder(inputPath, true); err != nil {
			return err
		}
	} else {
		// Legacy .ts file conversion
		if filepath.Ext(inputPath) != ".ts" {
			return fmt.Errorf("input file must be .ts")
		}

		baseName := filepath.Base(inputPath)
		nameWithoutExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]

		opts := capture.DefaultSaveOptions(nameWithoutExt)
		opts.ConvertToMP4, opts.DeleteTS = true, true

		if err := a.saver.ConvertToMP4(inputPath, opts); err != nil {
			return err
		}
	}

	a.EmitClipsUpdate()
	return nil
}

func (a *App) EmitClipsUpdate() {
	if a.app != nil {
		a.app.Event.Emit("clips-updated")
	}
}

// --- Internal methods ---

func (a *App) setState(status Status, errorMsg string) {
	a.state.Status = status
	a.state.ErrorMessage = errorMsg

	if a.OnStateChange != nil {
		go a.OnStateChange(a.state)
	}

	// Notify frontend
	if a.app != nil {
		a.app.Event.Emit("state-changed", a.state)
	}

	// Notify tray manager
	if a.onTrayStateChange != nil {
		go a.onTrayStateChange(a.state)
	}
}

// --- DTOs for Wails binding ---

// DisplayInfo is display info for frontend
type DisplayInfo struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	RefreshRate int    `json:"refreshRate"`
	IsPrimary   bool   `json:"isPrimary"`
}

// EncoderInfo is encoder info for frontend
type EncoderInfo struct {
	Name    string `json:"name"`
	Codec   string `json:"codec"`
	GPUName string `json:"gpuName"`
}
