package audio

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"rewind/internal/buffer"

	"github.com/gen2brain/malgo"
)

const (
	SampleRate     = 48000
	Channels       = 2
	Format         = malgo.FormatF32
	BytesPerSample = 4
	BytesPerFrame  = BytesPerSample * Channels
)

type CaptureManager struct {
	ctx         *malgo.AllocatedContext
	streams     []*Stream
	mixedBuffer *buffer.Buffer
	running     bool
	mu          sync.Mutex
	quitChan    chan struct{}
}

type Stream struct {
	device  *malgo.Device
	buffer  *buffer.Buffer
	volume  float32
	isReady bool
}

func NewCaptureManager() (*CaptureManager, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	return &CaptureManager{
		ctx:         ctx,
		mixedBuffer: buffer.New(0),
		quitChan:    make(chan struct{}),
	}, nil
}

func (cm *CaptureManager) Close() {
	cm.Stop()
	if cm.ctx != nil {
		cm.ctx.Uninit()
		cm.ctx.Free()
	}
}

func (cm *CaptureManager) StartCapture(micID, sysID string, micVol, sysVol int, durationSec int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running {
		return fmt.Errorf("already running")
	}

	slog.Info("starting audio capture (WASAPI)", "mic", micID, "sys", sysID, "micVol", micVol, "sysVol", sysVol, "duration", durationSec)

	cm.streams = nil

	// Create buffer based on duration
	bufferSize := SampleRate * BytesPerFrame * durationSec
	cm.mixedBuffer = buffer.New(bufferSize)

	// Calculate gains based on 0-200 range (100 = 1.0)
	micGain := float32(micVol) / 100.0
	sysGain := float32(sysVol) / 100.0

	// Clamp gains to be safe (max 2.0 = 200%)
	if micGain < 0 {
		micGain = 0
	} else if micGain > 2.0 {
		micGain = 2.0
	}
	if sysGain < 0 {
		sysGain = 0
	} else if sysGain > 2.0 {
		sysGain = 2.0
	}

	type deviceEntry struct {
		id     string
		loop   bool
		volume float32
	}

	var deviceIDs []deviceEntry
	if micID != "" {
		deviceIDs = append(deviceIDs, deviceEntry{micID, false, micGain})
	}
	if sysID != "" {
		deviceIDs = append(deviceIDs, deviceEntry{sysID, true, sysGain})
	}

	if len(deviceIDs) == 0 {
		return fmt.Errorf("no devices selected")
	}

	for _, dev := range deviceIDs {
		id, err := ParseDeviceID(dev.id)
		if err != nil {
			slog.Error("invalid device id", "id", dev.id, "error", err)
			continue
		}

		stream := &Stream{
			buffer: buffer.New(SampleRate * BytesPerFrame * 2),
			volume: dev.volume,
		}

		deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
		deviceConfig.Capture.Format = Format
		deviceConfig.Capture.Channels = Channels
		deviceConfig.SampleRate = SampleRate

		if dev.loop {
			deviceConfig.DeviceType = malgo.Loopback
			deviceConfig.Playback.DeviceID = id.Pointer()
		} else {
			deviceConfig.Capture.DeviceID = id.Pointer()
		}

		onRecv := func(pOutput, pInput []byte, framecount uint32) {
			stream.buffer.Write(pInput)
		}

		callbacks := malgo.DeviceCallbacks{Data: onRecv}

		device, err := malgo.InitDevice(cm.ctx.Context, deviceConfig, callbacks)
		if err != nil {
			slog.Error("failed to init device", "id", dev.id, "error", err)
			continue
		}

		if err := device.Start(); err != nil {
			device.Uninit()
			slog.Error("failed to start device", "id", dev.id, "error", err)
			continue
		}

		stream.device = device
		stream.isReady = true
		cm.streams = append(cm.streams, stream)
		slog.Info("audio stream started", "loopback", dev.loop, "volume", dev.volume)
	}

	if len(cm.streams) == 0 {
		return fmt.Errorf("failed to start any audio streams")
	}

	cm.running = true
	cm.quitChan = make(chan struct{})
	go cm.mixLoop()

	return nil
}

func (cm *CaptureManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return
	}

	close(cm.quitChan)
	cm.running = false

	for _, s := range cm.streams {
		s.device.Uninit()
	}
	cm.streams = nil
}

func (cm *CaptureManager) GetBuffer() *buffer.Buffer {
	return cm.mixedBuffer
}

func (cm *CaptureManager) IsRunning() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.running
}

func (cm *CaptureManager) mixLoop() {
	const framesPerChunk = 960
	chunkSize := framesPerChunk * BytesPerFrame

	mixBuf := make([]float32, framesPerChunk*Channels)
	byteBuf := make([]byte, chunkSize)
	streamBuf := make([]byte, chunkSize)

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-cm.quitChan:
			return
		case <-ticker.C:
			for i := range mixBuf {
				mixBuf[i] = 0
			}

			// Mix all streams with volume
			for _, s := range cm.streams {
				n, _ := s.buffer.Read(streamBuf)
				if n == 0 {
					continue
				}

				numSamples := n / BytesPerSample
				for i := 0; i < numSamples && i < len(mixBuf); i++ {
					bits := binary.LittleEndian.Uint32(streamBuf[i*4 : (i+1)*4])
					sample := math.Float32frombits(bits)
					// Apply Volume/Gain
					mixBuf[i] += sample * s.volume
				}
			}

			// Hard Clip Limiter
			for i := range mixBuf {
				if mixBuf[i] > 1.0 {
					mixBuf[i] = 1.0
				} else if mixBuf[i] < -1.0 {
					mixBuf[i] = -1.0
				}
			}

			for i, sample := range mixBuf {
				bits := math.Float32bits(sample)
				binary.LittleEndian.PutUint32(byteBuf[i*4:(i+1)*4], bits)
			}

			cm.mixedBuffer.Write(byteBuf)
		}
	}
}
