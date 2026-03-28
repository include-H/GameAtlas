#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

append_godebug_flag() {
  local current="${GODEBUG:-}"
  local flag="goindex=0"

  case ",$current," in
    *",$flag,"*)
      printf '%s' "$current"
      ;;
    ",," )
      printf '%s' "$flag"
      ;;
    * )
      printf '%s,%s' "$current" "$flag"
      ;;
  esac
}

export GODEBUG="$(append_godebug_flag)"

cd "$SCRIPT_DIR"

echo "运行后端测试..."
go test ./...

echo "运行后端 vet..."
go vet ./...
