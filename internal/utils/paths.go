package utils

import (
	"os"
	"path/filepath"
)

const AppName = "Rewind"

func GetAppDataDir() (string, error) {
	localAppData, err := os.UserCacheDir()
	if err != nil {
		return "", err
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

func GetTempSegmentsDir() (string, error) {
	tempDir := os.TempDir()
	segmentsDir := filepath.Join(tempDir, AppName, "segments")
	if err := os.MkdirAll(segmentsDir, 0755); err != nil {
		return "", err
	}
	return segmentsDir, nil
}

func GetFFmpegPath() string {
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)

		ffmpegPath := filepath.Join(exeDir, "ffmpeg.exe")
		if _, err := os.Stat(ffmpegPath); err == nil {
			return ffmpegPath
		}

		ffmpegPath = filepath.Join(exeDir, "bin", "ffmpeg.exe")
		if _, err := os.Stat(ffmpegPath); err == nil {
			return ffmpegPath
		}
	}

	if _, err := os.Stat("bin/ffmpeg.exe"); err == nil {
		return "bin/ffmpeg.exe"
	}
	if _, err := os.Stat("ffmpeg.exe"); err == nil {
		return "ffmpeg.exe"
	}

	return "ffmpeg"
}

func CleanupTempSegments() error {
	tempDir, err := GetTempSegmentsDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(tempDir, entry.Name())
			if err := os.RemoveAll(subDir); err != nil {
				return err
			}
		} else {
			ext := filepath.Ext(entry.Name())
			if ext == ".mp4" || ext == ".ts" || ext == ".txt" || ext == ".pcm" {
				filePath := filepath.Join(tempDir, entry.Name())
				if err := os.Remove(filePath); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
