package app

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"rewind/internal/utils"
)

const configFileName = "settings.json"

func getConfigFilePath() (string, error) {
	configDir, err := utils.GetConfigDir()
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

	//a.config = cfg   // todo:
	if a.config.OutputDir != cfg.OutputDir {
		slog.Info("output directory changed", "old", a.config.OutputDir, "new", cfg.OutputDir)
		a.config.OutputDir = cfg.OutputDir
	}

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
