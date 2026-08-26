package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/pkg/process"
)

func TestRedGreen(t *testing.T) {
	allPassed := true
	var results []string

	// Test 1: Verify Runner timeout handling
	t.Run("TestRunnerTimeoutHandling", func(t *testing.T) {
		runner := process.NewRunner()
		ctx := context.Background()

		// Create a command that will run longer than the timeout
		// Using sleep command with a short timeout to test timeout handling
		timeout := 100 * time.Millisecond
		result, err := runner.RunWithTimeout(ctx, timeout, "sleep", "10")

		if err != nil {
			results = append(results, fmt.Sprintf("FAIL: TestRunnerTimeoutHandling - unexpected error: %v", err))
			allPassed = false
			return
		}

		// After timeout, TimedOut should be true
		if !result.TimedOut {
			results = append(results, "FAIL: TestRunnerTimeoutHandling - expected TimedOut=true but got false")
			allPassed = false
		} else {
			results = append(results, "PASS: TestRunnerTimeoutHandling")
		}
	})

	// Test 2: Verify Runner context cancellation handling
	t.Run("TestRunnerCancellationHandling", func(t *testing.T) {
		runner := process.NewRunner()
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel the context before running
		cancel()

		timeout := 5 * time.Second
		result, err := runner.RunWithTimeout(ctx, timeout, "echo", "test")

		if err != nil {
			results = append(results, fmt.Sprintf("FAIL: TestRunnerCancellationHandling - unexpected error: %v", err))
			allPassed = false
			return
		}

		// After context cancellation, Killed should be true
		if !result.Killed {
			results = append(results, "FAIL: TestRunnerCancellationHandling - expected Killed=true but got false")
			allPassed = false
		} else {
			results = append(results, "PASS: TestRunnerCancellationHandling")
		}
	})

	// Test 3: Verify Runner normal execution handling
	t.Run("TestRunnerNormalExecution", func(t *testing.T) {
		runner := process.NewRunner()
		ctx := context.Background()

		timeout := 5 * time.Second
		result, err := runner.RunWithTimeout(ctx, timeout, "echo", "hello")

		if err != nil {
			results = append(results, fmt.Sprintf("FAIL: TestRunnerNormalExecution - unexpected error: %v", err))
			allPassed = false
			return
		}

		// Normal execution should not be timed out or killed
		if result.TimedOut {
			results = append(results, "FAIL: TestRunnerNormalExecution - TimedOut should be false")
			allPassed = false
		} else if result.Killed {
			results = append(results, "FAIL: TestRunnerNormalExecution - Killed should be false")
			allPassed = false
		} else if result.ExitCode != 0 {
			results = append(results, fmt.Sprintf("FAIL: TestRunnerNormalExecution - expected ExitCode=0 but got %d", result.ExitCode))
			allPassed = false
		} else {
			results = append(results, "PASS: TestRunnerNormalExecution")
		}
	})

	// Test 4: Verify Limiter context handling
	t.Run("TestLimiterContextHandling", func(t *testing.T) {
		limiter := process.NewLimiter(process.DefaultLimiterConfig())

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// ApplyLimits should return an error when context is cancelled
		_, _, err := limiter.ApplyLimits(ctx, "echo", []string{"test"})

		// The correct behavior should be to return an error when context is cancelled
		if err == nil {
			results = append(results, "FAIL: TestLimiterContextHandling - expected error for cancelled context but got nil")
			allPassed = false
		} else {
			results = append(results, "PASS: TestLimiterContextHandling")
		}
	})

	// Print results
	fmt.Println("\n============================================")
	fmt.Println("Test Results:")
	fmt.Println("============================================")
	for _, r := range results {
		fmt.Println(r)
	}
	fmt.Println("============================================")

	if allPassed {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		os.Exit(0)
	} else {
		fmt.Println("RED（红灯，缺陷未修复）")
		os.Exit(1)
	}
}

func TestRedGreenStdin(t *testing.T) {
	allPassed := true
	var results []string

	// Test Runner with stdin timeout handling
	t.Run("TestRunnerStdinTimeout", func(t *testing.T) {
		runner := process.NewRunner()
		ctx := context.Background()

		timeout := 100 * time.Millisecond
		result, err := runner.RunWithStdinTimeout(ctx, "test", timeout, "sleep", "10")

		if err != nil {
			results = append(results, fmt.Sprintf("FAIL: TestRunnerStdinTimeout - unexpected error: %v", err))
			allPassed = false
			return
		}

		if !result.TimedOut {
			results = append(results, "FAIL: TestRunnerStdinTimeout - expected TimedOut=true but got false")
			allPassed = false
		} else {
			results = append(results, "PASS: TestRunnerStdinTimeout")
		}
	})

	// Print results
	fmt.Println("\n============================================")
	fmt.Println("Test Results (Stdin):")
	fmt.Println("============================================")
	for _, r := range results {
		fmt.Println(r)
	}
	fmt.Println("============================================")

	if allPassed {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	} else {
		fmt.Println("RED（红灯，缺陷未修复）")
	}
}

func TestRedGreenSummary(t *testing.T) {
	// Run all tests and produce final result
	hasFailures := false
	var failures []string

	// Test timeout handling
	runner := process.NewRunner()
	ctx := context.Background()
	timeout := 100 * time.Millisecond

	result, err := runner.RunWithTimeout(ctx, timeout, "sleep", "10")
	if err != nil {
		failures = append(failures, fmt.Sprintf("Timeout test: unexpected error: %v", err))
		hasFailures = true
	} else if !result.TimedOut {
		failures = append(failures, "Timeout test: expected TimedOut=true")
		hasFailures = true
	}

	// Test cancellation handling
	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	result2, err2 := runner.RunWithTimeout(ctx2, 5*time.Second, "echo", "test")
	if err2 != nil {
		failures = append(failures, fmt.Sprintf("Cancel test: unexpected error: %v", err2))
		hasFailures = true
	} else if !result2.Killed {
		failures = append(failures, "Cancel test: expected Killed=true")
		hasFailures = true
	}

	// Test limiter context handling
	limiter := process.NewLimiter(process.DefaultLimiterConfig())
	ctx3, cancel3 := context.WithCancel(context.Background())
	cancel3()

	_, _, err3 := limiter.ApplyLimits(ctx3, "echo", []string{"test"})
	if err3 == nil {
		failures = append(failures, "Limiter test: expected error for cancelled context")
		hasFailures = true
	}

	// Print summary
	fmt.Println("\n============================================")
	fmt.Println("Final Test Summary:")
	fmt.Println("============================================")

	if hasFailures {
		fmt.Println("Test Results:")
		for _, f := range failures {
			fmt.Printf("  - FAIL: %s\n", f)
		}
		fmt.Println("============================================")
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fail()
	} else {
		fmt.Println("  - All tests PASSED")
		fmt.Println("============================================")
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
	fmt.Println("============================================")
}

// Suppress unused variable warning
var _ = strings.Contains