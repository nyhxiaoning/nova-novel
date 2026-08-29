#!/bin/bash
set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_PORT="${NOVA_BACKEND_PORT:-8085}"
FRONTEND_PORT="${NOVA_FRONTEND_PORT:-5173}"
FRONTEND_URL="http://localhost:${FRONTEND_PORT}"
BACKEND_URL="http://localhost:${BACKEND_PORT}"

cd "${ROOT_DIR}"

MODE="${1:-all}"  # all | fe | be

case "$MODE" in
  fe|frontend)
    echo "==> Nova 前端开发服务启动"
    echo "  前端地址: ${FRONTEND_URL}"
    echo ""

    if ! command -v pnpm >/dev/null 2>&1; then
        echo "错误: 未找到 pnpm，请先安装 pnpm"
        exit 1
    fi

    if [ ! -d "web/node_modules" ]; then
        echo "==> 安装前端依赖"
        (cd web && pnpm install)
    fi

    echo "  按 Ctrl+C 停止服务"
    cd web && exec pnpm dev --port "${FRONTEND_PORT}"
    ;;

  be|backend)
    echo "==> Nova 后端开发服务启动"
    echo "  后端地址: ${BACKEND_URL}"
    echo ""

    echo "==> 拉取 Go 依赖"
    go mod tidy

    echo "  按 Ctrl+C 停止服务"
    exec go run ./cmd/nova --port "${BACKEND_PORT}" --no-open
    ;;

  all)
    echo "==> Nova 开发服务启动"
    echo "  前端地址: ${FRONTEND_URL}"
    echo "  后端地址: ${BACKEND_URL}"
    echo ""

    if ! command -v pnpm >/dev/null 2>&1; then
        echo "错误: 未找到 pnpm，请先安装 pnpm"
        exit 1
    fi

    if [ ! -d "web/node_modules" ]; then
        echo "==> 安装前端依赖"
        (cd web && pnpm install)
    fi

    echo "==> 拉取 Go 依赖"
    go mod tidy

    echo "==> 启动前后端"
    echo "  按 Ctrl+C 停止服务"
    echo ""

    exec go run ./cmd/nova --port "${BACKEND_PORT}" --dev --no-open
    ;;

  *)
    echo "用法: ./bootstrap.sh [all|fe|be]"
    echo "  all  - 启动前后端 (默认)"
    echo "  fe   - 仅启动前端 (Vite dev server)"
    echo "  be   - 仅启动后端 (Go server)"
    exit 1
    ;;
esac
