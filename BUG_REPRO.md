# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

代码沙箱服务在长时间运行或频繁执行超时任务后，系统中出现大量僵尸进程（`<defunct>`），表现为 `ps aux` 输出中存在状态为 `Z` 的进程条目。同时，进程管理模块的活跃进程计数在超时任务完成后无法归零，内存占用持续增长。

## 2. 环境信息（Environment）

- **操作系统**: Linux（支持 `/proc` 文件系统）
- **Go 版本**: go1.x
- **项目模块**: github.com/codesandbox/codesandbox
- **运行参数**: 无需特殊参数，直接运行 `go test` 即可复现
- **硬件信息**: CPU 核数不限

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test -v -count=1 -run '^TestRedGreen$' .` 运行验证用例
3. 观察测试输出结果
4. （可选）在测试运行期间另开终端执行 `ps aux | grep '<defunct>'` 查看僵尸进程

## 4. 实际结果（Actual Behavior / Observed Output）

- 测试失败（RED），输出如下：
  ```
  === RUN   TestRedGreen
  RED (红灯，缺陷未修复)
    ActiveProcesses before sweep: 1, WaitForProcesses reaped: 0, after sweep: 1
  --- FAIL: TestRedGreen (0.31s)
  FAIL
  ```
- 退出码为 1
- `ActiveProcesses` 在超时任务执行后返回值大于 0
- 调用清理方法后，活跃进程数仍未减少
- 系统中可观察到 `<defunct>` 僵尸进程

## 5. 期望结果（Expected Behavior）

- `go test -v -count=1 -run '^TestRedGreen$' .` 通过（GREEN），退出码为 0
- 测试输出 "GREEN (绿灯，缺陷已修复)"
- 超时任务执行后活跃进程计数正确归零
- 系统中无 `<defunct>` 僵尸进程积累
- `go build ./...` 与 `go vet ./...` 全部通过

## 6. 触发频率（Frequency）

- 必现（100%），只要执行超时路径的进程操作即可稳定复现
- 频繁执行超时任务时僵尸进程积累速度加快

## 7. 影响范围（Impact / Scope）

- **进程表资源泄漏**: 僵尸进程占用系统进程表条目，大量积累后可能导致无法创建新进程
- **内存泄漏**: 进程管理模块内部状态持续增长，长时间运行的服务会出现内存溢出
- **监控失准**: 活跃进程计数无法反映真实状态，导致运维监控误判
- **服务稳定性下降**: 进程表耗尽后代码沙箱服务无法执行任何代码

## 8. 附加说明（Additional Notes / Workaround）

目前无可靠的临时规避方案。定期重启服务可以暂时清除僵尸进程，但无法解决根本问题。
