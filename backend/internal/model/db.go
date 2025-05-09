package model

import (
	"log"
	"nfa-dashboard/config"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() {
	var err error

	// 配置GORM日志
	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)

	// 连接数据库
	DB, err = gorm.Open(mysql.Open(config.GetDSN()), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	log.Println("数据库连接成功")

	// 注意：我们使用的是现有数据库表，不需要自动迁移模型
	// 如果需要验证表结构，可以使用以下代码
	// err = DB.Migrator().HasTable(&School{})
	// if !err {
	// 	log.Fatalf("数据库表 nfa_school 不存在")
	// }
	// err = DB.Migrator().HasTable(&SchoolTraffic{})
	// if !err {
	// 	log.Fatalf("数据库表 nfa_school_traffic 不存在")
	// }
}
