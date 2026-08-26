# ==================== 基础镜像 ====================
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git gcc musl-dev

# 复制 go.mod 和 go.sum（利用 Docker 缓存层）
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=1.0.0" -o /codesandbox ./cmd/server

# ==================== 运行时镜像 ====================
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache \
    python3 \
    python3-pip \
    python3-dev \
    nodejs \
    npm \
    openjdk17-jre-headless \
    bash \
    curl \
    ca-certificates \
    tzdata

# 设置 Python 默认版本
RUN ln -sf python3 /usr/bin/python && \
    ln -sf python3 /usr/bin/python2

# 创建运行时用户
RUN addgroup -S sandbox && adduser -S -G sandbox sandbox

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /codesandbox /app/codesandbox

# 复制静态资源
COPY web/static /app/web/static

# 创建数据目录
RUN mkdir -p /app/data/tmp /app/data/executions /app/data/templates && \
    chown -R sandbox:sandbox /app

# 设置环境变量
ENV SANDBOX_PORT=8080 \
    SANDBOX_HOST=0.0.0.0 \
    SANDBOX_DATA_DIR=/app/data \
    SANDBOX_LOG_LEVEL=info \
    SANDBOX_MAX_CODE_LENGTH=100000 \
    SANDBOX_MAX_EXEC_TIME=30s \
    SANDBOX_MAX_MEMORY_MB=256 \
    SANDBOX_MAX_OUTPUT_BYTES=1048576

# 暴露端口
EXPOSE 8080

# 切换到非特权用户
USER sandbox

# 设置健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 启动服务
ENTRYPOINT ["/app/codesandbox"]
