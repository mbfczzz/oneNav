#!/usr/bin/env bash
# ============================================================================
# 更新已部署的 ZMark:拉最新代码 -> 重建前端+后端 -> 重启服务。
# 复用现有 backend-go/.env(数据库/管理员配置不变,无需重新输入)。
# 用法(在仓库目录):bash update.sh
# ============================================================================
set -euo pipefail
cd "$(dirname "$0")"
APP_NAME="zmark"
BRANCH="${BRANCH:-main}"
SUDO=""; [ "$(id -u)" = "0" ] || SUDO="sudo"

[ -f backend-go/.env ] || { echo "✗ 未找到 backend-go/.env,请先用 deploy.sh 完成首次部署"; exit 1; }

echo "==> 拉取最新代码"
git fetch --depth=1 origin "$BRANCH"
git reset --hard "origin/$BRANCH"     # .env / node_modules / dist 已被 .gitignore,不受影响

echo "==> 重建前端"
if [ -f package-lock.json ]; then npm ci || npm install; else npm install; fi
VITE_API_BASE=/api npm run build

echo "==> 重建后端"
( cd backend-go && go build -o "${APP_NAME}-server" . )

echo "==> 重启服务"
if command -v systemctl >/dev/null 2>&1 && systemctl cat "$APP_NAME" >/dev/null 2>&1; then
  $SUDO systemctl restart "$APP_NAME"
  RUNNER="systemd"
else
  pkill -f "${APP_NAME}-server" 2>/dev/null || true
  ( cd backend-go && nohup "./${APP_NAME}-server" >../zmark.log 2>&1 & )
  RUNNER="nohup"
fi

PORT="$(sed -n 's/^PORT=//p' backend-go/.env | head -1)"; PORT="${PORT:-8080}"
sleep 2
if curl -fsS "http://127.0.0.1:${PORT}/api/health" >/dev/null 2>&1; then
  echo "✅ 更新完成并重启成功($RUNNER,端口 $PORT)"
else
  echo "⚠️ 已重启但健康检查未过,看日志:journalctl -u $APP_NAME -n 50"
fi
