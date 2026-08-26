# Code Sandbox 在线代码执行沙箱系统 - Docker 部署评测

## 一、项目概述

Code Sandbox 是一个基于 Go 语言实现的在线代码运行沙箱系统，支持 Python、JavaScript、Java 三种主流编程语言的代码在线执行。系统提供了安全的代码执行环境、执行历史记录、代码模板管理等核心功能。

### 技术特性

| 特性 | 说明 |
|------|------|
| 编程语言 | Go 1.21+ |
| Web 框架 | 标准库 net/http |
| 代码执行 | 子进程隔离 + 资源限制 |
| 存储方案 | 内存存储 + 文件存储双模式 |
| 日志系统 | 结构化日志，支持多级别 |
| 优雅关闭 | 信号监听 + 上下文超时 |
| 安全机制 | 代码扫描 + 资源限制 + 沙箱隔离 |

---

## 二、功能特性

### 2.1 代码执行

- **多语言支持**: Python、JavaScript (Node.js)、Java
- **资源限制**: CPU 时间、内存、文件大小、进程数
- **输出捕获**: 标准输出 + 标准错误
- **超时控制**: 可配置执行超时时间
- **安全扫描**: 提交前进行代码安全检查

### 2.2 执行历史

- 历史记录存储与查询
- 按语言筛选历史
- 关键词搜索历史
- 历史记录分页浏览
- 自动清理过期记录

### 2.3 模板管理

- 预置 8 种语言模板
- 模板 CRUD 操作
- 模板分类与标签
- 模板使用次数统计
- 模板搜索功能

### 2.4 系统管理

- 健康检查端点
- 语言支持查询
- 代码验证接口
- 统计信息接口

---

## 三、API 接口

### 3.1 核心接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/executions` | 提交代码执行 |
| GET | `/api/executions` | 列出执行历史 |
| GET | `/api/executions/{id}` | 获取单个执行记录 |
| DELETE | `/api/executions/{id}` | 删除执行记录 |
| GET | `/api/executions/recent` | 获取最近执行 |
| DELETE | `/api/executions/cleanup` | 清理过期记录 |

### 3.2 模板接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/templates` | 创建模板 |
| GET | `/api/templates` | 列出模板 |
| GET | `/api/templates/{id}` | 获取单个模板 |
| PUT | `/api/templates/{id}` | 更新模板 |
| DELETE | `/api/templates/{id}` | 删除模板 |
| GET | `/api/templates/search?q=` | 搜索模板 |

### 3.3 系统接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/api/languages` | 支持的语言列表 |
| POST | `/api/validate` | 代码验证 |
| GET | `/api/stats` | 系统统计信息 |

### 3.4 示例请求

#### 提交代码执行

```bash
curl -X POST http://localhost:8080/api/executions \
  -H "Content-Type: application/json" \
  -d '{
    "language": "python",
    "code": "print(\"Hello, World!\")",
    "stdin": "",
    "timeout": 10
  }'
```

#### 执行结果

```json
{
  "success": true,
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "language": "python",
    "code": "print(\"Hello, World!\")",
    "status": "success",
    "output": "Hello, World!\n",
    "error": "",
    "execution_time_ms": 150,
    "memory_used_kb": 2048,
    "exit_code": 0,
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

---

## 四、Docker 部署

### 4.1 快速部署

```bash
# 1. 构建镜像
cd project-root
./build_docker.sh

# 2. 运行容器
docker run -d \
  --name codesandbox \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e SANDBOX_LOG_LEVEL=info \
  codesandbox:latest

# 3. 查看日志
docker logs -f codesandbox

# 4. 健康检查
curl http://localhost:8080/health
```

### 4.2 环境变量配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SANDBOX_PORT` | 8080 | 服务端口 |
| `SANDBOX_HOST` | 0.0.0.0 | 监听地址 |
| `SANDBOX_DATA_DIR` | /app/data | 数据存储目录 |
| `SANDBOX_LOG_LEVEL` | info | 日志级别 (debug/info/warn/error) |
| `SANDBOX_MAX_CODE_LENGTH` | 100000 | 代码最大长度（字节） |
| `SANDBOX_MAX_EXEC_TIME` | 30s | 最大执行时间 |
| `SANDBOX_MAX_MEMORY_MB` | 256 | 最大内存限制（MB） |
| `SANDBOX_MAX_OUTPUT_BYTES` | 1048576 | 最大输出大小（字节） |

### 4.3 多阶段构建

项目采用多阶段 Docker 构建：

**构建阶段 (builder)**
- 基础镜像: `golang:1.21-alpine`
- 用于编译 Go 源码
- 包含完整的 Go 工具链

**运行阶段 (runtime)**
- 基础镜像: `alpine:3.19`
- 仅包含运行时所需文件
- 预装 Python3、Node.js、OpenJDK 17
- 镜像体积优化

### 4.4 Docker Compose 部署

```yaml
version: '3.8'
services:
  codesandbox:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - SANDBOX_LOG_LEVEL=info
      - SANDBOX_MAX_EXEC_TIME=30s
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

---

## 五、项目结构

```
codesandbox/
├── cmd/
│   └── server/
│       └── main.go              # 入口点
├── internal/
│   ├── config/
│   │   ├── config.go            # 配置结构
│   │   ├── defaults.go         # 默认配置
│   │   └── env_config.go       # 环境变量配置
│   ├── handler/
│   │   ├── router.go           # 路由配置
│   │   ├── execution_handler.go # 执行处理器
│   │   ├── template_handler.go  # 模板处理器
│   │   ├── history_handler.go   # 历史处理器
│   │   ├── language_handler.go  # 语言处理器
│   │   ├── health_handler.go    # 健康检查
│   │   ├── validator.go         # 代码验证
│   │   └── middleware.go       # 中间件
│   ├── model/
│   │   ├── execution.go        # 执行模型
│   │   ├── template.go         # 模板模型
│   │   ├── history.go          # 历史模型
│   │   ├── language.go         # 语言模型
│   │   ├── result.go           # 结果模型
│   │   ├── config.go           # 配置模型
│   │   └── request.go          # 请求模型
│   ├── service/
│   │   ├── execution_service.go # 执行业务逻辑
│   │   ├── template_service.go  # 模板业务逻辑
│   │   ├── history_service.go   # 历史业务逻辑
│   │   ├── language_service.go  # 语言业务逻辑
│   │   ├── sandbox_service.go   # 沙箱服务
│   │   ├── resource_limiter.go  # 资源限制
│   │   ├── code_validator.go    # 代码验证
│   │   └── security_scanner.go  # 安全扫描
│   └── store/
│       ├── execution_store.go   # 执行存储接口
│       ├── template_store.go    # 模板存储接口
│       ├── history_store.go     # 历史存储接口
│       ├── memory_store.go      # 内存存储实现
│       └── file_store.go        # 文件存储实现
├── pkg/
│   ├── logger/
│   │   ├── logger.go            # 日志核心
│   │   └── format.go            # 日志格式化
│   ├── response/
│   │   ├── response.go          # 响应封装
│   │   └── error.go             # 错误定义
│   ├── process/
│   │   ├── runner.go            # 进程运行器
│   │   ├── manager.go           # 进程管理器
│   │   ├── limiter.go           # 资源限制器
│   │   └── capture.go           # 输出捕获
│   ├── cache/
│   │   └── cache.go             # 内存缓存
│   ├── concurrent/
│   │   └── concurrent.go        # 并发工具
│   ├── fileutil/
│   │   └── fileutil.go          # 文件工具
│   ├── hash/
│   │   └── hash.go              # 哈希工具
│   ├── netutil/
│   │   └── netutil.go           # 网络工具
│   ├── osutil/
│   │   └── osutil.go            # OS 工具
│   ├── pool/
│   │   └── pool.go              # 对象池
│   ├── ratelimit/
│   │   └── ratelimit.go         # 速率限制
│   ├── retry/
│   │   └── retry.go             # 重试机制
│   ├── stringutil/
│   │   ├── stringutil.go        # 字符串工具
│   │   └── validate.go          # 字符串验证
│   ├── syncutil/
│   │   └── map.go               # 并发 Map
│   ├── timeutil/
│   │   └── timeutil.go          # 时间工具
│   ├── uuid/
│   │   └── uuid.go              # UUID 生成
│   ├── validation/
│   │   └── validation.go        # 验证工具
│   └── jsonutil/
│       └── jsonutil.go          # JSON 工具
├── web/
│   └── static/
│       └── index.html           # 前端页面
├── go.mod
├── go.sum
├── Dockerfile
├── build_docker.sh
└── BUG_CATALOG.md
```

---

## 六、安全设计

### 6.1 进程隔离
- 代码在独立子进程中执行
- 使用 `ulimit` 限制系统资源
- 非特权用户运行

### 6.2 资源限制
- **CPU 时间限制**: 防止无限循环消耗 CPU
- **内存限制**: 防止内存溢出
- **文件大小限制**: 防止磁盘写入攻击
- **进程数量限制**: 防止 fork bomb

### 6.3 代码安全
- 关键字黑名单检查
- 危险函数识别
- 语法验证

### 6.4 网络隔离
- 沙箱容器不暴露网络
- 仅开放必要端口

---

## 七、性能指标

| 指标 | 目标值 |
|------|--------|
| 单代码执行延迟 | < 500ms (简单代码) |
| 并发执行能力 | 100+ 请求/秒 |
| 内存占用 | < 50MB (空闲状态) |
| 启动时间 | < 2s |
| 健康检查响应 | < 10ms |

---

## 八、评测说明

### 8.1 功能完整性
- [x] Python 代码执行
- [x] JavaScript 代码执行
- [x] Java 代码执行
- [x] 执行历史记录
- [x] 模板管理
- [x] 健康检查
- [x] 优雅关闭
- [x] 结构化日志
- [x] 资源限制
- [x] 代码安全扫描

### 8.2 代码质量
- [x] 纯 Go 标准库实现
- [x] 清晰的分层架构
- [x] 完整的接口抽象
- [x] 完善的错误处理
- [x] 线程安全的并发设计

### 8.3 部署友好性
- [x] 多阶段 Docker 构建
- [x] 健康检查配置
- [x] 环境变量配置
- [x] 数据持久化支持
- [x] 非特权用户运行

---

## 九、快速开始指南

### 本地开发

```bash
# 1. 克隆项目
git clone <repository-url>
cd codesandbox

# 2. 安装依赖
go mod download

# 3. 运行服务
go run ./cmd/server

# 4. 验证服务
curl http://localhost:8080/health
```

### Docker 构建

```bash
# 1. 确保 Docker 已安装
docker --version

# 2. 构建镜像
./build_docker.sh

# 3. 运行容器
docker run -d -p 8080:8080 codesandbox:latest

# 4. 查看日志
docker logs -f <container-id>
```

### 测试执行

```bash
# Python 示例
curl -X POST http://localhost:8080/api/executions \
  -H "Content-Type: application/json" \
  -d '{"language": "python", "code": "for i in range(5): print(f\"Hello {i}\")"}'

# JavaScript 示例
curl -X POST http://localhost:8080/api/executions \
  -H "Content-Type: application/json" \
  -d '{"language": "javascript", "code": "for(let i=0; i<5; i++) { console.log(`Hello ${i}`) }"}'

# Java 示例
curl -X POST http://localhost:8080/api/executions \
  -H "Content-Type: application/json" \
  -d '{"language": "java", "code": "public class Main { public static void main(String[] args) { System.out.println(\"Hello, World!\"); } }"}'
```

---

## 十、常见问题

### Q1: 容器启动后无法访问？
检查端口映射：`docker ps` 确认容器状态，使用 `-p` 参数映射端口。

### Q2: 代码执行超时？
调整 `SANDBOX_MAX_EXEC_TIME` 环境变量，默认 30 秒。

### Q3: 内存不足？
调整 `SANDBOX_MAX_MEMORY_MB` 环境变量，默认 256MB。

### Q4: 如何持久化数据？
挂载数据目录到容器：`-v /host/path:/app/data`

### Q5: 如何查看执行日志？
使用 `docker logs -f <container-id>` 查看应用日志。

---

## 附录

### A. 预置模板列表

| 语言 | 模板名称 | 描述 |
|------|----------|------|
| Python | Hello World | 基本输出示例 |
| Python | Fibonacci | 递归算法示例 |
| Python | Sorting | 排序算法示例 |
| JavaScript | Hello World | 基本输出示例 |
| JavaScript | Array Methods | 数组操作示例 |
| JavaScript | Async Demo | 异步操作示例 |
| Java | Hello World | 基本输出示例 |
| Java | Calculator | 计算器示例 |

### B. 状态码说明

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 404 | 资源未找到 |
| 500 | 服务器内部错误 |

### C. 日志级别

| 级别 | 说明 |
|------|------|
| debug | 调试信息 |
| info | 常规信息 |
| warn | 警告信息 |
| error | 错误信息 |
| fatal | 致命错误 |

---

**项目版本**: 1.0.0
**技术栈**: Go 1.21+, Alpine Linux, Docker
**许可证**: MIT License
