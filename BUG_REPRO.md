# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

短链接服务在批量创建大量短链接后，系统出现文件操作失败的问题。单次创建短链接正常，但当批量创建数百条短链接时，后续的创建请求会报错，涉及文件打开/写入操作的全部失败，表现为资源耗尽类错误。

## 2. 环境信息（Environment）

- 操作系统：Linux
- Go 版本：go1.22
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：go test . -count=1 -run '^TestRedGreen$'
- 硬件信息：与并发/性能相关可补充 CPU 核数等

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test -v -count=1 -run '^TestRedGreen$' .` 运行红绿灯测试
3. 观察测试输出中 FD count 变化情况和最终判定结果

## 4. 实际结果（Actual Behavior / Observed Output）

- 测试显示：
```
FD count before test: 29
FD count after creating 500 URLs: 529 (delta: 500)
RED（红灯，缺陷未修复）
文件描述符泄露检测: 基线 29, 创建后 529, 泄露增量 500
--- FAIL: TestRedGreen (0.04s)
FAIL
EXIT_CODE=1
```
- 每创建一个短链接，文件描述符泄露一个，500 次创建泄露 500 个 FD
- 判定结果为 RED（红灯）
- 退出码为 1（非 0）
- 无 DATA RACE 报告（非并发竞争类缺陷）

## 5. 期望结果（Expected Behavior）

- 创建任意数量的短链接后，文件描述符数量保持稳定，无泄露
- RED/GREEN 判定结果应为 GREEN（绿灯）
- 退出码为 0
- go build ./... 与 go vet ./... 全部通过
- 重定向功能正常，数据完整性保持

## 6. 触发频率（Frequency）

必现（100%）：只要批量创建超过一定数量的短链接（测试中为 500 个），必然触发文件描述符泄露，导致 RED 判定。

## 7. 影响范围（Impact / Scope）

- 批量操作场景下短链接创建全部失败
- 文件描述符资源被耗尽，影响系统内所有涉及文件操作的功能
- 服务无法正常持久化数据，可能导致数据丢失
- 严重时可能影响整个进程的文件操作能力

## 8. 附加说明（Additional Notes / Workaround）

- 临时规避：减少批量操作的数量，或在批次之间重启服务以释放泄露的文件描述符
- 根本解决：需要在文件写入操作完成后正确关闭文件句柄，确保资源及时释放
