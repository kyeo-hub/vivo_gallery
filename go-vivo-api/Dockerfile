# 构建阶段
FROM golang:1.21-alpine AS builder

# 使用国内镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装构建依赖
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# 设置国内 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

# 复制所有文件
COPY . .

# 下载依赖
RUN go mod tidy && go mod download

# 编译静态二进制
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o vivo-api

# 运行阶段
FROM alpine:latest

# 使用国内镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装 ca-certificates 用于 HTTPS 请求
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 复制二进制文件
COPY --from=builder /app/vivo-api .

# 创建数据目录
RUN mkdir -p /data

# 暴露端口
EXPOSE 8080

# 数据卷挂载点
VOLUME ["/data"]

# 运行
CMD ["./vivo-api"]
