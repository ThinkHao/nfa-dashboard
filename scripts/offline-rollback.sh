#!/usr/bin/env bash
set -euo pipefail

# offline-rollback.sh
# 回滚到上一个成功版本（仅保留 2 个版本：当前与上一个）。

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUNDLE_ROOT="${SCRIPT_DIR}/.."
COMPOSE_DIR="${BUNDLE_ROOT}/compose"
COMPOSE_FILE="${COMPOSE_DIR}/docker-compose.offline.yml"
ENV_FILE="${COMPOSE_DIR}/.env"
RELEASES_DIR="${BUNDLE_ROOT}/releases"

log() { echo "[offline-rollback] $*"; }
err() { echo "[offline-rollback][ERROR] $*" >&2; }

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "缺少命令: $1"
    exit 1
  fi
}

need_cmd docker
if ! docker compose version >/dev/null 2>&1; then
  err "需要 docker compose v2"
  exit 1
fi

if [[ ! -d "$RELEASES_DIR" ]]; then
  err "未找到 releases 目录，无法回滚"
  exit 1
fi

# 从当前 .env 获取当前版本（IMAGE_TAG），用于排除
CURRENT_TAG=""
if [[ -f "$ENV_FILE" ]]; then
  CURRENT_TAG=$(grep -E '^IMAGE_TAG=' "$ENV_FILE" | head -n1 | cut -d'=' -f2 || true)
fi

# 找到上一个版本目录（按时间倒序，排除当前）
PREV_VERSION=""
while IFS= read -r ver; do
  [[ -z "$ver" ]] && continue
  if [[ -n "$CURRENT_TAG" && "$ver" == "$CURRENT_TAG" ]]; then
    continue
  fi
  PREV_VERSION="$ver"
  break
done < <(ls -1t "$RELEASES_DIR" 2>/dev/null)

if [[ -z "$PREV_VERSION" || ! -d "$RELEASES_DIR/$PREV_VERSION" ]]; then
  err "没有可回滚的历史版本"
  exit 1
fi

log "准备回滚到版本: $PREV_VERSION"

# 恢复该版本的 .env
if [[ ! -f "$RELEASES_DIR/$PREV_VERSION/.env" ]]; then
  err "历史版本缺少 .env: $RELEASES_DIR/$PREV_VERSION/.env"
  exit 1
fi
cp "$RELEASES_DIR/$PREV_VERSION/.env" "$ENV_FILE"

# 确保 .env 中 IMAGE_TAG 为目标版本（兜底）
if grep -qE '^IMAGE_TAG=' "$ENV_FILE"; then
  sed -i.bak "s/^IMAGE_TAG=.*/IMAGE_TAG=${PREV_VERSION}/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
else
  echo "IMAGE_TAG=${PREV_VERSION}" >> "$ENV_FILE"
fi

# 回滚启动
log "启动回滚 compose"
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --remove-orphans

log "回滚完成：当前版本应为 ${PREV_VERSION}"
