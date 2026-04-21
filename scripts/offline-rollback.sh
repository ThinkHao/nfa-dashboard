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

bool_true() {
  local v="${1:-}"
  case "${v,,}" in
    1|true|yes|on) return 0 ;;
    *) return 1 ;;
  esac
}

resolve_compose_path() {
  local p="$1"
  if [[ "$p" = /* ]]; then
    echo "$p"
  else
    echo "${COMPOSE_DIR}/${p#./}"
  fi
}

generate_self_signed_cert() {
  local cert_file="$1"
  local key_file="$2"
  local cert_cn="$3"
  local cert_sans="$4"

  if openssl req -x509 -newkey rsa:2048 -sha256 -days 365 -nodes \
    -keyout "$key_file" \
    -out "$cert_file" \
    -subj "/CN=${cert_cn}" \
    -addext "subjectAltName=${cert_sans}" >/dev/null 2>&1; then
    return 0
  fi

  local tmp_conf
  tmp_conf="$(mktemp)"
  {
    echo "[req]"
    echo "distinguished_name = dn"
    echo "x509_extensions = v3_req"
    echo "prompt = no"
    echo
    echo "[dn]"
    echo "CN = ${cert_cn}"
    echo
    echo "[v3_req]"
    echo "subjectAltName = @alt_names"
    echo
    echo "[alt_names]"
  } > "$tmp_conf"

  local dns_i=1
  local ip_i=1
  local token=""
  IFS=',' read -ra _sans_arr <<< "$cert_sans"
  for token in "${_sans_arr[@]}"; do
    token="$(echo "$token" | xargs)"
    [[ -z "$token" ]] && continue
    if [[ "$token" =~ ^DNS:(.+)$ ]]; then
      echo "DNS.${dns_i} = ${BASH_REMATCH[1]}" >> "$tmp_conf"
      dns_i=$((dns_i + 1))
    elif [[ "$token" =~ ^IP:(.+)$ ]]; then
      echo "IP.${ip_i} = ${BASH_REMATCH[1]}" >> "$tmp_conf"
      ip_i=$((ip_i + 1))
    fi
  done

  openssl req -x509 -newkey rsa:2048 -sha256 -days 365 -nodes \
    -keyout "$key_file" \
    -out "$cert_file" \
    -config "$tmp_conf" \
    -extensions v3_req >/dev/null 2>&1
  rm -f "$tmp_conf"
}

ensure_frontend_nginx_conf() {
  local enable_https cert_src key_src cert_cn cert_sans cert_dest key_dest
  enable_https="$(grep -E '^ENABLE_HTTPS=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  cert_src="$(grep -E '^SSL_CERT_PATH=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  key_src="$(grep -E '^SSL_KEY_PATH=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  cert_cn="$(grep -E '^SSL_CERT_CN=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  cert_sans="$(grep -E '^SSL_CERT_SANS=' "$ENV_FILE" | head -n1 | cut -d'=' -f2-)"
  cert_dest="${COMPOSE_DIR}/nginx/certs/server.crt"
  key_dest="${COMPOSE_DIR}/nginx/certs/server.key"

  mkdir -p "${COMPOSE_DIR}/nginx/certs"

  if bool_true "$enable_https"; then
    cert_cn="${cert_cn:-localhost}"
    cert_sans="${cert_sans:-DNS:localhost,IP:127.0.0.1}"
    if [[ -n "$cert_src" || -n "$key_src" ]]; then
      if [[ -z "$cert_src" || -z "$key_src" ]]; then
        err "ENABLE_HTTPS=true 时，SSL_CERT_PATH 与 SSL_KEY_PATH 要么都配置，要么都留空自动生成"
        exit 1
      fi
      cert_src="$(resolve_compose_path "$cert_src")"
      key_src="$(resolve_compose_path "$key_src")"
      if [[ ! -f "$cert_src" || ! -f "$key_src" ]]; then
        err "证书文件不存在: cert=$cert_src key=$key_src"
        exit 1
      fi
      cp "$cert_src" "$cert_dest"
      cp "$key_src" "$key_dest"
    elif [[ ! -f "$cert_dest" || ! -f "$key_dest" ]]; then
      need_cmd openssl
      generate_self_signed_cert "$cert_dest" "$key_dest" "$cert_cn" "$cert_sans"
    fi

    cat > "${COMPOSE_DIR}/nginx/nginx.conf" <<'EOF'
server {
  listen 80 ssl;
  server_name _;
  gzip on;
  gzip_types text/plain text/css application/json application/javascript application/xml+rss;
  root /usr/share/nginx/html;

  ssl_certificate     /etc/nginx/certs/server.crt;
  ssl_certificate_key /etc/nginx/certs/server.key;
  ssl_session_timeout 1d;
  ssl_session_cache shared:SSL:10m;
  ssl_session_tickets off;
  ssl_protocols TLSv1.2 TLSv1.3;
  ssl_ciphers HIGH:!aNULL:!MD5;
  ssl_prefer_server_ciphers off;

  add_header Strict-Transport-Security "max-age=31536000" always;
  add_header X-Content-Type-Options "nosniff" always;
  add_header X-Frame-Options "DENY" always;
  add_header Referrer-Policy "strict-origin-when-cross-origin" always;

  location / {
    try_files $uri $uri/ /index.html;
  }

  location /api/ {
    proxy_pass http://backend:8081;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto https;
    proxy_http_version 1.1;
  }

  location ~* \.(js|css|png|jpg|jpeg|gif|svg|ico|woff2?)$ {
    expires 7d;
    add_header Cache-Control "public";
    try_files $uri =404;
  }
}
EOF
  else
    cat > "${COMPOSE_DIR}/nginx/nginx.conf" <<'EOF'
server {
  listen 80;
  server_name _;
  gzip on;
  gzip_types text/plain text/css application/json application/javascript application/xml+rss;
  root /usr/share/nginx/html;

  location / {
    try_files $uri $uri/ /index.html;
  }

  location /api/ {
    proxy_pass http://backend:8081;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_http_version 1.1;
  }

  location ~* \.(js|css|png|jpg|jpeg|gif|svg|ico|woff2?)$ {
    expires 7d;
    add_header Cache-Control "public";
    try_files $uri =404;
  }
}
EOF
  fi
}

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

ensure_frontend_nginx_conf

# 回滚启动
log "启动回滚 compose"
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --remove-orphans

log "回滚完成：当前版本应为 ${PREV_VERSION}"
