@echo off
:: NFA Dashboard 一键部署脚本 (Windows版)
:: 支持安装、更新和卸载功能

setlocal enabledelayedexpansion

:: 默认安装路径
set "INSTALL_DIR=C:\nfa-dashboard"
set "SERVICE_NAME=NFADashboard"

:: 颜色定义
set "RED=[91m"
set "GREEN=[92m"
set "YELLOW=[93m"
set "BLUE=[94m"
set "NC=[0m"

:: 打印帮助信息
:print_usage
if "%~1"=="" (
    echo %BLUE%NFA Dashboard 一键部署脚本 (Windows版)%NC%
    echo 用法: %0 [选项]
    echo 选项:
    echo   %GREEN%install%NC%    安装 NFA Dashboard
    echo   %YELLOW%update%NC%     更新 NFA Dashboard
    echo   %RED%uninstall%NC%  卸载 NFA Dashboard
    echo   %BLUE%help%NC%       显示此帮助信息
    echo.
    echo 示例:
    echo   %0 install --domain example.com --db-host localhost --db-user root --db-pass password
    echo   %0 update
    echo   %0 uninstall
    exit /b 0
)

:: 检查管理员权限
:check_admin
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo %RED%错误: 此脚本需要以管理员权限运行%NC%
    echo 请右键点击脚本，选择"以管理员身份运行"
    exit /b 1
)

:: 检查系统依赖
:check_dependencies
echo %BLUE%检查系统依赖...%NC%

:: 检查是否安装了NSSM (Non-Sucking Service Manager)
where nssm >nul 2>&1
if %errorlevel% neq 0 (
    echo %YELLOW%未找到NSSM，将自动下载并安装...%NC%
    
    :: 创建临时目录
    mkdir %TEMP%\nssm-install >nul 2>&1
    
    :: 下载NSSM
    powershell -Command "Invoke-WebRequest -Uri 'https://nssm.cc/release/nssm-2.24.zip' -OutFile '%TEMP%\nssm.zip'"
    
    :: 解压NSSM
    powershell -Command "Expand-Archive -Path '%TEMP%\nssm.zip' -DestinationPath '%TEMP%\nssm-install' -Force"
    
    :: 复制NSSM到系统目录
    if exist "%TEMP%\nssm-install\nssm-2.24\win64\nssm.exe" (
        copy "%TEMP%\nssm-install\nssm-2.24\win64\nssm.exe" "C:\Windows\System32\" >nul
    ) else (
        copy "%TEMP%\nssm-install\nssm-2.24\win32\nssm.exe" "C:\Windows\System32\" >nul
    )
    
    :: 清理临时文件
    rmdir /s /q %TEMP%\nssm-install >nul 2>&1
    del %TEMP%\nssm.zip >nul 2>&1
)

echo %GREEN%依赖检查完成%NC%
exit /b 0

:: 配置数据库连接
:configure_database
echo %BLUE%配置数据库连接...%NC%

:: 替换配置文件中的数据库信息
powershell -Command "(Get-Content '%INSTALL_DIR%\backend\config\config.yaml') -replace 'host:.*', 'host: %DB_HOST%' | Set-Content '%INSTALL_DIR%\backend\config\config.yaml'"
powershell -Command "(Get-Content '%INSTALL_DIR%\backend\config\config.yaml') -replace 'port:.*', 'port: %DB_PORT%' | Set-Content '%INSTALL_DIR%\backend\config\config.yaml'"
powershell -Command "(Get-Content '%INSTALL_DIR%\backend\config\config.yaml') -replace 'username:.*', 'username: %DB_USER%' | Set-Content '%INSTALL_DIR%\backend\config\config.yaml'"
powershell -Command "(Get-Content '%INSTALL_DIR%\backend\config\config.yaml') -replace 'password:.*', 'password: %DB_PASS%' | Set-Content '%INSTALL_DIR%\backend\config\config.yaml'"
powershell -Command "(Get-Content '%INSTALL_DIR%\backend\config\config.yaml') -replace 'dbname:.*', 'dbname: %DB_NAME%' | Set-Content '%INSTALL_DIR%\backend\config\config.yaml'"

echo %GREEN%数据库配置完成%NC%
exit /b 0

:: 安装Windows服务
:install_service
echo %BLUE%安装Windows服务...%NC%

:: 使用NSSM创建服务
nssm install %SERVICE_NAME% "%INSTALL_DIR%\backend\nfa-dashboard-backend.exe"
nssm set %SERVICE_NAME% AppDirectory "%INSTALL_DIR%\backend"
nssm set %SERVICE_NAME% Description "NFA Dashboard Backend Service"
nssm set %SERVICE_NAME% Start SERVICE_AUTO_START
nssm set %SERVICE_NAME% AppStdout "%INSTALL_DIR%\backend\logs\service.log"
nssm set %SERVICE_NAME% AppStderr "%INSTALL_DIR%\backend\logs\service.log"

:: 启动服务
nssm start %SERVICE_NAME%

echo %GREEN%Windows服务安装完成%NC%
exit /b 0

:: 安装NFA Dashboard
:install_dashboard
echo %BLUE%开始安装 NFA Dashboard...%NC%

:: 创建安装目录
mkdir "%INSTALL_DIR%" >nul 2>&1
mkdir "%INSTALL_DIR%\backend\logs" >nul 2>&1

:: 解压文件到安装目录
echo %BLUE%解压文件到安装目录...%NC%
powershell -Command "Expand-Archive -Path '%~dp0\..\nfa-dashboard-windows-amd64.zip' -DestinationPath '%INSTALL_DIR%' -Force"

:: 配置数据库
call :configure_database

:: 安装Windows服务
call :install_service

echo %GREEN%NFA Dashboard 安装完成!%NC%
echo 您可以通过访问 http://%DOMAIN% 来使用 NFA Dashboard
exit /b 0

:: 更新NFA Dashboard
:update_dashboard
echo %BLUE%开始更新 NFA Dashboard...%NC%

:: 备份配置文件
echo %BLUE%备份配置文件...%NC%
if exist "%INSTALL_DIR%\backend\config\config.yaml" (
    copy "%INSTALL_DIR%\backend\config\config.yaml" "%TEMP%\nfa-config.yaml.bak" >nul
)

:: 停止服务
echo %BLUE%停止服务...%NC%
nssm stop %SERVICE_NAME% >nul 2>&1

:: 解压文件到安装目录
echo %BLUE%更新文件...%NC%
powershell -Command "Expand-Archive -Path '%~dp0\..\nfa-dashboard-windows-amd64.zip' -DestinationPath '%INSTALL_DIR%' -Force"

:: 恢复配置文件
echo %BLUE%恢复配置文件...%NC%
if exist "%TEMP%\nfa-config.yaml.bak" (
    copy "%TEMP%\nfa-config.yaml.bak" "%INSTALL_DIR%\backend\config\config.yaml" >nul
    del "%TEMP%\nfa-config.yaml.bak" >nul
)

:: 启动服务
echo %BLUE%启动服务...%NC%
nssm start %SERVICE_NAME%

echo %GREEN%NFA Dashboard 更新完成!%NC%
exit /b 0

:: 卸载NFA Dashboard
:uninstall_dashboard
echo %RED%开始卸载 NFA Dashboard...%NC%

:: 确认卸载
set /p CONFIRM="确定要卸载 NFA Dashboard 吗? [y/N] "
if /i not "%CONFIRM%"=="y" (
    echo %YELLOW%卸载已取消%NC%
    exit /b 0
)

:: 停止并删除服务
echo %BLUE%停止并删除服务...%NC%
nssm stop %SERVICE_NAME% >nul 2>&1
nssm remove %SERVICE_NAME% confirm >nul 2>&1

:: 删除安装目录
echo %BLUE%删除安装目录...%NC%
rmdir /s /q "%INSTALL_DIR%" >nul 2>&1

echo %GREEN%NFA Dashboard 卸载完成!%NC%
exit /b 0

:: 解析命令行参数
:parse_args
set "DOMAIN=localhost"
set "DB_HOST=localhost"
set "DB_PORT=3306"
set "DB_USER=root"
set "DB_PASS="
set "DB_NAME=nfa_v2"

:parse_args_loop
if "%~1"=="" goto :eof

if "%~1"=="--domain" (
    set "DOMAIN=%~2"
    shift
    shift
    goto parse_args_loop
)

if "%~1"=="--db-host" (
    set "DB_HOST=%~2"
    shift
    shift
    goto parse_args_loop
)

if "%~1"=="--db-port" (
    set "DB_PORT=%~2"
    shift
    shift
    goto parse_args_loop
)

if "%~1"=="--db-user" (
    set "DB_USER=%~2"
    shift
    shift
    goto parse_args_loop
)

if "%~1"=="--db-pass" (
    set "DB_PASS=%~2"
    shift
    shift
    goto parse_args_loop
)

if "%~1"=="--db-name" (
    set "DB_NAME=%~2"
    shift
    shift
    goto parse_args_loop
)

if "%~1"=="--install-dir" (
    set "INSTALL_DIR=%~2"
    shift
    shift
    goto parse_args_loop
)

shift
goto parse_args_loop

:: 主函数
:main
if "%~1"=="" (
    call :print_usage
    exit /b 0
)

set "ACTION=%~1"
shift

if "%ACTION%"=="install" (
    call :check_admin
    call :parse_args %*
    call :check_dependencies
    call :install_dashboard
    exit /b 0
)

if "%ACTION%"=="update" (
    call :check_admin
    call :check_dependencies
    call :update_dashboard
    exit /b 0
)

if "%ACTION%"=="uninstall" (
    call :check_admin
    call :uninstall_dashboard
    exit /b 0
)

if "%ACTION%"=="help" (
    call :print_usage
    exit /b 0
)

if "%ACTION%"=="--help" (
    call :print_usage
    exit /b 0
)

if "%ACTION%"=="-h" (
    call :print_usage
    exit /b 0
)

echo %RED%错误: 未知操作 '%ACTION%'%NC%
call :print_usage
exit /b 1

:: 调用主函数
:start
call :main %*
exit /b
