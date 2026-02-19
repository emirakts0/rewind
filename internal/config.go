package internal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const configFileName = "settings.json"

// Config represents the application configuration
type Config struct {
	DisplayIndex       int    `json:"displayIndex"`
	FPS                int    `json:"fps"`
	Bitrate            int    `json:"bitrate"`
	RecordSeconds      int    `json:"recordSeconds"`
	SegmentDurationSec int    `json:"segmentDurationSec"`
	OutputDir          string `json:"outputDir"`
	MicrophoneDevice   int    `json:"microphoneDevice"`
	SystemAudioDevice  int    `json:"systemAudioDevice"`
	MicrophoneVolume   int    `json:"microphoneVolume"`
	SystemAudioVolume  int    `json:"systemAudioVolume"`
	ShowCursor         bool   `json:"showCursor"`
	ShowBorder         bool   `json:"showBorder"`
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() Config {
	outputDir := "./clips"
	if dir, err := GetClipsDir(); err == nil {
		outputDir = dir
	}

	return Config{
		DisplayIndex:       0,
		FPS:                30,
		Bitrate:            15,
		RecordSeconds:      30,
		SegmentDurationSec: 5,
		OutputDir:          outputDir,
		MicrophoneDevice:   -1,
		SystemAudioDevice:  -1,
		MicrophoneVolume:   100,
		SystemAudioVolume:  100,
		ShowCursor:         true,
		ShowBorder:         false,
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

// ValidateConfig checks if config values are valid and fixes them if needed
func ValidateConfig(cfg *Config) (bool, error) {
	needsSave := false

	// Validate monitor index
	monitors, err := GetMonitors()
	if err != nil {
		slog.Warn("failed to get monitors for validation", "error", err)
	} else if len(monitors) > 0 {
		if cfg.DisplayIndex >= len(monitors) || cfg.DisplayIndex < 0 {
			slog.Warn("invalid display index in config, resetting to 0", "configured", cfg.DisplayIndex, "available", len(monitors))
			cfg.DisplayIndex = 0
			needsSave = true
		}
	}

	// Validate audio devices
	devices, err := ListAudioDevices()
	if err != nil {
		slog.Warn("failed to get audio devices for validation", "error", err)
	} else {
		inputCount := 0
		outputCount := 0
		for _, d := range devices {
			if d.IsInput {
				inputCount++
			} else {
				outputCount++
			}
		}

		// Validate microphone device
		if cfg.MicrophoneDevice >= inputCount {
			slog.Warn("invalid microphone device in config, disabling", "configured", cfg.MicrophoneDevice, "available", inputCount)
			cfg.MicrophoneDevice = -1
			needsSave = true
		}

		// Validate system audio device
		if cfg.SystemAudioDevice >= outputCount {
			slog.Warn("invalid system audio device in config, disabling", "configured", cfg.SystemAudioDevice, "available", outputCount)
			cfg.SystemAudioDevice = -1
			needsSave = true
		}
	}

	return needsSave, nil
}

// ValidateAndFixConfig checks if config values are valid and fixes them if needed
func ValidateAndFixConfig(cfg *Config) error {
	needsSave, err := ValidateConfig(cfg)
	if err != nil {
		return err
	}

	if needsSave {
		if err := saveConfigToFile(*cfg); err != nil {
			slog.Warn("failed to save corrected config", "error", err)
			return err
		}
		slog.Info("config validated and corrected")
	} else {
		slog.Info("config validation passed")
	}

	return nil
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
