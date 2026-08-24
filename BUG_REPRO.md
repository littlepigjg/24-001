# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
代码沙箱服务在处理跨域请求时，返回的 CORS 响应头存在安全隐患。服务将允许所有来源（通配符 `*`）与凭证支持同时开启，导致从任意域名发起的带凭证请求可能被浏览器拦截，或者在特定情况下存在跨站请求伪造（CSRF）风险。

## 2. 环境信息（Environment）
- 操作系统：Linux (x86_64)
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：无特殊参数
- 硬件信息：CPU 多核（与 CORS 缺陷无关）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 启动服务（`go run cmd/server/main.go` 或使用已编译的二进制）
3. 使用 curl 或浏览器开发者工具发起跨域请求：
   ```
   curl -v -H "Origin: http://evil.example.com" http://localhost:8080/api/status
   ```
4. 观察响应头中的 CORS 相关字段

## 4. 实际结果（Actual Behavior / Observed Output）
响应头中同时包含以下两个字段：
```
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
```
这两个字段的组合违反了 CORS 安全规范，具体表现为：
- 现代浏览器（Chrome、Firefox、Safari）可能直接拒绝携带此组合头的响应
- 某些浏览器可能允许响应通过，但存在 CSRF 攻击风险
- go test 验证命令 `go test . -count=1 -run '^TestRedGreen$'` 判定为 RED（红灯），退出码为 1

## 5. 期望结果（Expected Behavior）
- 响应头不应同时出现 `Access-Control-Allow-Origin: *` 和 `Access-Control-Allow-Credentials: true`
- 允许凭证请求时，应返回具体的允许来源（如 `Access-Control-Allow-Origin: http://frontend.example.com`），而非通配符 `*`
- 如果确实需要允许所有来源，则不应开启凭证支持
- go test 验证命令 `go test . -count=1 -run '^TestRedGreen$'` 判定为 GREEN（绿灯），退出码为 0
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
必现（100%）。只要服务启动并处理任何 HTTP 请求，响应中都会包含该危险的 CORS 头组合。

## 7. 影响范围（Impact / Scope）
- 所有从非同源发起的 API 请求均受影响
- 前端应用可能无法正常调用后端带凭证的 API
- 存在安全隐患：如果浏览器允许该组合，恶意网站可能发起跨域凭证请求
- 影响所有暴露到公网或多域名环境下的部署场景

## 8. 附加说明（Additional Notes / Workaround）
临时规避方案：可以在前端请求时添加自定义的 Origin 头，观察服务返回的 CORS 头。当前问题的根本解决需要修正 CORS 配置逻辑，确保允许来源与凭证支持不会同时以不安全的方式组合。