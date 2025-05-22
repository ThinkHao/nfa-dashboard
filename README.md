# NFA Dashboard

NFA Dashboard 是一个用于网络流量分析和结算的仪表板系统。该系统包含前端和后端两部分，前端使用Vue 3 + TypeScript构建，后端使用Go语言开发。

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
- macOS (amd64/arm64): `nfa-dashboard-darwin-amd64.tar.gz` / `nfa-dashboard-darwin-arm64.tar.gz`
- Windows (amd64): `nfa-dashboard-windows-amd64.zip`

### 部署方法

#### Linux/macOS部署

1. 下载对应平台的压缩包
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

部署脚本支持的参数：
- `--domain`: 网站域名（默认：localhost）
- `--db-host`: 数据库主机地址（默认：localhost）
- `--db-port`: 数据库端口（默认：3306）
- `--db-user`: 数据库用户名（默认：root）
- `--db-pass`: 数据库密码
- `--db-name`: 数据库名称（默认：nfa_v2）
- `--install-dir`: 安装目录（默认：/opt/nfa-dashboard）

#### Windows部署

1. 下载Windows版压缩包 `nfa-dashboard-windows-amd64.zip`
2. 解压压缩包
3. 以管理员身份运行部署脚本
   ```
   cd scripts
   
   # 安装
   deploy.bat install --domain example.com --db-host localhost --db-user root --db-pass password
   
   # 更新
   deploy.bat update
   
   # 卸载
   deploy.bat uninstall
   ```

Windows版部署脚本支持与Linux/macOS版相同的参数，但安装目录默认为`C:\nfa-dashboard`。

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
- Linux/macOS/Windows服务器
- Nginx (Linux/macOS)
- MySQL 5.7或以上

## 许可证

[待定]
