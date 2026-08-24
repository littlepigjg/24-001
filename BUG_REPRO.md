# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
代码沙箱服务在处理HTTP POST请求时存在请求体资源泄露问题。每个请求的Body在被读取后未被正确关闭，导致HTTP连接无法被连接池复用。在持续的高并发请求下，服务器文件描述符会逐渐耗尽，最终导致所有新请求失败。

## 2. 环境信息（Environment）
- 操作系统：Linux (AMD64)
- Go版本：go1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：无需特殊运行参数，直接执行 go test 即可复现
- 硬件信息：CPU核数不限，问题与硬件无关

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test . -count=1 -run '^TestRedGreen$' -v` 运行验证测试
3. 观察测试输出，所有子测试均报告 "request body was not closed"

## 4. 实际结果（Actual Behavior / Observed Output）
```
=== RUN   TestRedGreen
=== RUN   TestRedGreen/DecodeJSONBody_closes_body_on_success
    RED (红灯，缺陷未修复)
    request body was not closed after DecodeJSONBody success
=== RUN   TestRedGreen/DecodeJSONBody_closes_body_on_decode_error
    RED (红灯，缺陷未修复)
    request body was not closed after DecodeJSONBody decode error
=== RUN   TestRedGreen/ValidateRequestBody_closes_body_on_success
    RED (红灯，缺陷未修复)
    request body was not closed after ValidateRequestBody success
=== RUN   TestRedGreen/ValidateRequestBody_closes_body_on_read_error
    RED (红灯，缺陷未修复)
    request body was not closed after ValidateRequestBody read error
=== RUN   TestRedGreen/handler_Execute_closes_body_on_validation_error
    RED (红灯，缺陷未修复)
    request body was not closed after handler Execute
=== RUN   TestRedGreen/handler_BatchExecute_closes_body
    RED (红灯，缺陷未修复)
    request body was not closed after handler BatchExecute
--- FAIL: TestRedGreen (0.00s)
```
- RED/GREEN判定结果：RED（红灯，缺陷未修复）
- 退出码：1（测试失败）
- 其他异常现象：在实际部署中，长时间运行后HTTP连接数持续增长，文件描述符泄露，最终导致"too many open files"错误

## 5. 期望结果（Expected Behavior）
- 所有子测试均报告 GREEN（绿灯，缺陷已修复）
- 请求体在所有处理路径（成功路径、错误路径、解码失败路径）上均被正确关闭
- go build ./... 编译通过
- go vet ./... 无静态分析警告
- 长时间高并发压测下，HTTP连接数保持稳定，无文件描述符泄露

## 6. 触发频率（Frequency）
必现（100%）：任何POST请求都会触发请求体未关闭问题。在实际HTTP服务器环境中，随着请求量增加，资源泄露逐渐累积，最终导致服务不可用。

## 7. 影响范围（Impact / Scope）
- HTTP连接泄露：每个POST请求泄露一个HTTP连接
- 文件描述符耗尽：持续高并发下，系统文件描述符被耗尽，所有网络请求失败
- 服务可用性下降：从单个请求失败逐渐演变为整个服务不可用
- 无数据损坏风险：此缺陷主要影响资源管理，不涉及业务数据错误

## 8. 附加说明（Additional Notes / Workaround）
无临时规避方法。此问题需要在代码层面修复请求体的生命周期管理，确保在所有处理路径上正确调用Close()方法释放资源。
