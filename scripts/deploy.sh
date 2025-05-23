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
    
    # 创建Nginx配置
    cat > $NGINX_CONF << EOL
server {
    listen 80;
    server_name $DOMAIN;
    
    location / {
        root $INSTALL_DIR/frontend;
        index index.html;
        try_files \$uri \$uri/ /index.html;
    }
    
    location /api/ {
        proxy_pass http://localhost:8081/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOL
    
    # 检查Nginx配置并重新加载
    nginx -t && systemctl reload nginx
    
    echo -e "${GREEN}Nginx配置完成${NC}"
}

# 配置systemd服务
configure_systemd() {
    echo -e "${BLUE}配置系统服务...${NC}"
    
    # 创建systemd服务文件
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
    
    # 重新加载systemd配置
    systemctl daemon-reload
    
    echo -e "${GREEN}系统服务配置完成${NC}"
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
    
    # 复制文件到安装目录
    echo -e "${BLUE}复制文件到安装目录...${NC}"
    
    # 复制前端文件
    mkdir -p $INSTALL_DIR/frontend
    cp -r "$(dirname "$0")/../frontend"/* $INSTALL_DIR/frontend/
    
    # 复制后端文件
    mkdir -p $INSTALL_DIR/backend
    cp -r "$(dirname "$0")/../backend"/* $INSTALL_DIR/backend/
    
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
    systemctl enable $SERVICE_NAME
    systemctl start $SERVICE_NAME
    
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
    
    # 复制前端文件
    rm -rf $INSTALL_DIR/frontend
    mkdir -p $INSTALL_DIR/frontend
    cp -r "$(dirname "$0")/../frontend"/* $INSTALL_DIR/frontend/
    
    # 复制后端文件（保留config目录）
    find $INSTALL_DIR/backend -type f -not -path "*/config/*" -delete
    cp -r "$(dirname "$0")/../backend"/* $INSTALL_DIR/backend/
    
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
    systemctl start $SERVICE_NAME
    
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
