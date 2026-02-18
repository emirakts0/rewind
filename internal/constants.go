package internal

import "time"

const (
	// Recording constraints
	MinFPS = 1
	MaxFPS = 240

	// Audio configuration
	DefaultSampleRate = 48000
	DefaultChannels   = 2

	// Save operation
	SaveDebounceSeconds  = 5
	SaveDebounceDuration = SaveDebounceSeconds * time.Second

	// Buffer monitoring
	BufferUpdateInterval = 1 * time.Second

	// Unit conversions
	BytesToMB      = 1024 * 1024
	MbpsToBytes    = 1000000
	MaxBufferUsage = 100.0

	// Error handling
	CriticalErrorDelay = 100 * time.Millisecond
)

// Valid segment durations in seconds
var ValidSegmentDurations = []int{2, 5, 10}
