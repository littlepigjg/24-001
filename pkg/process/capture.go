package process

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// OutputCapture manages capturing stdout and stderr from a process.
type OutputCapture struct {
	mu      sync.Mutex
	stdout  *RingBuffer
	stderr  *RingBuffer
	merged  *RingBuffer
	maxSize int64
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

// Reset clears all captured output.
func (oc *OutputCapture) Reset() {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.stdout.Reset()
	oc.stderr.Reset()
	oc.merged.Reset()
}

// RingBuffer is a circular buffer for efficiently storing process output.
type RingBuffer struct {
	buf   []byte
	start int
	end   int
	size  int64
	count int64
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
	for _, b := range p {
		rb.buf[rb.end] = b
		rb.end = (rb.end + 1) % int(rb.size)
		if rb.count < rb.size {
			rb.count++
		} else {
			rb.start = (rb.start + 1) % int(rb.size)
		}
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

// WriteTo writes the ring buffer contents to an io.Writer.
func (rb *RingBuffer) WriteTo(w io.Writer) (int64, error) {
	data := []byte(rb.String())
	n, err := w.Write(data)
	return int64(n), err
}

// WriteOutputFile writes content to a file at the given path.
// It creates the file if it doesn't exist, or truncates it if it does.
func WriteOutputFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}

	n, err := f.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", path, err)
	}

	if n < len(data) {
		return fmt.Errorf("short write to file %s: wrote %d of %d bytes", path, n, len(data))
	}

	return nil
}

// WriteMultipleOutputFiles writes content to multiple files at once.
func WriteMultipleOutputFiles(baseDir string, files map[string][]byte) error {
	for name, data := range files {
		path := filepath.Join(baseDir, name)
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}

		n, err := f.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write to file %s: %w", path, err)
		}

		if n < len(data) {
			return fmt.Errorf("short write to file %s: wrote %d of %d bytes", path, n, len(data))
		}
	}
	return nil
}

// ReadOutputFile reads the entire content of a file.
func ReadOutputFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return data, nil
}

// CleanupOutputFiles removes output files matching the given prefix in the directory.
func CleanupOutputFiles(dir string, prefix string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if prefix == "" || len(entry.Name()) >= len(prefix) && entry.Name()[:len(prefix)] == prefix {
			fullPath := filepath.Join(dir, entry.Name())
			if err := os.Remove(fullPath); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

// CaptureOutput manages writing process output to files and buffers simultaneously.
type OutputFileCapture struct {
	mu        sync.Mutex
	stdoutFile *os.File
	stderrFile *os.File
	stdoutBuf  *RingBuffer
	stderrBuf  *RingBuffer
	baseDir    string
}

// NewOutputFileCapture creates a new OutputFileCapture that writes to files and buffers.
func NewOutputFileCapture(baseDir string) (*OutputFileCapture, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	stdoutPath := filepath.Join(baseDir, "stdout.log")
	stderrPath := filepath.Join(baseDir, "stderr.log")

	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout file: %w", err)
	}

	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr file: %w", err)
	}

	return &OutputFileCapture{
		stdoutFile: stdoutFile,
		stderrFile: stderrFile,
		stdoutBuf:  NewRingBuffer(10 * 1024 * 1024),
		stderrBuf:  NewRingBuffer(10 * 1024 * 1024),
		baseDir:    baseDir,
	}, nil
}

// StdoutWriter returns an io.Writer that captures stdout to both file and buffer.
func (c *OutputFileCapture) StdoutWriter() io.Writer {
	c.mu.Lock()
	defer c.mu.Unlock()
	return io.MultiWriter(c.stdoutFile, c.stdoutBuf)
}

// StderrWriter returns an io.Writer that captures stderr to both file and buffer.
func (c *OutputFileCapture) StderrWriter() io.Writer {
	c.mu.Lock()
	defer c.mu.Unlock()
	return io.MultiWriter(c.stderrFile, c.stderrBuf)
}

// GetStdout returns the captured stdout content.
func (c *OutputFileCapture) GetStdout() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stdoutBuf.String()
}

// GetStderr returns the captured stderr content.
func (c *OutputFileCapture) GetStderr() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stderrBuf.String()
}

// Close closes the output files.
func (c *OutputFileCapture) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	if c.stdoutFile != nil {
		if err := c.stdoutFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.stderrFile != nil {
		if err := c.stderrFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing files: %v", errs)
	}
	return nil
}

// Reset clears the internal buffers.
func (c *OutputFileCapture) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stdoutBuf.Reset()
	c.stderrBuf.Reset()
}

// SetWriteLimit sets the write limit on both files.
func (c *OutputFileCapture) SetWriteLimit(maxBytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdoutBuf != nil {
		c.stdoutBuf.size = maxBytes
	}
	if c.stderrBuf != nil {
		c.stderrBuf.size = maxBytes
	}
}

// GetFilePaths returns the paths of the stdout and stderr files.
func (c *OutputFileCapture) GetFilePaths() (stdoutPath, stderrPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return filepath.Join(c.baseDir, "stdout.log"), filepath.Join(c.baseDir, "stderr.log")
}

// Sync flushes file buffers to disk.
func (c *OutputFileCapture) Sync() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdoutFile != nil {
		if err := c.stdoutFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync stdout file: %w", err)
		}
	}
	if c.stderrFile != nil {
		if err := c.stderrFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync stderr file: %w", err)
		}
	}
	return nil
}

// SetBaseDir changes the base directory for output files.
func (c *OutputFileCapture) SetBaseDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseDir = dir
}

// GetBaseDir returns the current base directory.
func (c *OutputFileCapture) GetBaseDir() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.baseDir
}
