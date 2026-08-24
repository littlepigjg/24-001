# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

代码沙箱执行服务在处理用户提交的代码执行请求时存在两个问题：

1. **上下文超时/取消不生效**：当调用方传入带有截止时间（deadline）或可取消（cancel）的 context 时，实际执行的命令不会在 context 到期或取消后终止，而是继续运行直到代码自身的超时时间结束。
2. **并发读取状态不一致**：在执行进行过程中或执行完成后，通过不同接口读取同一执行记录的状态，可能出现不一致的情况。

## 2. 环境信息（Environment）

- 操作系统：Linux
- Go 版本：go version go1.23.x
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：go test -v -race -count=1 -run '^TestRedGreen$' .
- CPU 核数：与并发行为相关，建议至少 2 核

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go vet ./...` 确保静态检查无报错
3. 执行以下测试命令：
   ```bash
   go test -v -race -count=1 -run '^TestRedGreen$' .
   ```
4. 观察测试输出和 race detector 警告

## 4. 实际结果（Actual Behavior / Observed Output）

- **测试1（context_cancellation_propagates）**：输出 "RED (红灯，缺陷未修复) - 等待执行结果超时，执行未被原始上下文取消"，执行没有因 context 取消而终止
- **测试2（deadline_propagation）**：输出 "RED (红灯，缺陷未修复) - 等待执行结果超时，上下文截止时间未传播"，执行没有在 context 截止时间后终止
- **测试3（concurrent_snapshot_consistency）**：输出 "RED (红灯，缺陷未修复) - GetResult 返回的指针在执行完成后状态被并发修改: 原始快照=running, 当前指针=completed"，快照与指针状态不一致
- **go test -race 报告**：多次 DATA RACE 警告，涉及执行状态字段的并发读写
- **退出码**：1 (FAIL)

## 5. 期望结果（Expected Behavior）

- **测试1**：执行在 context 取消后立即终止，状态变为 canceled 或 timed_out，输出 "GREEN (绿灯，缺陷已修复)"
- **测试2**：执行在 context 截止时间（500ms）后终止，状态变为 timed_out 或 canceled，输出 "GREEN (绿灯，缺陷已修复)"
- **测试3**：GetResult 返回的状态与 GetResultWithSnapshot 返回的快照一致，无数据竞争，输出 "GREEN (绿灯，缺陷已修复)"
- **go test -race**：全部通过且无 DATA RACE 报告
- **退出码**：0 (PASS)
- **go build ./... 与 go vet ./...**：全部通过

## 6. 触发频率（Frequency）

- 上下文超时/取消不生效：100% 必现（每次使用 context deadline 或 cancel 时都会触发）
- 并发读取不一致：100% 必现（在执行完成窗口内读取即可稳定触发）
- 数据竞争：使用 -race 检测时 100% 可复现

## 7. 影响范围（Impact / Scope）

- **服务可用性**：context 超时失效会导致执行进程无法按预期终止，长时间占用系统资源
- **资源泄漏**：无法取消的执行会持续占用 goroutine、进程、内存等资源，高并发场景下可能导致资源耗尽
- **数据一致性**：并发读取不一致的状态会导致前端展示错误信息，用户体验混乱
- **线上风险**：如果依赖 context 超时做服务熔断或限流，此缺陷会导致熔断失效，可能引发雪崩

## 8. 附加说明（Additional Notes / Workaround）

临时规避方案：
- 可以通过执行服务的 Cancel 方法主动取消执行（该方法不依赖 context 传播），但需要业务层额外处理
- 对于短执行任务（<1s），context 超时问题不明显，因为执行本身很快就会完成
- 建议在修复前，业务层增加兜底超时逻辑作为保护