# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

短链接服务在高并发场景下出现严重不稳定行为。当多个请求同时创建短链接时，服务可能直接崩溃（panic），或者部分短链接创建请求静默失败，导致数据丢失。使用 `-race` 参数运行测试时可以检测到数据竞争（DATA RACE）警告。单请求或低并发场景下功能正常，问题仅在并发量较高时出现。

## 2. 环境信息（Environment）

- 操作系统：Linux
- Go 版本：go 1.22+
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：需带 `-race` 标志和 `-count=N` 重复执行
- 硬件信息：多核心 CPU（并发缺陷在多核环境下更易触发）

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行以下测试命令触发并发场景：
   ```
   go test . -race -count=20 -run '^TestRedGreen$' -timeout 60s
   ```
3. 观察测试输出和退出码

## 4. 实际结果（Actual Behavior / Observed Output）

- 具体错误信息：
  ```
  WARNING: DATA RACE
  Write at 0x... by goroutine XX:
    runtime.mapassign_faststr()
    github.com/codesandbox/codesandbox/...Save()
  panic: runtime error: index out of range [...] with length [...]
    encoding/json.mapEncoder.encode()
    github.com/codesandbox/codesandbox/...persistLocked()
  ```
- 判定结果：RED（红灯，缺陷未修复）
- 其他异常现象：
  - 短链接快照数据条目数少于预期值（数据丢失）
  - 部分 goroutine 返回错误，提示短链接创建失败
  - 程序崩溃后中断所有并发操作
- go test -race 报告多个 DATA RACE 警告，涉及 map 并发读写

## 5. 期望结果（Expected Behavior）

- 无 panic、无数据竞争（go test -race 无警告）
- 判定结果应为 GREEN（绿灯，缺陷已修复）
- 150 个短链接全部成功创建并持久化，快照条目数精确等于 150
- 每个短链接均能正确查询和重定向
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）

必现（100%）：在 30 个 goroutine 并发、每个 goroutine 5 次操作的场景下，带 `-race` 和 `-count=20` 运行可稳定复现。不带 `-race` 时也可触发 panic 或数据丢失，但概率略有下降。

## 7. 影响范围（Impact / Scope）

- 服务在高并发下崩溃，导致所有在处理中的请求失败
- 数据不一致：部分短链接记录丢失，无法通过 code 查询到
- 数据损坏：JSON 持久化文件可能包含不完整或被截断的数据
- 线上可用性下降：并发量上来时服务不稳定
- 后续的重定向操作可能因短链接记录缺失而返回 404 或跳转到错误页面

## 8. 附加说明（Additional Notes / Workaround）

目前无可靠的临时规避方案。降低并发量（如减少到 5 个 goroutine）可以降低触发概率，但不能完全避免数据竞争。建议在修复前避免将该服务部署到高并发生产环境。