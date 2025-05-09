package main

import (
	"fmt"
	"log"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/controller"
	"nfa-dashboard/internal/middleware"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
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

	// API路由
	api := r.Group("/api/v1")
	{
		// 学校相关接口
		api.GET("/schools", schoolController.GetAllSchools)
		api.GET("/regions", schoolController.GetAllRegions)
		api.GET("/cps", schoolController.GetAllCPs)
		api.GET("/traffic", schoolController.GetTrafficData)
		api.GET("/traffic/summary", schoolController.GetTrafficSummary)
	}

	// 启动服务器
	port := config.AppConfig.Server.Port
	log.Printf("服务器启动在 http://localhost:%d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
