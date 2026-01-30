package utils

import (
	"os"
	"path/filepath"
)

const AppName = "Rewind"

func GetAppDataDir() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		localAppData = cacheDir
	}

	appDataDir := filepath.Join(localAppData, AppName)
	return appDataDir, nil
}

func getSubDir(name string) (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(appDataDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return dir, nil
}

func GetClipsDir() (string, error)  { return getSubDir("clips") }
func GetLogsDir() (string, error)   { return getSubDir("logs") }
func GetConfigDir() (string, error) { return getSubDir("config") }

func ResolveAbsPath(path string, baseDir string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	if baseDir != "" {
		return filepath.Join(baseDir, path), nil
	}

	return filepath.Abs(path)
}

func ResolveAndValidatePath(path string, baseDir string) (string, error) {
	absPath, err := ResolveAbsPath(path, baseDir)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(absPath); err != nil {
		return "", err
	}

	return absPath, nil
}
