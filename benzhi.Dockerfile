# benzhi.Dockerfile - URL Shortener Service
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译

FROM golang:1.22

WORKDIR /app

# 禁用 CGO 以支持跨架构构建（避免 QEMU 模拟环境下的 gcc 问题）
ENV CGO_ENABLED=0

# 复制所有源代码（本项目仅用标准库，无需 go mod download）
COPY . .

# 预编译验证
RUN go build ./...

# 默认启动命令
CMD ["go", "run", "./cmd/server"]