package bootstrap

import (
	"nfa-dashboard/internal/controller"
	"nfa-dashboard/internal/middleware"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/scheduler"
	"nfa-dashboard/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

func BuildEngine() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.Gzip())
	r.Use(middleware.Audit())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	schoolRepo := repository.NewSchoolRepository()
	schoolService := service.NewSchoolService(schoolRepo)
	trafficScopeRuleRepo := repository.NewTrafficScopeRuleRepository()
	trafficScopeSchoolRepo := repository.NewTrafficScopeSchoolRepository()

	settlementRepo := repository.NewSettlementRepository()
	settlementService := service.NewSettlementService(settlementRepo)

	formulaRepo := repository.NewSettlementFormulaRepository()
	formulaService := service.NewSettlementFormulaService(formulaRepo)
	formulaController := controller.NewSettlementFormulaController(formulaService)

	settlementResultRepo := repository.NewSettlementResultRepository()
	settlementResultService := service.NewSettlementResultService(settlementResultRepo, formulaRepo)
	settlementController := controller.NewSettlementController(settlementService, settlementResultService)

	entitiesRepo := repository.NewEntitiesRepository()
	userRepo := repository.NewUserRepository()

	settlementDataRepo := repository.NewSettlementDataRepository()
	settlementDataService := service.NewSettlementDataService(settlementDataRepo, userRepo, entitiesRepo, settlementRepo)
	settlementDataController := controller.NewSettlementDataController(settlementDataService)

	ratesRepo := repository.NewRatesRepository()
	rateDiscountRepo := repository.NewRateDiscountRepository()
	ratesSvc := service.NewRatesService(ratesRepo, rateDiscountRepo, userRepo)
	ratesController := controller.NewSettlementRatesController(ratesSvc, settlementRepo)

	customerFieldsRepo := repository.NewCustomerFieldsRepository()
	customerFieldsSvc := service.NewCustomerFieldsService(customerFieldsRepo)
	customerFieldsController := controller.NewCustomerFieldsController(customerFieldsSvc)

	filterRulesRepo := repository.NewFilterRulesRepository()
	systemSettingsRepo := repository.NewSystemSettingsRepository()
	systemSettingsSvc := service.NewSystemSettingsService(systemSettingsRepo)
	settlementParticipationSvc := service.NewSettlementParticipationService(trafficScopeSchoolRepo, filterRulesRepo, 2*time.Minute)
	filterRulesSvc := service.NewFilterRulesService(filterRulesRepo)
	filterRulesController := controller.NewFilterRulesController(filterRulesSvc, settlementParticipationSvc)

	syncRulesRepo := repository.NewSyncRulesRepository()
	syncRulesSvc := service.NewSyncRulesService(syncRulesRepo)
	syncRulesController := controller.NewSyncRulesController(syncRulesSvc)

	ratesSyncSvc := service.NewRatesSyncService(syncRulesRepo, ratesRepo, schoolRepo)
	ratesSyncController := controller.NewRatesSyncController(ratesSyncSvc)

	rateDiscountSvc := service.NewRateDiscountService(rateDiscountRepo)
	rateDiscountController := controller.NewRateDiscountController(rateDiscountSvc)

	btRepo := repository.NewBusinessTypeRepository()
	btService := service.NewBusinessTypeService(btRepo)
	btController := controller.NewBusinessTypeController(btService)
	entitiesSvc := service.NewEntitiesService(entitiesRepo, btRepo)
	entitiesController := controller.NewSettlementEntitiesController(entitiesSvc)

	authService := service.NewAuthService(userRepo)
	authController := controller.NewAuthController(authService)
	authMW := middleware.NewAuthMiddleware(authService)

	roleRepo := repository.NewRoleRepository()
	permRepo := repository.NewPermissionRepository()

	roleService := service.NewRoleService(roleRepo, permRepo)
	roleController := controller.NewSystemRoleController(roleService)

	permService := service.NewPermissionService(permRepo)
	permController := controller.NewSystemPermissionController(permService)
	bindingController := controller.NewSystemBindingController()

	userService := service.NewUserService(userRepo, roleRepo)
	systemUserController := controller.NewSystemUserController(userService)

	userSchoolRepo := repository.NewUserSchoolRepository()
	userSchoolService := service.NewUserSchoolService(userRepo, schoolRepo, userSchoolRepo)
	userSchoolController := controller.NewSystemUserSchoolController(userSchoolService)
	trafficScopeService := service.NewTrafficScopeService(trafficScopeRuleRepo, trafficScopeSchoolRepo, userSchoolRepo, userRepo)
	schoolController := controller.NewSchoolController(schoolService, trafficScopeService, systemSettingsSvc, settlementParticipationSvc)
	trafficScopeController := controller.NewSystemTrafficScopeController(trafficScopeService, userService, schoolService)
	systemSettingsController := controller.NewSystemSettingsController(systemSettingsSvc)

	opLogRepo := repository.NewOperationLogRepository()
	opLogService := service.NewOperationLogService(opLogRepo)
	opLogController := controller.NewOperationLogController(opLogService)

	settlementScheduler := scheduler.NewSettlementScheduler(settlementService)
	settlementScheduler.Start()

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/refresh", authController.Refresh)
			auth.GET("/profile", authMW.AuthRequired(), authController.Profile)
		}

		v2 := r.Group("/api/v2")
		{
			v2.GET("/schools", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllSchoolsV2)
			v2.GET("/regions", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllRegionsV2)
			v2.GET("/cps", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllCPsV2)
			v2.GET("/traffic", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficDataV2)
			v2.GET("/traffic/summary", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficSummaryV2)

			settlementV2 := v2.Group("/settlement", authMW.AuthRequired())
			{
				settlementV2.GET("/data", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementsV2)
				settlementV2.GET("/daily-details", authMW.PermissionRequired("settlement.read"), settlementController.GetDailySettlementDetailsV2)
			}
		}

		api.GET("/schools", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllSchools)
		api.GET("/regions", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllRegions)
		api.GET("/cps", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllCPs)
		api.GET("/traffic", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficData)
		api.GET("/traffic/summary", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficSummary)

		settlement := api.Group("/settlement", authMW.AuthRequired())
		{
			settlement.GET("/config", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementConfig)
			settlement.PUT("/config", authMW.PermissionRequired("settlement.calculate"), settlementController.UpdateSettlementConfig)

			settlement.GET("/tasks", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementTasks)
			settlement.GET("/tasks/:id", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementTaskByID)
			settlement.POST("/tasks/daily", authMW.PermissionRequired("settlement.calculate"), settlementController.CreateDailySettlementTask)
			settlement.POST("/tasks/weekly", authMW.PermissionRequired("settlement.calculate"), settlementController.CreateWeeklySettlementTask)
			settlement.DELETE("/tasks/:id", authMW.PermissionRequired("settlement.calculate"), settlementController.DeleteSettlementTask)

			settlement.GET("/data", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlements)
			settlement.GET("/daily-details", authMW.PermissionRequired("settlement.read"), settlementController.GetDailySettlementDetails)
			settlement.GET("/results", authMW.PermissionRequired("settlement.results.read"), settlementController.GetSettlementResults)
			settlement.GET("/results/channels", authMW.PermissionRequired("settlement.results.read"), settlementController.GetChannelSettlementResults)

			settlement.GET("/data/customer", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListCustomerData)
			settlement.GET("/data/customer/monthly", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListCustomerMonthlyData)
			settlement.POST("/data/customer/monthly/rebuild", authMW.PermissionRequired("settlement.data.recalculate"), settlementDataController.RebuildMonthlyData)
			settlement.GET("/data/customer/export", authMW.PermissionRequired("settlement.data.export"), settlementDataController.ExportCustomerData)
			settlement.POST("/data/customer/recalculate", authMW.PermissionRequired("settlement.data.recalculate"), settlementDataController.RecalculateCustomerData)
			settlement.GET("/data/customer/owners", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListUsedOwnerEntities)
			settlement.GET("/data/customer/channel-owners", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListUsedChannelOwners)
			settlement.GET("/data/customer/owner-subjects", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListUsedOwnerSubjects)

			formulas := settlement.Group("/formulas")
			{
				formulas.GET("", authMW.PermissionRequired("settlement.formula.read"), formulaController.List)
				formulas.GET("/:id", authMW.PermissionRequired("settlement.formula.read"), formulaController.Get)
				formulas.POST("", authMW.PermissionRequired("settlement.formula.write"), formulaController.Create)
				formulas.PUT("/:id", authMW.PermissionRequired("settlement.formula.write"), formulaController.Update)
				formulas.DELETE("/:id", authMW.PermissionRequired("settlement.formula.write"), formulaController.Delete)
			}

			rates := settlement.Group("/rates")
			{
				rates.GET("/customer", authMW.PermissionRequired("rates.customer.read"), ratesController.ListCustomerRates)
				rates.POST("/customer", authMW.PermissionRequired("rates.customer.write"), ratesController.UpsertCustomerRate)
				rates.GET("/customer/export", authMW.PermissionRequired("rates.customer.export"), ratesController.ExportCustomerRates)
				rates.GET("/customer/export-xlsx", authMW.PermissionRequired("rates.customer.export"), ratesController.ExportCustomerRatesXLSX)
				rates.GET("/customer/import-template", authMW.PermissionRequired("rates.customer.export"), ratesController.CustomerRatesImportTemplate)
				rates.POST("/customer/import", authMW.PermissionRequired("rates.customer.import"), ratesController.ImportCustomerRates)
				rates.POST("/customer/import/tasks", authMW.PermissionRequired("rates.customer.import"), ratesController.CreateCustomerImportTask)
				rates.GET("/customer/import/tasks/:id", authMW.PermissionRequired("rates.customer.import"), ratesController.GetCustomerImportTask)
				rates.POST("/customer/import/tasks/:id/continue", authMW.PermissionRequired("rates.customer.import"), ratesController.ContinueCustomerImportTask)
				rates.GET("/customer/import/tasks/:id/errors.csv", authMW.PermissionRequired("rates.customer.import"), ratesController.DownloadCustomerImportTaskErrorsCSV)
				rates.GET("/customer/import/tasks/:id/created-users.csv", authMW.PermissionRequired("rates.customer.import"), ratesController.DownloadCustomerImportTaskCreatedUsersCSV)
				rates.GET("/node", authMW.PermissionRequired("rates.node.read"), ratesController.ListNodeRates)
				rates.POST("/node", authMW.PermissionRequired("rates.node.write"), ratesController.UpsertNodeRate)
				rates.GET("/final", authMW.PermissionRequired("rates.final.read"), ratesController.ListFinalCustomerRates)
				rates.GET("/final-discounted", authMW.PermissionRequired("rates.final.read"), ratesController.ListFinalCustomerRatesDiscounted)
				rates.POST("/final", authMW.PermissionRequired("rates.final.write"), ratesController.UpsertFinalCustomerRate)
				rates.POST("/final/init-from-customer", authMW.PermissionRequired("rates.final.write"), ratesController.InitFinalCustomerRatesFromCustomer)
				rates.POST("/final/refresh", authMW.PermissionRequired("rates.final.write"), ratesController.RefreshFinalCustomerRates)
				rates.POST("/final/cleanup-invalid", authMW.PermissionRequired("rates.final.write"), ratesController.CleanupInvalidFinalCustomerRates)

				fields := rates.Group("/customer-fields")
				{
					fields.GET("", authMW.PermissionRequired("rates.customer_fields.read"), customerFieldsController.List)
					fields.POST("", authMW.PermissionRequired("rates.customer_fields.write"), customerFieldsController.Create)
					fields.PUT("/:id", authMW.PermissionRequired("rates.customer_fields.write"), customerFieldsController.Update)
					fields.DELETE("/:id", authMW.PermissionRequired("rates.customer_fields.write"), customerFieldsController.Delete)
				}

				filterRules := rates.Group("/filter-rules")
				{
					filterRules.GET("/options", authMW.PermissionRequired("rates.filter_rules.read"), filterRulesController.ListOptions)
					filterRules.GET("", authMW.PermissionRequired("rates.filter_rules.read"), filterRulesController.List)
					filterRules.POST("", authMW.PermissionRequired("rates.filter_rules.write"), filterRulesController.Create)
					filterRules.PUT("/:id", authMW.PermissionRequired("rates.filter_rules.write"), filterRulesController.Update)
					filterRules.DELETE("/:id", authMW.PermissionRequired("rates.filter_rules.write"), filterRulesController.Delete)
					filterRules.PUT("/:id/priority", authMW.PermissionRequired("rates.filter_rules.write"), filterRulesController.UpdatePriority)
					filterRules.PUT("/:id/enabled", authMW.PermissionRequired("rates.filter_rules.write"), filterRulesController.SetEnabled)
				}

				rules := rates.Group("/sync-rules")
				{
					rules.GET("/options", authMW.PermissionRequired("rates.sync_rules.read"), syncRulesController.ListOptions)
					rules.GET("", authMW.PermissionRequired("rates.sync_rules.read"), syncRulesController.List)
					rules.POST("", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.Create)
					rules.PUT("/:id", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.Update)
					rules.DELETE("/:id", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.Delete)
					rules.PUT("/:id/priority", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.UpdatePriority)
					rules.PUT("/:id/enabled", authMW.PermissionRequired("rates.sync_rules.write"), syncRulesController.SetEnabled)
				}

				discountRules := rates.Group("/discount-rules")
				{
					discountRules.GET("", authMW.PermissionRequired("rates.discount_rule.read"), rateDiscountController.List)
					discountRules.GET("/:id", authMW.PermissionRequired("rates.discount_rule.read"), rateDiscountController.Get)
					discountRules.POST("", authMW.PermissionRequired("rates.discount_rule.manage"), rateDiscountController.Create)
					discountRules.PUT("/:id", authMW.PermissionRequired("rates.discount_rule.manage"), rateDiscountController.Update)
					discountRules.DELETE("/:id", authMW.PermissionRequired("rates.discount_rule.manage"), rateDiscountController.Delete)
					discountRules.PUT("/:id/items", authMW.PermissionRequired("rates.discount_rule.manage"), rateDiscountController.ReplaceItems)
				}

				sync := rates.Group("/sync")
				{
					sync.POST("/execute", authMW.PermissionRequired("rates.sync.execute"), ratesSyncController.Execute)
				}
			}

			entities := settlement.Group("/entities")
			{
				entities.GET("", authMW.PermissionRequired("entities.read"), entitiesController.ListEntities)
				entities.POST("", authMW.PermissionRequired("entities.write"), entitiesController.CreateEntity)
				entities.PUT("/:id", authMW.PermissionRequired("entities.write"), entitiesController.UpdateEntity)
				entities.DELETE("/:id", authMW.PermissionRequired("entities.write"), entitiesController.DeleteEntity)
			}

			bt := settlement.Group("/business-types")
			{
				bt.GET("", authMW.PermissionRequired("business_types.read"), btController.List)
				bt.POST("", authMW.PermissionRequired("business_types.write"), btController.Create)
				bt.PUT("/:id", authMW.PermissionRequired("business_types.write"), btController.Update)
				bt.DELETE("/:id", authMW.PermissionRequired("business_types.write"), btController.Delete)
			}
		}

		system := api.Group("/system", authMW.AuthRequired())
		{
			roles := system.Group("/roles", authMW.PermissionRequired("system.role.manage"))
			{
				roles.GET("", roleController.ListRoles)
				roles.POST("", roleController.CreateRole)
				roles.PUT("/:id", roleController.UpdateRole)
				roles.DELETE("/:id", roleController.DeleteRole)
				roles.GET("/:id/permissions", roleController.GetRolePermissions)
				roles.PUT("/:id/permissions", roleController.SetRolePermissions)
			}

			system.GET("/permissions", authMW.PermissionRequired("system.role.manage"), permController.ListPermissions)
			system.POST("/permissions", authMW.PermissionRequired("system.permission.manage"), permController.CreatePermission)
			system.GET("/permissions/:id", authMW.PermissionRequired("system.permission.manage"), permController.GetPermission)
			system.PUT("/permissions/:id", authMW.PermissionRequired("system.permission.manage"), permController.UpdatePermission)
			system.DELETE("/permissions/:id", authMW.PermissionRequired("system.permission.manage"), permController.DisablePermission)
			system.POST("/permissions/sync", authMW.PermissionRequired("system.permission.manage"), permController.SyncPermissions)

			users := system.Group("/users", authMW.PermissionRequired("system.user.manage"))
			{
				users.POST("", systemUserController.CreateUser)
				users.GET("", systemUserController.ListUsers)
				users.PUT("/:id/status", systemUserController.UpdateUserStatus)
				users.PUT("/:id/roles", systemUserController.SetUserRoles)
				users.PUT("/:id/alias", systemUserController.UpdateUserAlias)
			}

			system.POST("/user-schools/owner", authMW.PermissionRequired("system.user.manage"), userSchoolController.SetOwner)
			system.GET("/binding/allowed-user-roles", authMW.PermissionRequired("system.user.manage"), bindingController.GetAllowedUserRoles)
			system.GET("/traffic-scopes/users", authMW.PermissionRequired("traffic.scope.manage"), trafficScopeController.ListUsers)
			system.GET("/traffic-scopes/options", authMW.PermissionRequired("traffic.scope.manage"), trafficScopeController.ListOptions)
			system.GET("/traffic-scopes/:user_id", authMW.PermissionRequired("traffic.scope.manage"), trafficScopeController.ListRules)
			system.PUT("/traffic-scopes/:user_id", authMW.PermissionRequired("traffic.scope.manage"), trafficScopeController.ReplaceRules)
			system.GET("/traffic-scopes/:user_id/preview", authMW.PermissionRequired("traffic.scope.manage"), trafficScopeController.Preview)
			system.GET("/settings/traffic", authMW.PermissionRequired("system.user.manage"), systemSettingsController.GetTrafficSettings)
			system.PUT("/settings/traffic", authMW.PermissionRequired("system.user.manage"), systemSettingsController.UpdateTrafficSettings)
			system.GET("/operation-logs", authMW.PermissionRequired("operation_logs.read"), opLogController.List)
			system.GET("/operation-logs/export", authMW.PermissionRequired("operation_logs.read"), opLogController.Export)
		}
	}

	return r
}
