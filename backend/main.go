package main

import (
	"fmt"
	"log"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/controller"
	"nfa-dashboard/internal/middleware"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/scheduler"
	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	config.LoadConfig()

	// 初始化数据库连接
	model.InitDB()

	// 创建Gin引擎
	r := gin.Default()

	// 注册中间件
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.Audit())

	// 创建依赖
	schoolRepo := repository.NewSchoolRepository()
	schoolService := service.NewSchoolService(schoolRepo)
	schoolController := controller.NewSchoolController(schoolService)

	// 创建结算系统依赖
	settlementRepo := repository.NewSettlementRepository()
	settlementService := service.NewSettlementService(settlementRepo)
	settlementController := controller.NewSettlementController(settlementService)

	// 认证与权限依赖
	userRepo := repository.NewUserRepository()
	authService := service.NewAuthService(userRepo)
	authController := controller.NewAuthController(authService)
	authMW := middleware.NewAuthMiddleware(authService)

	// 系统管理依赖（角色/权限/用户）
	roleRepo := repository.NewRoleRepository()
	permRepo := repository.NewPermissionRepository()

	roleService := service.NewRoleService(roleRepo, permRepo)
	roleController := controller.NewSystemRoleController(roleService)

	permService := service.NewPermissionService(permRepo)
	permController := controller.NewSystemPermissionController(permService)

	userService := service.NewUserService(userRepo, roleRepo)
	systemUserController := controller.NewSystemUserController(userService)

	// 操作日志依赖
	opLogRepo := repository.NewOperationLogRepository()
	opLogService := service.NewOperationLogService(opLogRepo)
	opLogController := controller.NewOperationLogController(opLogService)

	// 创建并启动结算调度器
	settlementScheduler := scheduler.NewSettlementScheduler(settlementService)
	settlementScheduler.Start()

	// API路由
	api := r.Group("/api/v1")
	{
		// 认证接口
		auth := api.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/refresh", authController.Refresh)
			auth.GET("/profile", authMW.AuthRequired(), authController.Profile)
		}

		// 学校相关接口
		api.GET("/schools", schoolController.GetAllSchools)
		api.GET("/regions", schoolController.GetAllRegions)
		api.GET("/cps", schoolController.GetAllCPs)
		api.GET("/traffic", schoolController.GetTrafficData)
		api.GET("/traffic/summary", schoolController.GetTrafficSummary)

		// 结算系统相关接口（需要登录）
		settlement := api.Group("/settlement", authMW.AuthRequired())
		{
			// 结算配置相关接口
			settlement.GET("/config", settlementController.GetSettlementConfig)
			settlement.PUT("/config", authMW.PermissionRequired("settlement.calculate"), settlementController.UpdateSettlementConfig)

			// 结算任务相关接口
			settlement.GET("/tasks", settlementController.GetSettlementTasks)
			settlement.GET("/tasks/:id", settlementController.GetSettlementTaskByID)
			settlement.POST("/tasks/daily", authMW.PermissionRequired("settlement.calculate"), settlementController.CreateDailySettlementTask)
			settlement.POST("/tasks/weekly", authMW.PermissionRequired("settlement.calculate"), settlementController.CreateWeeklySettlementTask)
			settlement.DELETE("/tasks/:id", authMW.PermissionRequired("settlement.calculate"), settlementController.DeleteSettlementTask)

			// 结算数据相关接口
			settlement.GET("/data", settlementController.GetSettlements)
			settlement.GET("/daily-details", settlementController.GetDailySettlementDetails)
		}

		// 系统管理接口（需要登录）
		system := api.Group("/system", authMW.AuthRequired())
		{
			// 角色管理（需要 system.role.manage）
			roles := system.Group("/roles", authMW.PermissionRequired("system.role.manage"))
			{
				roles.GET("", roleController.ListRoles)
				roles.POST("", roleController.CreateRole)
				roles.PUT(":id", roleController.UpdateRole)
				roles.DELETE(":id", roleController.DeleteRole)
				roles.GET(":id/permissions", roleController.GetRolePermissions)
				roles.PUT(":id/permissions", roleController.SetRolePermissions)
			}

			// 权限列表（同样归属角色管理查看）
			system.GET("/permissions", authMW.PermissionRequired("system.role.manage"), permController.ListPermissions)

			// 权限管理（需要 system.permission.manage）
			system.POST("/permissions", authMW.PermissionRequired("system.permission.manage"), permController.CreatePermission)
			system.GET("/permissions/:id", authMW.PermissionRequired("system.permission.manage"), permController.GetPermission)
			system.PUT("/permissions/:id", authMW.PermissionRequired("system.permission.manage"), permController.UpdatePermission)
			system.DELETE("/permissions/:id", authMW.PermissionRequired("system.permission.manage"), permController.DisablePermission)
			system.POST("/permissions/sync", authMW.PermissionRequired("system.permission.manage"), permController.SyncPermissions)

			// 用户管理（需要 system.user.manage）
			users := system.Group("/users", authMW.PermissionRequired("system.user.manage"))
			{
				users.POST("", systemUserController.CreateUser)
				users.GET("", systemUserController.ListUsers)
				users.PUT(":id/status", systemUserController.UpdateUserStatus)
				users.PUT(":id/roles", systemUserController.SetUserRoles)
				users.PUT(":id/alias", systemUserController.UpdateUserAlias)
			}

			// 操作日志查询与导出（需要 operation_logs.read）
			system.GET("/operation-logs", authMW.PermissionRequired("operation_logs.read"), opLogController.List)
			system.GET("/operation-logs/export", authMW.PermissionRequired("operation_logs.read"), opLogController.Export)
		}
	}

	// 启动服务器
	port := config.AppConfig.Server.Port
	log.Printf("服务器启动在 http://localhost:%d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
