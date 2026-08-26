package codesandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/pkg/osutil"
	"github.com/codesandbox/codesandbox/pkg/process"
)

func TestRedGreen(t *testing.T) {
	r := process.NewRunner()
	ctx := context.Background()

	cmd := exec.Command("sleep", "0.1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start test process: %v", err)
	}
	testPid := cmd.Process.Pid
	time.Sleep(200 * time.Millisecond)

	if !osutil.IsProcessZombie(testPid) {
		t.Fatalf("failed to create zombie process for testing")
	}

	result, err := r.RunWithTimeout(ctx, 10*time.Millisecond, "sleep", "5")
	if err != nil {
		t.Fatalf("RunWithTimeout returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("RunWithTimeout returned nil result")
	}

	time.Sleep(50 * time.Millisecond)

	active := r.ActiveProcesses()
	if active > 0 {
		reaped := r.WaitForProcesses()
		time.Sleep(50 * time.Millisecond)
		active2 := r.ActiveProcesses()
		if active2 > 0 {
			fmt.Println("RED (红灯，缺陷未修复)")
			fmt.Printf("  ActiveProcesses before sweep: %d, WaitForProcesses reaped: %d, after sweep: %d\n", active, reaped, active2)
			t.Fail()
			return
		}
	}

	e := process.NewExecutor()
	if e.GetActiveProcessCount() > 0 {
		e.SweepZombieProcesses()
		time.Sleep(50 * time.Millisecond)
		if e.GetActiveProcessCount() > 0 {
			fmt.Println("RED (红灯，缺陷未修复)")
			t.Fail()
			return
		}
	}

	if osutil.IsProcessZombie(testPid) {
		proc, err := os.FindProcess(testPid)
		if err == nil {
			proc.Wait()
		}
		time.Sleep(20 * time.Millisecond)
	}

	cmd2 := exec.Command("sleep", "0.1")
	cmd2.Start()
	pid2 := cmd2.Process.Pid
	time.Sleep(200 * time.Millisecond)
	if osutil.IsProcessZombie(pid2) {
		proc2, _ := os.FindProcess(pid2)
		if proc2 != nil {
			proc2.Wait()
		}
		time.Sleep(20 * time.Millisecond)
	}

	fmt.Println("GREEN (绿灯，缺陷已修复)")
}
