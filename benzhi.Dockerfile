# benzhi.Dockerfile - 轻量级延迟任务调度器
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译
# 修复：移除 curl 安装步骤，golang:1.22 镜像自带 wget

FROM golang:1.22

WORKDIR /app

# 复制所有源代码（本项目仅用标准库，无需 go mod download）
COPY . .

# 默认启动命令
CMD ["go", "run", "./cmd/server"]