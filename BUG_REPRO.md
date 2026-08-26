# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
URL 短链接服务的过期检查和清理功能存在异常：创建时间在有效期内的 URL 被错误地判定为已过期，并在清理操作中被错误删除或禁用。具体表现为，一个设置了 24 小时有效期的 URL，在创建仅 23 小时后就被标记为过期并清除。

## 2. 环境信息（Environment）
- 操作系统：Linux（本报告在 Linux 环境下复现）
- Go 版本：go 1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：默认配置，存储类型为内存存储
- 时区设置：UTC 或 UTC+8（东八区），问题在非 UTC 时区部署时表现更明显
- CPU 核数：与本缺陷无关

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 go build ./... 确保编译通过
2. 初始化 URL 短链接服务，创建一个新的 URLStore 和 AccessLogStore 实例
3. 创建一个 ShortURL 记录，设置创建时间为当前时间前 23 小时（createdAt = now.Add(-23h)）
4. 设置 ExpiresAt 为 createdAt 加 24 小时（ExpiresAt = createdAt.Add(24h)）
5. 保存该 ShortURL 到存储中
6. 调用过期检查接口判断该 URL 是否已过期
7. 调用清理接口，设置 maxAge 为 24 小时，执行清理操作
8. 观察清理后的 URL 状态（是否被删除或禁用）

## 4. 实际结果（Actual Behavior / Observed Output）
- 过期检查接口错误地返回"已过期"，尽管 URL 实际只创建了 23 小时，在 24 小时有效期内
- 清理操作错误地删除或禁用了该 URL，尽管它应该保留到 24 小时后才过期
- 清理后再次查询该 URL，发现其 Disabled 字段已被设置为 true，或已从存储中移除
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- go test -race 无数据竞争报告（本缺陷为纯逻辑错误）

## 5. 期望结果（Expected Behavior）
- 过期检查接口正确返回"未过期"，因为 URL 实际只创建了 23 小时，尚未到 24 小时的有效期
- 清理操作正确保留该 URL，不应对其进行删除或禁用
- 清理后 URL 保持启用状态，Disabled 字段为 false
- RED/GREEN 判定结果：GREEN（绿灯，缺陷已修复）
- go build 与 go vet 全部通过
- 跨时区部署时（如 UTC、UTC+8），过期检查和清理操作结果一致

## 6. 触发频率（Frequency）
- 必现（100%）：只要创建的 URL 创建时间在 maxAge 有效期内（如 23 小时 vs 24 小时），清理操作必然错误地将其判定为过期并清除
- 在任何时区均能稳定复现，因为缺陷是硬编码的时间偏移逻辑

## 7. 影响范围（Impact / Scope）
- URL 短链接服务的过期检查功能完全失效：有效期内的 URL 被错误标记为已过期
- 清理操作会错误删除有效数据，导致用户在 URL 有效期内无法访问
- 数据丢失风险：有效期内的 URL 被提前删除，无法恢复
- 用户体验严重受损：创建的短链接在使用期内就失效了
- 运维风险：定期清理任务会持续错误清除有效数据
- 跨时区部署时问题更严重，可能导致不同时区的服务实例行为不一致

## 8. 附加说明（Additional Notes / Workaround）
- 临时规避方法：在修复前，可以暂时禁用自动清理功能，手动管理 URL 生命周期
- 建议的排查方向：检查过期检查和清理操作中的时间计算逻辑，确认时间比较是否使用了一致的时区基准
- 相关日志示例：
  ```
  [INFO] Cleaned up 1 expired URLs
  但被清理的 URL 实际创建时间为 now-23h，在 24h 有效期内
  ```
