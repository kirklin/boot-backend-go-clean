# 使用官方 Golang 镜像作为基础镜像
FROM golang:1.23 AS builder

# 设置 Go 代理以提高构建速度
ENV GOPROXY=https://goproxy.cn,direct

# 创建工作目录
WORKDIR /app

# 只复制 go.mod 和 go.sum 文件，避免每次修改代码都重建缓存
COPY go.mod go.sum ./

# 下载 Go 依赖
RUN go mod download

# 复制源代码到容器
COPY . .

# 如果存在 .env 文件，则复制它
COPY .env* ./

# 编译 Go 应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags='-extldflags "-static"' -o main ./cmd/main.go

# 使用更小的镜像作为最终镜像
FROM alpine:latest

# 安装 CA 证书
RUN apk --no-cache add ca-certificates

# 设置容器的工作目录
WORKDIR /root/

# 将构建好的二进制文件从构建镜像复制到最终镜像
COPY --from=builder /app/main .

# 如果存在 .env 文件，则从构建阶段复制它
COPY --from=builder /app/.env* ./

# 运行二进制文件
CMD ["./main"]
