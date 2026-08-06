#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:3000/api/health}"

BACKEND_PID=""

check_dependency() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少依赖: $1"
    exit 1
  fi
}

cleanup() {
  if [[ -n "${BACKEND_PID}" ]] && kill -0 "${BACKEND_PID}" >/dev/null 2>&1; then
    kill "${BACKEND_PID}" >/dev/null 2>&1 || true
    wait "${BACKEND_PID}" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

check_dependency go
check_dependency npm
check_dependency curl

echo "检查前端依赖..."
if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
  echo "安装前端依赖..."
  (cd "$FRONTEND_DIR" && npm install)
fi

echo "构建前端..."
(cd "$FRONTEND_DIR" && npm run build)

# 兼容 Clash 类 HTTP 代理被误配成 GOPROXY 的环境：Go 模块协议不支持 CONNECT 代理，
# 把 http:// 开头的 GOPROXY 改为 HTTPS_PROXY 出站 + 官方模块代理。
if [[ "${GOPROXY:-}" == http://* ]]; then
  proxy_url="${GOPROXY%%,*}"
  export HTTPS_PROXY="${HTTPS_PROXY:-$proxy_url}"
  export HTTP_PROXY="${HTTP_PROXY:-$proxy_url}"
  export GOPROXY="https://proxy.golang.org,direct"
  echo "检测到 GOPROXY 为 HTTP 代理（$proxy_url），已切换为 HTTPS_PROXY + proxy.golang.org"
fi

# 本地回环请求必须绕过代理：127.0.0.1 在代理端会被当成代理机自身，健康检查/本地联调会卡死。
export NO_PROXY="${NO_PROXY:-127.0.0.1,localhost}"
export no_proxy="$NO_PROXY"

echo "预热 Go 依赖..."
(
  cd "$BACKEND_DIR"
  go mod download
)

echo "启动 Go 后端..."
(
  cd "$BACKEND_DIR"
  go run ./cmd/server
) &
BACKEND_PID=$!

echo "等待后端就绪..."
for _ in $(seq 1 60); do
  if curl -fsS "$BACKEND_URL" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! curl -fsS "$BACKEND_URL" >/dev/null 2>&1; then
  echo "后端启动失败或超时，未检测到健康检查接口: $BACKEND_URL"
  exit 1
fi

echo "启动 Vite 前端..."
echo "前端地址: http://127.0.0.1:5173"
echo "后端地址: http://127.0.0.1:3000"
echo "按 Ctrl+C 停止服务"

cd "$FRONTEND_DIR"
npm run dev
