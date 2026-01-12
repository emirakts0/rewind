package output

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	hiddenexec "rewind/internal/utils"
	stdruntime "runtime"
	"runtime/debug"
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
}

func DefaultSaveOptions(filename string) *SaveOptions {
	return &SaveOptions{
		Filename:     filename,
		ConvertToMP4: true,
		DeleteTS:     true,
	}
}

type Snapshotter interface {
	Snapshot() []byte
}

func (s *Saver) Save(src Snapshotter, opts *SaveOptions) error {
	data := src.Snapshot()
	if len(data) == 0 {
		return fmt.Errorf("buffer is empty")
	}

	go s.processSave(data, opts)
	return nil
}

func (s *Saver) processSave(data []byte, opts *SaveOptions) {
	// 1. Write to temporary TS file
	tsPath, err := s.writeTempFile(data, opts)

	// Explicitly release the strong reference to the huge buffer
	// This makes it eligible for GC immediately, while the conversion runs
	data = nil
	stdruntime.GC()
	debug.FreeOSMemory()

	if err != nil {
		slog.Error("failed to write temp file", "error", err)
		return
	}

	// 2. Convert to MP4
	if opts.ConvertToMP4 {
		s.convertToMP4(tsPath, opts)
	} else {
		slog.Info("clip saved", "path", tsPath)
	}
}

func (s *Saver) writeTempFile(data []byte, opts *SaveOptions) (string, error) {
	tsPath := filepath.Join(s.outputDir, opts.Filename+".ts")
	f, err := os.Create(tsPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 8*1024*1024)
	if _, err := w.Write(data); err != nil {
		return "", err
	}
	if err := w.Flush(); err != nil {
		return "", err
	}

	return tsPath, nil
}

func (s *Saver) convertToMP4(tsPath string, opts *SaveOptions) {
	mp4Path := filepath.Join(s.outputDir, opts.Filename+".mp4")
	absTs, _ := filepath.Abs(tsPath)
	absMp4, _ := filepath.Abs(mp4Path)

	cmd := hiddenexec.Command(s.ffmpegPath, "-y", "-i", absTs, "-c", "copy", absMp4)
	if err := cmd.Run(); err == nil {
		slog.Info("clip saved", "path", mp4Path)
		if opts.DeleteTS {
			os.Remove(absTs)
		}
	} else {
		slog.Error("conversion failed", "error", err)
	}
}
