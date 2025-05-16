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

	// 创建依赖
	schoolRepo := repository.NewSchoolRepository()
	schoolService := service.NewSchoolService(schoolRepo)
	schoolController := controller.NewSchoolController(schoolService)

	// 创建结算系统依赖
	settlementRepo := repository.NewSettlementRepository()
	settlementService := service.NewSettlementService(settlementRepo)
	settlementController := controller.NewSettlementController(settlementService)

	// 创建并启动结算调度器
	settlementScheduler := scheduler.NewSettlementScheduler(settlementService)
	settlementScheduler.Start()

	// API路由
	api := r.Group("/api/v1")
	{
		// 学校相关接口
		api.GET("/schools", schoolController.GetAllSchools)
		api.GET("/regions", schoolController.GetAllRegions)
		api.GET("/cps", schoolController.GetAllCPs)
		api.GET("/traffic", schoolController.GetTrafficData)
		api.GET("/traffic/summary", schoolController.GetTrafficSummary)

		// 结算系统相关接口
		settlement := api.Group("/settlement")
		{
			// 结算配置相关接口
			settlement.GET("/config", settlementController.GetSettlementConfig)
			settlement.PUT("/config", settlementController.UpdateSettlementConfig)

			// 结算任务相关接口
			settlement.GET("/tasks", settlementController.GetSettlementTasks)
			settlement.GET("/tasks/:id", settlementController.GetSettlementTaskByID)
			settlement.POST("/tasks/daily", settlementController.CreateDailySettlementTask)
			settlement.POST("/tasks/weekly", settlementController.CreateWeeklySettlementTask)
			settlement.DELETE("/tasks/:id", settlementController.DeleteSettlementTask)

			// 结算数据相关接口
			settlement.GET("/data", settlementController.GetSettlements)
		}
	}

	// 启动服务器
	port := config.AppConfig.Server.Port
	log.Printf("服务器启动在 http://localhost:%d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
