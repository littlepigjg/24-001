# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
在并发场景下提交代码执行请求后，数据库中的执行记录出现数据不一致：部分提交的记录丢失（提交 5 条仅能查到 0~2 条），部分记录状态卡在 `pending` 无法更新为最终状态（completed/failed）。高峰期高并发下问题稳定复现。

## 2. 环境信息（Environment）
- 操作系统：Linux (x86_64)
- Go 版本：go1.21+
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：`go test -race . -count=3 -run '^TestRedGreen$|^TestConcurrentSubmit$' -v`
- 并发数量：5 goroutines 同时提交执行请求

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录 `/home/admin/code/24/001/24-001-25`，执行 `go build ./...` 确保编译通过
2. 执行 `go test -race . -count=3 -run '^TestRedGreen$|^TestConcurrentSubmit$' -v` 运行验证测试
3. 观察测试输出结果和退出码

## 4. 实际结果（Actual Behavior / Observed Output）
- **TestRedGreen**（3/3 次）：输出 `RED（红灯，缺陷未修复）—— 检测到僵尸执行记录（status=pending）`，退出码 1
- **TestConcurrentSubmit**（3/3 次）：输出 `RED（红灯，缺陷未修复）—— 并发场景下有 3~5 条记录丢失（提交 5 条，仅找到 0~2 条）`，退出码 1
- 日志中出现错误信息：`Failed to update execution status: execution with ID xxx not found`
- 部分执行记录写入数据库后无法被后续更新操作找到（记录丢失或状态不一致）
- **无 DATA RACE 警告**（并发访问已加锁保护，不再有数据竞争）

## 5. 期望结果（Expected Behavior）
- **TestRedGreen**：3 次均输出 `GREEN（绿灯，缺陷已修复）—— 无僵尸执行记录`，退出码 0
- **TestConcurrentSubmit**：3 次均输出 `GREEN（绿灯，缺陷已修复）—— 并发提交稳定，5 条记录全部正确保存并更新`，退出码 0
- 无任何错误日志输出
- go test -race 无 DATA RACE 警告
- go build ./... 编译通过，go vet ./... 无报错
- 并发提交的所有执行记录均被正确保存并更新，无数据丢失、无僵尸记录

## 6. 触发频率（Frequency）
必现（100%）：在并发场景下，每次运行验证测试都能稳定检测到僵尸记录和数据丢失。单次运行 TestRedGreen 即可 100% 触发僵尸记录问题；并发提交测试 TestConcurrentSubmit 在 5 个 goroutine 并发下 100% 出现记录丢失。

## 7. 影响范围（Impact / Scope）
- 并发提交的执行请求可能丢失，用户提交的代码无法被执行
- 执行记录状态卡在 pending，前端用户看到任务一直"执行中"
- 僵尸记录占用存储空间，长期积累导致数据库膨胀
- 高峰期高并发下服务不可用率上升，影响线上稳定性
- 错误日志大量输出 `not found` 错误，掩盖其他真正的故障

## 8. 附加说明（Additional Notes / Workaround）
临时规避方法：串行提交执行请求（每次只提交 1 条，等待完成后再提交下一条），可以降低触发概率但无法根治。建议在修复前避免高并发批量提交操作。