# NFA Dashboard

NFA Dashboard 是一个用于网络流量分析和结算的仪表板系统。该系统包含前端和后端两部分，前端使用Vue 3 + TypeScript构建，后端使用Go语言开发。

本项目仅支持在Linux服务器上部署。

## 项目结构

```
nfa-dashboard/
├── backend/         # Go语言后端
├── frontend/        # Vue 3前端
├── docs/            # 文档
├── sql/             # SQL脚本
└── scripts/         # 部署脚本
```

## 开发环境

### SQL 迁移约定

- 增量迁移唯一入口是 `sql/migrations/`
- 不要在根目录 `sql/` 下新增 `NNN_*.sql`
- 新建迁移请使用：
  ```bash
  python scripts/sql_migration_guard.py create "your migration title"
  ```
- 若已经在别处草拟了 SQL，请接管到迁移目录：
  ```bash
  python scripts/sql_migration_guard.py adopt "your migration title" --source path/to/file.sql
  ```
- 提交前建议运行：
  ```bash
  python scripts/sql_migration_guard.py check
  ```
- 如果迁移包含 schema 变更，请在文件头声明 contract：
  ```sql
  -- contract: none
  -- contract: table=example_table
  -- contract: column=example_table.example_column
  ```
- 如果迁移引入运行时必需的表或列，请同步更新 `scripts/offline-deploy.sh` 中的 `assert_db_schema()`

### 前端开发

前端使用Vue 3 + TypeScript + Vite构建。

```bash
# 进入前端目录
cd frontend/frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 运行单元测试
npm run test:unit
```

### 后端开发

后端使用Go语言开发。

```bash
# 进入后端目录
cd backend

# 获取依赖
go mod download

# 运行后端服务
go run main.go

# 构建后端
go build -o nfa-dashboard-backend main.go
```

### HTTPS（自签名证书）

当服务映射到公网时，建议优先启用 HTTPS。后端已支持直接读取证书并以 TLS 启动。

1. 生成自签名证书（Linux/OpenSSL）：

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:2048 -sha256 -days 365 -nodes \
  -keyout certs/server.key \
  -out certs/server.crt \
  -subj "/CN=YOUR_SERVER_IP"
```

2. 修改 `backend/config/config.yaml`：

```yaml
server:
  tls:
    enabled: true
    cert_file: ./certs/server.crt
    key_file: ./certs/server.key
```

3. 启动后端：

```bash
cd backend
go run main.go
```

4. 首次访问时浏览器会提示证书不受信任（自签名证书正常现象）。

### 公网映射安全基线

- 强制使用复杂 `AUTH_SECRET`，不要保留默认值 `dev-secret-change-me`
- 不要把数据库明文密码提交到仓库，生产环境用环境变量注入
- 限制 CORS 来源：将 `server.security.cors.allowed_origins` 改为你的实际前端地址
- 启用登录限流（已内置，默认开启）
- 仅开放必要端口（建议前端/反向代理暴露 443，后端只对内网开放）
- 定期轮换管理员密码，至少 8 位并包含大小写、数字、符号

## 发布与部署

### 发布新版本

项目使用GitHub Actions自动构建和发布。当您推送一个新的标签（如`v1.0.0`）时，会自动触发构建流程并创建一个新的Release。

```bash
# 创建新标签
git tag v1.0.0

# 推送标签到GitHub
git push origin v1.0.0
```

构建完成后，GitHub Release页面会自动生成以下发布包：
- Linux (amd64/arm64): `nfa-dashboard-linux-amd64.tar.gz` / `nfa-dashboard-linux-arm64.tar.gz`

### 部署方法

#### Linux部署

1. 下载对应架构的压缩包（amd64或arm64）
2. 解压压缩包
   ```bash
   tar -xzf nfa-dashboard-linux-amd64.tar.gz
   ```
3. 运行部署脚本
   ```bash
   cd scripts
   chmod +x deploy.sh
   
   # 安装
   ./deploy.sh install --domain example.com --db-host localhost --db-user root --db-pass password
   
   # 更新
   ./deploy.sh update
   
   # 卸载
   ./deploy.sh uninstall
   ```

### 离线部署（一键升级，Linux amd64）

> 适用于无法联网拉取镜像的环境。离线包内置镜像与脚本，支持健康检查与自动回滚，仅保留最近 2 个版本。

- 约束与前置条件
  - 仅支持 Linux amd64
  - 服务器已安装 Docker 20+ 与 docker compose v2
  - 使用外置 MySQL 5.7（compose 离线方案不包含数据库容器）
  - 后端健康检查 URL：`GET /health`（脚本会等待通过）

- 离线包目录结构（解压后）
  ```
  nfa-dashboard/
  ├─ compose/
  │  ├─ docker-compose.offline.yml
  │  └─ .env.example
  ├─ scripts/
  │  ├─ offline-deploy.sh        # 升级/部署
  │  └─ offline-rollback.sh      # 回滚到上一个版本
  ├─ images-amd64.tar.gz         # 预打包镜像
  ├─ sha256sums.txt              # 可选：文件校验
  └─ releases/                   # 部署后生成的回滚点目录（最多保留 2 个版本）
  ```

- 准备环境变量（.env 来源说明）
  - 脚本仅从当前包内的 `compose/.env` 合并生成新配置；不会自动读取包外目录
  - 升级前请确保以下其一：
    - 同目录覆盖升级：保留上次的 `compose/.env`
    - 或将“旧版本”中的 `compose/.env` 复制到新包 `compose/.env`
  - 脚本会以 `compose/.env.example` 为基准合并旧值，保留 example 中没有的自定义键，并强制写入 `IMAGE_TAG=<离线包版本>`
  - 必填项：`DB_HOST`、`DB_USER`、`DB_PASS`、`DB_NAME`、`AUTH_SECRET`

- 执行升级
  ```bash
  cd scripts
  chmod +x offline-deploy.sh offline-rollback.sh
  ./offline-deploy.sh
  ```
  - 流程：校验 → 导入镜像 → 合并/校验 .env → `docker compose -f compose/docker-compose.offline.yml --env-file compose/.env up -d` → 健康检查
  - 成功后会把本次使用的 `compose/.env` 与 compose 文件保存到 `releases/<版本>/`，最多保留最近 2 个版本

- 验证
  - 前端：http://<host>:`${FRONTEND_PORT}`（默认 8080）
  - 健康检查：http://<host>:`${APP_PORT}`/health（默认 8081），返回 `{"status":"ok"}`

- 回滚
  - 手动回滚：
    ```bash
    cd scripts && ./offline-rollback.sh
    ```
  - 脚本还会在升级健康检查失败时尝试自动回滚到上一个版本（如果 `releases/` 中存在）

- 常见问题
  - 问：旧的 `.env` 不在当前包目录，如何继承？
    - 答：请将旧版本的 `compose/.env` 复制到新包的 `compose/.env` 后再执行脚本，脚本会自动合并到新 `.env`
  - 问：`.env` 中的镜像标签如何确定？
    - 答：脚本会从 `bundle.yaml` 或 `.env.example` 解析版本并写入 `IMAGE_TAG`，无需手工修改

部署脚本支持的参数：
- `--domain`: 网站域名（默认：localhost）
- `--db-host`: 数据库主机地址（默认：localhost）
- `--db-port`: 数据库端口（默认：3306）
- `--db-user`: 数据库用户名（默认：root）
- `--db-pass`: 数据库密码
- `--db-name`: 数据库名称（默认：nfa_v2）
- `--install-dir`: 安装目录（默认：/opt/nfa-dashboard）

### 配置管理

数据库连接信息等敏感配置可以通过以下方式管理：

1. **命令行参数**：在运行部署脚本时通过参数指定
   ```bash
   ./deploy.sh install --db-host mydb.example.com --db-user admin --db-pass secure_password
   ```

2. **环境变量**：使用环境变量配置（需要修改后端代码支持）
   ```bash
   # 复制环境变量模板
   cp scripts/env.template .env
   
   # 编辑环境变量
   vi .env
   
   # 加载环境变量并启动服务
   source .env && ./nfa-dashboard-backend
   ```

3. **配置文件**：直接编辑配置文件
   ```bash
   vi /opt/nfa-dashboard/backend/config/config.yaml
   ```

## 系统要求

### 服务器要求
- CPU: 2核或以上
- 内存: 2GB或以上
- 磁盘: 10GB可用空间

### 软件要求
- Linux服务器
- Nginx
- MySQL 5.7或以上

## 许可证

[待定]
