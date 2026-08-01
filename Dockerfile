# ---- 前端构建 ----
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# ---- 后端构建 ----
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=frontend-builder /app/frontend/dist ./web/dist
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags '-s -w' -o /game-server ./cmd/server

# ---- 运行镜像 ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates sqlite-libs
WORKDIR /app
COPY --from=backend-builder /game-server .
EXPOSE 3000
VOLUME ["/app/data"]
CMD ["./game-server"]
