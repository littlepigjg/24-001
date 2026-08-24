# Bug Reproduction Guide: Nil Pointer Dereference via Cross-Layer Contract Breach

## Bug ID
2-001-001

## Bug Category
nil pointer dereference (空指针解引用)

## Affected Components
- **Store Layer**: `internal/store/memory_store.go` — `GetExecution`, `GetHistory`, `GetTemplate`
- **Service Layer**: `internal/service/execution_service.go` — `Cancel`, `GetResult`, `runExecution`
- **Service Layer**: `internal/service/template_service.go` — `Get`, `Update`, `Delete`

## Symptom Description
When querying or canceling a non-existent execution ID, the service panics with a nil pointer dereference. The panic occurs because:

1. The store layer's `GetExecution` returns `(nil, nil)` instead of `(nil, error)` when a `panicGuard` hook is set and returns `true` for the queried ID
2. The service layer ignores the error return value (using `_` discard) and directly accesses fields on the nil pointer
3. This causes a nil pointer dereference panic at `exec.Status` access in `Cancel`, or silently returns `(nil, nil)` in `GetResult`

## Root Cause Analysis

### Breaking the Error Propagation Contract

The original contract between the store layer and service layer is:
> **When a resource is not found, `Get*` methods MUST return `(nil, error)` — never `(nil, nil)`.**

The defect breaks this contract in `memory_store.go`:

```go
// internal/store/memory_store.go, lines 54-65
func (s *MemoryStore) GetExecution(id string) (*model.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exec, exists := s.executions[id]
	if !exists {
		if s.panicGuard != nil && s.panicGuard(id) {
			return nil, nil  // ← CONTRACT VIOLATION: returns nil, nil instead of nil, error
		}
		return nil, fmt.Errorf("execution with ID %s not found", id)
	}
	return exec, nil
}
```

### Service Layer Ignores the Error

The service layer compounds the issue by ignoring errors:

```go
// internal/service/execution_service.go, lines 244-252
func (s *ExecutionService) Cancel(id string) error {
	exec, _ := s.store.GetExecution(id)  // ← error discarded with _
	if exec.Status != model.StatusRunning && exec.Status != model.StatusPending {
		// ↑ exec is nil when GetExecution returns nil, nil → PANIC
		return fmt.Errorf("execution is not in a cancellable state: %s", exec.Status)
	}
	exec.UpdateStatus(model.StatusCanceled)
	return s.store.UpdateExecution(exec)
}
```

### Chain of Failures

```
Caller requests Cancel("non-existent-id")
  → execution_service.Cancel("non-existent-id")
    → memory_store.GetExecution("non-existent-id")
      → executions map lookup → key not found
      → panicGuard("non-existent-id") returns true
      → returns (nil, nil)  ← BUG: breaks error contract
    → exec, _ := (nil, nil)  ← error discarded
    → exec.Status  ← PANIC: nil pointer dereference
```

## How to Reproduce

### Prerequisites
- Go 1.21+ installed
- Project dependencies installed (`go mod tidy`)

### Reproduction Steps

1. **Set the panic guard** on the store to activate the defect:
   ```go
   store.SetPanicGuard(func(id string) bool {
       return true  // activates guard for all IDs
   })
   ```

2. **Call Cancel with a non-existent ID**:
   ```go
   err := executionService.Cancel("non-existent-id")
   // This will panic with nil pointer dereference
   ```

3. **Call GetResult with a non-existent ID**:
   ```go
   exec, err := executionService.GetResult("non-existent-id")
   // Returns (nil, nil) — silently wrong, no error reported
   ```

### Automated Reproduction
The test file `red_green_test.go` provides automated reproduction:

```bash
go test -v -count=1 -run '^TestRedGreen$' .
```

Expected output (with defect present):
```
=== RUN   TestRedGreen
=== RUN   TestRedGreen/Cancel_non-existent_execution
RED (红灯，缺陷未修复)
=== RUN   TestRedGreen/GetResult_non-existent_execution
RED (红灯，缺陷未修复)
=== RUN   TestRedGreen/Cancel_existing_execution_still_works
GREEN (绿灯，缺陷已修复)
--- FAIL: TestRedGreen (0.00s)
```

## Bug Locations (Key Code Snippets)

### 1. Defect Trigger: `memory_store.go` — Guard-based nil return

**File**: `internal/store/memory_store.go`
**Function**: `GetExecution` (line 54), `GetHistory` (line 113), `GetTemplate` (line 260)
**Lines**: 59-61 (in GetExecution), 118-120 (in GetHistory), 265-267 (in GetTemplate)

```go
if s.panicGuard != nil && s.panicGuard(id) {
    return nil, nil  // ← Should return nil, fmt.Errorf(...)
}
```

### 2. Defect Amplifier: `execution_service.go` — Error discarded

**File**: `internal/service/execution_service.go`
**Function**: `Cancel` (line 244), `GetResult` (line 238)
**Lines**: 239, 245

```go
exec, _ := s.store.GetExecution(id)  // ← Should check error
```

### 3. Defect Amplifier: `template_service.go` — Error discarded

**File**: `internal/service/template_service.go`
**Function**: `Get` (line 61), `Update` (line 67), `Delete` (line 88)
**Lines**: 62, 77, 89

```go
tmpl, _ := s.store.GetTemplate(id)  // ← Should check error
```

## Non-Defect Hooks (Do NOT Modify)
These are diagnostic/fault-injection APIs that must survive any fix:

| Symbol | File | Type |
|--------|------|------|
| `PanicGuardFn` | `memory_store.go` | Type definition |
| `SetPanicGuard` | `memory_store.go` | Method on `MemoryStore` |
| `GetWithGuard` | `memory_store.go` | Method on `MemoryStore` |
| `SaveWithGuard` | `memory_store.go` | Method on `MemoryStore` |
| `RawSnapshot` | `memory_store.go` | Method on `MemoryStore` |

## Fix Verification
After applying the fix:
```bash
go test -v -count=1 -run '^TestRedGreen$' .
```
Expected output (after fix):
```
=== RUN   TestRedGreen
=== RUN   TestRedGreen/Cancel_non-existent_execution
GREEN (绿灯，缺陷已修复)
=== RUN   TestRedGreen/GetResult_non-existent_execution
GREEN (绿灯，缺陷已修复)
=== RUN   TestRedGreen/Cancel_existing_execution_still_works
GREEN (绿灯，缺陷已修复)
--- PASS: TestRedGreen (0.00s)
    --- PASS: TestRedGreen/Cancel_non-existent_execution (0.00s)
    --- PASS: TestRedGreen/GetResult_non-existent_execution (0.00s)
    --- PASS: TestRedGreen/Cancel_existing_execution_still_works (0.00s)
PASS
```
