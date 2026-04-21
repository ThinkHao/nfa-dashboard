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
	addr := fmt.Sprintf(":%d", port)
	if config.AppConfig.Server.TLS.Enabled {
		log.Printf("服务器启动在 https://localhost:%d", port)
		if err := r.RunTLS(addr, config.AppConfig.Server.TLS.CertFile, config.AppConfig.Server.TLS.KeyFile); err != nil {
			log.Fatalf("HTTPS服务器启动失败: %v", err)
		}
		return
	}
	log.Printf("服务器启动在 http://localhost:%d", port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
