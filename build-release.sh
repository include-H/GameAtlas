#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
RELEASE_ROOT="$ROOT_DIR/release"
EMBEDDED_WEB_DIR="$BACKEND_DIR/web/dist"

VERSION="${1:-$(date +%Y%m%d-%H%M%S)}"
PACKAGE_DIR="$RELEASE_ROOT/game-release-$VERSION"

check_dependency() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少依赖: $1"
    exit 1
  fi
}

cleanup_embedded_web() {
  rm -rf "$EMBEDDED_WEB_DIR"
  mkdir -p "$EMBEDDED_WEB_DIR"
  touch "$EMBEDDED_WEB_DIR/.gitkeep"
}

write_runtime_env() {
  local target="$1"
  cat > "$target" <<'EOF'
APP_ENV=production
HOST=0.0.0.0
PORT=3000

# 发布包相对路径
DB_PATH=data/db.db
STATIC_DIR=frontend/dist
ASSETS_DIR=data/gamelist
MIGRATIONS_DIR=migrations

# 游戏库根目录
ALLOWED_LIBRARY_ROOTS=ROM
PRIMARY_ROM_ROOT=ROM

# 可选代理
PROXY=
HTTP_PROXY=
HTTPS_PROXY=
STEAM_PROXY=

LOG_LEVEL=info
READ_HEADER_TIMEOUT=5s
SHUTDOWN_TIMEOUT=10s
EOF
}

check_dependency go
check_dependency npm

echo "清理旧发布目录..."
rm -rf "$PACKAGE_DIR"
mkdir -p "$PACKAGE_DIR"

echo "构建前端..."
(
  cd "$FRONTEND_DIR"
  npm run build
)

echo "准备内嵌前端资源..."
cleanup_embedded_web
cp -R "$FRONTEND_DIR/dist/." "$EMBEDDED_WEB_DIR/"

echo "构建后端..."
(
  cd "$BACKEND_DIR"
  go build -trimpath -ldflags="-s -w" -o "$PACKAGE_DIR/game-server" ./cmd/server
)

echo "准备运行目录..."
mkdir -p \
  "$PACKAGE_DIR/data/gamelist" \
  "$PACKAGE_DIR/ROM"

echo "写入运行配置..."
write_runtime_env "$PACKAGE_DIR/.env"
cp "$PACKAGE_DIR/.env" "$PACKAGE_DIR/.env.example"

echo "复制参考文档..."
cp "$BACKEND_DIR/README.md" "$PACKAGE_DIR/README-backend.md"

cat > "$PACKAGE_DIR/start.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"
exec ./game-server
EOF
chmod +x "$PACKAGE_DIR/start.sh"

cleanup_embedded_web

echo
echo "发布包已生成:"
echo "  $PACKAGE_DIR"
echo
echo "目录结构:"
echo "  game-server"
echo "  .env"
echo "  data/db.db        # 首次运行后自动创建"
echo "  data/gamelist"
echo "  ROM"
echo
echo "启动方式:"
echo "  cd \"$PACKAGE_DIR\""
echo "  ./start.sh"
