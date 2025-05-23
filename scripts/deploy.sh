#!/bin/bash
# NFA Dashboard 一键部署脚本
# 支持安装、更新和卸载功能

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # 无颜色

# 默认安装路径
INSTALL_DIR="/opt/nfa-dashboard"
SERVICE_NAME="nfa-dashboard"
NGINX_CONF="/etc/nginx/conf.d/nfa-dashboard.conf"
SYSTEMD_SERVICE="/etc/systemd/system/${SERVICE_NAME}.service"

# 打印帮助信息
print_usage() {
    echo -e "${BLUE}NFA Dashboard 一键部署脚本${NC}"
    echo -e "用法: $0 [选项]"
    echo -e "选项:"
    echo -e "  ${GREEN}install${NC}    安装 NFA Dashboard"
    echo -e "  ${YELLOW}update${NC}     更新 NFA Dashboard"
    echo -e "  ${RED}uninstall${NC}  卸载 NFA Dashboard"
    echo -e "  ${BLUE}help${NC}       显示此帮助信息"
    echo
    echo -e "示例:"
    echo -e "  $0 install --domain example.com --db-host localhost --db-user root --db-pass password"
    echo -e "  $0 update"
    echo -e "  $0 uninstall"
}

# 检查是否为root用户
check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        echo -e "${RED}错误: 此脚本需要以root用户运行${NC}"
        exit 1
    fi
}

# 检查系统依赖
check_dependencies() {
    echo -e "${BLUE}检查系统依赖...${NC}"
    
    # 检查操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
    else
        OS=$(uname -s)
    fi
    
    # 安装依赖
    case $OS in
        "Ubuntu"*|"Debian"*)
            apt-get update
            apt-get install -y nginx curl
            ;;
        "CentOS"*|"Red Hat"*|"Fedora"*)
            yum install -y epel-release
            yum install -y nginx curl
            ;;
        "Darwin")
            echo -e "${YELLOW}在MacOS上运行，假设已安装依赖${NC}"
            ;;
        *)
            echo -e "${YELLOW}未识别的操作系统: $OS${NC}"
            echo -e "${YELLOW}请确保已安装以下依赖: nginx, curl${NC}"
            ;;
    esac
    
    echo -e "${GREEN}依赖检查完成${NC}"
}

# 配置数据库连接
configure_database() {
    echo -e "${BLUE}配置数据库连接...${NC}"
    
    # 检查配置文件是否存在，如果不存在但有模板文件，则复制模板
    if [ ! -f "$INSTALL_DIR/backend/config/config.yaml" ]; then
        if [ -f "$INSTALL_DIR/backend/config/config.yaml.template" ]; then
            echo -e "${BLUE}使用配置模板创建配置文件...${NC}"
            cp "$INSTALL_DIR/backend/config/config.yaml.template" "$INSTALL_DIR/backend/config/config.yaml"
        else
            echo -e "${RED}错误: 找不到配置文件或模板${NC}"
            echo -e "${YELLOW}创建默认配置文件...${NC}"
            mkdir -p "$INSTALL_DIR/backend/config"
            cat > "$INSTALL_DIR/backend/config/config.yaml" << EOL
server:
  port: 8081

database:
  host: localhost
  port: 3306
  username: root
  password: 
  dbname: nfa_v2

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
EOL
        fi
    fi
    
    # 替换配置文件中的数据库信息
    sed -i.bak "s/host:.*/host: $DB_HOST/g" $INSTALL_DIR/backend/config/config.yaml
    sed -i.bak "s/port:.*/port: $DB_PORT/g" $INSTALL_DIR/backend/config/config.yaml
    sed -i.bak "s/username:.*/username: $DB_USER/g" $INSTALL_DIR/backend/config/config.yaml
    sed -i.bak "s/password:.*/password: $DB_PASS/g" $INSTALL_DIR/backend/config/config.yaml
    sed -i.bak "s/dbname:.*/dbname: $DB_NAME/g" $INSTALL_DIR/backend/config/config.yaml
    
    # 清理备份文件
    rm -f $INSTALL_DIR/backend/config/config.yaml.bak
    
    echo -e "${GREEN}数据库配置完成${NC}"
}

# 配置Nginx
configure_nginx() {
    echo -e "${BLUE}配置Nginx...${NC}"
    
    # 检查Nginx是否安装
    if ! command -v nginx &> /dev/null; then
        echo -e "${YELLOW}警告: Nginx未安装，跳过Nginx配置${NC}"
        return 0
    fi
    
    # 检查Nginx配置目录是否存在
    NGINX_CONF_DIR="/etc/nginx/conf.d"
    if [ ! -d "$NGINX_CONF_DIR" ]; then
        echo -e "${YELLOW}警告: Nginx配置目录 $NGINX_CONF_DIR 不存在${NC}"
        # 尝试其他常见的Nginx配置目录
        for dir in "/etc/nginx/sites-available" "/etc/nginx/sites-enabled" "/usr/local/etc/nginx/conf.d"; do
            if [ -d "$dir" ]; then
                NGINX_CONF_DIR="$dir"
                echo -e "${BLUE}使用替代Nginx配置目录: $NGINX_CONF_DIR${NC}"
                break
            fi
        done
    fi
    
    # 更新Nginx配置文件路径
    NGINX_CONF="$NGINX_CONF_DIR/nfa-dashboard.conf"
    
    # 创建Nginx配置
    mkdir -p "$(dirname "$NGINX_CONF")"
    cat > $NGINX_CONF << EOL
server {
    listen 80;
    server_name $DOMAIN;
    
    # 设置跨域头
    add_header Access-Control-Allow-Origin *;
    add_header Access-Control-Allow-Methods 'GET, POST, PUT, DELETE, OPTIONS';
    add_header Access-Control-Allow-Headers 'DNT,X-Mx-ReqToken,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization';
    
    # 处理OPTIONS请求
    if (\$request_method = 'OPTIONS') {
        return 204;
    }
    
    location / {
        root $INSTALL_DIR/frontend;
        index index.html;
        try_files \$uri \$uri/ /index.html;
    }
    
    # 代理所有API请求到后端服务
    location /api/ {
        proxy_pass http://localhost:8081/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }
}
EOL
    
    # 检查Nginx配置
    if nginx -t; then
        # 尝试重新加载Nginx
        if systemctl is-active --quiet nginx; then
            systemctl reload nginx || echo -e "${YELLOW}警告: 无法重新加载Nginx，请手动重启${NC}"
        else
            systemctl start nginx || echo -e "${YELLOW}警告: 无法启动Nginx，请手动启动${NC}"
        fi
    else
        echo -e "${RED}错误: Nginx配置测试失败${NC}"
    fi
    
    echo -e "${GREEN}Nginx配置完成${NC}"
}

# 配置systemd服务
configure_systemd() {
    echo -e "${BLUE}配置系统服务...${NC}"
    
    # 检查是否使用systemd
    if ! command -v systemctl &> /dev/null; then
        echo -e "${YELLOW}警告: 系统不支持systemd，跳过服务配置${NC}"
        echo -e "${YELLOW}请手动启动后端服务: $INSTALL_DIR/backend/nfa-dashboard-backend${NC}"
        return 0
    fi
    
    # 检查systemd服务目录
    SYSTEMD_DIR="/etc/systemd/system"
    if [ ! -d "$SYSTEMD_DIR" ]; then
        echo -e "${RED}错误: systemd服务目录不存在${NC}"
        return 1
    fi
    
    # 创建systemd服务文件
    mkdir -p "$(dirname "$SYSTEMD_SERVICE")"
    cat > $SYSTEMD_SERVICE << EOL
[Unit]
Description=NFA Dashboard Backend Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR/backend
ExecStart=$INSTALL_DIR/backend/nfa-dashboard-backend
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOL
    
    # 设置正确的权限
    chmod 644 $SYSTEMD_SERVICE
    
    # 重新加载systemd配置
    systemctl daemon-reload || {
        echo -e "${RED}错误: 无法重新加载systemd配置${NC}"
        return 1
    }
    
    echo -e "${GREEN}系统服务配置完成${NC}"
    return 0
}

# 检查必要的文件和目录是否存在
check_files() {
    # 检查前端目录
    if [ ! -d "$(dirname "$0")/../frontend" ]; then
        echo -e "${RED}错误: 找不到前端目录${NC}"
        return 1
    fi
    
    # 检查后端目录
    if [ ! -d "$(dirname "$0")/../backend" ]; then
        echo -e "${RED}错误: 找不到后端目录${NC}"
        return 1
    fi
    
    # 检查后端可执行文件
    if [ ! -f "$(dirname "$0")/../backend/nfa-dashboard-backend" ]; then
        echo -e "${RED}错误: 找不到后端可执行文件${NC}"
        return 1
    fi
    
    # 检查配置文件
    if [ ! -f "$(dirname "$0")/../backend/config/config.yaml.template" ] && [ ! -f "$(dirname "$0")/../backend/config/config.yaml" ]; then
        echo -e "${RED}错误: 找不到配置文件${NC}"
        return 1
    fi
    
    return 0
}

# 安装NFA Dashboard
install_dashboard() {
    echo -e "${BLUE}开始安装 NFA Dashboard...${NC}"
    
    # 检查必要的文件和目录
    if ! check_files; then
        echo -e "${RED}错误: 缺少必要的文件或目录${NC}"
        echo -e "${YELLOW}请确保您在解压缩包后的正确目录中运行此脚本${NC}"
        exit 1
    fi
    
    # 创建安装目录
    mkdir -p $INSTALL_DIR
    mkdir -p $INSTALL_DIR/frontend
    mkdir -p $INSTALL_DIR/backend/config
    
    # 复制文件到安装目录
    echo -e "${BLUE}复制文件到安装目录...${NC}"
    
    # 检查前端目录结构
    FRONTEND_SRC="$(dirname "$0")/../frontend"
    if [ -d "$FRONTEND_SRC/frontend/dist" ]; then
        # 新结构: frontend/frontend/dist
        cp -r "$FRONTEND_SRC/frontend/dist"/* $INSTALL_DIR/frontend/
    elif [ -d "$FRONTEND_SRC/dist" ]; then
        # 旧结构: frontend/dist
        cp -r "$FRONTEND_SRC/dist"/* $INSTALL_DIR/frontend/
    else
        # 直接复制所有前端文件
        cp -r "$FRONTEND_SRC"/* $INSTALL_DIR/frontend/
    fi
    
    # 检查后端目录结构
    BACKEND_SRC="$(dirname "$0")/../backend"
    # 复制后端可执行文件
    if [ -f "$BACKEND_SRC/nfa-dashboard-backend" ]; then
        cp "$BACKEND_SRC/nfa-dashboard-backend" $INSTALL_DIR/backend/
    fi
    
    # 复制配置文件模板
    if [ -f "$BACKEND_SRC/config/config.yaml.template" ]; then
        cp "$BACKEND_SRC/config/config.yaml.template" $INSTALL_DIR/backend/config/
    elif [ -f "$BACKEND_SRC/config/config.yaml" ]; then
        cp "$BACKEND_SRC/config/config.yaml" $INSTALL_DIR/backend/config/config.yaml.template
    fi
    
    # 设置可执行权限
    chmod +x $INSTALL_DIR/backend/nfa-dashboard-backend
    
    # 配置数据库
    configure_database
    
    # 配置Nginx
    configure_nginx
    
    # 配置systemd服务
    configure_systemd
    
    # 启动服务
    echo -e "${BLUE}启动服务...${NC}"
    if command -v systemctl &> /dev/null; then
        systemctl enable $SERVICE_NAME || echo -e "${YELLOW}警告: 无法启用服务${NC}"
        systemctl start $SERVICE_NAME || {
            echo -e "${RED}错误: 无法启动服务${NC}"
            echo -e "${YELLOW}尝试手动启动后端...${NC}"
            nohup $INSTALL_DIR/backend/nfa-dashboard-backend > /var/log/nfa-dashboard.log 2>&1 &
            echo -e "${GREEN}后端服务已手动启动${NC}"
        }
    else
        echo -e "${YELLOW}系统不支持systemd，手动启动后端...${NC}"
        nohup $INSTALL_DIR/backend/nfa-dashboard-backend > /var/log/nfa-dashboard.log 2>&1 &
        echo -e "${GREEN}后端服务已手动启动${NC}"
    fi
    
    echo -e "${GREEN}NFA Dashboard 安装完成!${NC}"
    echo -e "您可以通过访问 http://$DOMAIN 来使用 NFA Dashboard"
}

# 更新NFA Dashboard
update_dashboard() {
    echo -e "${BLUE}开始更新 NFA Dashboard...${NC}"
    
    # 检查必要的文件和目录
    if ! check_files; then
        echo -e "${RED}错误: 缺少必要的文件或目录${NC}"
        echo -e "${YELLOW}请确保您在解压缩包后的正确目录中运行此脚本${NC}"
        exit 1
    fi
    
    # 备份配置文件
    echo -e "${BLUE}备份配置文件...${NC}"
    if [ -f $INSTALL_DIR/backend/config/config.yaml ]; then
        cp $INSTALL_DIR/backend/config/config.yaml /tmp/nfa-config.yaml.bak
    fi
    
    # 停止服务
    echo -e "${BLUE}停止服务...${NC}"
    systemctl stop $SERVICE_NAME || true
    
    # 复制文件到安装目录
    echo -e "${BLUE}更新文件...${NC}"
    
    # 检查前端目录结构
    FRONTEND_SRC="$(dirname "$0")/../frontend"
    
    # 修改前端 API 配置
    echo -e "${BLUE}更新前端 API 配置...${NC}"
    API_CONFIG_FILE=""
    
    if [ -f "$FRONTEND_SRC/frontend/src/api/index.ts" ]; then
        API_CONFIG_FILE="$FRONTEND_SRC/frontend/src/api/index.ts"
    elif [ -f "$FRONTEND_SRC/src/api/index.ts" ]; then
        API_CONFIG_FILE="$FRONTEND_SRC/src/api/index.ts"
    fi
    
    if [ -n "$API_CONFIG_FILE" ]; then
        echo -e "${BLUE}找到 API 配置文件: $API_CONFIG_FILE${NC}"
        
        # 备份原始文件
        cp "$API_CONFIG_FILE" "${API_CONFIG_FILE}.bak"
        
        # 替换硬编码的 localhost 为动态获取的主机名
        if grep -q "baseURL: 'http://localhost:8081'" "$API_CONFIG_FILE"; then
            echo -e "${BLUE}替换硬编码的 API 地址...${NC}"
            sed -i.bak "s|baseURL: 'http://localhost:8081'|baseURL: window.location.protocol + \"//\" + window.location.hostname + \":8081\"|g" "$API_CONFIG_FILE"
            rm -f "${API_CONFIG_FILE}.bak"
        fi
    else
        echo -e "${YELLOW}未找到 API 配置文件，跳过更新${NC}"
    fi
    
    # 重新构建前端
    echo -e "${BLUE}重新构建前端...${NC}"
    if [ -f "$FRONTEND_SRC/frontend/package.json" ]; then
        cd "$FRONTEND_SRC/frontend"
        if command -v npm &> /dev/null; then
            npm install --production=false && npm run build-only
        else
            echo -e "${YELLOW}未找到 npm，跳过前端构建${NC}"
        fi
    elif [ -f "$FRONTEND_SRC/package.json" ]; then
        cd "$FRONTEND_SRC"
        if command -v npm &> /dev/null; then
            npm install --production=false && npm run build-only
        else
            echo -e "${YELLOW}未找到 npm，跳过前端构建${NC}"
        fi
    else
        echo -e "${YELLOW}未找到前端 package.json，跳过构建${NC}"
    fi
    
    # 复制前端文件
    echo -e "${BLUE}复制前端文件...${NC}"
    rm -rf $INSTALL_DIR/frontend
    mkdir -p $INSTALL_DIR/frontend
    
    if [ -d "$FRONTEND_SRC/frontend/dist" ]; then
        # 新结构: frontend/frontend/dist
        cp -r "$FRONTEND_SRC/frontend/dist"/* $INSTALL_DIR/frontend/
    elif [ -d "$FRONTEND_SRC/dist" ]; then
        # 旧结构: frontend/dist
        cp -r "$FRONTEND_SRC/dist"/* $INSTALL_DIR/frontend/
    else
        # 直接复制所有前端文件
        cp -r "$FRONTEND_SRC"/* $INSTALL_DIR/frontend/
    fi
    
    # 检查后端目录结构
    BACKEND_SRC="$(dirname "$0")/../backend"
    # 复制后端文件（保留config目录）
    find $INSTALL_DIR/backend -type f -not -path "*/config/*" -delete
    
    # 复制后端可执行文件
    if [ -f "$BACKEND_SRC/nfa-dashboard-backend" ]; then
        cp "$BACKEND_SRC/nfa-dashboard-backend" $INSTALL_DIR/backend/
    fi
    
    # 复制配置文件模板（如果不存在配置文件）
    if [ ! -f "$INSTALL_DIR/backend/config/config.yaml" ] && [ -f "$BACKEND_SRC/config/config.yaml.template" ]; then
        cp "$BACKEND_SRC/config/config.yaml.template" $INSTALL_DIR/backend/config/config.yaml.template
    fi
    
    # 恢复配置文件
    echo -e "${BLUE}恢复配置文件...${NC}"
    if [ -f /tmp/nfa-config.yaml.bak ]; then
        cp /tmp/nfa-config.yaml.bak $INSTALL_DIR/backend/config/config.yaml
        rm /tmp/nfa-config.yaml.bak
    fi
    
    # 设置可执行权限
    chmod +x $INSTALL_DIR/backend/nfa-dashboard-backend
    
    # 启动服务
    echo -e "${BLUE}启动服务...${NC}"
    if command -v systemctl &> /dev/null; then
        systemctl start $SERVICE_NAME || {
            echo -e "${RED}错误: 无法启动服务${NC}"
            echo -e "${YELLOW}尝试手动启动后端...${NC}"
            nohup $INSTALL_DIR/backend/nfa-dashboard-backend > /var/log/nfa-dashboard.log 2>&1 &
            echo -e "${GREEN}后端服务已手动启动${NC}"
        }
    else
        echo -e "${YELLOW}系统不支持systemd，手动启动后端...${NC}"
        nohup $INSTALL_DIR/backend/nfa-dashboard-backend > /var/log/nfa-dashboard.log 2>&1 &
        echo -e "${GREEN}后端服务已手动启动${NC}"
    fi
    
    echo -e "${GREEN}NFA Dashboard 更新完成!${NC}"
}

# 卸载NFA Dashboard
uninstall_dashboard() {
    echo -e "${RED}开始卸载 NFA Dashboard...${NC}"
    
    # 确认卸载
    read -p "确定要卸载 NFA Dashboard 吗? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}卸载已取消${NC}"
        exit 0
    fi
    
    # 停止服务
    echo -e "${BLUE}停止服务...${NC}"
    systemctl stop $SERVICE_NAME || true
    systemctl disable $SERVICE_NAME || true
    
    # 删除服务文件
    echo -e "${BLUE}删除服务文件...${NC}"
    rm -f $SYSTEMD_SERVICE
    systemctl daemon-reload
    
    # 删除Nginx配置
    echo -e "${BLUE}删除Nginx配置...${NC}"
    rm -f $NGINX_CONF
    nginx -t && systemctl reload nginx
    
    # 删除安装目录
    echo -e "${BLUE}删除安装目录...${NC}"
    rm -rf $INSTALL_DIR
    
    echo -e "${GREEN}NFA Dashboard 卸载完成!${NC}"
}

# 解析命令行参数
parse_args() {
    DOMAIN="localhost"
    DB_HOST="localhost"
    DB_PORT="3306"
    DB_USER="root"
    DB_PASS=""
    DB_NAME="nfa_v2"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --domain)
                DOMAIN="$2"
                shift 2
                ;;
            --db-host)
                DB_HOST="$2"
                shift 2
                ;;
            --db-port)
                DB_PORT="$2"
                shift 2
                ;;
            --db-user)
                DB_USER="$2"
                shift 2
                ;;
            --db-pass)
                DB_PASS="$2"
                shift 2
                ;;
            --db-name)
                DB_NAME="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
}

# 主函数
main() {
    if [ $# -eq 0 ]; then
        print_usage
        exit 0
    fi
    
    ACTION=$1
    shift
    
    case $ACTION in
        install)
            check_root
            parse_args "$@"
            check_dependencies
            install_dashboard
            ;;
        update)
            check_root
            check_dependencies
            update_dashboard
            ;;
        uninstall)
            check_root
            uninstall_dashboard
            ;;
        help|--help|-h)
            print_usage
            ;;
        *)
            echo -e "${RED}错误: 未知操作 '$ACTION'${NC}"
            print_usage
            exit 1
            ;;
    esac
}

main "$@"
