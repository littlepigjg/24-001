# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
模板搜索功能在使用特定长度和组成的搜索关键字时，出现严重的性能退化甚至请求挂起。当搜索字符串由较多的不同字符组成（如 20 个不同字母）时，搜索请求耗时从毫秒级飙升至数秒以上，最终导致请求超时。短关键词或重复字符的关键词搜索表现正常。

## 2. 环境信息（Environment）
- 操作系统：Linux (x86_64)
- Go 版本：go1.26.5 linux/amd64
- 项目模块：github.com/codesandbox/codesandbox
- 关键依赖：无外部依赖
- 运行参数：go test . -count=1 -run '^TestRedGreen$'
- 硬件信息：CPU 核数（未指定）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 启动测试：执行 `go test . -count=1 -run '^TestRedGreen$' -v`
3. 观察测试输出，注意 "malicious_search_detected" 子测试的结果
4. 当搜索关键字为 "abcdefghijklmnopqrst"（20 个不同字符）时，搜索操作耗时超过 3 秒触发超时
5. 测试报告 FAIL，打印 "RED (红灯，缺陷未修复)"

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出：
  ```
  === RUN   TestRedGreen/normal_search_works
  === RUN   TestRedGreen/malicious_search_detected
  RED (红灯，缺陷未修复)
      red_green_test.go:71: Search with malicious input took too long (timeout 3s) - defect triggered
  === RUN   TestRedGreen/search_with_special_chars
  --- FAIL: TestRedGreen (3.01s)
      --- PASS: TestRedGreen/normal_search_works (0.00s)
      --- FAIL: TestRedGreen/malicious_search_detected (3.00s)
      --- PASS: TestRedGreen/search_with_special_chars (0.00s)
  FAIL
  ```
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- 其他异常现象：搜索请求挂起，CPU 使用率短暂飙升后超时返回
- go test 退出码：1

## 5. 期望结果（Expected Behavior）
- 所有搜索请求（包括 20+ 字符的多样化关键字）均在 3 秒内正常返回
- RED/GREEN 判定结果：GREEN（绿灯，缺陷已修复）
- 测试全部 PASS，退出码为 0
- 短关键词搜索功能不受影响，仍能正确返回匹配结果
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
- 必现（100%）：当搜索关键字为 20 个及以上不同字符组成的字符串时，必然触发超时
- 字符数量越多、多样性越高，性能退化越严重
- 少于 12 个字符的搜索关键词通常不会触发

## 7. 影响范围（Impact / Scope）
- 搜索接口存在 DoS 风险：恶意用户可以通过提交长字符串搜索关键字使服务挂起
- 所有依赖搜索功能的 API 端点受影响（模板搜索、列表过滤、历史搜索）
- 由于搜索在多个字段上重复执行，开销被指数级放大
- 可能导致服务整体响应能力下降，影响所有用户的正常请求
- 在线上环境可能造成 goroutine 泄漏、连接堆积等连锁问题

## 8. 附加说明（Additional Notes / Workaround）
- 临时规避方法：在 API 网关或前端限制搜索关键字长度（如不超过 10 个字符）
- 可对搜索请求增加超时机制，避免长时间挂起
- 建议后续增加搜索输入的长度和字符验证
