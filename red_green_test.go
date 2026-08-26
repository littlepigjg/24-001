package codesandbox

import (
	"context"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/pkg/process"
)

func TestRedGreen(t *testing.T) {
	e := process.NewExecutor()

	customMemory := int64(1234567)
	customTimeout := 5 * time.Second

	opts := process.ExecuteOptions{
		Timeout:    customTimeout,
		MemoryLimit: customMemory,
	}

	_, err := e.ExecutePython(context.Background(), "print('hello')", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snap := e.GetLimiterSnapshot()

	if snap.MemoryLimit != customMemory {
		t.Log("RED（红灯，缺陷未修复）")
		t.FailNow()
	}

	if snap.TimeLimit != customTimeout {
		t.Log("RED（红灯，缺陷未修复）")
		t.FailNow()
	}

	t.Log("GREEN（绿灯，缺陷已修复）")
}
