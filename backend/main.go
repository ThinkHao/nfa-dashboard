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

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 创建依赖
	schoolRepo := repository.NewSchoolRepository()
	schoolService := service.NewSchoolService(schoolRepo)
	schoolController := controller.NewSchoolController(schoolService)

	// 结算系统依赖
	settlementRepo := repository.NewSettlementRepository()
	settlementService := service.NewSettlementService(settlementRepo)

	// 结算公式依赖（持久化）
	formulaRepo := repository.NewSettlementFormulaRepository()
	formulaService := service.NewSettlementFormulaService(formulaRepo)
	formulaController := controller.NewSettlementFormulaController(formulaService)

	// 结算结果依赖
	settlementResultRepo := repository.NewSettlementResultRepository()
	settlementResultService := service.NewSettlementResultService(settlementResultRepo, formulaRepo)

	settlementController := controller.NewSettlementController(settlementService, settlementResultService)

	// 结算子模块：费率与业务对象依赖与控制器
	ratesRepo := repository.NewRatesRepository()
	ratesSvc := service.NewRatesService(ratesRepo)
	ratesController := controller.NewSettlementRatesController(ratesSvc)

	// 客户费率-自定义字段定义依赖与控制器
	customerFieldsRepo := repository.NewCustomerFieldsRepository()
	customerFieldsSvc := service.NewCustomerFieldsService(customerFieldsRepo)
	customerFieldsController := controller.NewCustomerFieldsController(customerFieldsSvc)

	// 客户费率-同步规则依赖与控制器
	syncRulesRepo := repository.NewSyncRulesRepository()
	syncRulesSvc := service.NewSyncRulesService(syncRulesRepo)
	syncRulesController := controller.NewSyncRulesController(syncRulesSvc)

	// 客户费率-执行同步服务与控制器
	ratesSyncSvc := service.NewRatesSyncService(syncRulesRepo, ratesRepo, schoolRepo)
	ratesSyncController := controller.NewRatesSyncController(ratesSyncSvc)

	entitiesRepo := repository.NewEntitiesRepository()
	// 业务类型依赖（供实体类型校验与单独管理）
	btRepo := repository.NewBusinessTypeRepository()
	btService := service.NewBusinessTypeService(btRepo)
	btController := controller.NewBusinessTypeController(btService)
	entitiesSvc := service.NewEntitiesService(entitiesRepo, btRepo)
	entitiesController := controller.NewSettlementEntitiesController(entitiesSvc)

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
	// 绑定配置控制器
	bindingController := controller.NewSystemBindingController()

	userService := service.NewUserService(userRepo, roleRepo)
	systemUserController := controller.NewSystemUserController(userService)

	// 用户-院校绑定：仓储/服务/控制器
	userSchoolRepo := repository.NewUserSchoolRepository()
	userSchoolService := service.NewUserSchoolService(userRepo, schoolRepo, userSchoolRepo)
	userSchoolController := controller.NewSystemUserSchoolController(userSchoolService)

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

		// API v2 路由（基于 user_id 的权限过滤）
		v2 := r.Group("/api/v2")
		{
			// 学校与流量相关接口（需要登录与权限）
			v2.GET("/schools", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllSchoolsV2)
			v2.GET("/traffic", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficDataV2)
			v2.GET("/traffic/summary", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficSummaryV2)

			// 结算系统相关接口（需要登录）
			settlementV2 := v2.Group("/settlement", authMW.AuthRequired())
			{
				settlementV2.GET("/data", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementsV2)
				settlementV2.GET("/daily-details", authMW.PermissionRequired("settlement.read"), settlementController.GetDailySettlementDetailsV2)
			}
		}

		// 学校与流量相关接口（需要登录与权限）
		api.GET("/schools", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllSchools)
		api.GET("/regions", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllRegions)
		api.GET("/cps", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllCPs)
		api.GET("/traffic", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficData)
		api.GET("/traffic/summary", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficSummary)

		// 结算系统相关接口（需要登录）
		settlement := api.Group("/settlement", authMW.AuthRequired())
		{
			// 结算配置相关接口
			settlement.GET("/config", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementConfig)
			settlement.PUT("/config", authMW.PermissionRequired("settlement.calculate"), settlementController.UpdateSettlementConfig)

			// 结算任务相关接口
			settlement.GET("/tasks", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementTasks)
			settlement.GET("/tasks/:id", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementTaskByID)
			settlement.POST("/tasks/daily", authMW.PermissionRequired("settlement.calculate"), settlementController.CreateDailySettlementTask)
			settlement.POST("/tasks/weekly", authMW.PermissionRequired("settlement.calculate"), settlementController.CreateWeeklySettlementTask)
			settlement.DELETE("/tasks/:id", authMW.PermissionRequired("settlement.calculate"), settlementController.DeleteSettlementTask)

			// 结算数据相关接口
			settlement.GET("/data", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlements)
			settlement.GET("/daily-details", authMW.PermissionRequired("settlement.read"), settlementController.GetDailySettlementDetails)
			settlement.GET("/results", authMW.PermissionRequired("settlement.results.read"), settlementController.GetSettlementResults)

			// 结算公式 CRUD
			formulas := settlement.Group("/formulas")
			{
				formulas.GET("", authMW.PermissionRequired("settlement.formula.read"), formulaController.List)
				formulas.GET("/:id", authMW.PermissionRequired("settlement.formula.read"), formulaController.Get)
				formulas.POST("", authMW.PermissionRequired("settlement.formula.write"), formulaController.Create)
				formulas.PUT("/:id", authMW.PermissionRequired("settlement.formula.write"), formulaController.Update)
				formulas.DELETE("/:id", authMW.PermissionRequired("settlement.formula.write"), formulaController.Delete)
			}

			// 费率模块（归属结算系统）
			rates := settlement.Group("/rates")
			{
				// 客户业务费率
				rates.GET("/customer", authMW.PermissionRequired("rates.customer.read"), ratesController.ListCustomerRates)
				rates.POST("/customer", authMW.PermissionRequired("rates.customer.write"), ratesController.UpsertCustomerRate)
				// 节点业务费率
				rates.GET("/node", authMW.PermissionRequired("rates.node.read"), ratesController.ListNodeRates)
				rates.POST("/node", authMW.PermissionRequired("rates.node.write"), ratesController.UpsertNodeRate)
				// 最终客户费率
				rates.GET("/final", authMW.PermissionRequired("rates.final.read"), ratesController.ListFinalCustomerRates)
				rates.POST("/final", authMW.PermissionRequired("rates.final.write"), ratesController.UpsertFinalCustomerRate)
				rates.POST("/final/init-from-customer", authMW.PermissionRequired("rates.final.write"), ratesController.InitFinalCustomerRatesFromCustomer)
				rates.POST("/final/refresh", authMW.PermissionRequired("rates.final.write"), ratesController.RefreshFinalCustomerRates)
				// 清理无效的最终客户费率（仅 auto；任一关键费率字段为空）
				rates.POST("/final/cleanup-invalid", authMW.PermissionRequired("rates.final.write"), ratesController.CleanupInvalidFinalCustomerRates)

				// 客户费率-自定义字段定义
				fields := rates.Group("/customer-fields")
				{
					fields.GET("", authMW.PermissionRequired("rates.customer_fields.read"), customerFieldsController.List)
					fields.POST("", authMW.PermissionRequired("rates.customer_fields.write"), customerFieldsController.Create)
					fields.PUT("/:id", authMW.PermissionRequired("rates.customer_fields.write"), customerFieldsController.Update)
					fields.DELETE("/:id", authMW.PermissionRequired("rates.customer_fields.write"), customerFieldsController.Delete)
				}

				// 客户费率-同步规则
				rules := rates.Group("/sync-rules")
				{
					rules.GET("", authMW.PermissionRequired("rates.sync_rules.read"), syncRulesController.List)
					rules.POST("", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.Create)
					rules.PUT("/:id", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.Update)
					rules.DELETE("/:id", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.Delete)
					rules.PUT("/:id/priority", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.UpdatePriority)
					rules.PUT("/:id/enabled", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.SetEnabled)
				}

				// 客户费率-执行同步
				sync := rates.Group("/sync")
				{
					sync.POST("/execute", authMW.PermissionRequired("rates.sync.execute"), ratesSyncController.Execute)
				}
			}

			// 业务对象（归属结算系统）
			entities := settlement.Group("/entities")
			{
				entities.GET("", authMW.PermissionRequired("entities.read"), entitiesController.ListEntities)
				entities.POST("", authMW.PermissionRequired("entities.write"), entitiesController.CreateEntity)
				entities.PUT("/:id", authMW.PermissionRequired("entities.write"), entitiesController.UpdateEntity)
				entities.DELETE("/:id", authMW.PermissionRequired("entities.write"), entitiesController.DeleteEntity)
			}

			// 业务类型管理（归属结算系统）
			bt := settlement.Group("/business-types")
			{
				bt.GET("", authMW.PermissionRequired("business_types.read"), btController.List)
				bt.POST("", authMW.PermissionRequired("business_types.write"), btController.Create)
				bt.PUT("/:id", authMW.PermissionRequired("business_types.write"), btController.Update)
				bt.DELETE("/:id", authMW.PermissionRequired("business_types.write"), btController.Delete)
			}
		}

		// 系统管理接口（需要登录）
		system := api.Group("/system", authMW.AuthRequired())
		{
			// 角色管理（需要 system.role.manage）
			roles := system.Group("/roles", authMW.PermissionRequired("system.role.manage"))
			{
				roles.GET("", roleController.ListRoles)
				roles.POST("", roleController.CreateRole)
				roles.PUT("/:id", roleController.UpdateRole)
				roles.DELETE("/:id", roleController.DeleteRole)
				roles.GET("/:id/permissions", roleController.GetRolePermissions)
				roles.PUT("/:id/permissions", roleController.SetRolePermissions)
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
				users.PUT("/:id/status", systemUserController.UpdateUserStatus)
				users.PUT("/:id/roles", systemUserController.SetUserRoles)
				users.PUT("/:id/alias", systemUserController.UpdateUserAlias)
			}

			// 用户-院校绑定（需要 system.user.manage）
			system.POST("/user-schools/owner", authMW.PermissionRequired("system.user.manage"), userSchoolController.SetOwner)

			// 绑定配置查询（需要 system.user.manage）
			system.GET("/binding/allowed-user-roles", authMW.PermissionRequired("system.user.manage"), bindingController.GetAllowedUserRoles)

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
