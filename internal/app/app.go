package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"rewind/internal/buffer"
	"rewind/internal/capture"
	"rewind/internal/hardware"
	"rewind/internal/output"
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
	DisplayIndex  int    `json:"displayIndex"`
	EncoderName   string `json:"encoderName"`
	FPS           int    `json:"fps"`
	Bitrate       string `json:"bitrate"`
	RecordSeconds int    `json:"recordSeconds"`
	OutputDir     string `json:"outputDir"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		DisplayIndex:  0,
		EncoderName:   "", // auto-select
		FPS:           60,
		Bitrate:       "15M",
		RecordSeconds: 30,
		OutputDir:     "./clips",
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

	// Configuration
	config     Config
	ffmpegPath string

	// Hardware info (detected once)
	sysInfo *hardware.SystemInfo

	// Runtime state
	state        State
	capturer     *capture.Capturer
	ringBuffer   *buffer.Ring
	saver        *output.Saver
	startTime    time.Time
	lastSaveTime time.Time

	// Event callbacks (for Wails)
	OnStateChange func(state State)
	OnClipSaved   func(filename string)
	OnStartup     func(ctx context.Context)
}

// New creates a new App instance
func New(ffmpegPath string) *App {
	return &App{
		config:     DefaultConfig(),
		ffmpegPath: ffmpegPath,
		state:      State{Status: StatusIdle},
	}
}

// Startup is called when the Wails app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	slog.Info("Rewind starting up...")

	if a.OnStartup != nil {
		a.OnStartup(ctx)
	}
}

// Shutdown is called when the Wails app is closing
func (a *App) Shutdown(ctx context.Context) {
	slog.Info("Rewind shutting down...")

	// Stop recording if active
	if a.IsRecording() {
		a.Stop()
	}

	// Clear buffer
	if a.ringBuffer != nil {
		a.ringBuffer.Clear()
	}
}

// Initialize detects hardware and prepares the app
func (a *App) Initialize() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	hardware.FFmpegPath = a.ffmpegPath

	sysInfo, err := hardware.Detect()
	if err != nil {
		return fmt.Errorf("hardware detection failed: %w", err)
	}

	a.sysInfo = sysInfo

	// Auto-select encoder if not set
	if a.config.EncoderName == "" {
		a.autoSelectEncoder()
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
			Index:     d.Index,
			Name:      d.FriendlyName,
			Width:     d.Width,
			Height:    d.Height,
			IsPrimary: d.IsPrimary,
		})
	}
	return displays
}

// GetEncoders returns all available encoders
func (a *App) GetEncoders() []EncoderInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.sysInfo == nil {
		return nil
	}

	var encoders []EncoderInfo
	for _, e := range a.sysInfo.GetAvailableEncoders() {
		gpuName := "CPU"
		if e.GPUIndex >= 0 {
			if gpu := a.sysInfo.GPUs.FindByIndex(e.GPUIndex); gpu != nil {
				gpuName = gpu.Name
			}
		}
		encoders = append(encoders, EncoderInfo{
			Name:    e.Name,
			Codec:   e.Codec,
			GPUName: gpuName,
		})
	}
	return encoders
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
	return nil
}

// GetState returns the current state
func (a *App) GetState() State {
	a.mu.RLock()
	defer a.mu.RUnlock()

	state := a.state
	if a.ringBuffer != nil && a.state.Status == StatusRecording {
		total := a.ringBuffer.Size()
		used := a.ringBuffer.UsedBytes()
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

	// Build capture config
	captureCfg := capture.DefaultConfig()
	captureCfg.DisplayIndex = a.config.DisplayIndex
	captureCfg.EncoderName = a.config.EncoderName
	captureCfg.FPS = a.config.FPS
	captureCfg.Bitrate = a.config.Bitrate
	captureCfg.RecordSeconds = a.config.RecordSeconds
	captureCfg.OutputDir = a.config.OutputDir
	captureCfg.FFmpegPath = a.ffmpegPath

	if err := captureCfg.Resolve(a.sysInfo); err != nil {
		return fmt.Errorf("config resolution failed: %w", err)
	}

	// Create components
	bufSize := capture.CalculateBufferSize(a.config.Bitrate, a.config.RecordSeconds)
	a.ringBuffer = buffer.NewRing(bufSize)
	a.saver = output.NewSaver(a.ffmpegPath, a.config.OutputDir)

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

	os.MkdirAll(a.config.OutputDir, os.ModePerm)

	slog.Info("recording started",
		"display", a.config.DisplayIndex,
		"encoder", captureCfg.EncoderDisplayName(),
	)

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
	opts := output.DefaultSaveOptions(filename)

	if err := a.saver.Save(a.ringBuffer, opts); err != nil {
		return "", fmt.Errorf("save failed: %w", err)
	}

	a.lastSaveTime = time.Now()

	if a.OnClipSaved != nil {
		go a.OnClipSaved(filename + ".mp4")
	}

	slog.Info("clip saved", "filename", filename)
	return filename + ".mp4", nil
}

// IsRecording returns true if currently recording
func (a *App) IsRecording() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state.Status == StatusRecording
}

// --- Internal methods ---

func (a *App) setState(status Status, errorMsg string) {
	a.state.Status = status
	a.state.ErrorMessage = errorMsg

	if a.OnStateChange != nil {
		go a.OnStateChange(a.state)
	}
}

func (a *App) autoSelectEncoder() {
	if a.sysInfo == nil {
		return
	}

	encoders := a.sysInfo.GetAvailableEncoders()
	for _, e := range encoders {
		if e.Name != "libx264" {
			a.config.EncoderName = e.Name
			return
		}
	}
	a.config.EncoderName = "libx264"
}

// --- DTOs for Wails binding ---

// DisplayInfo is display info for frontend
type DisplayInfo struct {
	Index     int    `json:"index"`
	Name      string `json:"name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	IsPrimary bool   `json:"isPrimary"`
}

// EncoderInfo is encoder info for frontend
type EncoderInfo struct {
	Name    string `json:"name"`
	Codec   string `json:"codec"`
	GPUName string `json:"gpuName"`
}
