package buffer

import "sync"

type Ring struct {
	data []byte
	head int
	size int
	full bool
	mu   sync.Mutex
}

func NewRing(size int) *Ring {
	return &Ring{
		data: make([]byte, size),
		size: size,
	}
}

func (rb *Ring) Write(p []byte) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	n := len(p)
	written := 0

	for written < n {
		space := rb.size - rb.head
		toWrite := n - written
		if toWrite > space {
			toWrite = space
		}
		copy(rb.data[rb.head:], p[written:written+toWrite])
		rb.head += toWrite
		written += toWrite
		if rb.head == rb.size {
			rb.head = 0
			rb.full = true
		}
	}

	return n
}

func (rb *Ring) Snapshot() []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full && rb.head == 0 {
		return nil
	}

	if !rb.full {
		snap := make([]byte, rb.head)
		copy(snap, rb.data[:rb.head])
		return snap
	}

	snap := make([]byte, rb.size)
	p1 := copy(snap, rb.data[rb.head:])
	copy(snap[p1:], rb.data[:rb.head])
	return snap
}

func (rb *Ring) Size() int {
	return rb.size
}

func (rb *Ring) UsedBytes() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.full {
		return rb.size
	}
	return rb.head
}

func (rb *Ring) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.full = false
}
