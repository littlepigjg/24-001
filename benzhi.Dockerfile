# benzhi.Dockerfile - 代码沙箱服务
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译

FROM golang:1.22

WORKDIR /app

# 禁用 cgo 以支持多架构交叉编译
ENV CGO_ENABLED=0

# 复制所有源代码
COPY . .

# 预编译验证
RUN go build ./...

# 默认启动命令
CMD ["go", "run", "./cmd/server"]