#!/usr/bin/env bash
# ============================================================================
# ZMark 一键部署脚本 —— 在与 MySQL 同一台 Linux 服务器上运行。
#
# 全自动:缺 Go / Node / git 会自动安装;构建前端+后端;写 backend-go/.env;
#         以 systemd(root)或 nohup 常驻启动;放行系统防火墙端口;健康检查。
#         后端单进程同时服务前端与 /api(同源,无需 Nginx)。
#
# 用法(代码已在服务器上,在仓库根目录执行):
#   DB_PASSWORD=你的MySQL密码 bash deploy.sh
#   bash deploy.sh                              # 交互式输入密码
#   PORT=80 ADMIN_PASS=强密码 DB_PASSWORD=xxx bash deploy.sh
#   SKIP_DEPS=1 ... bash deploy.sh              # 跳过自动装依赖
# 可覆盖:DB_HOST DB_PORT DB_NAME DB_USERNAME DB_PASSWORD PORT
#         ADMIN_USER ADMIN_PASS TABLE_PREFIX AUTH_TTL_HOURS GO_VERSION SKIP_DEPS
# ============================================================================
set -euo pipefail
cd "$(dirname "$0")"
ROOT="$(pwd)"
APP_NAME="zmark"
ENV_FILE="backend-go/.env"
GO_VERSION="${GO_VERSION:-1.22.5}"

# ---------- 配置(同机部署默认连本地 MySQL)----------
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_NAME="${DB_NAME:-beadforge}"
DB_USERNAME="${DB_USERNAME:-root}"
PORT="${PORT:-8080}"
ADMIN_USER="${ADMIN_USER:-admin}"
TABLE_PREFIX="${TABLE_PREFIX:-onenav_}"
AUTH_TTL_HOURS="${AUTH_TTL_HOURS:-168}"

read_env() { [ -f "$ENV_FILE" ] && sed -n "s/^$1=//p" "$ENV_FILE" | head -1 || true; }
DB_PASSWORD="${DB_PASSWORD:-$(read_env DB_PASSWORD)}"
ADMIN_PASS="${ADMIN_PASS:-$(read_env ADMIN_PASS)}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"

# ---------- sudo / 包管理器 ----------
SUDO=""; [ "$(id -u)" = "0" ] || SUDO="sudo"
PKG=""
for m in apt-get dnf yum; do command -v "$m" >/dev/null 2>&1 && { PKG="$m"; break; }; done
arch() { case "$(uname -m)" in x86_64|amd64) echo amd64;; aarch64|arm64) echo arm64;; *) echo amd64;; esac; }
ver_ge() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]; }  # ver_ge A B -> A>=B

install_base() {
  local need=()
  for c in git curl tar; do command -v "$c" >/dev/null 2>&1 || need+=("$c"); done
  [ "${#need[@]}" -eq 0 ] && return
  echo "==> 安装基础工具: ${need[*]}"
  case "$PKG" in
    apt-get) $SUDO apt-get update -y && $SUDO apt-get install -y "${need[@]}" ;;
    dnf) $SUDO dnf install -y "${need[@]}" ;;
    yum) $SUDO yum install -y "${need[@]}" ;;
    *) echo "✗ 未识别包管理器,请手动安装: ${need[*]}"; exit 1 ;;
  esac
}
install_node() {
  if command -v node >/dev/null 2>&1; then
    local maj; maj="$(node -v | sed 's/v//; s/\..*//')"
    [ "${maj:-0}" -ge 18 ] && return
  fi
  echo "==> 安装 Node.js 20"
  case "$PKG" in
    apt-get) curl -fsSL https://deb.nodesource.com/setup_20.x | $SUDO -E bash - && $SUDO apt-get install -y nodejs ;;
    dnf) curl -fsSL https://rpm.nodesource.com/setup_20.x | $SUDO -E bash - && $SUDO dnf install -y nodejs ;;
    yum) curl -fsSL https://rpm.nodesource.com/setup_20.x | $SUDO -E bash - && $SUDO yum install -y nodejs ;;
    *) echo "✗ 请手动安装 Node 18+"; exit 1 ;;
  esac
}
install_go() {
  if command -v go >/dev/null 2>&1; then
    local v; v="$(go version | awk '{print $3}' | sed 's/go//')"
    ver_ge "$v" 1.22 && return
  fi
  echo "==> 安装 Go ${GO_VERSION}"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-$(arch).tar.gz" -o /tmp/go.tgz
  $SUDO rm -rf /usr/local/go && $SUDO tar -C /usr/local -xzf /tmp/go.tgz
  export PATH="$PATH:/usr/local/go/bin"
  grep -q '/usr/local/go/bin' /etc/profile 2>/dev/null || echo 'export PATH=$PATH:/usr/local/go/bin' | $SUDO tee -a /etc/profile >/dev/null
}
open_firewall() {
  if command -v firewall-cmd >/dev/null 2>&1 && $SUDO firewall-cmd --state >/dev/null 2>&1; then
    $SUDO firewall-cmd --permanent --add-port="${PORT}"/tcp >/dev/null 2>&1 || true
    $SUDO firewall-cmd --reload >/dev/null 2>&1 || true
    echo "==> firewalld 已放行 ${PORT}/tcp"
  elif command -v ufw >/dev/null 2>&1 && $SUDO ufw status 2>/dev/null | grep -qi active; then
    $SUDO ufw allow "${PORT}"/tcp >/dev/null 2>&1 || true
    echo "==> ufw 已放行 ${PORT}/tcp"
  fi
}

# ---------- 自动装依赖 ----------
if [ -z "${SKIP_DEPS:-}" ]; then
  echo "==> [0/5] 检查并安装依赖 (Go/Node/git)"
  install_base
  install_node
  install_go
fi
command -v go   >/dev/null 2>&1 || { echo "✗ 未找到 go,请手动安装 Go 1.22+"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "✗ 未找到 node,请手动安装 Node 18+"; exit 1; }
command -v npm  >/dev/null 2>&1 || { echo "✗ 未找到 npm"; exit 1; }

# ---------- 数据库密码 ----------
if [ -z "$DB_PASSWORD" ]; then
  read -rsp "请输入 MySQL 密码 (DB_PASSWORD): " DB_PASSWORD; echo
fi
[ -z "$DB_PASSWORD" ] && { echo "✗ DB_PASSWORD 不能为空"; exit 1; }

echo "==> [1/5] 构建前端 (VITE_API_BASE=/api)"
if [ -f package-lock.json ]; then npm ci || npm install; else npm install; fi
VITE_API_BASE=/api npm run build

echo "==> [2/5] 构建后端 (Go)"
( cd backend-go && go build -o "${APP_NAME}-server" . )
BIN="$ROOT/backend-go/${APP_NAME}-server"

echo "==> [3/5] 写入 $ENV_FILE"
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

echo "==> [4/5] 启动服务"
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
  RUNNER="systemd(服务名 ${APP_NAME};日志 journalctl -u ${APP_NAME} -f)"
else
  pkill -f "${APP_NAME}-server" 2>/dev/null || true
  ( cd backend-go && nohup "$BIN" >"$ROOT/zmark.log" 2>&1 & )
  RUNNER="nohup(日志 $ROOT/zmark.log)"
fi

echo "==> [5/5] 放行端口并健康检查"
open_firewall || true
ok=""
for i in $(seq 1 15); do
  curl -fsS "http://127.0.0.1:${PORT}/api/health" >/dev/null 2>&1 && { ok=1; break; }
  sleep 1
done

IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
echo
if [ -n "$ok" ]; then echo "✅ 部署成功 · ${RUNNER}"; else echo "⚠️  已启动但健康检查未过,请查看日志 · ${RUNNER}"; fi
echo "   访问: http://${IP:-<服务器IP>}:${PORT}/    管理员: ${ADMIN_USER} / (你设置的密码)"
echo "   ⚠️ 云服务器还需在【安全组】入方向放行 TCP ${PORT}(脚本只能开系统防火墙,改不了云控制台)。"
echo "   80 端口: PORT=80 重新运行;HTTPS/域名可在前面挂 Nginx。"
