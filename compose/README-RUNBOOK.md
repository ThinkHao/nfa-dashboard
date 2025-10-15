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
