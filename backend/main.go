package main

import (
	"fmt"
	"log"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/bootstrap"
	"nfa-dashboard/internal/model"
)

func main() {
	config.LoadConfig()
	model.InitDB()

	r := bootstrap.BuildEngine()
	port := config.AppConfig.Server.Port
	log.Printf("服务器启动在 http://localhost:%d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
