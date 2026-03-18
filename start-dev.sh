#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:3000/api/health}"
BACKEND_ENV_FILE="$BACKEND_DIR/.env"
DEFAULT_DEV_DB="$BACKEND_DIR/data/app.db"
DEFAULT_DEV_ASSETS_DIR="$BACKEND_DIR/data/gamelist"

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

echo "预热 Go 依赖..."
(
  cd "$BACKEND_DIR"
  go mod download
)

echo "启动 Go 后端..."
(
  cd "$BACKEND_DIR"
  if [[ ! -f "$BACKEND_ENV_FILE" ]] && [[ -f "$DEFAULT_DEV_DB" ]] && [[ -z "${DB_PATH:-}" ]]; then
    export DB_PATH="data/app.db"
    echo "未检测到 backend/.env，开发环境默认使用示例数据库: $DB_PATH"
  fi
  if [[ ! -f "$BACKEND_ENV_FILE" ]] && [[ -d "$DEFAULT_DEV_ASSETS_DIR" ]] && [[ -z "${ASSETS_DIR:-}" ]]; then
    export ASSETS_DIR="data/gamelist"
    echo "未检测到 backend/.env，开发环境默认使用示例素材目录: $ASSETS_DIR"
  fi
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
