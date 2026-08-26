# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
短链跳转服务存在一个上下文未正确传递的问题：当客户端在发起跳转请求后提前断开连接或请求超时时，服务端仍然继续执行访问计数和日志记录操作，导致访问量虚增、日志残留无效条目。按正常逻辑，已取消的请求应该完全跳过后续的记录操作。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：go test . -count=1（默认即复现，无需 -race 或并发）
- 硬件信息：无特殊要求

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录：`cd /path/to/project`
2. 执行 `go build ./...` 确保编译通过
3. 执行 `go test -v -run '^TestRedGreen$' . -count=1` 运行验证测试
4. 观察测试输出与退出码

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出：`RED (红灯，缺陷未修复) — visitsAfter=1 bookkeepingAfter=2 expected=0/1`
- go test 退出码为 1
- 具体表现：创建短链后用已取消的 context 调用跳转接口，访问计数仍被递增到 1，bookkeepingCount 从 1 增至 2，pendingWrites 队列中多了一条不该有的记录
- 这说明即使客户端已取消请求，服务端仍在执行 bookkeeping 操作
- go test -race 报告：可能无 DATA RACE（本缺陷为逻辑错误，非数据竞争）

## 5. 期望结果（Expected Behavior）
- 使用已取消的 context 调用跳转接口后：
  - 访问计数应保持为 0（不应被递增）
  - bookkeepingCount 应保持为 1（仅创建时的那一次）
  - pendingWrites 不应新增条目
- go test 退出码应为 0
- 测试输出：`GREEN (绿灯，缺陷已修复)`
- `go build ./...` 与 `go vet ./...` 全部通过
- `go test -race -count=20 .` 无数据竞争报告

## 6. 触发频率（Frequency）
- 必现（100%）：只要 context 在调用跳转接口前被取消，必然触发
- 无需 -race 或并发即可稳定复现

## 7. 影响范围（Impact / Scope）
- 访问量统计失真：已取消的请求被计入有效访问
- 日志污染：无效请求产生的日志条目影响后续审计和分析
- 资源浪费：不必要的 map 操作和 slice 追加在高并发下累积
- 状态不一致：pendingWrites 在后续 Close 时会被错误应用，进一步放大数据偏差
- 对于依赖访问量统计的业务场景（如付费短链、流量分析），可能造成直接经济损失

## 8. 附加说明（Additional Notes / Workaround）
当前无可用的临时 workaround。一旦请求被取消，bookkeeping 仍会执行，没有开关可以关闭此行为。
