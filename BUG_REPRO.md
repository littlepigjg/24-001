# BUG_REPRO.md - 优雅关闭不完整缺陷复现报告

## 缺陷概述

**缺陷类型**: 优雅关闭不完整 (Graceful Shutdown Incomplete)

**严重程度**: 高

**影响范围**: 所有在服务器关闭时正在执行的代码任务

**根因**: `ExecutionService.Shutdown()` 方法在关闭时先强制杀死所有运行中的进程，
然后才等待 goroutine 完成。这导致 goroutine 检测到关闭信号后跳过结果保存，
造成执行结果数据丢失。

## 缺陷代码位置

### 文件1: `pkg/process/manager.go`

**行号**: 225-249 (Shutdown)

```go
func (r *Runner) Shutdown(ctx context.Context) error {
    r.shutdownOnce.Do(func() {
        close(r.shutdownCh)  // 关闭信号
    })
    
    // 杀死所有运行中的进程（包括子进程）
    for _, cmd := range cmds {
        if cmd.Process != nil {
            processGroupID := -cmd.Process.Pid
            syscall.Kill(processGroupID, syscall.SIGKILL)
        }
    }
    return nil
}
```

**行号**: 251-258 (IsShutdownTriggered)

```go
func (r *Runner) IsShutdownTriggered() bool {
    select {
    case <-r.shutdownCh:
        return true
    default:
        return false
    }
}
```

### 文件2: `internal/service/execution_service.go`

**行号**: 160-168 (runExecution 中的检查)

```go
// 执行完成后检查关闭状态
if s.executor.RunnerAccess().IsShutdownTriggered() {
    // BUG: 检测到关闭后跳过结果保存
    exec.Status = model.StatusCanceled
    exec.ErrorMessage = "execution interrupted by server shutdown"
    return  // 结果未保存！
}
```

**行号**: 289-309 (Shutdown 方法 - 缺陷核心)

```go
func (s *ExecutionService) Shutdown(timeout time.Duration) error {
    // BUG: 先杀死进程！
    s.executor.Shutdown(sdCtx)
    
    // 然后才等待 goroutine（但结果已丢失）
    s.WaitForCompletion(timeout)
}
```

**问题**: 操作顺序错误 - 应该先等待，再杀死。

### 文件3: `pkg/process/runner.go`

**行号**: 31-34 (委托方法)

```go
func (e *Executor) Shutdown(ctx context.Context) error {
    return e.runner.Shutdown(ctx)  // 委托给 Runner
}
```

**行号**: 41-44 (Runner 访问)

```go
func (e *Executor) RunnerAccess() *Runner {
    return e.runner  // 暴露 Runner 供外部检查
}
```

### 文件4: `cmd/server/main.go`

**行号**: 139-143 (触发缺陷)

```go
// 短超时加剧数据丢失
if err := execSvc.Shutdown(500 * time.Millisecond); err != nil {
    log.Warnf("Execution service shutdown warning: %v", err)
}
```

## 复现步骤

### 前提条件
- Go 1.x 编译环境
- 项目代码已编译 (`go build ./...`)

### 复现步骤
1. 在项目根目录执行:
   ```bash
   go test -v -count=1 -run '^TestRedGreen$' .
   ```

2. 观察测试输出:
   ```
   RED (红灯，缺陷未修复)
   --- FAIL: TestRedGreen
   ```

3. 关键输出信息:
   - `Execution status: canceled` - 执行被取消
   - `Execution result is nil: true` - 结果未保存
   - `Execution result was NOT properly saved after shutdown - DEFECT EXISTS`

### 预期行为（修复后）
- 执行状态应为 `completed`
- 执行结果应包含 stdout 输出
- 测试应输出 `GREEN (绿灯，缺陷已修复)`

## 数据流分析

1. 用户提交代码执行请求 (`ExecutionService.Execute`)
2. 创建 goroutine 运行 `runExecution`
3. Goroutine 调用 `executor.ExecuteShell` 执行代码（sleep 0.5s）
4. 服务器收到关闭信号
5. `Shutdown()` 被调用:
   a. 调用 `executor.Shutdown()` → `Runner.Shutdown()`
      - 关闭 `shutdownCh` 通道
      - 向进程组发送 SIGKILL
   b. 调用 `WaitForCompletion(2s)` 等待 goroutine
6. Goroutine 中:
   - `cmd.Run()` 因进程被杀返回
   - `runExecution` 检查 `IsShutdownTriggered()` → true
   - 跳过结果保存，直接返回
   - `wg.Done()` 被调用
7. `WaitForCompletion` 返回
8. 结果查询返回 `result = nil`，状态为 `canceled`

## 修复方案

修改 `internal/service/execution_service.go` 中的 `Shutdown` 方法：

```go
func (s *ExecutionService) Shutdown(timeout time.Duration) error {
    // 修复: 先等待 goroutine 完成并保存结果
    if err := s.WaitForCompletion(timeout); err != nil {
        // 超时后再杀死剩余进程
        s.executor.Shutdown(sdCtx)
        // 再次等待处理被杀进程的结果
        s.WaitForCompletion(500 * time.Millisecond)
        return err
    }
    return nil
}
```

**修复涉及文件**:
1. `internal/service/execution_service.go` - 修改 `Shutdown` 方法
2. (可选) `pkg/process/manager.go` - 调整关闭逻辑

## 测试场景说明

测试使用 `sleep 0.5` 命令和 `Shutdown(2s)` 超时：
- **缺陷存在**: 先杀进程 → 结果丢失 → RED
- **缺陷修复**: 先等待 → 执行在 500ms 内完成 → 结果保存 → GREEN