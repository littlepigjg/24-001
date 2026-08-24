# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
短链接服务在处理自定义码（custom code）时存在路径遍历漏洞。当用户提交包含路径分隔符（如 `/`）或目录回溯序列（如 `..`）的自定义码时，系统未能正确拒绝该请求，导致服务可能在文件系统的非预期位置创建或尝试读取文件，存在安全风险。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块/依赖：github.com/codesandbox/codesandbox
- 运行参数：go test . -count=1 -run '^TestRedGreen$'
- 硬件信息：CPU 架构 x86_64

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，确认 go.mod 中的 module 名为 github.com/codesandbox/codesandbox
2. 执行 go build ./... 确保编译通过
3. 执行 go vet ./... 确保无静态检查警告
4. 执行 go test . -count=1 -run '^TestRedGreen$'
5. 观察测试输出结果

## 4. 实际结果（Actual Behavior / Observed Output）
测试输出显示：
- 正常短链接创建成功（code: normal01）
- 包含路径遍历序列的自定义码短链接也创建成功（code: ../canary_target）
- 重定向服务成功处理了包含危险字符的code
- 最终判定为 RED（红灯）
- 具体错误输出：
  ```
  === RUN   TestRedGreen
  [INFO] Loaded 0 URLs from storage
  [INFO] AccessLogStore opened with log path
  [INFO] Short URL created: normal01 -> https://example.com/normal
  [INFO] Short URL created: ../canary_target -> https://example.com/dangerous
  [INFO] Redirect: ../canary_target -> https://example.com/dangerous
  RED (红灯，缺陷未修复)
  --- FAIL: TestRedGreen (0.00s)
  FAIL    github.com/codesandbox/codesandbox
  ```
- go test -race 可能检测到与文件操作相关的异常（若涉及并发场景）

## 5. 期望结果（Expected Behavior）
修复后按相同步骤运行应出现：
- 包含路径遍历字符的自定义码在创建时被立即拒绝，返回验证错误
- 系统不会在非预期目录下创建或读取文件
- 正常短链接的创建、查询、重定向功能不受影响
- 测试输出判定为 GREEN（绿灯，缺陷已修复）
- go build ./... 与 go vet ./... 全部通过
- 正常功能的回归测试全部通过

## 6. 触发频率（Frequency）
必现（100%）。只要使用包含 `..`、`/`、`\` 等路径遍历字符的自定义码，必然触发漏洞。

## 7. 影响范围（Impact / Scope）
- 安全风险：攻击者可能通过构造特殊的自定义码，在文件系统的任意位置写入或读取文件
- 数据完整性：可能覆盖系统关键配置文件或注入恶意内容
- 服务可用性：异常的文件路径访问可能导致服务 panic 或文件系统错误
- 权限边界突破：路径遍历可能绕过存储目录的安全边界，访问到不应被短链接服务触及的文件
- 信息泄露：通过精心构造的code，可能读取到系统中的敏感文件内容

## 8. 附加说明（Additional Notes / Workaround）
临时规避方案：在调用短链接创建接口前，客户端侧先对自定义码做严格校验，只允许字母、数字、短横线和下划线。但此方案仅能缓解风险，不能替代服务端的修复。服务端必须在所有涉及code的入口进行统一校验，确保纵深防御。
