package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Binding  BindingConfig  `mapstructure:"binding"`
	RatesOwnerRoles RatesOwnerRolesConfig `mapstructure:"rates_owner_roles"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AuthConfig struct {
	Secret                  string `mapstructure:"secret"`
	AccessTokenTTLMinutes   int    `mapstructure:"access_token_ttl_minutes"`
	RefreshTokenTTLMinutes  int    `mapstructure:"refresh_token_ttl_minutes"`
}

type BindingConfig struct {
	// 新字段：客户费归属（销售）可选的系统用户角色名
	AllowedSalesRoles []string `mapstructure:"allowed_sales_roles"`
	// 新字段：线路费归属可选的系统用户角色名
	AllowedLineRoles  []string `mapstructure:"allowed_line_roles"`
}

// RatesOwnerRolesConfig 控制费率页面“归属”下拉可选角色
type RatesOwnerRolesConfig struct {
	CustomerFee     []string `mapstructure:"customer_fee"`
	NetworkLineFee  []string `mapstructure:"network_line_fee"`
}

var AppConfig Config

func LoadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode config into struct: %s", err)
	}

	log.Println("配置加载成功")
}

func GetDSN() string {
	db := AppConfig.Database
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db.Username, db.Password, db.Host, db.Port, db.DBName)
}

func GetJWTSecret() string {
	if AppConfig.Auth.Secret == "" {
		return "dev-secret-change-me"
	}
	return AppConfig.Auth.Secret
}

func GetAccessTokenTTLMinutes() int {
	if AppConfig.Auth.AccessTokenTTLMinutes <= 0 {
		return 60
	}
	return AppConfig.Auth.AccessTokenTTLMinutes
}

func GetRefreshTokenTTLMinutes() int {
	if AppConfig.Auth.RefreshTokenTTLMinutes <= 0 {
		return 43200
	}
	return AppConfig.Auth.RefreshTokenTTLMinutes
}

// 新增：分别获取销售与线路的角色白名单
func GetAllowedSalesRoles() []string { return AppConfig.Binding.AllowedSalesRoles }
func GetAllowedLineRoles() []string  { return AppConfig.Binding.AllowedLineRoles }

// GetOwnerRoles returns allowed role names for a specific owner type on rates page
// t: "customer_fee" | "network_line_fee"
func GetOwnerRoles(t string) []string {
	switch t {
	case "customer_fee":
		return AppConfig.RatesOwnerRoles.CustomerFee
	case "network_line_fee":
		return AppConfig.RatesOwnerRoles.NetworkLineFee
	default:
		return nil
	}
}
