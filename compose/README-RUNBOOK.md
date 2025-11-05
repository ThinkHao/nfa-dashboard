# NFA Dashboard - Docker Compose Runbook

## 前置条件
- 已备份生产 MySQL 5.7 数据库（结构+数据）
- 服务器安装 Docker 20+ 与 docker-compose v2

## 一、数据库安装/迁移（生产手工一次性执行）
1. 登录生产数据库
2. 执行 `sql/dist/install_full.sql`
   - 该脚本适配 MySQL 5.7：
     - 新表使用 CREATE TABLE IF NOT EXISTS
     - 列/索引变更通过 information_schema 判断 + 动态 SQL 执行
     - 权限种子与授权使用 INSERT IGNORE/ON DUPLICATE KEY UPDATE

## 二、准备环境变量
复制 `compose/.env.example` 为 `.env` 并填写：
```
DB_HOST=...
DB_PORT=3306
DB_USER=...
DB_PASS=...
DB_NAME=nfa
APP_PORT=8081
FRONTEND_PORT=8080
```

## 三、启动服务
在 `compose/` 目录下：
```
docker compose --env-file .env up -d --build
```
- 首次会构建镜像并启动前后端
- Nginx 将 `/api` 反代到 `backend:8081`
- 后端健康检查：`GET /health`

## 四、验证
- 访问前端：http://<host>:${FRONTEND_PORT}
- 验证后端健康：http://<host>:${APP_PORT}/health 返回 {"status":"ok"}
- 登录系统，验证结算结果、角色权限等功能

## 五、可选：开发环境内置 MySQL（非生产）
```
docker compose --env-file .env --profile with-db up -d
```
- 数据持久化卷：`mysql-data`（宿主机命名卷：nfa_mysql_data）

## 六、回滚建议
- 镜像回滚：切换回上一个 tag 并重启 compose
- 数据回滚：使用执行 install_full.sql 前的数据库备份

## 七、离线部署（无网络环境）

适用于无法联网拉取镜像的环境，使用离线包内置镜像与脚本进行一键升级，保留最近 2 个版本。

- 约束与前置条件
  - 仅支持 Linux amd64
  - 已安装 Docker 20+ 与 docker compose v2
  - 使用外置 MySQL 5.7（离线方案不包含数据库容器）
  - 后端健康检查 URL：`GET /health`

- .env 准备与继承
  - 离线脚本仅从当前包的 `compose/.env` 合并生成新配置；不会自动读取包外目录
  - 建议做法：
    - 覆盖升级：保留上次的 `compose/.env`；或
    - 将旧版本的 `compose/.env` 复制到新包的 `compose/.env`
  - 合并规则：以 `compose/.env.example` 为基准，旧键覆盖默认值，example 中不存在的旧键将被原样保留；并强制写入 `IMAGE_TAG=<离线包版本>`

- 升级执行
  ```bash
  cd scripts
  chmod +x offline-deploy.sh offline-rollback.sh
  ./offline-deploy.sh
  ```
  - 流程：校验 → 导入镜像（images-amd64.tar.gz） → 合并/校验 .env →
    `docker compose -f compose/docker-compose.offline.yml --env-file compose/.env up -d` → 健康检查

- 验证
  - 前端：http://<host>:${FRONTEND_PORT}（默认 8080）
  - 健康检查：http://<host>:${APP_PORT}/health（默认 8081），返回 `{ "status": "ok" }`

- 回滚
  - 手动回滚：`cd scripts && ./offline-rollback.sh`
  - 升级失败时脚本会尝试自动回滚到 `releases/` 中的上一个版本

- 常见问题（FAQ）
  - 旧 `.env` 不在当前包目录，如何继承？
    - 将旧版本 `compose/.env` 复制到新包 `compose/.env` 后再执行脚本，脚本会自动合并
  - `.env` 的镜像标签如何确定？
    - 脚本会从 `bundle.yaml` 或 `.env.example` 解析版本并写入 `IMAGE_TAG`
