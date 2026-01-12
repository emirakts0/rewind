package capture

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	hiddenexec "rewind/internal/utils"
	"strings"

	"sync"
)

type Capturer struct {
	config  *Config
	cmd     *exec.Cmd
	stdout  io.ReadCloser
	stdErr  io.ReadCloser
	running bool
	mu      sync.Mutex

	OnData  func(data []byte)
	OnError func(err error)
}

func NewCapturer(cfg *Config) (*Capturer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Capturer{config: cfg}, nil
}

func (c *Capturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("capturer already running")
	}

	builder := NewFFmpegCommandBuilder(c.config)
	args := builder.BuildArgs()

	c.cmd = hiddenexec.Command(c.config.FFmpegPath, args...)

	var err error
	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.stdErr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	slog.Info("starting ffmpeg", "command", c.config.FFmpegPath+" "+strings.Join(args, " "))

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	c.running = true
	go c.readLoop()

	return nil
}

func (c *Capturer) readLoop() {
	reader := bufio.NewReaderSize(c.stdout, 4*1024*1024)
	buf := make([]byte, 1024*1024)

	for {
		n, err := reader.Read(buf)
		if n > 0 && c.OnData != nil {
			// Pass buffer slice directly to avoid allocation loop.
			// The subscriber (Ring.Write) copies the data, so this is safe
			// as long as it returns before we overwrite buf.
			c.OnData(buf[:n])
		}
		if err != nil {
			if err != io.EOF && c.OnError != nil {
				c.OnError(err)
			}
			break
		}
	}

	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
}

func (c *Capturer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill ffmpeg: %w", err)
		}
		c.cmd.Wait()
	}

	c.running = false
	return nil
}

func (c *Capturer) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *Capturer) Config() *Config {
	return c.config
}
