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
	"syscall"
	"time"

	"rewind/internal/buffer"
	"rewind/internal/capture"
	"rewind/internal/hardware"
	"rewind/internal/output"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	ConvertToMP4  bool   `json:"convertToMP4"`
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
		ConvertToMP4:  true,
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
	opts := output.DefaultSaveOptions(filename)
	opts.ConvertToMP4 = a.config.ConvertToMP4
	opts.DeleteTS = a.config.ConvertToMP4 // Only delete TS if we converted to MP4

	if err := a.saver.Save(a.ringBuffer, opts); err != nil {
		return "", fmt.Errorf("save failed: %w", err)
	}

	a.lastSaveTime = time.Now()

	ext := ".ts"
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

// --- Internal methods ---

func (a *App) setState(status Status, errorMsg string) {
	a.state.Status = status
	a.state.ErrorMessage = errorMsg

	if a.OnStateChange != nil {
		go a.OnStateChange(a.state)
	}
}

// SelectDirectory opens a directory selection dialog
func (a *App) SelectDirectory() (string, error) {
	slog.Info("SelectDirectory called")
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Output Directory",
		DefaultDirectory: a.config.OutputDir,
	})

	if err != nil {
		return "", err
	}

	return selection, nil
}

func (a *App) autoSelectEncoder() {
	if a.sysInfo == nil {
		return
	}

	hardware.ValidateEncoders(a.sysInfo.GPUs)

	best := hardware.FindBestEncoder(a.sysInfo.GPUs)
	if best != nil {
		slog.Info("auto-selected encoder", "encoder", best.Name)
		a.config.EncoderName = best.Name
	} else {
		slog.Info("auto-selected fallback cpu encoder")
		a.config.EncoderName = "libx264"
	}
}

// EstimateMemory calculates the estimated buffer size based on bitrate and duration
func (a *App) EstimateMemory(bitrate string, seconds int) string {
	size := capture.CalculateBufferSize(bitrate, seconds)
	// Convert to MegaBytes (1024^2)
	mb := float64(size) / (1024 * 1024)
	return fmt.Sprintf("~%.0fMB", mb)
}

// Clip represents a saved video file
type Clip struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// GetClips returns a list of saved clips in the output directory
func (a *App) GetClips() ([]Clip, error) {
	files, err := os.ReadDir(a.config.OutputDir)
	if err != nil {
		// If dir doesn't exist, return empty
		if os.IsNotExist(err) {
			return []Clip{}, nil
		}
		return nil, err
	}

	var clips []Clip
	for _, f := range files {
		if f.IsDir() {
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

		absPath, _ := filepath.Abs(filepath.Join(a.config.OutputDir, f.Name()))
		clips = append(clips, Clip{
			Name:    f.Name(),
			Path:    absPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	// Sort by newest first
	// (Simple bubble sort or just let frontend sort. Let's let frontend sort or rely on OS order?
	// OS order is arbitrary. Let's do a quick reverse sort if needed, but array sort in Go is verbose.
	// Frontend can sort.)
	return clips, nil
}

// OpenClip opens a clip in the default system player
func (a *App) OpenClip(path string) error {
	slog.Info("opening clip", "path", path)
	cmd := exec.Command("explorer", path)
	return cmd.Start()
}

// ConvertToMP4 converts a .ts file to .mp4 and deletes the original
func (a *App) ConvertToMP4(inputPath string) error {
	if filepath.Ext(inputPath) != ".ts" {
		return fmt.Errorf("input file must be .ts")
	}

	outputPath := inputPath[:len(inputPath)-3] + ".mp4"

	// Use ffmpeg to convert (copy codec)
	cmd := exec.Command(a.ffmpegPath,
		"-i", inputPath,
		"-c", "copy",
		"-y", // overwrite if exists
		outputPath,
	)

	// Hide command window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Run(); err != nil {
		slog.Error("conversion failed", "error", err)
		return err
	}

	// Delete original file
	if err := os.Remove(inputPath); err != nil {
		slog.Error("failed to delete .ts file", "error", err)
	}

	a.EmitClipsUpdate()
	return nil
}

func (a *App) EmitClipsUpdate() {
	runtime.EventsEmit(a.ctx, "clips-updated")
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
