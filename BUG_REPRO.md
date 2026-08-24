# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
短链接服务在高并发场景下出现请求卡死不响应的问题。单线程调用创建短码和跳转接口均正常，但当多个请求并发执行时，服务会出现整体卡死现象，客户端请求在超时时间内无法得到任何响应，最终超时断开。

## 2. 环境信息（Environment）
- 操作系统：Linux (x86_64)
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 关键依赖：无外部依赖
- 运行参数：go test . -count=1 -run '^TestRedGreen$'
- 硬件信息：4 核 CPU

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 go build ./... 确保编译通过
2. 执行 go test -race . -count=1 -run '^TestRedGreen$' 运行验证测试
3. 观察测试输出结果

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出：
```
=== RUN   TestRedGreen
RED（红灯，缺陷未修复）
    red_green_test.go:81: Create timed out - possible deadlock detected
--- FAIL: TestRedGreen (0.50s)
FAIL
exit status 1
```
- 并发创建短码请求在 500ms 内无返回，触发超时保护
- RED/GREEN 判定结果：RED（红灯）
- 服务进程未 panic，也未 crash，表现为请求挂起等待
- 无数据竞争报告（死锁不产生 race detector 警告）

## 5. 期望结果（Expected Behavior）
- 所有创建和跳转请求均在 500ms 内正常返回
- 短码创建返回正确的 ShortURL 对象，包含正确的 Code 和 RawURL
- 跳转请求返回正确的 RedirectResult，包含正确的 RawURL 和 302 状态码
- RED/GREEN 判定结果应为 GREEN
- go test -race -count=20 全部通过且无数据竞争
- go build ./... 与 go vet ./... 全部通过

## 6. 触发频率（Frequency）
- 单线程调用 Create：约 100% 触发（直接死锁）
- 并发调用 Create + HandleRedirect：约 100% 触发
- 纯 Save 操作：不触发（Save 内部正确释放锁后再调用清理）
- 纯 Get 操作：约 100% 触发（Get 同样持有读锁调用清理）
- 纯 RawSnapshot 操作：约 100% 触发（RawSnapshot 持有读锁调用清理）

## 7. 影响范围（Impact / Scope）
- 服务所有涉及读取短码的接口不可用（创建、查询、跳转）
- 高并发下整个服务对外完全失去响应
- 已创建的短码数据无法通过查询接口获取
- 虽然服务进程仍在运行，但实际等同于不可用状态
- 无数据损坏风险（死锁不会破坏已有数据）

## 8. 附加说明（Additional Notes / Workaround）
- 无有效临时 workaround
- 重启服务可暂时恢复但高并发下会立即复现
- 问题与存储的并发访问路径直接相关