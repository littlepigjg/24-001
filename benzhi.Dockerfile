# benzhi.Dockerfile - 轻量级延迟任务调度器
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译

FROM golang:1.22

WORKDIR /app

# 复制所有源代码（本项目仅用标准库，无需 go mod download）
COPY . .

# 预编译验证（CGO_ENABLED=0 确保跨架构构建兼容）
RUN CGO_ENABLED=0 go build ./...

# 默认启动命令（运行时也禁用 CGO 以确保跨架构兼容）
CMD ["sh", "-c", "CGO_ENABLED=0 go run ./cmd/server"]