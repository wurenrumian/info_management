# syntax=docker/dockerfile:1

# 使用国内镜像代理（实测可拉取）
FROM docker.m.daocloud.io/library/golang:1.25-alpine AS builder

WORKDIR /app

# Go 依赖下载使用国内代理
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM docker.m.daocloud.io/library/alpine:3.20 AS runtime

WORKDIR /app

# Alpine 包管理切换国内源，并安装证书/时区/健康检查依赖
# 其中 poppler-utils 提供 pdftotext（知识库 PDF 正文抽取必需）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata wget poppler-utils \
    && adduser -D -H -u 10001 appuser

COPY --from=builder /out/server /app/server

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]
