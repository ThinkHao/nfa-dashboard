package bootstrap

import (
	"log"

	"nfa-dashboard/config"
	"nfa-dashboard/internal/controller"
	"nfa-dashboard/internal/middleware"
	"nfa-dashboard/internal/notify"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/scheduler"
	"nfa-dashboard/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

func BuildEngine() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.SecurityHeaders())
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
	edcRepo := repository.NewEDCRepository()
	edcScopeRepo := repository.NewEDCTrafficScopeRepository()

	settlementRepo := repository.NewSettlementRepository()
	settlementDataRepo := repository.NewSettlementDataRepository()
	notifier := notify.NewFromConfig(config.GetFeishuWebhookURL())
	settlementService := service.NewSettlementService(settlementRepo, settlementDataRepo, notifier)

	settlementController := controller.NewSettlementController(settlementService)

	entitiesRepo := repository.NewEntitiesRepository()
	userRepo := repository.NewUserRepository()

	settlementDataService := service.NewSettlementDataService(settlementDataRepo, userRepo, entitiesRepo, settlementRepo)
	settlementDataController := controller.NewSettlementDataController(settlementDataService)

	ratesRepo := repository.NewRatesRepository()
	rateDiscountRepo := repository.NewRateDiscountRepository()
	ratesSvc := service.NewRatesService(ratesRepo, rateDiscountRepo, userRepo)
	ratesController := controller.NewSettlementRatesController(ratesSvc, settlementRepo)
	edcNodeSettlementRepo := repository.NewEDCNodeSettlementRepository()
	edcNodeSettlementSvc := service.NewEDCNodeSettlementService(edcNodeSettlementRepo, ratesRepo, settlementRepo)
	edcNodeSettlementController := controller.NewEDCNodeSettlementController(settlementService, edcNodeSettlementSvc)

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
	edcScopeService := service.NewEDCTrafficScopeService(edcScopeRepo, userRepo)
	edcService := service.NewEDCService(edcRepo)
	schoolController := controller.NewSchoolController(schoolService, trafficScopeService, systemSettingsSvc, settlementParticipationSvc)
	edcController := controller.NewEDCController(edcService, edcScopeService)
	trafficScopeController := controller.NewSystemTrafficScopeController(trafficScopeService, userService, schoolService)
	edcTrafficScopeController := controller.NewSystemEDCTrafficScopeController(edcScopeService, userService)
	systemSettingsController := controller.NewSystemSettingsController(systemSettingsSvc)

	opLogRepo := repository.NewOperationLogRepository()
	opLogService := service.NewOperationLogService(opLogRepo)
	opLogController := controller.NewOperationLogController(opLogService)

	settlementScheduler := scheduler.NewSettlementScheduler(settlementService, edcNodeSettlementSvc, notifier)
	if config.IsSchedulerEnabled() {
		settlementScheduler.Start()
	} else {
		log.Println("scheduler.enabled=false，本实例不启动结算调度器")
	}

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", middleware.LoginRateLimit(), authController.Login)
			auth.POST("/refresh", authController.Refresh)
			auth.GET("/profile", authMW.AuthRequired(), authController.Profile)
			auth.POST("/change-password", authMW.AuthRequired(), authController.ChangePassword)
		}

		v2 := r.Group("/api/v2")
		{
			v2.GET("/schools", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllSchoolsV2)
			v2.GET("/regions", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllRegionsV2)
			v2.GET("/cps", authMW.AuthRequired(), authMW.PermissionRequired("school.read"), schoolController.GetAllCPsV2)
			v2.GET("/traffic", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficDataV2)
			v2.GET("/traffic/summary", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), schoolController.GetTrafficSummaryV2)
			v2.GET("/edc/entities", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), edcController.ListEntities)
			v2.GET("/edc/regions", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), edcController.ListRegions)
			v2.GET("/edc/cps", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), edcController.ListCPs)
			v2.GET("/edc/traffic", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), edcController.GetTrafficData)
			v2.GET("/edc/traffic/summary", authMW.AuthRequired(), authMW.PermissionRequired("traffic.read"), edcController.GetTrafficSummary)

			settlementV2 := v2.Group("/settlement", authMW.AuthRequired())
			{
				settlementV2.GET("/data", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlementsV2)
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
			settlement.POST("/tasks/node-daily95", authMW.PermissionRequired("settlement.calculate"), edcNodeSettlementController.CreateNodeDailyTask)
			settlement.POST("/tasks/node-monthly95", authMW.PermissionRequired("settlement.calculate"), edcNodeSettlementController.CreateNodeMonthlyTask)
			settlement.DELETE("/tasks/:id", authMW.PermissionRequired("settlement.calculate"), settlementController.DeleteSettlementTask)

			settlement.GET("/data", authMW.PermissionRequired("settlement.read"), settlementController.GetSettlements)

			settlement.GET("/data/customer", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListCustomerData)
			settlement.GET("/data/customer/monthly", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListCustomerMonthlyData)
			settlement.POST("/data/customer/monthly/rebuild", authMW.PermissionRequired("settlement.data.recalculate"), settlementDataController.RebuildMonthlyData)
			settlement.GET("/data/customer/export", authMW.PermissionRequired("settlement.data.export"), settlementDataController.ExportCustomerData)
			settlement.POST("/data/customer/recalculate", authMW.PermissionRequired("settlement.data.recalculate"), settlementDataController.RecalculateCustomerData)
			settlement.GET("/data/customer/owners", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListUsedOwnerEntities)
			settlement.GET("/data/customer/channel-owners", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListUsedChannelOwners)
			settlement.GET("/data/customer/owner-subjects", authMW.PermissionRequired("settlement.data.read"), settlementDataController.ListUsedOwnerSubjects)
			settlement.GET("/data/node", authMW.PermissionRequired("settlement.data.read"), edcNodeSettlementController.ListNodeDaily)
			settlement.GET("/data/node/monthly", authMW.PermissionRequired("settlement.data.read"), edcNodeSettlementController.ListNodeMonthly)

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
				rates.GET("/node-groups", authMW.PermissionRequired("rates.node.read"), ratesController.ListNodeSettlementGroups)
				rates.POST("/node-groups", authMW.PermissionRequired("rates.node.write"), ratesController.CreateNodeSettlementGroup)
				rates.PUT("/node-groups/:id", authMW.PermissionRequired("rates.node.write"), ratesController.UpdateNodeSettlementGroup)
				rates.DELETE("/node-groups/:id", authMW.PermissionRequired("rates.node.write"), ratesController.DisableNodeSettlementGroup)
				rates.GET("/final-node", authMW.PermissionRequired("rates.node.read"), ratesController.ListFinalNodeRates)
				rates.POST("/final-node", authMW.PermissionRequired("rates.node.write"), ratesController.UpsertFinalNodeRate)
				rates.POST("/final-node/init-from-node", authMW.PermissionRequired("rates.node.write"), ratesController.InitFinalNodeRatesFromNode)
				rates.POST("/final-node/refresh", authMW.PermissionRequired("rates.node.write"), ratesController.RefreshFinalNodeRates)
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
			system.GET("/edc-traffic-scopes/:user_id", authMW.PermissionRequired("traffic.scope.manage"), edcTrafficScopeController.ListRules)
			system.PUT("/edc-traffic-scopes/:user_id", authMW.PermissionRequired("traffic.scope.manage"), edcTrafficScopeController.ReplaceRules)
			system.GET("/settings/traffic", authMW.PermissionRequired("system.user.manage"), systemSettingsController.GetTrafficSettings)
			system.PUT("/settings/traffic", authMW.PermissionRequired("system.user.manage"), systemSettingsController.UpdateTrafficSettings)
			system.GET("/operation-logs", authMW.PermissionRequired("operation_logs.read"), opLogController.List)
			system.GET("/operation-logs/export", authMW.PermissionRequired("operation_logs.read"), opLogController.Export)
		}
	}

	return r
}
