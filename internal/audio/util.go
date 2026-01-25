package audio

// CalculateMixedBufferSize returns the size of the final mixed audio buffer which
// stores the last 'seconds' of audio for both mic and system combined.
func CalculateMixedBufferSize(seconds int) int {
	return SampleRate * BytesPerFrame * seconds
}

// CalculateStreamBufferSize returns the size of the temporary buffer used by
// an individual audio stream (mic or system) before mixing.
func CalculateStreamBufferSize(seconds int) int {
	return SampleRate * BytesPerFrame * seconds
}
