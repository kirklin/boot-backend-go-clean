# 使用官方 Golang Alpine 镜像作为构建器
FROM golang:1.25-alpine AS builder

# 设置 Go 代理以提高构建速度
ENV GOPROXY=https://goproxy.cn,direct

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 并下载依赖项
# 这利用了 Docker 的缓存机制，只有在 go.mod 或 go.sum 发生变化时才会重新下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
# 只复制需要的代码，以保持构建上下文的清洁和优化缓存
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/


# 编译应用程序，创建一个静态链接的二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags='-extldflags "-static"' -o main ./cmd/main.go

# --- 第二阶段：创建最终的轻量级镜像 ---
FROM alpine:latest

# 为应用程序创建一个非 root 用户和组
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 安装 CA 证书和时区数据
RUN apk --no-cache add ca-certificates tzdata

# 创建应用程序目录
RUN mkdir /app

# 将目录所有权赋予我们的非 root 用户
RUN chown -R appuser:appgroup /app

# 设置工作目录
WORKDIR /app

# 从构建器阶段复制编译好的二进制文件
COPY --from=builder /app/main .

# 复制 .env 文件到工作目录
# 确保在运行 docker-compose build 之前，项目根目录下存在 .env 文件
COPY .env .

# 切换到非 root 用户
USER appuser

# 运行应用程序
CMD ["./main"]
