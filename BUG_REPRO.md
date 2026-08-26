# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
短链服务在分页处理和批量操作时，当 pageSize 参数为负值（如 -1）时会直接 panic 崩溃，错误信息为 "slice bounds out of range"。问题出现在多个业务场景：创建短链时设置 MaxVisits=-1、分页查询时传入 page_size=-1、以及访问重定向使用负 MaxVisits 创建的短链时。服务没有对这些异常参数做安全降级或返回友好错误，而是直接触发运行时崩溃。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：无特殊参数
- 硬件信息：N/A（纯软件逻辑缺陷，与硬件无关）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 编写测试脚本或使用 Go 测试框架，按以下步骤操作
3. 创建配置对象并将存储的 pageSize 设置为 -1
4. 初始化 URLStore、AccessLogStore、URLService
5. 调用 URLService.List(ctx, 2, -1) 或调用 URLService.Create(ctx, req) 其中 req.MaxVisits=-1
6. 观察程序是否触发 panic

具体代码示例：
```go
cfg := config.Default()
cfg.Storage.SetPageSize(-1)
us, _ := store.NewURLStore(cfg)
urlSvc, _ := service.NewURLService(cfg, us)
// 触发缺陷操作：
urlSvc.List(ctx, 2, -1) // 或 urlSvc.Create(ctx, &model.CreateReq{RawURL: "...", MaxVisits: -1})
```

## 4. 实际结果（Actual Behavior / Observed Output）
- 具体错误信息：`runtime error: slice bounds out of range [:-2]`
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- 其他异常现象：panic 发生在底层切片操作中，导致整个请求处理 goroutine 崩溃。如果没有上层 recover 机制，panic 会传播到 HTTP handler 层导致 500 错误。
- go test -race 结果：N/A（本缺陷不是竞态条件，是确定性的逻辑错误）

## 5. 期望结果（Expected Behavior）
- 无 panic、无数据竞争
- RED/GREEN 判定结果应为 GREEN
- 具体正确行为：
  - 当 pageSize 为负值时，应安全降级为默认值（如 100）或返回参数错误
  - URLService.List(ctx, 2, -1) 应将 pageSize 修正为默认值后正常返回分页数据
  - URLService.Create(ctx, req) 在 MaxVisits=-1 时应正常创建短链，不受底层 pageSize 影响
  - RedirectService.HandleRedirect 应正常处理 MaxVisits=-1 的短链重定向
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
必现（100%）。只要传入负数 pageSize 或通过 MaxVisits=-1 间接传递负数 pageSize，每次都稳定触发 panic。

## 7. 影响范围（Impact / Scope）
- 服务 panic 崩溃：任何涉及分页操作或批量处理的 API 调用都可能触发
- 接口返回错误结果：如果上层有 recover 机制，会得到不正确的空结果或错误信息
- 影响所有短链相关功能：创建、查询、重定向等核心操作都可能受影响
- 数据一致性：panic 可能导致部分写入操作未完成，造成数据不一致

## 8. 附加说明（Additional Notes / Workaround）
临时规避方法：
1. 在调用分页接口前，确保 pageSize 参数为正整数
2. 创建短链时不要设置 MaxVisits=-1，改用 0 或具体正整数
3. 如果必须使用 MaxVisits=-1（表示无限制），可以设置为一个较大的正数（如 999999）作为临时替代
