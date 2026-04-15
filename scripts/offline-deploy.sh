#!/usr/bin/env bash
set -euo pipefail

# offline-deploy.sh
# 用于离线一键升级（Linux amd64）。
# 功能：预检 -> 校验 -> 镜像导入 -> .env 合并校验 -> 执行 DB 迁移/预检 -> compose 启动 -> 健康检查 -> 写回滚点（仅保留 2 个版本）。

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUNDLE_ROOT="${SCRIPT_DIR}/.."
COMPOSE_DIR="${BUNDLE_ROOT}/compose"
SCRIPTS_DIR="${BUNDLE_ROOT}/scripts"
RELEASES_DIR="${BUNDLE_ROOT}/releases"
IMAGES_TAR="${BUNDLE_ROOT}/images-amd64.tar.gz"
COMPOSE_FILE="${COMPOSE_DIR}/docker-compose.offline.yml"
ENV_EXAMPLE="${COMPOSE_DIR}/.env.example"
ENV_FILE="${COMPOSE_DIR}/.env"
BUNDLE_META="${BUNDLE_ROOT}/bundle.yaml"
SHA_FILE="${BUNDLE_ROOT}/sha256sums.txt"
MIGRATIONS_DIR="${BUNDLE_ROOT}/sql/migrations"

log() { echo "[offline-deploy] $*"; }
warn() { echo "[offline-deploy][WARN] $*" >&2; }
err() { echo "[offline-deploy][ERROR] $*" >&2; }

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "缺少命令: $1"
    exit 1
  fi
}

MYSQL_CMD=()
init_mysql_cmd() {
  local db_host="$1"
  local db_port="$2"
  local db_user="$3"
  local db_pass="$4"
  local db_name="$5"
  MYSQL_CMD=(mysql --protocol=TCP -h "$db_host" -P "$db_port" -u "$db_user" --default-character-set=utf8mb4 "$db_name")
  if [[ -n "$db_pass" ]]; then
    MYSQL_CMD+=("-p${db_pass}")
  fi
}

mysql_query() {
  local sql="$1"
  "${MYSQL_CMD[@]}" -Nse "$sql"
}

mysql_exec_file() {
  local file="$1"
  "${MYSQL_CMD[@]}" < "$file"
}

run_db_migrations() {
  need_cmd mysql
  if [[ ! -d "$MIGRATIONS_DIR" ]]; then
    warn "未找到迁移目录: $MIGRATIONS_DIR，跳过迁移执行"
    return
  fi

  local db_host db_port db_user db_pass db_name
  db_host="$(grep -E '^DB_HOST=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  db_port="$(grep -E '^DB_PORT=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  db_user="$(grep -E '^DB_USER=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  db_pass="$(grep -E '^DB_PASS=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  db_name="$(grep -E '^DB_NAME=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"

  db_port="${db_port:-3306}"
  if [[ -z "$db_host" || -z "$db_user" || -z "$db_name" ]]; then
    err "数据库连接信息缺失（至少需要 DB_HOST/DB_USER/DB_NAME）"
    exit 1
  fi

  init_mysql_cmd "$db_host" "$db_port" "$db_user" "$db_pass" "$db_name"
  log "检查数据库连接: ${db_user}@${db_host}:${db_port}/${db_name}"
  mysql_query "SELECT 1;" >/dev/null

  mysql_query "CREATE TABLE IF NOT EXISTS schema_migrations (
version VARCHAR(64) NOT NULL PRIMARY KEY,
filename VARCHAR(255) NOT NULL,
applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"

  mapfile -t migration_files < <(find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name '[0-9][0-9][0-9]_*.sql' -printf '%f\n' | sort -V)
  if (( ${#migration_files[@]} == 0 )); then
    warn "迁移目录无可执行 SQL（规则: NNN_*.sql），跳过迁移执行"
    return
  fi

  declare -A versions_seen=()
  for filename in "${migration_files[@]}"; do
    local version="${filename%%_*}"
    if [[ -n "${versions_seen[$version]+x}" ]]; then
      err "检测到重复迁移版本号: ${version}（${versions_seen[$version]} 与 ${filename}）"
      exit 1
    fi
    versions_seen["$version"]="$filename"
  done

  log "开始执行增量迁移（自动扫描 NNN_*.sql）"
  for filename in "${migration_files[@]}"; do
    local version="${filename%%_*}"
    local migration_file="${MIGRATIONS_DIR}/${filename}"
    local applied
    applied="$(mysql_query "SELECT COUNT(*) FROM schema_migrations WHERE version='${version}';")"
    if [[ "$applied" == "0" ]]; then
      log "执行迁移 ${version} -> ${filename}"
      mysql_exec_file "$migration_file"
      mysql_query "INSERT INTO schema_migrations(version, filename) VALUES ('${version}', '${filename}');"
    else
      log "跳过已执行迁移 ${version} -> ${filename}"
    fi
  done
  log "增量迁移执行完成"
}

assert_db_schema() {
  log "执行关键 schema 预检"
  local checks=(
    "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly';|settlement_customer_monthly 表缺失"
    "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='traffic_scope_rules';|traffic_scope_rules 表缺失"
    "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='traffic_scope_rule_groups';|traffic_scope_rule_groups 表缺失"
    "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='traffic_scope_rule_conditions';|traffic_scope_rule_conditions 表缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='increment_start_at';|rate_customer.increment_start_at 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='stock_ratio';|rate_customer.stock_ratio 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='increment_ratio';|rate_customer.increment_ratio 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='daily_increment_value';|rate_customer.daily_increment_value 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer' AND COLUMN_NAME='stock_ratio';|settlement_customer.stock_ratio 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer' AND COLUMN_NAME='increment_ratio';|settlement_customer.increment_ratio 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer' AND COLUMN_NAME='daily_increment_value';|settlement_customer.daily_increment_value 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly' AND COLUMN_NAME='stock_ratio';|settlement_customer_monthly.stock_ratio 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly' AND COLUMN_NAME='increment_ratio';|settlement_customer_monthly.increment_ratio 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly' AND COLUMN_NAME='daily_increment_value';|settlement_customer_monthly.daily_increment_value 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='nfa_settlement_task' AND COLUMN_NAME='task_stage';|nfa_settlement_task.task_stage 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='nfa_settlement_task' AND COLUMN_NAME='total_count';|nfa_settlement_task.total_count 列缺失"
    "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='nfa_settlement_task' AND COLUMN_NAME='task_meta';|nfa_settlement_task.task_meta 列缺失"
  )
  for item in "${checks[@]}"; do
    local sql="${item%%|*}"
    local msg="${item#*|}"
    local cnt
    cnt="$(mysql_query "$sql")"
    if [[ "$cnt" == "0" ]]; then
      err "$msg"
      err "schema 预检失败，停止启动新版本"
      exit 1
    fi
  done
  log "schema 预检通过"
}

# 1) 预检
need_cmd docker
if ! docker compose version >/dev/null 2>&1; then
  err "需要 docker compose v2，请安装 Docker 20+ 并启用 compose v2"
  exit 1
fi

# 2) 校验文件完整性（可选）
if [[ -f "$SHA_FILE" ]]; then
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$BUNDLE_ROOT" && sha256sum -c "$SHA_FILE") || {
      err "文件校验失败，请重新下载离线包"
      exit 1
    }
    log "SHA256 校验通过"
  else
    warn "未找到 sha256sum，跳过校验"
  fi
else
  warn "未找到 sha256sums.txt，跳过校验"
fi

# 3) 解析版本（从 bundle.yaml 可选读取）
VERSION=""
if [[ -f "$BUNDLE_META" ]]; then
  VERSION=$(grep -E "^\s*release:\s*" "$BUNDLE_META" | head -n1 | awk -F':' '{gsub(/ /,"",$2); print $2}') || true
fi
if [[ -z "$VERSION" ]]; then
  # 兜底：从 .env.example 的 IMAGE_TAG 读取
  if [[ -f "$ENV_EXAMPLE" ]]; then
    VERSION=$(grep -E "^IMAGE_TAG=" "$ENV_EXAMPLE" | head -n1 | cut -d'=' -f2) || true
  fi
fi
if [[ -z "$VERSION" ]]; then
  warn "未能解析版本号（bundle.yaml 或 .env.example），继续执行但无法写入回滚点"
fi
log "当前升级目标版本: ${VERSION:-unknown}"

# 4) 导入镜像
if [[ ! -f "$IMAGES_TAR" ]]; then
  err "未找到镜像包: $IMAGES_TAR"
  exit 1
fi
log "导入镜像: $IMAGES_TAR"
docker load -i "$IMAGES_TAR"

# 5) .env 合并与校验
if [[ ! -f "$ENV_EXAMPLE" ]]; then
  err "缺少 $ENV_EXAMPLE"
  exit 1
fi
mkdir -p "$COMPOSE_DIR"

TMP_ENV_NEW="${ENV_FILE}.new"
TMP_ENV_OLD="${ENV_FILE}.old"

if [[ -f "$ENV_FILE" ]]; then
  cp "$ENV_FILE" "$TMP_ENV_OLD"
else
  : > "$TMP_ENV_OLD"
fi

# 读取旧 env 到 map
# shellcheck disable=SC2034
declare -A OLD
while IFS= read -r line; do
  [[ -z "$line" || "$line" =~ ^# ]] && continue
  if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
    k="${BASH_REMATCH[1]}"; v="${BASH_REMATCH[2]}"
    OLD["$k"]="$v"
  fi
done < "$TMP_ENV_OLD"

# 合并：以 .env.example 为基准，旧值优先，新增键补 example
{
  echo "# Generated by offline-deploy.sh at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
  while IFS= read -r line; do
    if [[ -z "$line" ]]; then echo ""; continue; fi
    if [[ "$line" =~ ^# ]]; then echo "$line"; continue; fi
    if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
      k="${BASH_REMATCH[1]}"; defv="${BASH_REMATCH[2]}"
      if [[ -n "${OLD[$k]+x}" ]]; then
        echo "$k=${OLD[$k]}"
      else
        echo "$k=$defv"
      fi
    else
      echo "$line"
    fi
  done < "$ENV_EXAMPLE"

  # 追加旧 env 中未在 example 出现的键（保留）
  echo ""
  echo "# Preserved keys from previous .env (not in .env.example)"
  while IFS= read -r line; do
    [[ -z "$line" || "$line" =~ ^# ]] && continue
    if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
      k="${BASH_REMATCH[1]}"; v="${BASH_REMATCH[2]}"
      if ! grep -qE "^${k}=" "$ENV_EXAMPLE"; then
        echo "$k=$v"
      fi
    fi
  done < "$TMP_ENV_OLD"
} > "$TMP_ENV_NEW"

# 强制写入 IMAGE_TAG=VERSION（如能解析）
if [[ -n "$VERSION" ]]; then
  if grep -qE '^IMAGE_TAG=' "$TMP_ENV_NEW"; then
    sed -i.bak "s/^IMAGE_TAG=.*/IMAGE_TAG=${VERSION}/" "$TMP_ENV_NEW" && rm -f "$TMP_ENV_NEW.bak"
  else
    echo "IMAGE_TAG=${VERSION}" >> "$TMP_ENV_NEW"
  fi
fi

# 校验必填项（以 .env.example 为准）
required_keys=(DB_HOST DB_USER DB_PASS DB_NAME AUTH_SECRET)
placeholders=("your.mysql.host" "your_password" "change_me_in_prod")

missing=()
while IFS= read -r line; do
  [[ -z "$line" || "$line" =~ ^# ]] && continue
  if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
    k="${BASH_REMATCH[1]}"; v="${BASH_REMATCH[2]}"
    for rk in "${required_keys[@]}"; do
      if [[ "$k" == "$rk" ]]; then
        vv="${v%\r}"
        if [[ -z "$vv" ]]; then
          missing+=("$k")
        else
          for p in "${placeholders[@]}"; do
            if [[ "$vv" == "$p" ]]; then
              missing+=("$k")
              break
            fi
          done
        fi
      fi
    done
  fi
done < "$TMP_ENV_NEW"

if (( ${#missing[@]} > 0 )); then
  err "必填项缺失或为占位值: ${missing[*]}"
  err "请编辑 ${TMP_ENV_NEW} 填写正确后重试"
  exit 1
fi

# 备份旧 env，替换为新 env
if [[ -f "$ENV_FILE" ]]; then
  cp "$ENV_FILE" "${ENV_FILE}.bak-$(date +'%Y%m%d%H%M%S')"
fi
mv "$TMP_ENV_NEW" "$ENV_FILE"
rm -f "$TMP_ENV_OLD"

# 6) 数据库迁移与 schema 预检（在启动新版本前执行，失败则阻断）
run_db_migrations
assert_db_schema

# 7) 启动 compose
log "启动服务 (docker compose)"
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --remove-orphans

# 8) 健康检查
APP_PORT=$(grep -E '^APP_PORT=' "$ENV_FILE" | head -n1 | cut -d'=' -f2)
APP_PORT=${APP_PORT:-8081}
HEALTH_URL="http://127.0.0.1:${APP_PORT}/health"
log "等待健康检查: $HEALTH_URL"

ok=false
for i in $(seq 1 60); do
  if command -v curl >/dev/null 2>&1; then
    if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then ok=true; break; fi
  else
    if wget -qO- "$HEALTH_URL" >/dev/null 2>&1; then ok=true; break; fi
  fi
  sleep 2
  echo -n "."
done

echo ""
if [[ "$ok" != true ]]; then
  err "健康检查失败"
  # 自动回滚到上一个版本（如果有）
  if [[ -d "$RELEASES_DIR" ]]; then
    # 选择最新的除当前之外的一个版本目录
    prev="$(ls -1t "$RELEASES_DIR" 2>/dev/null | grep -v "^${VERSION}$" | head -n1 || true)"
    if [[ -n "$prev" && -d "$RELEASES_DIR/$prev" ]]; then
      warn "尝试自动回滚到版本: $prev"
      cp "$RELEASES_DIR/$prev/.env" "$ENV_FILE"
      docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --remove-orphans || true
      err "已触发回滚，请检查日志后重试升级"
      exit 2
    fi
  fi
  err "无可用回滚版本，请检查服务配置后重试"
  exit 2
fi
log "健康检查通过"

# 9) 写回滚点（仅保留 2 个版本）
if [[ -n "$VERSION" ]]; then
  mkdir -p "$RELEASES_DIR/$VERSION"
  cp "$COMPOSE_FILE" "$RELEASES_DIR/$VERSION/compose.yml"
  cp "$ENV_FILE" "$RELEASES_DIR/$VERSION/.env"
  # 仅保留 2 个版本
  count=$(ls -1 "$RELEASES_DIR" 2>/dev/null | wc -l | awk '{print $1}')
  if [[ "$count" -gt 2 ]]; then
    ls -1t "$RELEASES_DIR" | tail -n +3 | xargs -I{} rm -rf "$RELEASES_DIR/{}"
  fi
  log "已保存回滚点: $RELEASES_DIR/$VERSION (仅保留最近2个)"
else
  warn "未能确定版本号，未保存回滚点"
fi

log "部署完成"
