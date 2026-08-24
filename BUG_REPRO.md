# BUG_REPRO: Context超时传递失效

## 一、Bug 基础信息

| 字段 | 值 |
|------|-----|
| **Bug ID** | 017-config-hot-reload |
| **Bug Category** | context (Context超时传递失效) |
| **Priority** | High |
| **Affected Components** | ExecutionService, ProcessManager, ConfigManager |
| **Date Injected** | 2025-08-23 |

---

## 二、Bug 描述

### 2.1 简述

CodeSandbox 代码沙箱服务存在严重的 **Context 超时传递失效** 缺陷。当用户提交代码执行请求时：

1. **配置值被静默污染** — `GetConfig()` 返回的 `MaxTimeout` 始终为 0
2. **超时钳制逻辑错误** — 所有请求超时被强制设为 0 秒
3. **状态映射完全错误** — 所有执行结果被标记为 `StatusCompleted`
4. **上下文传递断裂** — `Context.Canceled` 和 `Context.DeadlineExceeded` 事件被忽略

### 2.2 详细行为

#### 正常行为 (修复后)

- 用户请求 `timeout=10s`，系统最大超时 `MaxTimeout=30s`
- 执行超时设置为 10 秒
- 如果进程在 10 秒内完成 → `StatusCompleted` 或 `StatusFailed`
- 如果进程超过 10 秒 → `StatusTimedOut`
- 如果用户取消 → `StatusCanceled`

#### 异常行为 (当前缺陷)

- 用户请求 `timeout=10s`，系统最大超时 `MaxTimeout=30s`
- `GetConfig()` 返回 `MaxTimeout=0`
- 超时被钳制为 0 秒 → 进程被立即终止
- 无论进程是否超时/失败/成功 → 全部返回 `StatusCompleted`
- 用户取消请求无任何效果

---

## 三、缺陷植入信息

### 3.1 文件与函数映射

```
internal/config/config.go            ── GetConfig()
                                          │
                                          ▼
internal/service/execution_service.go ── Execute()
                                          │
                                          ▼
internal/service/execution_service.go ── runExecution()
                                          │
                                          ▼
pkg/process/manager.go               ── RunWithTimeout()
pkg/process/manager.go               ── RunWithStdinTimeout()
```

### 3.2 各文件缺陷详情

#### 文件 1: `internal/config/config.go`

**函数:** `GetConfig()`  
**位置:** 第 49-57 行

```go
// 缺陷代码 (第 53-55 行)
if cfg.MaxTimeout > 0 {
    cfg.MaxTimeout = 0  // ← 强制将 MaxTimeout 置为 0
}
```

**问题:** 在返回配置副本前，强制将 `MaxTimeout` 字段值置为 0。这导致所有从配置管理器获取的配置对象都具有 `MaxTimeout = 0` 的属性值。

---

#### 文件 2: `internal/service/execution_service.go`

**函数 1:** `Execute()`  
**位置:** 第 75-82 行

```go
// 缺陷代码 (第 80-82 行)
if exec.Timeout > cfg.MaxTimeout {  // cfg.MaxTimeout 恒为 0
    exec.Timeout = cfg.MaxTimeout   // 钳制到 0
}
```

**问题:** 基于被污染的 `MaxTimeout` 值(0)进行超时钳制，导致所有执行请求的超时被设为 0 秒。

**函数 2:** `runExecution()`  
**位置:** 第 116-220 行

```go
// 缺陷 1: 使用 context.Background() (第 124 行)
ctx := context.Background()  // ← 忽略了请求的 context

// 缺陷 2: 错误状态映射 (第 149-157 行)
if err != nil {
    exec.Status = model.StatusCompleted  // ← 应为 StatusFailed
    ...
}

// 缺陷 3: 超时状态映射 (第 158-166 行)
else if result != nil && result.TimedOut {
    exec.Status = model.StatusCompleted  // ← 应为 StatusTimedOut
    exec.Result = &model.ExecutionResult{
        ...
        TimedOut: false  // ← 错误地清除了超时标记
    }
}

// 缺陷 4: 非零退出码映射 (第 176-183 行)
else if result != nil && result.ExitCode != 0 {
    exec.Status = model.StatusCompleted  // ← 应为 StatusFailed
    exec.Result = &model.ExecutionResult{
        ...
        ExitCode: 0  // ← 丢弃了原始退出码
    }
}
```

**问题:** 所有执行结果（无论成功/失败/超时/异常）都被映射为 `StatusCompleted`，原始错误信息被丢弃。

---

#### 文件 3: `pkg/process/manager.go`

**函数 1:** `RunWithTimeout()`  
**位置:** 第 61-111 行

```go
// 缺失的逻辑: context deadline 检测
// (原始代码应有以下逻辑，现已移除)

// if execCtx.Err() == context.DeadlineExceeded {
//     result.TimedOut = true
//     result.ExitCode = -1
// }
```

**问题:** 移除了对 `context.DeadlineExceeded` 的检测。当 context 超时时，不会设置 `TimedOut = true` 和 `ExitCode = -1`。

**函数 2:** `RunWithStdinTimeout()`  
**位置:** 第 119-170 行

```go
// 与 RunWithTimeout() 相同的问题
// 缺失 context deadline 检测
```

---

## 四、缺陷触发链路

```
用户请求: POST /executions
  {
    "language": "python",
    "code": "import time; time.sleep(30)",
    "timeout": 10
  }
           │
           ▼
  ┌─────────────────────────────────────────────┐
  │ 1. Handler.CreateExecution(ctx, req)         │
  │    调用 execSvc.Execute(ctx, req)           │
  └─────────────────────────────────────────────┘
           │
           ▼
  ┌─────────────────────────────────────────────┐
  │ 2. Execute()                                │
  │    cfg = cfgMgr.GetConfig()                 │
  │    → cfg.MaxTimeout = 0 (缺陷①: 配置污染)   │
  │    exec.Timeout = min(10, 0) = 0 (缺陷②)   │
  │    go runExecution(exec)                    │
  └─────────────────────────────────────────────┘
           │
           ▼
  ┌─────────────────────────────────────────────┐
  │ 3. runExecution()                           │
  │    ctx = context.Background() (缺陷③)      │
  │    executor.ExecutePython(ctx, code, opts)  │
  │    opts.Timeout = 0 * time.Second          │
  └─────────────────────────────────────────────┘
           │
           ▼
  ┌─────────────────────────────────────────────┐
  │ 4. Executor.ExecutePython()                 │
  │    runner.RunWithTimeout(ctx, 0, ...)       │
  └─────────────────────────────────────────────┘
           │
           ▼
  ┌─────────────────────────────────────────────┐
  │ 5. RunWithTimeout()                        │
  │    execCtx = context.WithTimeout(ctx, 0)   │
  │    → context 立即超时                        │
  │    → 进程被立即杀掉                          │
  │    → 未设置 TimedOut/ExitCode (缺陷④)      │
  └─────────────────────────────────────────────┘
           │
           ▼
  ┌─────────────────────────────────────────────┐
  │ 6. runExecution() 接收结果                  │
  │    err != nil                               │
  │    → StatusCompleted (缺陷⑤: 状态污染)     │
  │    → ErrorMessage: "signal: killed"         │
  │    → ExitCode: 0 (原始退出码被丢弃)         │
  └─────────────────────────────────────────────┘
           │
           ▼
  ┌─────────────────────────────────────────────┐
  │ 7. 存储到 MemoryStore                       │
  │    status = "completed"  ← 错误！           │
  └─────────────────────────────────────────────┘
           │
           ▼
  API 响应:
  {
    "id": "...",
    "status": "completed",  ← 错误！应为 "timed_out"
    "result": {
      "stdout": "",
      "stderr": "signal: killed",
      "exit_code": 0,      ← 错误！应为 -1
      "duration": 0
    }
  }
```

---

## 五、复现步骤

### 5.1 运行缺陷验证测试

```bash
cd /home/admin/code/24/001/24-001-17
go test . -v -count=1 -run '^TestRedGreen$'
```

### 5.2 预期输出 (缺陷存在时)

```
=== RUN   TestRedGreen
FAIL: Test 1 - Config MaxTimeout corrupted: expected 30, got 0
FAIL: Test 2 - Timeout not clamped: expected 30, got 0
FAIL: Test 3 - Error mapped to wrong status: expected failed, got completed
FAIL: Test 4 - Execution not cancelled by context: expected timed_out/canceled, got completed
RED (红灯，缺陷未修复)
--- FAIL: TestRedGreen (0.15s)
FAIL
FAIL    github.com/codesandbox/codesandbox      0.158s
FAIL
```

### 5.3 测试点说明

| 测试 | 检查内容 | 正常结果 | 缺陷结果 |
|------|---------|---------|---------|
| Test 1 | Config.MaxTimeout 是否被污染 | 30 (原值) | 0 (被置零) |
| Test 2 | 超时值是否正确钳制 | 30 (钳制后) | 0 (钳制到零) |
| Test 3 | 错误进程状态是否正确 | `failed` | `completed` |
| Test 4 | Context 超时是否传播 | `timed_out`/`canceled` | `completed` |

---

## 六、修复方案

### 6.1 修改 1: `internal/config/config.go` — GetConfig()

```go
// 修复后:
func (m *Manager) GetConfig() *model.AppConfig {
    m.mu.RLock()
    defer m.mu.RUnlock()
    cfg := *m.config
    return &cfg  // 移除 MaxTimeout 置零逻辑
}
```

### 6.2 修改 2: `internal/service/execution_service.go` — Execute()

```go
// 修复后 (移除错误的超时钳制):
// 保留基于原始 MaxTimeout 的合理钳制
if cfg.MaxTimeout > 0 && exec.Timeout > cfg.MaxTimeout {
    exec.Timeout = cfg.MaxTimeout
}
```

### 6.3 修改 3: `internal/service/execution_service.go` — runExecution()

```go
// 修复后:
// a) 传递原始 context（需要将 ctx 保存到 Execution 中）
// b) 正确映射状态:
//    - err != nil      → StatusFailed
//    - result.TimedOut → StatusTimedOut
//    - result.Killed   → StatusKilled
//    - ExitCode != 0   → StatusFailed
//    - 正常完成        → StatusCompleted
```

### 6.4 修改 4: `pkg/process/manager.go` — RunWithTimeout()

```go
// 修复后 (恢复 context deadline 检测):
if err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
        result.ExitCode = exitErr.ExitCode()
    } else {
        result.Stderr = err.Error()
        if execCtx.Err() == context.DeadlineExceeded {
            result.TimedOut = true
            result.ExitCode = -1
        }
    }
}
```

### 6.5 修改 5: `pkg/process/manager.go` — RunWithStdinTimeout()

```go
// 同 RunWithTimeout()，恢复 context deadline 检测
```

---

## 七、修复验证

修复完成后，执行以下命令验证：

```bash
cd /home/admin/code/24/001/24-001-17
go test . -v -count=1 -run '^TestRedGreen$'
```

预期输出：

```
=== RUN   TestRedGreen
PASS: Test 1 - Config MaxTimeout correctly preserved
PASS: Test 2 - Timeout correctly clamped to MaxTimeout
PASS: Test 3 - Error correctly mapped to StatusFailed
PASS: Test 4 - Context timeout propagated correctly
GREEN (绿灯，缺陷已修复)
--- PASS: TestRedGreen (0.15s)
PASS
ok      github.com/codesandbox/codesandbox      0.158s
```

---

## 八、门禁验证

| 门禁 | 命令 | 状态 |
|------|------|------|
| Gate A | `go build ./...` | ✅ PASSED |
| Gate B | `go vet ./...` | ✅ PASSED |
| Gate C | `go test -c -o /dev/null .` | ✅ PASSED |

---

## 九、跨分支验证

| 检查项 | 状态 |
|--------|------|
| 临时分支编译 (`go test -c`) | ✅ PASSED |
| 临时分支测试 (RED 结果一致) | ✅ PASSED |
| Hook 符号一致性 | ✅ PASSED (两侧均为 0) |

---

*BUG_REPRO.md — 缺陷复现报告*