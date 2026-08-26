# benzhi.Dockerfile - Code Sandbox 服务
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译

FROM golang:1.22

WORKDIR /app

# 复制源代码
COPY . .

# 预下载依赖
RUN go mod download

# 预编译验证
RUN go build ./...

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV SANDBOX_PORT=8080 \
    SANDBOX_HOST=0.0.0.0 \
    SANDBOX_DATA_DIR=/app/data

# 默认启动命令（保留 Go 工具链，使用 go run 启动）
CMD ["go", "run", "./cmd/server"]