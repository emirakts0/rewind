package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IsVideoFile checks if a filename has a video extension
func IsVideoFile(filename string) bool {
	return filepath.Ext(filename) == ".mp4"
}

// GetClipInfo extracts clip information from a directory entry
func GetClipInfo(dirPath string, fileInfo os.DirEntry) (*Clip, error) {
	if fileInfo.IsDir() || !IsVideoFile(fileInfo.Name()) {
		return nil, nil
	}

	info, err := fileInfo.Info()
	if err != nil {
		return nil, err
	}

	return &Clip{
		Name:    fileInfo.Name(),
		Path:    filepath.Join(dirPath, fileInfo.Name()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// ListClipsInDirectory returns all video clips in the specified directory
func ListClipsInDirectory(outputDir string) ([]Clip, error) {
	files, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Clip{}, nil
		}
		return nil, err
	}

	var clips []Clip
	for _, f := range files {
		clip, err := GetClipInfo(outputDir, f)
		if err != nil {
			continue
		}
		if clip != nil {
			clips = append(clips, *clip)
		}
	}

	return clips, nil
}

// DeleteClipFiles deletes the specified clip files with validation
func DeleteClipFiles(paths []string, outputDir string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no clips specified")
	}

	var errors []string
	for _, path := range paths {
		if !strings.HasPrefix(path, outputDir) {
			errors = append(errors, fmt.Sprintf("%s: not in output directory", filepath.Base(path)))
			continue
		}

		if err := os.Remove(path); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete some clips: %s", strings.Join(errors, "; "))
	}

	return nil
}

// OpenInExplorer opens a file or directory in Windows Explorer
func OpenInExplorer(path string) error {
	cmd := exec.Command("explorer", path)
	return cmd.Start()
}

// EnsureDirectoryExists creates a directory if it doesn't exist
func EnsureDirectoryExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}

// GenerateClipFilename generates a timestamped filename for a clip
func GenerateClipFilename() string {
	return fmt.Sprintf("clip_%s.mp4", time.Now().Format("20060102_150405"))
}

// Clip represents a saved video clip
type Clip struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}
