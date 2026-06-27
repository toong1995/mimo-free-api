# syntax=docker/dockerfile:1

# ===== Stage 1: 构建前端 =====
FROM node:20-slim AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ===== Stage 2: 构建后端 =====
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
# 前端构建产物（static/）由上一阶段生成
COPY --from=frontend /app/static ./static
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o mimo-gateway .

# ===== Stage 3: 运行时 =====
FROM alpine:3.19
# ca-certificates: HTTPS 请求小米服务端；tzdata: 时区；wget: healthcheck
RUN apk --no-cache add ca-certificates tzdata wget && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=backend /app/mimo-gateway .
COPY config.example.json /app/config.example.json

# 数据目录（统计 + 配置持久化），交给命名卷挂载
RUN mkdir -p /app/data && chown -R app:app /app

# 以非 root 用户运行
USER app

EXPOSE 8080
VOLUME ["/app/data"]

# 存活探针：访问首页（无需鉴权），失败则重启
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://localhost:8080/ || exit 1

CMD ["./mimo-gateway"]
