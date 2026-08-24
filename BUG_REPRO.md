# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
项目的日志级别过滤功能出现异常行为。当用户将日志级别设置为较高严重级别（如 error）时，较低级别（INFO、WARN）的日志消息仍然会输出到控制台；反之，当设置为 warn 级别时，本应输出的 ERROR 级别日志却被过滤掉了。整个日志级别过滤逻辑呈现混乱状态，高优先级日志可能被遗漏，低优先级日志反而泄漏。

## 2. 环境信息（Environment）
- 操作系统：Linux (x86_64)
- Go 版本：go 1.22+
- 项目模块：github.com/codesandbox/codesandbox
- 项目类型：Go 代码沙箱服务
- 运行参数：无需特殊参数，默认配置即可复现
- 硬件信息：与硬件无关，任意 CPU 均可复现

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 编写一个简单的 Go 测试或直接在代码中创建日志实例
3. 创建 Logger 实例并设置日志级别为 LevelError
4. 依次调用 Debug、Info、Warn、Error 方法输出日志消息
5. 观察输出结果，记录各级别消息的出现情况
6. 再将日志级别改为 LevelWarn，重复步骤 3-5，观察结果

## 4. 实际结果（Actual Behavior / Observed Output）
- 当级别为 LevelError 时：
  - DEBUG 消息：被过滤（正确）
  - INFO 消息：**仍然输出（错误）**
  - WARN 消息：**仍然输出（错误）**
  - ERROR 消息：被过滤（错误）
  - 实际观察到 INFO 和 WARN 消息出现在输出中，而 ERROR 消息没有
- 当级别为 LevelWarn 时：
  - DEBUG 消息：被过滤（正确）
  - INFO 消息：被过滤（正确）
  - WARN 消息：正常输出（正确）
  - ERROR 消息：**被过滤（错误）**
- 测试用例运行结果：RED（红灯），退出码 1
- go test -race：无数据竞争（非并发缺陷）

## 5. 期望结果（Expected Behavior）
- 当级别为 LevelError 时：
  - DEBUG、INFO、WARN 消息不输出
  - ERROR、FATAL 消息正常输出
- 当级别为 LevelWarn 时：
  - DEBUG、INFO 消息不输出
  - WARN、ERROR、FATAL 消息正常输出
- 测试用例通过，判定为 GREEN（绿灯）
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
必现（100%）。只要设置日志级别并调用过滤方法，每次都会触发错误行为。

## 7. 影响范围（Impact / Scope）
- 线上环境日志刷屏：设置高日志级别无法有效过滤低级别日志，导致日志量过大，关键信息被淹没
- 错误日志丢失：设置 warn 级别时 ERROR 日志被过滤，导致线上错误无法被及时发现和排查
- 日志级别配置失效：用户对日志级别的配置完全不起作用，整个日志系统的可信度下降
- 影响所有使用日志系统的模块，包括服务启动日志、请求处理日志、错误处理日志等

## 8. 附加说明（Additional Notes / Workaround）
目前没有有效的临时规避方法。唯一的临时措施是将日志级别始终设为最低级别（DEBUG），但这会导致大量日志输出。建议尽快修复。