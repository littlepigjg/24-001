# BUG_CATALOG - 缺陷候选清单

> 本文档列出了可在项目中注入的 30 个跨文件缺陷。每个缺陷均标注了涉及的文件路径和函数名，方便评估者定位。
> 缺陷按类型分类，覆盖了并发、空指针、切片、错误处理、上下文、延迟、资源泄露、边界条件、性能、安全等场景。

---

## 缺陷清单

### 1: 并发写入竞态（Concurrent Write Race）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 1 |
| 分类 | 并发/竞态 |
| 难度 | ★★★★☆ |
| 描述 | 多个 goroutine 同时写入共享 map 时未加锁保护，导致 data race 或 fatal error。 |
| 植入位置 | `internal/store/memory_store.go: Put` 函数 — 直接对 `s.data[key]` 赋值而未获取写锁 |
| 预期表现 | 高并发下触发 `fatal error: concurrent map writes` 或 data race 检测报警 |
| 触发方式 | 使用 `go test -race` 运行并发测试，或发送大量并发 API 请求 |
| 跨文件影响 | `internal/handler/router.go: SetupRoutes` → `internal/store/memory_store.go: MemoryStore.Put` → `pkg/syncutil/map.go: SafeMap` |

---

### 2: 空指针解引用（Nil Pointer Dereference）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 2 |
| 分类 | 空指针 |
| 难度 | ★★☆☆☆ |
| 描述 | Get 方法在 key 不存在时返回 nil，调用方未做 nil 检查直接访问字段。 |
| 植入位置 | `internal/store/memory_store.go: Get` 函数 — 当 key 不存在时返回 `nil, nil` 而非 `nil, ErrNotFound` |
| 预期表现 | 访问不存在的记录时触发 `nil pointer dereference` panic |
| 触发方式 | 发送 GET 请求查询不存在的 execution_id |
| 跨文件影响 | `internal/handler/execution_handler.go: GetExecution` → `internal/service/execution_service.go: Get` → `internal/store/memory_store.go: Get` |

---

### 3: 切片越界访问（Slice Out-of-Bounds）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 3 |
| 分类 | 切片/数组 |
| 难度 | ★★☆☆☆ |
| 描述 | 分页计算 pageSize 时未校验参数合法性，pageSize=-1 导致切片越界。 |
| 植入位置 | `internal/service/template_service.go: List` — 未检查 pageSize 是否为正数，负数传入切片索引 |
| 预期表现 | 传入 page_size=-1 时触发 `slice bounds out of range` panic |
| 触发方式 | 发送 `GET /api/templates?page_size=-1` 请求 |
| 跨文件影响 | `internal/handler/template_handler.go: ListTemplates` → `internal/service/template_service.go: List` → `internal/store/memory_store.go: List` |

---

### 4: 错误信息丢失（Error Information Lost）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 4 |
| 分类 | 错误处理 |
| 难度 | ★★☆☆☆ |
| 描述 | 底层错误被 `errors.New("unknown error")` 替换，丢失了原始错误信息。 |
| 植入位置 | `internal/service/execution_service.go: Submit` — 将 `exec.Run()` 返回的错误替换为通用错误消息 |
| 预期表现 | 执行失败时只能看到 "execution failed" 而无法获知具体原因（如超时、编译错误） |
| 触发方式 | 提交一个有语法错误的 Python 代码 |
| 跨文件影响 | `internal/service/execution_service.go: Submit` → `pkg/process/runner.go: Run` → `pkg/response/response.go: ErrorResponse` |

---

### 5: 上下文未传递（Context Not Propagated）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 5 |
| 分类 | 上下文 |
| 难度 | ★★★☆☆ |
| 描述 | HTTP 请求的 context 未传递给下游的子进程执行，导致请求取消无法停止正在运行的进程。 |
| 植入位置 | `internal/handler/execution_handler.go: Submit` — 使用 `context.Background()` 替代 `r.Context()` 传递给 service 层 |
| 预期表现 | 客户端取消请求后，服务器端子进程仍继续运行直到完成或超时 |
| 触发方式 | 提交一个长时间运行的代码，客户端在执行完成前断开连接 |
| 跨文件影响 | `internal/handler/execution_handler.go: Submit` → `internal/service/execution_service.go: Execute` → `pkg/process/runner.go: Run` |

---

### 6: 延迟执行未调用（Defer Not Called）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 6 |
| 分类 | 资源泄露 |
| 难度 | ★★★☆☆ |
| 描述 | 文件句柄打开后忘记 defer Close，导致文件描述符泄露。 |
| 植入位置 | `pkg/process/capture.go: CaptureOutput` — `os.Create` 后忘记 `defer file.Close()` |
| 预期表现 | 高并发执行后出现 "too many open files" 错误 |
| 触发方式 | 连续提交大量执行请求（>1000） |
| 跨文件影响 | `pkg/process/runner.go: Run` → `pkg/process/capture.go: CaptureOutput` → `internal/service/execution_service.go: Submit` |

---

### 7: 资源限制缺失（Missing Resource Limits）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 7 |
| 分类 | 安全/资源 |
| 难度 | ★★★★☆ |
| 描述 | 资源限制配置未应用到子进程，代码可以使用无限内存或执行时间。 |
| 植入位置 | `pkg/process/runner.go: Run` — 注释掉 `limiter.ApplyLimits()` 调用 |
| 预期表现 | 恶意代码可以耗尽服务器内存或 CPU 资源 |
| 触发方式 | 提交一个创建大量数组的代码（如 `a = [0]*10**9`） |
| 跨文件影响 | `internal/config/config.go: SandboxConfig` → `pkg/process/limiter.go: Limiter` → `pkg/process/runner.go: Run` |

---

### 8: 类型断言未检查（Type Assertion Without Check）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 8 |
| 分类 | 类型/接口 |
| 难度 | ★★★☆☆ |
| 描述 | 对 interface{} 进行强制类型断言而不检查第二返回值。 |
| 植入位置 | `pkg/cache/cache.go: Get` — 直接断言为 `string` 类型，不做类型检查 |
| 预期表现 | 存储非 string 类型数据时触发 `interface conversion: interface {} is int64, not string` panic |
| 触发方式 | 向缓存写入 int64 类型数据后读取 |
| 跨文件影响 | `internal/handler/health_handler.go: GetStats` → `pkg/cache/cache.go: Get` → `pkg/cache/cache.go: Set` |

---

### 9: 边界条件未处理（Boundary Condition Not Handled）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 9 |
| 分类 | 边界条件 |
| 难度 | ★★☆☆☆ |
| 描述 | 空代码提交时，代码长度检查绕过，导致空代码进入执行流程。 |
| 植入位置 | `internal/model/execution.go: Validate` — 空代码字符串跳过长度校验 |
| 预期表现 | 提交空代码时执行成功（返回空输出），但未做任何实际执行 |
| 触发方式 | 提交 `{"language": "python", "code": ""}` |
| 跨文件影响 | `internal/handler/execution_handler.go: Submit` → `internal/model/execution.go: Validate` → `internal/service/execution_service.go: Execute` |

---

### 10: 时区处理错误（Timezone Handling Error）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 10 |
| 分类 | 时间处理 |
| 难度 | ★★★☆☆ |
| 描述 | 使用 `time.Now().Add(-24*time.Hour)` 计算时间差，而未考虑时区偏移导致历史记录过滤不准确。 |
| 植入位置 | `internal/service/history_service.go: Cleanup` — 使用 `time.Now()` 的本地时间与 UTC 存储时间比较 |
| 预期表现 | 跨时区部署时，清理操作可能删除有效数据或保留过期数据 |
| 触发方式 | 在 UTC+8 时区部署，清理设置为 24 小时过期的记录 |
| 跨文件影响 | `internal/service/history_service.go: Cleanup` → `pkg/timeutil/timeutil.go: ParseTime` → `internal/store/file_store.go: ListHistory` |

---

### 11: 日志级别过滤失效（Log Level Filtering Broken）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 11 |
| 分类 | 日志/配置 |
| 难度 | ★★☆☆☆ |
| 描述 | 日志级别比较使用字符串比较而非枚举值，导致 WARN 级别日志在 INFO 模式下仍输出。 |
| 植入位置 | `pkg/logger/logger.go: Enabled` — 使用 `strings.Compare` 替代整数比较 |
| 预期表现 | 设置日志级别为 ERROR 时，INFO 和 DEBUG 日志仍然输出 |
| 触发方式 | 启动时设置 `SANDBOX_LOG_LEVEL=error`，观察日志输出 |
| 跨文件影响 | `pkg/logger/logger.go: Enabled` → `internal/config/config.go: ServerConfig` → `cmd/server/main.go: main` |

---

### 12: 缓存键未校验（Cache Key Not Validated）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 12 |
| 分类 | 边界条件 |
| 难度 | ★★☆☆☆ |
| 描述 | 缓存键包含特殊字符（如 `/`、`..`），导致路径遍历或缓存污染。 |
| 植入位置 | `internal/service/template_service.go: Get` — 使用模板 ID 作为缓存键，未过滤特殊字符 |
| 预期表现 | 使用包含 `../../../etc/passwd` 的 ID 查询时，可能导致路径遍历 |
| 触发方式 | 发送 `GET /api/templates/../../../etc/passwd` 请求 |
| 跨文件影响 | `internal/handler/template_handler.go: Get` → `internal/service/template_service.go: Get` → `pkg/fileutil/fileutil.go: FileExists` |

---

### 13: 死锁风险（Deadlock Risk）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 13 |
| 分类 | 并发/锁 |
| 难度 | ★★★★☆ |
| 描述 | 嵌套锁获取：在持有 RLock 的情况下尝试获取 Lock，导致死锁。 |
| 植入位置 | `internal/store/memory_store.go: List` — 在持有读锁时调用 `Cleanup`，后者尝试获取写锁 |
| 预期表现 | 并发请求时服务器卡死，无响应 |
| 触发方式 | 高并发列表请求与写请求交替执行 |
| 跨文件影响 | `internal/store/memory_store.go: List` → `internal/store/memory_store.go: Cleanup` → `cmd/server/main.go: Shutdown` |

---

### 14: 内存泄露（Memory Leak）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 14 |
| 分类 | 内存/性能 |
| 难度 | ★★★★☆ |
| 描述 | 每次执行创建新的临时目录而未清理，导致磁盘空间耗尽。 |
| 植入位置 | `pkg/process/runner.go: Run` — `os.MkdirTemp` 创建的目录在执行完成后未调用 `os.RemoveAll` |
| 预期表现 | 长时间运行后服务器磁盘空间不足，`no space left on device` |
| 触发方式 | 提交 10000 次执行请求后检查磁盘使用 |
| 跨文件影响 | `pkg/process/runner.go: Run` → `pkg/process/capture.go: CaptureOutput` → `pkg/osutil/osutil.go: TempFile` |

---

### 15: 注入攻击（Code Injection）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 15 |
| 分类 | 安全 |
| 难度 | ★★★★★ |
| 描述 | 用户输入的代码拼接进 shell 命令字符串，未做转义处理。 |
| 植入位置 | `pkg/process/limiter.go: ApplyLimits` — 将代码文件路径直接拼接到 shell 命令中 |
| 预期表现 | 特殊字符注入后执行任意 shell 命令 |
| 触发方式 | 提交代码文件名为 `test.py; cat /etc/passwd` 的请求 |
| 跨文件影响 | `pkg/process/limiter.go: ApplyLimits` → `pkg/process/runner.go: Run` → `internal/service/sandbox_service.go: Execute` |

---

### 16: 竞争条件（TOCTOU Race）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 16 |
| 分类 | 并发/安全 |
| 难度 | ★★★★☆ |
| 描述 | 检查文件是否存在与打开文件之间存在时间窗口，攻击者可以替换文件。 |
| 植入位置 | `pkg/process/runner.go: Run` — 使用 `os.Stat` 检查文件存在，然后 `os.Open` 打开，中间无锁保护 |
| 预期表现 | 攻击者在检查和打开之间替换文件为恶意内容 |
| 触发方式 | 提交代码后快速替换代码文件为其他内容 |
| 跨文件影响 | `pkg/process/runner.go: Run` → `pkg/fileutil/fileutil.go: FileExists` → `pkg/fileutil/fileutil.go: ReadFileContent` |

---

### 17: 配置热更新失效（Config Hot-Reload Broken）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 17 |
| 分类 | 配置/刷新 |
| 难度 | ★★★☆☆ |
| 描述 | 配置加载后未缓存，每次请求都重新读取配置文件，导致性能问题。 |
| 植入位置 | `internal/config/config.go: Load` — 每次请求都调用 `os.ReadFile` 读取配置 |
| 预期表现 | 高并发下 CPU 使用率过高，磁盘 I/O 增加 |
| 触发方式 | 发送大量请求时观察 CPU 使用率 |
| 跨文件影响 | `internal/handler/router.go: SetupRoutes` → `internal/config/config.go: Load` → `cmd/server/main.go: NewServer` |

---

### 18: 缓冲区溢出（Buffer Overflow）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 18 |
| 分类 | 内存/边界 |
| 难度 | ★★★★☆ |
| 描述 | 输出缓冲区大小固定，当子进程输出超过缓冲区时发生截断或 panic。 |
| 植入位置 | `pkg/process/capture.go: RingBuffer` — `Write` 方法中当 `pos+len(data) > bufSize` 时直接 panic |
| 预期表现 | 运行输出超过 1MB 的代码时 panic |
| 触发方式 | 提交打印大量内容的代码：`print('a'*10**6)` |
| 跨文件影响 | `pkg/process/capture.go: RingBuffer` → `pkg/process/runner.go: Run` → `internal/model/execution.go: ExecutionResult` |

---

### 19: HTTPS 强制但未配置（HTTPS Forced But Not Configured）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 19 |
| 分类 | 配置/部署 |
| 难度 | ★★★☆☆ |
| 描述 | 强制检查 HTTPS 但未提供 TLS 证书，导致服务无法启动。 |
| 植入位置 | `cmd/server/main.go: main` — `server.ListenAndServeTLS` 使用了不存在的证书文件 |
| 预期表现 | 服务器启动失败，报 `no such file or directory` 错误 |
| 触发方式 | 启动服务器时观察启动日志 |
| 跨文件影响 | `cmd/server/main.go: main` → `internal/config/config.go: ServerConfig` → `pkg/osutil/osutil.go: FileExists` |

---

### 20: 请求体未关闭（Request Body Not Closed）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 20 |
| 分类 | 资源泄露 |
| 难度 | ★★☆☆☆ |
| 描述 | HTTP 请求的 Body 在读取后忘记 Close，导致连接泄露。 |
| 植入位置 | `internal/handler/execution_handler.go: Submit` — 读取 `r.Body` 后忘记 `defer r.Body.Close()` |
| 预期表现 | 高并发后出现 `too many open connections` 错误 |
| 触发方式 | 发送大量请求后观察服务器连接数 |
| 跨文件影响 | `internal/handler/execution_handler.go: Submit` → `pkg/response/response.go: DecodeJSON` → `internal/handler/router.go: SetupRoutes` |

---

### 21: 超时设置不合理（Unreasonable Timeout）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 21 |
| 分类 | 超时/资源 |
| 难度 | ★★★☆☆ |
| 描述 | 执行超时设为 0 表示永不超时，导致恶意代码无限执行。 |
| 植入位置 | `internal/config/defaults.go: GetDefaultConfig` — `Timeout: 0` 或不设超时 |
| 预期表现 | 死循环代码永远不会被终止 |
| 触发方式 | 提交 `while True: pass` 代码 |
| 跨文件影响 | `internal/config/defaults.go: GetDefaultConfig` → `pkg/process/runner.go: Run` → `internal/service/execution_service.go: Execute` |

---

### 22: JSON 解析不宽容（Rigid JSON Parsing）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 22 |
| 分类 | 解析/兼容性 |
| 难度 | ★★☆☆☆ |
| 描述 | 使用 `json.Decoder` 严格模式，遇到未知字段时报错。 |
| 植入位置 | `pkg/response/response.go: DecodeJSON` — 未调用 `d.DisallowUnknownFields()` 的情况下反而添加了它 |
| 预期表现 | 客户端发送额外字段时收到 400 错误 |
| 触发方式 | 提交请求时添加 `{"extra_field": "value"}` |
| 跨文件影响 | `pkg/response/response.go: DecodeJSON` → `internal/handler/execution_handler.go: Submit` → `internal/model/execution.go: ExecutionRequest` |

---

### 23: 日志格式化错误（Log Format Error）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 23 |
| 分类 | 日志/格式化 |
| 难度 | ★★☆☆☆ |
| 描述 | 日志格式化使用 `%s` 格式化数字类型，导致输出 "nil" 或错误。 |
| 植入位置 | `pkg/logger/logger.go: Infof` — 使用 `%s` 格式化 `int64` 类型的耗时 |
| 预期表现 | 日志中显示 `execution took <nil>` 或 `execution took %!s(int64=123)` |
| 触发方式 | 提交任何代码执行，查看日志 |
| 跨文件影响 | `pkg/logger/logger.go: Infof` → `pkg/logger/format.go: FormatEntry` → `internal/handler/middleware.go: LoggingMiddleware` |

---

### 24: 正则表达式 DoS（Regex ReDoS）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 24 |
| 分类 | 安全/性能 |
| 难度 | ★★★★☆ |
| 描述 | 搜索功能使用易受攻击的正则表达式，恶意输入导致 CPU 100%。 |
| 植入位置 | `internal/service/template_service.go: Search` — 使用 `(?i)` 匹配所有字符的正则 |
| 预期表现 | 特殊构造的输入导致搜索请求挂起数秒 |
| 触发方式 | 搜索 `a+a+a+a+a+` 类的复杂模式 |
| 跨文件影响 | `internal/service/template_service.go: Search` → `internal/store/memory_store.go: SearchTemplates` → `pkg/stringutil/stringutil.go: Contains` |

---

### 25: 数据库操作非原子（Non-Atomic DB Operation）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 25 |
| 分类 | 数据一致性 |
| 难度 | ★★★★☆ |
| 描述 | 写入操作分为两步（先创建记录再更新状态），中间崩溃导致数据不一致。 |
| 植入位置 | `internal/service/execution_service.go: Submit` — 先 `CreateExecution` 再 `UpdateExecution`，中间无事务保护 |
| 预期表现 | 服务器在两步操作之间崩溃时，数据库存在"僵尸"执行记录 |
| 触发方式 | 在高并发压力测试期间强制重启服务器 |
| 跨文件影响 | `internal/service/execution_service.go: Submit` → `internal/store/file_store.go: CreateExecution` → `internal/store/file_store.go: UpdateExecution` |

---

### 26: 进程僵尸（Zombie Process）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 26 |
| 分类 | 进程/资源 |
| 难度 | ★★★★☆ |
| 描述 | 子进程执行完成后未调用 `cmd.Wait()` 或 `cmd.Process.Release()`，产生僵尸进程。 |
| 植入位置 | `pkg/process/runner.go: Run` — 忘记在 `cmd.Run()` 后调用 `cmd.Wait()` |
| 预期表现 | 长时间运行后服务器出现大量 `<defunct>` 进程 |
| 触发方式 | 执行 1000+ 次代码后运行 `ps aux | grep defunct` |
| 跨文件影响 | `pkg/process/runner.go: Run` → `pkg/process/manager.go: Manager` → `pkg/osutil/osutil.go: KillProcess` |

---

### 27: 时间复杂度为 O(n²) 的查找（O(n²) Search）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 27 |
| 分类 | 性能/算法 |
| 难度 | ★★★☆☆ |
| 描述 | 历史记录列表在内存中使用线性扫描，大数据量下性能低下。 |
| 植入位置 | `internal/store/memory_store.go: ListHistory` — 嵌套循环遍历所有记录 |
| 预期表现 | 存储 100000 条记录后，列表请求耗时数秒 |
| 触发方式 | 插入大量记录后发送列表请求 |
| 跨文件影响 | `internal/store/memory_store.go: ListHistory` → `internal/service/history_service.go: List` → `internal/handler/history_handler.go: ListHistory` |

---

### 28: 字符编码处理错误（Encoding Error）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 28 |
| 分类 | 编码/国际化 |
| 难度 | ★★★☆☆ |
| 描述 | 使用 `[]byte` 操作处理 Unicode 字符，导致中文字符被截断。 |
| 植入位置 | `pkg/process/capture.go: RingBuffer.Write` — 使用 `buf[pos:pos+len(data)]` 操作可能截断 UTF-8 多字节字符 |
| 预期表现 | 输出包含中文时出现乱码或截断 |
| 触发方式 | 提交包含中文字符串的 Python 代码 |
| 跨文件影响 | `pkg/process/capture.go: RingBuffer.Write` → `internal/service/execution_service.go: Submit` → `pkg/stringutil/stringutil.go: Truncate` |

---

### 29: 优雅关闭不完整（Graceful Shutdown Incomplete）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 29 |
| 分类 | 生命周期/关闭 |
| 难度 | ★★★★☆ |
| 描述 | 优雅关闭时未等待正在执行的子进程完成，强制终止导致数据丢失。 |
| 植入位置 | `cmd/server/main.go: Shutdown` — 只调用 `http.Server.Shutdown()` 而未终止正在运行的子进程 |
| 预期表现 | 服务器关闭时正在执行的代码被强制中断，执行记录未保存 |
| 触发方式 | 在代码执行期间发送 SIGTERM 信号 |
| 跨文件影响 | `cmd/server/main.go: main` → `pkg/process/manager.go: Manager` → `pkg/process/runner.go: Run` |

---

### 30: CORS 配置过于宽松（Overly Permissive CORS）

| 属性 | 值 |
|------|-----|
| 缺陷ID | 30 |
| 分类 | 安全/配置 |
| 难度 | ★★★☆☆ |
| 描述 | CORS 中间件允许所有来源（`Access-Control-Allow-Origin: *`）且开启了 credentials。 |
| 植入位置 | `internal/handler/router.go: SetupRoutes` — CORS 头设置为 `*`，同时 `Access-Control-Allow-Credentials: true` |
| 预期表现 | 浏览器安全拦截或允许任意来源的跨域请求 |
| 触发方式 | 从任意域名发起 API 请求 |
| 跨文件影响 | `internal/handler/router.go: SetupRoutes` → `pkg/response/response.go: SetCORSHeaders` → `cmd/server/main.go: main` |

---

## 缺陷统计

| 分类 | 数量 |
|------|------|
| 并发/竞态 | 2 |
| 空指针 | 1 |
| 切片/数组 | 1 |
| 错误处理 | 1 |
| 上下文 | 1 |
| 资源泄露 | 2 |
| 安全 | 4 |
| 类型/接口 | 1 |
| 边界条件 | 2 |
| 时间处理 | 1 |
| 日志/配置 | 1 |
| 并发/锁 | 1 |
| 内存/性能 | 1 |
| 并发/安全 | 1 |
| 配置/刷新 | 1 |
| 内存/边界 | 1 |
| 配置/部署 | 1 |
| 超时/资源 | 1 |
| 解析/兼容性 | 1 |
| 日志/格式化 | 1 |
| 安全/性能 | 1 |
| 数据一致性 | 1 |
| 进程/资源 | 1 |
| 性能/算法 | 1 |
| 编码/国际化 | 1 |
| 生命周期/关闭 | 1 |
| 安全/配置 | 1 |
| **总计** | **30** |
