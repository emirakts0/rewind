package buffer

import "sync"

// TODO: MUTEX BURDA GEREKSİZ OVERHEAD VEYA KİLİTLEME YARATIYOR MU İYİ DÜŞÜN.

// Buffer is a thread-safe circular buffer
type Buffer struct {
	buf  []byte
	head int // Absolute position
	tail int // Absolute position
	size int
	mu   sync.Mutex
}

func New(size int) *Buffer {
	return &Buffer{
		buf:  make([]byte, size),
		size: size,
	}
}

// Write appends data to the buffer. If buffer is full,
// it drops the oldest data (increments tail).
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n := len(p)
	if n > b.size {
		// If writing more than size, just write the last 'size' bytes
		p = p[n-b.size:]
		n = b.size
	}

	free := b.size - (b.head - b.tail)
	if free < n {
		drop := n - free
		b.tail += drop
	}

	for i := 0; i < n; i++ {
		b.buf[(b.head+i)%b.size] = p[i]
	}
	b.head += n
	return n, nil
}

// Read consumes data from the buffer.
func (b *Buffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	dataLen := b.head - b.tail
	if dataLen <= 0 {
		return 0, nil
	}

	req := len(p)
	if req > dataLen {
		req = dataLen
	}

	for i := 0; i < req; i++ {
		p[i] = b.buf[(b.tail+i)%b.size]
	}
	b.tail += req
	return req, nil
}

// Snapshot returns a copy of the valid data in the buffer
// without consuming it.
func (b *Buffer) Snapshot() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	dataLen := b.head - b.tail
	if dataLen <= 0 {
		return nil
	}

	result := make([]byte, dataLen)
	for i := 0; i < dataLen; i++ {
		result[i] = b.buf[(b.tail+i)%b.size]
	}
	return result
}

func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.head = 0
	b.tail = 0
}

func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.head - b.tail
}

func (b *Buffer) Size() int {
	return b.size
}
