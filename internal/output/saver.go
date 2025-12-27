package output

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	hiddenexec "rewind/internal/hardware"
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

	go s.saveAsync(data, opts)
	return nil
}

func (s *Saver) saveAsync(data []byte, opts *SaveOptions) {
	tsPath := filepath.Join(s.outputDir, opts.Filename+".ts")
	mp4Path := filepath.Join(s.outputDir, opts.Filename+".mp4")

	f, err := os.Create(tsPath)
	if err != nil {
		slog.Error("failed to create file", "path", tsPath, "error", err)
		return
	}

	w := bufio.NewWriterSize(f, 8*1024*1024)
	if _, err := w.Write(data); err != nil {
		slog.Error("failed to write data", "error", err)
		f.Close()
		return
	}
	w.Flush()
	f.Close()

	if opts.ConvertToMP4 {
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
	} else {
		slog.Info("clip saved", "path", tsPath)
	}
}
