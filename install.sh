#!/usr/bin/env bash
# ============================================================================
# ZMark 一键安装引导。装 git -> 拉取仓库 -> 执行 deploy.sh
# (deploy.sh 会自动安装 Go/Node、构建前后端、写 .env、常驻启动、放行端口)。
#
# 公开仓库一行命令:
#   DB_PASSWORD=数据库密码 bash -c "$(curl -fsSL https://raw.githubusercontent.com/mbfczzz/oneNav/main/install.sh)"
#
# 私有仓库一行命令(需 GitHub token,对该仓库有读权限):
#   export GH_TOKEN=你的token DB_PASSWORD=数据库密码 ADMIN_PASS=强密码
#   bash -c "$(curl -fsSL -H "Authorization: token $GH_TOKEN" -H 'Accept: application/vnd.github.raw' https://api.github.com/repos/mbfczzz/oneNav/contents/install.sh)"
#
# 透传给 deploy.sh 的环境变量同样有效:DB_PASSWORD PORT ADMIN_PASS DB_NAME ...
# ============================================================================
set -euo pipefail
OWNER_REPO="${OWNER_REPO:-mbfczzz/oneNav}"
DIR="${DIR:-oneNav}"
BRANCH="${BRANCH:-main}"

SUDO=""; [ "$(id -u)" = "0" ] || SUDO="sudo"

if ! command -v git >/dev/null 2>&1; then
  echo "==> 安装 git"
  if   command -v apt-get >/dev/null 2>&1; then $SUDO apt-get update -y && $SUDO apt-get install -y git
  elif command -v dnf     >/dev/null 2>&1; then $SUDO dnf install -y git
  elif command -v yum     >/dev/null 2>&1; then $SUDO yum install -y git
  else echo "✗ 请先手动安装 git"; exit 1; fi
fi

if [ -n "${GH_TOKEN:-}" ]; then
  URL="https://${GH_TOKEN}@github.com/${OWNER_REPO}.git"
else
  URL="https://github.com/${OWNER_REPO}.git"
fi

if [ -d "$DIR/.git" ]; then
  echo "==> 更新已有代码 ($DIR)"
  git -C "$DIR" remote set-url origin "$URL"
  git -C "$DIR" fetch --depth=1 origin "$BRANCH"
  git -C "$DIR" reset --hard "origin/$BRANCH"
else
  echo "==> 克隆仓库到 $DIR"
  git clone --depth=1 -b "$BRANCH" "$URL" "$DIR"
fi

# 不把 token 留在 .git/config 里
git -C "$DIR" remote set-url origin "https://github.com/${OWNER_REPO}.git" 2>/dev/null || true

cd "$DIR"
echo "==> 执行 deploy.sh"
exec bash deploy.sh
