package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/pkg/process"
)

func TestRedGreen(t *testing.T) {
	hasDefect := false

	t.Run("ring_buffer_handles_overflow", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				hasDefect = true
			}
		}()

		rb := process.NewRingBuffer(100)
		_, err := rb.Write(make([]byte, 200))
		if err == nil {
			hasDefect = true
		}
	})

	t.Run("output_capture_overflow_flag", func(t *testing.T) {
		oc := process.NewOutputCapture(100)
		_, err := oc.Stdout().Write(make([]byte, 500))
		if err == nil {
			hasDefect = true
		}
	})

	t.Run("runner_large_output_detection", func(t *testing.T) {
		runner := process.NewRunner()
		runner.MaxOutputSize = 512

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := runner.RunWithTimeout(ctx, 3*time.Second, "python3", "-c",
			"import sys; sys.stdout.write('x'*2000)")

		if err == nil && result != nil && result.ExitCode == 0 {
			if len(result.Stdout) < 2000 {
				hasDefect = true
			}
		}
	})

	t.Run("service_overflow_detection", func(t *testing.T) {
		runner := process.NewRunner()
		runner.MaxOutputSize = 512

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := runner.RunWithTimeout(ctx, 3*time.Second, "python3", "-c",
			"import sys; sys.stdout.write('x'*2000)")

		if err != nil || result == nil {
			t.Skip("runner returned error")
			return
		}

		if len(result.Stdout) > 0 && len(result.Stdout) < 2000 {
			wouldDetect := result.TimedOut && result.Stdout == ""
			if !wouldDetect {
				hasDefect = true
			}
		}
	})

	if hasDefect {
		fmt.Println("RED")
		t.Fatal("RED")
	} else {
		fmt.Println("GREEN")
	}
}

func TestRingBufferAdditional(t *testing.T) {
	hasDefect := false

	t.Run("write_exactly_fills_buffer", func(t *testing.T) {
		rb := process.NewRingBuffer(100)
		data := make([]byte, 100)
		for i := range data {
			data[i] = byte(i)
		}

		n, err := rb.Write(data)
		if err != nil {
			hasDefect = true
		}
		if n != 100 {
			hasDefect = true
		}
	})

	t.Run("write_zero_length_data", func(t *testing.T) {
		rb := process.NewRingBuffer(100)
		n, err := rb.Write([]byte{})
		if err != nil {
			hasDefect = true
		}
		if n != 0 {
			hasDefect = true
		}
	})

	t.Run("multiple_writes_overflow", func(t *testing.T) {
		rb := process.NewRingBuffer(200)
		var lastErr error
		for i := 0; i < 50; i++ {
			chunk := make([]byte, 10)
			for j := range chunk {
				chunk[j] = byte(i + j)
			}
			_, err := rb.Write(chunk)
			if err != nil {
				lastErr = err
				break
			}
		}
		if lastErr == nil {
			hasDefect = true
		}
	})

	if hasDefect {
		fmt.Println("RED")
		t.Fatal("RED")
	} else {
		fmt.Println("GREEN")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
