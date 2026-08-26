package process

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// OutputCapture manages capturing stdout and stderr from a process.
type OutputCapture struct {
	mu        sync.Mutex
	stdout    *RingBuffer
	stderr    *RingBuffer
	merged    *RingBuffer
	maxSize   int64
	overflowed bool
}

// NewOutputCapture creates a new OutputCapture with the given max size per buffer.
func NewOutputCapture(maxSize int64) *OutputCapture {
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // 10MB default
	}
	return &OutputCapture{
		stdout:  NewRingBuffer(maxSize),
		stderr:  NewRingBuffer(maxSize),
		merged:  NewRingBuffer(maxSize * 2),
		maxSize: maxSize,
	}
}

// Stdout returns a writer that captures stdout.
func (oc *OutputCapture) Stdout() io.Writer {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	return io.MultiWriter(oc.stdout, oc.merged)
}

// Stderr returns a writer that captures stderr.
func (oc *OutputCapture) Stderr() io.Writer {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	return io.MultiWriter(oc.stderr, oc.merged)
}

// SetStdout sets the stdout writer for the capture.
func (oc *OutputCapture) SetStdout(w io.Writer) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.stdout = NewRingBuffer(oc.maxSize)
	oc.merged = NewRingBuffer(oc.maxSize * 2)
}

// SetStderr sets the stderr writer for the capture.
func (oc *OutputCapture) SetStderr(w io.Writer) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.stderr = NewRingBuffer(oc.maxSize)
	oc.merged = NewRingBuffer(oc.maxSize * 2)
}

// GetStdout returns the captured stdout as a string.
func (oc *OutputCapture) GetStdout() string {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	return oc.stdout.String()
}

// GetStderr returns the captured stderr as a string.
func (oc *OutputCapture) GetStderr() string {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	return oc.stderr.String()
}

// GetMerged returns the merged stdout+stderr output.
func (oc *OutputCapture) GetMerged() string {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	return oc.merged.String()
}

// HasOverflowed checks if any buffer has overflowed.
func (oc *OutputCapture) HasOverflowed() bool {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	return oc.stdout.Overflowed() || oc.stderr.Overflowed() || oc.merged.Overflowed()
}

// Reset clears all captured output.
func (oc *OutputCapture) Reset() {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.stdout.Reset()
	oc.stderr.Reset()
	oc.merged.Reset()
	oc.overflowed = false
}

// RingBuffer is a circular buffer for efficiently storing process output.
type RingBuffer struct {
	buf       []byte
	start     int
	end       int
	size      int64
	count     int64
	overflowed bool
}

// NewRingBuffer creates a new RingBuffer with the given max size.
func NewRingBuffer(maxSize int64) *RingBuffer {
	if maxSize <= 0 {
		maxSize = 1024 * 1024 // 1MB
	}
	return &RingBuffer{
		buf:  make([]byte, maxSize),
		size: maxSize,
	}
}

// Write implements io.Writer.
func (rb *RingBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if rb.count >= rb.size {
		rb.overflowed = true
		return 0, fmt.Errorf("ring buffer overflow: output exceeds maximum size")
	}
	available := rb.size - rb.count
	if int64(n) > available {
		rb.overflowed = true
		for i := int64(0); i < available; i++ {
			rb.buf[rb.end] = p[i]
			rb.end = (rb.end + 1) % int(rb.size)
			rb.count++
		}
		return int(available), fmt.Errorf("ring buffer overflow: write exceeds available space")
	}
	for _, b := range p {
		rb.buf[rb.end] = b
		rb.end = (rb.end + 1) % int(rb.size)
		rb.count++
	}
	return n, nil
}

// String returns the contents of the ring buffer as a string.
func (rb *RingBuffer) String() string {
	if rb.count == 0 {
		return ""
	}
	result := make([]byte, rb.count)
	for i := int64(0); i < rb.count; i++ {
		idx := (rb.start + int(i)) % int(rb.size)
		result[i] = rb.buf[idx]
	}
	return string(result)
}

// Reset clears the ring buffer.
func (rb *RingBuffer) Reset() {
	rb.start = 0
	rb.end = 0
	rb.count = 0
}

// Len returns the number of bytes in the buffer.
func (rb *RingBuffer) Len() int64 {
	return rb.count
}

// Overflowed returns true if the ring buffer has overflowed.
func (rb *RingBuffer) Overflowed() bool {
	return rb.overflowed
}

// WriteTo writes the ring buffer contents to an io.Writer.
func (rb *RingBuffer) WriteTo(w io.Writer) (int64, error) {
	data := []byte(rb.String())
	n, err := w.Write(data)
	return int64(n), err
}

// Check if os.Stdout is used somewhere for compatibility
var _ = os.Stdout
