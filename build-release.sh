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

copy_optional_runtime_data() {
  local source_dir="$1"
  local target_dir="$2"
  local filenames=(
    "bg.jpg"
  )

  for filename in "${filenames[@]}"; do
    if [[ -f "$source_dir/$filename" ]]; then
      cp "$source_dir/$filename" "$target_dir/$filename"
    fi
  done
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
  # 兼容 Clash 类 HTTP 代理被误配成 GOPROXY 的环境（同 start-dev.sh）。
  if [[ "${GOPROXY:-}" == http://* ]]; then
    proxy_url="${GOPROXY%%,*}"
    export HTTPS_PROXY="${HTTPS_PROXY:-$proxy_url}"
    export HTTP_PROXY="${HTTP_PROXY:-$proxy_url}"
    export GOPROXY="https://proxy.golang.org,direct"
  fi
  go build -trimpath -ldflags="-s -w" -o "$PACKAGE_DIR/game-server" ./cmd/server
)

echo "准备运行目录..."
mkdir -p \
  "$PACKAGE_DIR/data/gamelist" \
  "$PACKAGE_DIR/ROM"

echo "复制可选自定义资源..."
copy_optional_runtime_data "$BACKEND_DIR/data" "$PACKAGE_DIR/data"

echo "复制参考文档..."
cp "$ROOT_DIR/README.md" "$PACKAGE_DIR/README.md"
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
echo "  data/db.db        # 首次运行后自动创建"
echo "  data/gamelist"
echo "  data/bg.jpg       # 如存在则作为共享背景"
echo "  ROM"
echo
echo "启动方式:"
echo "  cd \"$PACKAGE_DIR\""
echo "  ./start.sh"
