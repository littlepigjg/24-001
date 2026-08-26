# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
短链接服务在处理操作失败时，返回的错误消息过于笼统，丢失了底层详细错误信息。当创建重复短码、查询不存在的短码等场景发生时，调用方只能收到通用错误提示（如"failed to create short URL"、"redirect failed"），无法获知具体失败原因（如短码已存在、短码未找到等），给问题排查和前端错误提示带来困难。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：无特殊要求
- 硬件信息：与缺陷无关

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行单元测试：`go test . -count=1 -run '^TestRedGreen$' -v`
3. 观察测试输出，检查以下测试用例的错误消息内容

手动复现步骤：
1. 初始化配置：调用 config.Default() 获取默认配置
2. 创建存储层：调用 store.NewURLStore(cfg) 创建 URL 存储
3. 创建服务层：调用 service.NewURLService(cfg, store) 创建短链接服务
4. 调用 service.NewRedirectService(store, logStore) 创建重定向服务
5. 尝试创建一个自定义短码的短链接：Create(req{CustomCode: "testcode", RawURL: "https://example.com"})
6. 再次尝试创建相同短码的短链接：Create(req{CustomCode: "testcode", RawURL: "https://example.com/other"})
7. 观察返回的错误消息

## 4. 实际结果（Actual Behavior / Observed Output）
- 自动化测试结果：3/6 测试失败，输出 RED（红灯，缺陷未修复）
- 具体失败：
  - 创建重复短码：错误消息为 "failed to create short URL"，缺少 "code already exists" 具体信息
  - 访问不存在短码（通过重定向）：错误消息为 "redirect failed"，缺少 "not found" 具体信息
  - 访问不存在短码（通过查询）：错误消息为 "failed to get short URL"，缺少 "not found" 具体信息
- 退出码：1（测试失败）
- 无 panic、无数据竞争（缺陷不涉及并发）

## 5. 期望结果（Expected Behavior）
- 创建重复短码时：错误消息应包含 "code already exists"，调用方可精确识别重复原因
- 访问不存在短码时：错误消息应包含 "short URL not found"，调用方可精确识别未找到原因
- 所有错误链完整保留，支持 errors.Is / errors.As 匹配
- go test . -count=1 -run '^TestRedGreen$' 全部通过，输出 GREEN（绿灯，缺陷已修复）
- go build ./... 与 go vet ./... 全部通过

## 6. 触发频率（Frequency）
必现（100%）。任何涉及操作失败的场景（创建重复、查询不存在、禁用后重定向、过期后重定向等）均稳定触发错误信息丢失。

## 7. 影响范围（Impact / Scope）
- 前端无法展示精确的错误提示，用户体验差
- 运维排查问题时无法直接从错误日志定位根因，需要深入代码分析
- 上游调用方无法基于具体错误类型做差异化处理（如区分"短码不存在"和"短码已禁用"）
- 错误统计和监控系统无法按具体错误类型聚合，影响数据分析准确性

## 8. 附加说明（Additional Notes / Workaround）
目前无有效 workaround。临时缓解方式为在调用方增加额外的前置检查逻辑（如先 Get 再 Create），但无法完全规避所有场景的错误信息丢失问题。