# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

短链接服务在正常运行过程中，临时目录（`os.TempDir()`）下堆积了大量未清理的临时文件夹，以 `urlsvc-`、`redirect-`、`urlstore-`、`accesslog-` 为前缀。随着业务量增长，这些残留目录会持续占用磁盘空间，最终可能导致磁盘空间耗尽。问题出现在短链创建、重定向、更新和日志存储等多个操作路径中，且只有在特定条件下才会触发清理逻辑，其余场景均存在泄漏。

## 2. 环境信息（Environment）

- 操作系统：Linux (x86_64)
- Go 版本：go1.24.4
- 项目模块：github.com/codesandbox/codesandbox
- 运行参数：无需特殊参数，正常单元测试即可触发
- 硬件信息：CPU 4 核

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，确保项目可正常编译：`go build ./...`
2. 确保静态检查通过：`go vet ./...`
3. 执行缺陷检测测试：`go test -v -count=1 -run '^TestRedGreen$' .`
4. 观察测试输出中的 RED/GREEN 判定结果和各类型临时目录的泄漏数量

## 4. 实际结果（Actual Behavior / Observed Output）

测试输出：
```
=== RUN   TestRedGreen
    red_green_test.go:122: RED（红灯，缺陷未修复）
    red_green_test.go:123:   urlsvc leaked dirs: 50 (before=0, after_create=50, final=50)
    red_green_test.go:124:   redirect leaked dirs: 630 (before=0, after_create=0, final=630)
    red_green_test.go:125:   urlstore remaining: 70, accesslog remaining: 1
--- FAIL: TestRedGreen (0.47s)
FAIL
exit status 1
```

- urlsvc 类型目录泄漏：50 个（每个设置了 MaxVisits > 0 的短链创建都会泄漏一个）
- redirect 类型目录泄漏：630 个（同一短链重定向超过 1 次后，每次重定向都会泄漏一个）
- urlstore 类型目录残留：70 个（短链更新时产生的子目录未被追踪清理）
- accesslog 类型目录残留：1 个（日志存储关闭时未清理根目录）

## 5. 期望结果（Expected Behavior）

```
=== RUN   TestRedGreen
    red_green_test.go:128: GREEN（绿灯，缺陷已修复）
    red_green_test.go:129:   urlsvc dirs: 0, redirect dirs: 0, urlstore dirs: 0, accesslog dirs: 0
--- PASS: TestRedGreen (1.16s)
PASS
exit status 0
```

- 所有 urlsvc-* 目录在短链创建完成后被正确清理
- 所有 redirect-* 目录在每次重定向处理完成后被正确清理
- 所有 urlstore-* 子目录在 URLStore.Close() 时被正确清理
- accesslog-* 根目录在 AccessLogStore.Close() 时被正确清理
- go build ./... 编译通过
- go vet ./... 无警告

## 6. 触发频率（Frequency）

必现（100%）。只要执行短链创建（MaxVisits > 0）、重定向（次数 > 1）、短链更新和日志存储操作，就会产生临时目录泄漏。单元测试可稳定复现。

## 7. 影响范围（Impact / Scope）

- 磁盘空间持续消耗，长时间运行后可能导致磁盘空间不足
- 大量临时目录文件系统 inode 被占用，影响系统文件创建能力
- 在容器化部署环境中，可能导致容器因磁盘写满而崩溃
- 增加文件系统扫描和备份的负担，降低运维效率

## 8. 附加说明（Additional Notes / Workaround）

临时规避方法：可在运维层面定期清理 `/tmp` 下的 `urlsvc-*`、`redirect-*`、`urlstore-*`、`accesslog-*` 目录，但这只是权宜之计，根本修复需要在代码层面确保所有临时资源的生命周期正确管理。
