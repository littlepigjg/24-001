# benzhi.Dockerfile - 轻量级延迟任务调度器
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译
# 禁用 cgo 以支持跨架构构建（QEMU 模拟环境下 cgo 不可用）

FROM golang:1.22

WORKDIR /app

# 复制所有源代码（本项目仅用标准库，无需 go mod download）
COPY . .

# 禁用 cgo，确保跨架构构建兼容性
ENV CGO_ENABLED=0

# 预编译验证
RUN go build ./...

# 默认启动命令
CMD ["go", "run", "./cmd/server"]