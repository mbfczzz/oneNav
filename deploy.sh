#!/usr/bin/env bash
# ============================================================================
# ZMark 一键部署脚本 —— 在与 MySQL 同一台服务器上运行。
#
# 它会:1) 构建前端(SPA)  2) 构建 Go 后端  3) 写入 backend-go/.env
#       4) 以单进程方式启动(同一端口同时服务 SPA 和 /api,无需 Nginx)
#       优先用 systemd 常驻;无 systemd/非 root 时退回 nohup。
#
# 用法:
#   bash deploy.sh                      # 交互式输入数据库密码
#   DB_PASSWORD=xxx bash deploy.sh      # 非交互
#   PORT=80 DB_PASSWORD=xxx bash deploy.sh   # 用 80 端口(需 root)
# 可用环境变量覆盖:DB_HOST DB_PORT DB_NAME DB_USERNAME DB_PASSWORD
#                   PORT ADMIN_USER ADMIN_PASS TABLE_PREFIX AUTH_TTL_HOURS
# ============================================================================
set -euo pipefail
cd "$(dirname "$0")"
ROOT="$(pwd)"
APP_NAME="zmark"
ENV_FILE="backend-go/.env"

# ---------- 配置(同机部署默认连本地 MySQL)----------
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_NAME="${DB_NAME:-beadforge}"
DB_USERNAME="${DB_USERNAME:-root}"
PORT="${PORT:-8080}"
ADMIN_USER="${ADMIN_USER:-admin}"
TABLE_PREFIX="${TABLE_PREFIX:-onenav_}"
AUTH_TTL_HOURS="${AUTH_TTL_HOURS:-168}"

# 复用已有 .env 里的密码(便于重复运行),env 变量优先
read_env() { [ -f "$ENV_FILE" ] && sed -n "s/^$1=//p" "$ENV_FILE" | head -1 || true; }
DB_PASSWORD="${DB_PASSWORD:-$(read_env DB_PASSWORD)}"
ADMIN_PASS="${ADMIN_PASS:-$(read_env ADMIN_PASS)}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"

# ---------- 预检 ----------
need() { command -v "$1" >/dev/null 2>&1 || { echo "✗ 缺少依赖:$1,请先安装"; exit 1; }; }
need go
need node
need npm
need curl

if [ -z "$DB_PASSWORD" ]; then
  read -rsp "请输入 MySQL 密码 (DB_PASSWORD): " DB_PASSWORD
  echo
fi
[ -z "$DB_PASSWORD" ] && { echo "✗ DB_PASSWORD 不能为空"; exit 1; }

echo "==> [1/4] 构建前端 (VITE_API_BASE=/api)"
if [ -f package-lock.json ]; then npm ci || npm install; else npm install; fi
VITE_API_BASE=/api npm run build

echo "==> [2/4] 构建后端 (Go)"
( cd backend-go && go build -o "${APP_NAME}-server" . )
BIN="$ROOT/backend-go/${APP_NAME}-server"

echo "==> [3/4] 写入 $ENV_FILE"
umask 077
cat > "$ENV_FILE" <<EOF
DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_NAME=$DB_NAME
DB_USERNAME=$DB_USERNAME
DB_PASSWORD=$DB_PASSWORD
TABLE_PREFIX=$TABLE_PREFIX
PORT=$PORT
ADMIN_USER=$ADMIN_USER
ADMIN_PASS=$ADMIN_PASS
AUTH_TTL_HOURS=$AUTH_TTL_HOURS
WEB_DIR=$ROOT/dist
EOF

echo "==> [4/4] 启动服务"
if [ "$(id -u)" = "0" ] && command -v systemctl >/dev/null 2>&1; then
  cat > "/etc/systemd/system/${APP_NAME}.service" <<EOF
[Unit]
Description=ZMark navigation server
After=network.target mysql.service mysqld.service mariadb.service

[Service]
Type=simple
WorkingDirectory=$ROOT/backend-go
ExecStart=$BIN
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable "$APP_NAME" >/dev/null 2>&1 || true
  systemctl restart "$APP_NAME"
  RUNNER="systemd (服务名 ${APP_NAME});日志:journalctl -u ${APP_NAME} -f"
else
  pkill -f "${APP_NAME}-server" 2>/dev/null || true
  ( cd backend-go && nohup "$BIN" >"$ROOT/zmark.log" 2>&1 & )
  RUNNER="nohup;日志:$ROOT/zmark.log"
fi

# ---------- 健康检查 ----------
ok=""
for i in $(seq 1 15); do
  if curl -fsS "http://127.0.0.1:${PORT}/api/health" >/dev/null 2>&1; then ok=1; break; fi
  sleep 1
done

IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
echo
if [ -n "$ok" ]; then echo "✅ 部署成功(${RUNNER})"; else echo "⚠️  服务已启动但健康检查未通过,请查看日志(${RUNNER})"; fi
echo "   访问地址: http://${IP:-<服务器IP>}:${PORT}/"
echo "   管理员:   ${ADMIN_USER} / (你设置的密码)"
echo "   说明:    单进程同时提供前端与 /api;如需 80 端口,用 PORT=80 并以 root 运行,或在前面挂 Nginx/HTTPS。"
