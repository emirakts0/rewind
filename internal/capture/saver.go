package capture

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	hiddenexec "rewind/internal/utils"
	stdruntime "runtime"
	"runtime/debug"
	"time"
)

type Saver struct {
	ffmpegPath string
	outputDir  string
}

func NewSaver(ffmpegPath, outputDir string) *Saver {
	os.MkdirAll(outputDir, os.ModePerm)
	return &Saver{
		ffmpegPath: ffmpegPath,
		outputDir:  outputDir,
	}
}

type SaveOptions struct {
	Filename     string
	ConvertToMP4 bool
	DeleteTS     bool
	DurationSec  int
}

func DefaultSaveOptions(filename string) *SaveOptions {
	return &SaveOptions{
		Filename:     filename,
		ConvertToMP4: true,
		DeleteTS:     true,
	}
}

// ClipMetadata stores configuration used during recording for later conversion
type ClipMetadata struct {
	DurationSec int       `json:"durationSec"`
	HasAudio    bool      `json:"hasAudio"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Snapshotter interface {
	Snapshot() []byte
}

func (s *Saver) Save(src Snapshotter, opts *SaveOptions) error {
	return s.SaveWithAudio(src, nil, opts)
}

func (s *Saver) SaveWithAudio(videoSrc Snapshotter, audioSrc Snapshotter, opts *SaveOptions) error {
	videoData := videoSrc.Snapshot()
	if len(videoData) == 0 {
		return fmt.Errorf("buffer is empty")
	}

	var audioData []byte
	if audioSrc != nil {
		audioData = audioSrc.Snapshot()
	}

	go s.processSaveWithAudio(videoData, audioData, opts)
	return nil
}

func (s *Saver) processSaveWithAudio(videoData, audioData []byte, opts *SaveOptions) {
	// Mode 1: RAW Save (Create folder, save video and audio separately)
	if !opts.ConvertToMP4 {
		clipDir := filepath.Join(s.outputDir, opts.Filename)
		if err := os.MkdirAll(clipDir, os.ModePerm); err != nil {
			slog.Error("failed to create clip directory", "error", err)
			return
		}

		// Save Video
		videoPath := filepath.Join(clipDir, "video.ts")
		if err := s.writeData(videoPath, videoData); err != nil {
			slog.Error("failed to save raw video", "error", err)
		}

		// Free Video RAM immediately
		videoData = nil
		stdruntime.GC()
		debug.FreeOSMemory()

		// Save Audio
		hasAudio := len(audioData) > 0
		if hasAudio {
			audioPath := filepath.Join(clipDir, "audio.pcm")
			if err := s.writeData(audioPath, audioData); err != nil {
				slog.Error("failed to save raw audio", "error", err)
			}
		}

		// Save Metadata
		metadata := ClipMetadata{
			DurationSec: opts.DurationSec,
			HasAudio:    hasAudio,
			CreatedAt:   time.Now(),
		}
		metadataPath := filepath.Join(clipDir, "metadata.json")
		if err := s.writeMetadata(metadataPath, &metadata); err != nil {
			slog.Error("failed to save metadata", "error", err)
		}

		slog.Info("raw clip saved", "dir", clipDir)
		return
	}

	// Mode 2: MP4 Conversion (Temp files -> FFmpeg -> MP4)
	tsPath := filepath.Join(s.outputDir, opts.Filename+".ts")
	if err := s.writeData(tsPath, videoData); err != nil {
		slog.Error("failed to write video temp file", "error", err)
		return
	}

	videoData = nil
	stdruntime.GC()
	debug.FreeOSMemory()

	var pcmPath string
	if len(audioData) > 0 {
		pcmPath = filepath.Join(s.outputDir, opts.Filename+".pcm")
		if err := s.writeData(pcmPath, audioData); err != nil {
			slog.Error("failed to write audio temp file", "error", err)
			pcmPath = ""
		}
		audioData = nil
		stdruntime.GC()
		debug.FreeOSMemory()
	}

	if pcmPath != "" {
		s.mergeVideoAudio(tsPath, pcmPath, opts)
	} else {
		s.ConvertToMP4(tsPath, opts)
	}
}

func (s *Saver) writeData(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 8*1024*1024)
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

func (s *Saver) mergeVideoAudio(tsPath, pcmPath string, opts *SaveOptions) error {
	mp4Path := filepath.Join(s.outputDir, opts.Filename+".mp4")
	absTs, _ := filepath.Abs(tsPath)
	absPcm, _ := filepath.Abs(pcmPath)
	absMp4, _ := filepath.Abs(mp4Path)

	inputArgs := []string{}
	if opts.DurationSec > 0 {
		inputArgs = append(inputArgs, "-sseof", fmt.Sprintf("-%d", opts.DurationSec))
	}
	inputArgs = append(inputArgs, "-i", absTs)

	args := []string{"-y"}
	args = append(args, inputArgs...)
	args = append(args,
		"-f", "f32le", "-ar", "48000", "-ac", "2", "-i", absPcm,
		"-c:v", "copy",
		"-c:a", "aac", "-b:a", "192k",
		"-shortest",
		absMp4,
	)

	cmd := hiddenexec.Command(s.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		slog.Error("merge failed", "error", err)
		return err
	}

	slog.Info("clip saved with audio", "path", mp4Path)

	if opts.DeleteTS {
		os.Remove(absTs)
	}
	os.Remove(absPcm)

	return nil
}

func (s *Saver) ConvertToMP4(tsPath string, opts *SaveOptions) error {
	mp4Path := filepath.Join(s.outputDir, opts.Filename+".mp4")
	absTs, _ := filepath.Abs(tsPath)
	absMp4, _ := filepath.Abs(mp4Path)

	args := []string{"-y"}
	if opts.DurationSec > 0 {
		args = append(args, "-sseof", fmt.Sprintf("-%d", opts.DurationSec))
	}
	args = append(args, "-i", absTs, "-c", "copy", absMp4)

	cmd := hiddenexec.Command(s.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		slog.Error("conversion failed", "error", err)
		return err
	}

	slog.Info("clip saved", "path", mp4Path)
	if opts.DeleteTS {
		os.Remove(absTs)
	}
	return nil
}

func (s *Saver) writeMetadata(path string, metadata *ClipMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ReadMetadata reads metadata from a raw clip folder
func ReadMetadata(folderPath string) (*ClipMetadata, error) {
	metadataPath := filepath.Join(folderPath, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	var metadata ClipMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// ConvertRawFolder converts a raw clip folder to MP4
func (s *Saver) ConvertRawFolder(folderPath string, deleteRaw bool) error {
	// Read metadata
	metadata, err := ReadMetadata(folderPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	folderName := filepath.Base(folderPath)
	mp4Path := filepath.Join(s.outputDir, folderName+".mp4")
	absMp4, _ := filepath.Abs(mp4Path)

	videoPath := filepath.Join(folderPath, "video.ts")
	absVideo, _ := filepath.Abs(videoPath)

	audioPath := filepath.Join(folderPath, "audio.pcm")
	absAudio, _ := filepath.Abs(audioPath)

	args := []string{"-y"}

	// Add duration seeking for video
	if metadata.DurationSec > 0 {
		args = append(args, "-sseof", fmt.Sprintf("-%d", metadata.DurationSec))
	}
	args = append(args, "-i", absVideo)

	if metadata.HasAudio {
		// Add audio input and merge
		args = append(args,
			"-f", "f32le", "-ar", "48000", "-ac", "2", "-i", absAudio,
			"-c:v", "copy",
			"-c:a", "aac", "-b:a", "192k",
			"-shortest",
			absMp4,
		)
	} else {
		// Video only
		args = append(args, "-c", "copy", absMp4)
	}

	cmd := hiddenexec.Command(s.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		slog.Error("raw folder conversion failed", "error", err)
		return err
	}

	slog.Info("raw folder converted to mp4", "folder", folderPath, "output", mp4Path)

	// Delete raw folder if requested
	if deleteRaw {
		if err := os.RemoveAll(folderPath); err != nil {
			slog.Warn("failed to delete raw folder", "error", err)
		}
	}

	return nil
}
