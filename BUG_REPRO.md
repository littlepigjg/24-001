# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
代码执行沙箱服务的历史记录搜索功能在高并发或大数据量场景下表现异常：搜索结果不准确（漏报或误报）、大数据量下性能低下、并发调用时结果不一致。使用 `-race` 标志运行测试时可检测到数据竞争。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 存储类型：内存存储（MemoryStore）
- 运行参数：go test . -race -count=1 -run '^TestRedGreen$' -v
- CPU 核数：多核（并发场景测试）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 在项目中创建约 2000 条历史记录，每条记录的 Code 和 Stdout 字段中包含预定义关键词（如 ALPHA、BETA、GAMMA 等）
3. 启动 16 个 goroutine 并发调用 ListHistory 方法，每个 goroutine 执行 10 轮搜索，搜索关键词为上述预定义关键词
4. 同时验证边界匹配场景：创建一条记录，其 Stdout 字段值为 "XALPHA"，搜索关键词 "ALPHA"
5. 创建 3500 条记录后搜索一个不存在的关键词（如 "NONEXISTENT_KEYWORD"），测量搜索耗时
6. 带 `-race` 标志运行检测数据竞争

## 4. 实际结果（Actual Behavior / Observed Output）
- 并发搜索返回错误结果：部分返回的记录中不包含搜索关键词（如 record hist-305 不含 "DELTA"，record hist-1230 不含 "GAMMA"）
- 边界匹配失败：搜索 "ALPHA" 在字段 "XALPHA" 中返回 0 条结果（正确应返回 1 条）
- 性能退化：搜索不存在关键词在 3500 条记录上耗时 6.71 秒（应在 1 秒内完成）
- go test -race 报告：WARNING: DATA RACE — toLower() 函数中对 searchBuf 的并发读写冲突
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- 总测试耗时约 7 秒

## 5. 期望结果（Expected Behavior）
- 并发搜索所有 160 次迭代均返回正确结果：每条返回记录都包含搜索关键词
- 边界匹配正确：搜索 "ALPHA" 在字段 "XALPHA" 中正确返回匹配记录
- 性能正常：3500 条记录搜索不存在关键词耗时低于 1 秒
- go test -race 无 DATA RACE 警告
- RED/GREEN 判定结果：GREEN（绿灯，缺陷已修复）
- go build ./... 与 go vet ./... 全部通过

## 6. 触发频率（Frequency）
- 并发搜索结果错误：必现（100%），只要并发调用 ListHistory 就会触发
- 边界匹配失败：必现（100%），只要搜索关键词位于字符串末尾就会触发
- 性能退化：必现（100%），数据量超过 2000 条后可观测
- 数据竞争：必现（100%），使用 -race 标志运行时稳定复现

## 7. 影响范围（Impact / Scope）
- 历史记录搜索功能返回不可靠结果，影响用户对执行历史的查阅和审计
- 大数据量下搜索接口超时，影响前端页面加载和批量导出功能
- 并发场景下数据竞争可能导致程序在极端情况下 panic 或产生未定义行为
- 边界关键词搜索失败导致用户无法通过搜索定位特定代码执行记录
- 历史记录服务作为代码沙箱的核心功能之一，其可靠性直接影响产品可用性

## 8. 附加说明（Additional Notes / Workaround）
- 临时规避方案：单次、串行调用搜索接口（避免并发），可降低触发概率但无法根治
- 降低数据量可减少性能问题的影响，但无法消除竞态和边界匹配错误
- 建议修复后增加集成测试覆盖并发搜索和边界匹配场景