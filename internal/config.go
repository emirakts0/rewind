package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Sentinel errors returned by ReconcileConfig when a configured device is absent.
var (
	ErrDisplayNotFound     = errors.New("configured display not found")
	ErrMicNotFound         = errors.New("configured microphone not found")
	ErrSystemAudioNotFound = errors.New("configured system audio device not found")
)

const configFileName = "settings.json"

// Config represents the application configuration
type Config struct {
	DisplayIndex            int    `json:"displayIndex"`
	MonitorName             string `json:"monitorName"` // resolved name of DisplayIndex
	FPS                     int    `json:"fps"`
	Bitrate                 int    `json:"bitrate"`
	RecordSeconds           int    `json:"recordSeconds"`
	SegmentDurationSec      int    `json:"segmentDurationSec"`
	OutputDir               string `json:"outputDir"`
	MicrophoneDevice        int    `json:"microphoneDevice"`
	MicrophoneName          string `json:"microphoneName"` // resolved name of MicrophoneDevice (-1 → "")
	SystemAudioDevice       int    `json:"systemAudioDevice"`
	SystemAudioName         string `json:"systemAudioName"` // resolved name of SystemAudioDevice (-1 → "")
	MicrophoneVolume        int    `json:"microphoneVolume"`
	SystemAudioVolume       int    `json:"systemAudioVolume"`
	ShowCursor              bool   `json:"showCursor"`
	ShowBorder              bool   `json:"showBorder"`
	NotificationsEnabled    bool   `json:"notificationsEnabled"`
	NotificationsOnlyErrors bool   `json:"notificationsOnlyErrors"`
	NotificationsPosition   string `json:"notificationsPosition"`
	NotificationsDurationMs int    `json:"notificationsDurationMs"`
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() Config {
	outputDir := "./clips"
	if dir, err := GetClipsDir(); err == nil {
		outputDir = dir
	}

	return Config{
		DisplayIndex:            0,
		FPS:                     30,
		Bitrate:                 15,
		RecordSeconds:           30,
		SegmentDurationSec:      2,
		OutputDir:               outputDir,
		MicrophoneDevice:        -1,
		SystemAudioDevice:       -1,
		MicrophoneVolume:        100,
		SystemAudioVolume:       100,
		ShowCursor:              true,
		ShowBorder:              false,
		NotificationsEnabled:    true,
		NotificationsOnlyErrors: false,
		NotificationsPosition:   "top-left",
		NotificationsDurationMs: 3000,
	}
}

func getConfigFilePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, configFileName), nil
}

func (a *App) LoadConfig() error {
	configPath, err := getConfigFilePath()
	if err != nil {
		slog.Warn("failed to get config path, using defaults", "error", err)
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("no config file found, using defaults", "path", configPath)
			return nil
		}
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		slog.Warn("failed to parse config file, using defaults", "error", err)
		return nil
	}

	a.config = cfg
	slog.Info("config loaded", "path", configPath)
	return nil
}

func saveConfigToFile(cfg Config) error {
	slog.Info("saving config...")

	configPath, err := getConfigFilePath()
	if err != nil {
		slog.Warn("failed to get config path", "error", err)
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		slog.Warn("failed to marshal config", "error", err)
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		slog.Warn("failed to write config file", "error", err)
		return err
	}

	slog.Info("config saved", "path", configPath)
	return nil
}

// ReconcileConfig aligns configured device names with current hardware.
// Missing devices are reset to defaults and their sentinel errors are returned
func ReconcileConfig(cfg *Config) error {
	var reconcileErr error

	// Validate monitor
	monitors, err := GetMonitors()
	if err == nil && len(monitors) > 0 {
		newIndex := 0
		newName := monitors[0].Name

		for i, m := range monitors {
			if m.Name == cfg.MonitorName {
				newIndex = i
				newName = m.Name
				break
			}
		}

		if cfg.DisplayIndex != newIndex || cfg.MonitorName != newName {
			if cfg.MonitorName != "" && cfg.MonitorName != newName {
				slog.Warn("configured monitor missing, resetting to primary", "configured", cfg.MonitorName)
				reconcileErr = ErrDisplayNotFound
			}
			cfg.DisplayIndex = newIndex
			cfg.MonitorName = newName
		}
	}

	// Validate audio devices
	devices, err := ListAudioDevices()
	if err == nil {
		var inputs, outputs []AudioDeviceInfo
		for _, d := range devices {
			if d.IsInput {
				inputs = append(inputs, d)
			} else {
				outputs = append(outputs, d)
			}
		}

		// Microphone
		newMicIndex := -1
		newMicName := ""
		if cfg.MicrophoneName != "" {
			for i, d := range inputs {
				if d.Name == cfg.MicrophoneName {
					newMicIndex = i
					newMicName = d.Name
					break
				}
			}
			if newMicIndex == -1 {
				slog.Warn("configured microphone missing, disabling", "configured", cfg.MicrophoneName)
				reconcileErr = ErrMicNotFound
			}
		}
		if cfg.MicrophoneDevice != newMicIndex || cfg.MicrophoneName != newMicName {
			cfg.MicrophoneDevice = newMicIndex
			cfg.MicrophoneName = newMicName
		}

		// System Audio
		newSpeakerIndex := -1
		newSpeakerName := ""
		if cfg.SystemAudioName != "" {
			for i, d := range outputs {
				if d.Name == cfg.SystemAudioName {
					newSpeakerIndex = i
					newSpeakerName = d.Name
					break
				}
			}
			if newSpeakerIndex == -1 {
				slog.Warn("configured system audio missing, disabling", "configured", cfg.SystemAudioName)
				reconcileErr = ErrSystemAudioNotFound
			}
		}
		if cfg.SystemAudioDevice != newSpeakerIndex || cfg.SystemAudioName != newSpeakerName {
			cfg.SystemAudioDevice = newSpeakerIndex
			cfg.SystemAudioName = newSpeakerName
		}
	}

	if err := saveConfigToFile(*cfg); err != nil {
		slog.Warn("failed to save reconciled config", "error", err)
		return err
	}
	slog.Info("config reconciliation complete")

	return reconcileErr
}

// ValidateConfigValues validates configuration values for business rules
func ValidateConfigValues(cfg Config) error {
	if cfg.FPS <= 0 || cfg.FPS > MaxFPS {
		return fmt.Errorf("FPS must be between 1 and %d", MaxFPS)
	}
	if cfg.RecordSeconds <= 0 {
		return fmt.Errorf("record seconds must be positive")
	}
	if cfg.SegmentDurationSec != 2 && cfg.SegmentDurationSec != 5 && cfg.SegmentDurationSec != 10 {
		return fmt.Errorf("segment duration must be 2, 5, or 10 seconds")
	}
	return nil
}
