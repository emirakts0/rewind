package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"rewind/internal/native"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Status represents the current application state
type Status string

const (
	StatusIdle      Status = "idle"
	StatusRecording Status = "recording"
)

// Config represents user-configurable settings
type Config struct {
	DisplayIndex      int    `json:"displayIndex"`
	FPS               int    `json:"fps"`
	Bitrate           int    `json:"bitrate"` // in Mbps
	RecordSeconds     int    `json:"recordSeconds"`
	OutputDir         string `json:"outputDir"`
	MicrophoneDevice  int    `json:"microphoneDevice"`  // device index, -1 = disabled
	SystemAudioDevice int    `json:"systemAudioDevice"` // device index, -1 = disabled
	ShowCursor        bool   `json:"showCursor"`
	ShowBorder        bool   `json:"showBorder"`
}

// MemoryEstimate holds estimated memory usage
type MemoryEstimate struct {
	DiskMB   float64 `json:"diskMB"`   // Estimated disk usage for video segments
	MemoryMB float64 `json:"memoryMB"` // Estimated RAM usage for audio buffers
	TotalMB  float64 `json:"totalMB"`  // Total estimated usage
}

// EstimateMemoryUsage calculates estimated memory usage based on config
func (c *Config) EstimateMemoryUsage() MemoryEstimate {
	// Video disk usage estimation
	videoDiskMB := float64(c.Bitrate) * float64(c.RecordSeconds) / 8.0

	// Audio memory is minimal and only shown during recording
	audioMemoryMB := 0.0

	totalMB := videoDiskMB + audioMemoryMB

	return MemoryEstimate{
		DiskMB:   videoDiskMB,
		MemoryMB: audioMemoryMB,
		TotalMB:  totalMB,
	}
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	// Default to %LOCALAPPDATA%\Rewind\clips on Windows
	outputDir := "./clips"
	if cacheDir, err := os.UserCacheDir(); err == nil {
		outputDir = filepath.Join(cacheDir, "Rewind", "clips")
	}

	return Config{
		DisplayIndex:      0,
		FPS:               30,
		Bitrate:           15, // 15 Mbps
		RecordSeconds:     30,
		OutputDir:         outputDir,
		MicrophoneDevice:  -1,
		SystemAudioDevice: -1,
		ShowCursor:        true,
		ShowBorder:        false,
	}
}

// State holds the current application state
type State struct {
	Status        Status         `json:"status"`
	ErrorMessage  string         `json:"errorMessage,omitempty"`
	BufferUsage   float64        `json:"bufferUsage"`   // percentage 0-100
	RecordingFor  int            `json:"recordingFor"`  // seconds since recording started
	DiskUsageMB   float64        `json:"diskUsageMB"`   // actual disk space used by video segments in MB
	MemoryUsageMB float64        `json:"memoryUsageMB"` // actual memory used by audio buffers in MB
	Estimate      MemoryEstimate `json:"estimate"`      // estimated memory usage based on config
}

// App is the main application service
type App struct {
	mu  sync.RWMutex
	ctx context.Context

	app *application.App

	config     Config
	ffmpegPath string

	state        State
	replayHandle native.Handle
	startTime    time.Time
	lastSaveTime time.Time

	onTrayStateChange func(State)
}

// New creates a new App instance
func New(ffmpegPath string) *App {
	app := &App{
		config:     DefaultConfig(),
		ffmpegPath: ffmpegPath,
		state:      State{Status: StatusIdle},
	}

	if err := app.LoadConfig(); err != nil {
		slog.Warn("failed to load config", "error", err)
	}

	// Set up runtime error callback from Rust
	native.SetRuntimeErrorCallback(app.handleRuntimeError)

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

// ServiceStartup is called when the Wails v3 app starts
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.ctx = ctx
	slog.Info("Rewind service starting up...")
	return nil
}

// ServiceShutdown is called when the Wails v3 app is closing
func (a *App) ServiceShutdown() error {
	slog.Info("Rewind service shutting down...")

	if a.IsRecording() {
		a.StopRecording()
	}

	return nil
}

// ListAvailableDisplays returns all available displays/monitors
func (a *App) ListAvailableDisplays() []DisplayInfo {
	monitors, err := native.GetMonitors()
	if err != nil {
		slog.Error("failed to get monitors", "error", err)
		return nil
	}

	var displays []DisplayInfo
	for _, m := range monitors {
		displays = append(displays, DisplayInfo{
			Index:       m.Index,
			Name:        m.Name,
			Width:       int(m.Width),
			Height:      int(m.Height),
			RefreshRate: int(m.RefreshRate),
			IsPrimary:   m.Index == 0,
		})
	}
	return displays
}

// ListAudioInputDevices returns available microphone devices
func (a *App) ListAudioInputDevices() []string {
	devices, err := native.ListAudioDevices()
	if err != nil {
		slog.Error("failed to list audio devices", "error", err)
		return nil
	}

	var names []string
	for _, d := range devices {
		if d.IsInput {
			names = append(names, d.Name)
		}
	}
	return names
}

// ListAudioOutputDevices returns available speaker/loopback devices
func (a *App) ListAudioOutputDevices() []string {
	devices, err := native.ListAudioDevices()
	if err != nil {
		slog.Error("failed to list audio devices", "error", err)
		return nil
	}

	var names []string
	for _, d := range devices {
		if !d.IsInput {
			names = append(names, d.Name)
		}
	}
	return names
}

// GetConfig returns the current configuration
func (a *App) GetConfig() Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// UpdateConfig updates the application configuration (only when not recording)
func (a *App) UpdateConfig(cfg Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status == StatusRecording {
		return fmt.Errorf("cannot change config while recording")
	}

	if cfg.FPS <= 0 || cfg.FPS > 240 {
		return fmt.Errorf("FPS must be between 1 and 240")
	}
	if cfg.RecordSeconds <= 0 {
		return fmt.Errorf("record seconds must be positive")
	}

	a.config = cfg
	slog.Info("config updated", "config", cfg)

	if err := saveConfigToFile(cfg); err != nil {
		slog.Warn("failed to save config", "error", err)
	}

	return nil
}

// GetRecordingState returns the current recording state
func (a *App) GetRecordingState() State {
	a.mu.RLock()
	defer a.mu.RUnlock()

	state := a.state

	// Always include estimate based on current config
	state.Estimate = a.config.EstimateMemoryUsage()

	if a.replayHandle != 0 && a.state.Status == StatusRecording {
		elapsed := time.Since(a.startTime).Seconds()
		maxDuration := float64(a.config.RecordSeconds)
		if maxDuration > 0 {
			state.BufferUsage = (elapsed / maxDuration) * 100
			if state.BufferUsage > 100 {
				state.BufferUsage = 100
			}
		}
		state.RecordingFor = int(elapsed)
	}

	return state
}

// StartRecording begins screen capture and replay buffer recording
func (a *App) StartRecording() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status == StatusRecording {
		return fmt.Errorf("already recording")
	}

	// Get monitor info
	monitors, err := native.GetMonitors()
	if err != nil {
		return fmt.Errorf("failed to get monitors: %w", err)
	}

	if a.config.DisplayIndex >= len(monitors) {
		return fmt.Errorf("invalid display index: %d", a.config.DisplayIndex)
	}

	monitor := monitors[a.config.DisplayIndex]

	// Build audio config
	var micIdx, speakerIdx *int
	if a.config.MicrophoneDevice >= 0 {
		micIdx = &a.config.MicrophoneDevice
	}
	if a.config.SystemAudioDevice >= 0 {
		speakerIdx = &a.config.SystemAudioDevice
	}

	audioConfig := native.AudioConfig{
		SampleRate:         48000,
		Channels:           2,
		MicEnabled:         micIdx != nil,
		MicDeviceIndex:     micIdx,
		SpeakerEnabled:     speakerIdx != nil,
		SpeakerDeviceIndex: speakerIdx,
	}

	// Rust handles directory creation
	tempDir := filepath.Join(a.config.OutputDir, ".temp")

	// Build replay config
	config := native.ReplayRecordingConfig{
		Width:               monitor.Width,
		Height:              monitor.Height,
		Fps:                 uint32(a.config.FPS),
		VideoBitrate:        uint32(a.config.Bitrate * 1000000), // Mbps to bps
		Audio:               audioConfig,
		BufferDurationSecs:  uint64(a.config.RecordSeconds),
		SegmentDurationSecs: 5,
		ShowCursor:          a.config.ShowCursor,
		ShowBorder:          a.config.ShowBorder,
		FfmpegPath:          a.ffmpegPath,
		TempPath:            tempDir,
	}

	// Initialize replay buffer
	handle, err := native.InitReplayBuffer(a.config.DisplayIndex, config)
	if err != nil {
		return fmt.Errorf("failed to start replay buffer: %w", err)
	}

	a.replayHandle = handle
	a.startTime = time.Now()
	a.setState(StatusRecording, "")

	// Start background goroutine to update buffer status
	go a.updateBufferStatus()

	slog.Info("recording started",
		"display", a.config.DisplayIndex,
		"resolution", fmt.Sprintf("%dx%d", monitor.Width, monitor.Height),
		"fps", a.config.FPS,
		"bitrate", a.config.Bitrate,
	)

	return nil
}

// StopRecording stops the current recording session
func (a *App) StopRecording() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status != StatusRecording {
		return fmt.Errorf("not recording")
	}

	if a.replayHandle != 0 {
		a.replayHandle.Stop()
		a.replayHandle = 0
	}

	a.setState(StatusIdle, "")
	slog.Info("recording stopped")
	return nil
}

// SaveCurrentClip saves the replay buffer to a file
func (a *App) SaveCurrentClip() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status != StatusRecording {
		return "", fmt.Errorf("not recording")
	}

	// Debounce: 5 second cooldown between saves
	if time.Since(a.lastSaveTime) < 5*time.Second {
		remaining := 5 - int(time.Since(a.lastSaveTime).Seconds())
		return "", fmt.Errorf("please wait %d seconds before saving another clip", remaining)
	}

	if a.replayHandle == 0 {
		return "", fmt.Errorf("replay buffer not initialized")
	}

	filename := fmt.Sprintf("clip_%s.mp4", time.Now().Format("20060102_150405"))
	outputPath := filepath.Join(a.config.OutputDir, filename)

	if err := a.replayHandle.Save(outputPath); err != nil {
		return "", err
	}

	a.lastSaveTime = time.Now()
	a.emitClipsUpdated()

	slog.Info("clip saved", "filename", filename)
	return filename, nil
}

// IsRecording returns true if currently recording
func (a *App) IsRecording() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state.Status == StatusRecording
}

// ChooseOutputDirectory opens a directory picker dialog
func (a *App) ChooseOutputDirectory() (string, error) {
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

// ListSavedClips returns a list of saved clips in the output directory
func (a *App) ListSavedClips() ([]Clip, error) {
	files, err := os.ReadDir(a.config.OutputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Clip{}, nil
		}
		return nil, err
	}

	var clips []Clip
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".mp4" {
			continue
		}

		info, err := f.Info()
		if err != nil {
			continue
		}

		clips = append(clips, Clip{
			Name:    f.Name(),
			Path:    filepath.Join(a.config.OutputDir, f.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	return clips, nil
}

// OpenClipInExplorer opens a clip in the default system player
func (a *App) OpenClipInExplorer(path string) error {
	// Path is already absolute from ListSavedClips
	cmd := exec.Command("explorer", path)
	return cmd.Start()
}

func (a *App) emitClipsUpdated() {
	if a.app != nil {
		a.app.Event.Emit("clips-updated")
	}
}

// setState updates state and notifies listeners
func (a *App) setState(status Status, errorMsg string) {
	a.state.Status = status
	a.state.ErrorMessage = errorMsg
	a.state.Estimate = a.config.EstimateMemoryUsage()

	// Notify frontend
	if a.app != nil {
		a.app.Event.Emit("state-changed", a.state)
	}

	// Notify tray manager
	if a.onTrayStateChange != nil {
		go a.onTrayStateChange(a.state)
	}
}

// updateBufferStatus periodically updates buffer usage from Rust
func (a *App) updateBufferStatus() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		a.mu.RLock()
		if a.state.Status != StatusRecording || a.replayHandle == 0 {
			a.mu.RUnlock()
			return
		}
		handle := a.replayHandle
		startTime := a.startTime
		maxDuration := float64(a.config.RecordSeconds)
		a.mu.RUnlock()

		// Get status from Rust
		status, err := handle.GetStatus()
		if err != nil {
			slog.Debug("failed to get buffer status", "error", err)
			continue
		}

		// Calculate buffer usage
		elapsed := time.Since(startTime).Seconds()
		bufferUsage := (status.Duration / maxDuration) * 100
		if bufferUsage > 100 {
			bufferUsage = 100
		}

		// Convert bytes to MB
		diskUsageMB := float64(status.DiskUsage) / (1024 * 1024)
		memoryUsageMB := float64(status.MemoryUsage) / (1024 * 1024)

		// Update state
		a.mu.Lock()
		a.state.BufferUsage = bufferUsage
		a.state.RecordingFor = int(elapsed)
		a.state.DiskUsageMB = diskUsageMB
		a.state.MemoryUsageMB = memoryUsageMB
		a.state.Estimate = a.config.EstimateMemoryUsage()
		state := a.state
		a.mu.Unlock()

		// Emit state change event
		if a.app != nil {
			a.app.Event.Emit("state-changed", state)
		}
	}
}

// handleRuntimeError handles runtime errors/warnings from Rust during recording
func (a *App) handleRuntimeError(level string, message string) {
	a.mu.RLock()
	isRecording := a.state.Status == StatusRecording
	a.mu.RUnlock()

	// Only handle errors during recording
	if !isRecording {
		return
	}

	// Emit runtime error event to frontend
	if a.app != nil {
		a.app.Event.Emit("runtime-error", map[string]string{
			"level":   level,
			"message": message,
		})
	}

	// If it's a critical error, stop recording
	if level == "error" {
		// Check for critical errors that should stop recording
		criticalKeywords := []string{
			"Failed to send frame",
			"encoder",
			"capture",
			"monitor",
		}

		for _, keyword := range criticalKeywords {
			if strings.Contains(strings.ToLower(message), strings.ToLower(keyword)) {
				slog.Error("critical error detected, stopping recording", "message", message)
				go func() {
					time.Sleep(100 * time.Millisecond)
					if err := a.StopRecording(); err != nil {
						slog.Error("failed to stop recording after critical error", "error", err)
					}
				}()
				return
			}
		}
	}
}

// --- DTOs for Wails binding ---

type DisplayInfo struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	RefreshRate int    `json:"refreshRate"`
	IsPrimary   bool   `json:"isPrimary"`
}

type Clip struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}
