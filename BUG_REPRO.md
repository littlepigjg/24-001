# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

在代码沙箱服务的并发压测过程中，发现操作计数器出现严重丢失现象。当多个 goroutine 并发提交代码执行时，系统记录的操作次数远低于实际提交次数（丢失率约 68%），`go test -race` 能够稳定检测到 DATA RACE。单协程顺序执行时完全正常，问题仅在并发场景下出现。

## 2. 环境信息（Environment）

- 操作系统：Linux (amd64)
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：使用 `-race` 标志启用数据竞争检测
- 硬件信息：多核 CPU（并发问题与核心数正相关）

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test -race -count=1 -run '^TestRedGreen$' -v` 运行并发测试
3. 观察测试输出，查看计数器的预期值与实际值对比
4. 观察 `go test -race` 是否报告 DATA RACE

## 4. 实际结果（Actual Behavior / Observed Output）

- 测试输出计数器预期值与实际值严重不符，例如：
  ```
  预期 605 次操作, 实际 192 次, 丢失 413 次
  RED (红灯，缺陷未修复): 并发访问期间共有 413 次操作丢失
  ```
- `go test -race` 报告 DATA RACE，涉及 `unsafeCounter` 的并发读写
- 10次连续运行均稳定复现，丢失次数在 410-417 之间波动
- 测试以非零退出码（exit status 1）失败

## 5. 期望结果（Expected Behavior）

- 计数器的实际值与预期值完全一致，无任何丢失
- 测试输出 `GREEN (绿灯，缺陷已修复)`，退出码为 0
- `go test -race -count=20` 全部通过且无 DATA RACE 警告
- 高并发压测下无 panic、无数据竞争
- `go build ./...` 与 `go vet ./...` 全部通过

## 6. 触发频率（Frequency）

- 必现（100%）：10次连续运行均稳定复现，无 flaky 结果
- 丢失率稳定在约 68%（410-417 次丢失 / 605 次预期）
- `go test -race` 每次均可检测到 DATA RACE

## 7. 影响范围（Impact / Scope）

- 服务稳定性：并发场景下计数器严重失真，影响统计和监控数据的准确性
- 数据准确性：操作计数器丢失约 68%，严重影响审计和追踪
- 功能可靠性：并发提交代码执行时，操作记录大量丢失
- 线上可用性：高并发压测下统计数据不可信

## 8. 附加说明（Additional Notes / Workaround）

- 目前无可靠的临时规避方案，降低并发度可缓解但无法根除问题
- 问题仅在并发场景下出现，单协程或低并发下表现正常
- `go test -race` 可稳定检测到此缺陷，是可靠的回归测试手段
- 修复需确保 `unsafeCounter` 的 read-modify-write 操作使用 proper synchronization（如 mutex 或 atomic.AddInt64）