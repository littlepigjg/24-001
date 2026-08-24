# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

代码沙箱服务在接收客户端提交的执行请求时，如果请求体中包含额外的、未在 API 文档中定义的字段，服务会直接返回 400 错误拒绝请求。此外，当客户端主动取消请求后，后台正在执行的代码并不会随之停止，导致已取消的任务仍占用系统资源继续运行，执行状态与实际情况不一致。

## 2. 环境信息（Environment）

- 操作系统：Linux (x86_64)
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：go test . -count=1 -run '^TestRedGreen$'
- 硬件信息：CPU 多核

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 go build ./... 确保编译通过
2. 执行 go test . -count=1 -run '^TestRedGreen$' 运行验证测试
3. 观察测试结果，应显示 RED（红灯，缺陷未修复），有 4 项测试失败
4. 或者通过 HTTP 接口 POST /api/execute 提交包含额外字段的请求：
   ```json
   {"language":"python","code":"print('hello')","extra_field":"value"}
   ```
5. 观察服务器返回 400 错误，消息类似 "json decode failed: json: unknown field 'extra_field'"

## 4. 实际结果（Actual Behavior / Observed Output）

- go test 测试输出：
  ```
  === RUN   TestRedGreen/DecodeJSON_allows_extra_fields
  FAIL: DecodeJSON should allow unknown fields, got error: json decode failed: json: unknown field "extra_field"
  === RUN   TestRedGreen/DecodeJSONWithContext_allows_extra_fields
  FAIL: DecodeJSONWithContext should allow unknown fields, got error: json decode failed: json: unknown field "unknown_param"
  === RUN   TestRedGreen/DecodeExecutionRequest_handles_extra_fields
  FAIL: DecodeExecutionRequest should allow unknown fields, got error: failed to decode execution request: json: unknown field "meta"
  === RUN   TestRedGreen/DecodeExecutionRequestWithContext_handles_extra_fields
  FAIL: DecodeExecutionRequestWithContext should allow unknown fields, got error: failed to decode execution request: json: unknown field "comment"
  RED (红灯，缺陷未修复)
  共 4 项测试失败
  ```
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- HTTP 接口返回 400 Bad Request，错误信息包含 "unknown field"
- 请求取消后，后台执行仍继续，执行记录状态可能停留在 running 或产生不一致状态

## 5. 期望结果（Expected Behavior）

- go test . -count=1 -run '^TestRedGreen$' 所有测试通过，输出 GREEN（绿灯，缺陷已修复）
- HTTP 接口接收含额外字段的请求正常返回 200 成功，额外字段被忽略
- go build ./... 编译通过
- go vet ./... 无静态检查报错
- 客户端取消请求后，后台执行正确停止，执行状态正确更新为已取消
- 所有 JSON 解析函数正确忽略未知字段，不再因额外字段拒绝合法请求

## 6. 触发频率（Frequency）

必现（100%）：任何包含额外字段的 JSON 请求都会触发，无论是通过 HTTP 接口还是直接调用内部解析函数。

## 7. 影响范围（Impact / Scope）

- 所有提交代码执行的请求如果携带额外字段（如请求追踪 ID、客户端元数据、版本号等）均会被拒绝
- 客户端取消请求后，后台执行不响应取消信号，造成系统资源浪费
- 批量执行接口同样受影响
- 前端为了兼容不得不移除所有额外字段，限制了扩展能力
- 在高并发场景下，取消不及时可能导致大量僵尸执行任务堆积

## 8. 附加说明（Additional Notes / Workaround）

目前的临时规避方案是：客户端在发送请求前移除所有未定义的额外字段，但这只是权宜之计，无法满足后续功能扩展需求。建议尽快修复。