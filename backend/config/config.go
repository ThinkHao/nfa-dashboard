package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server          ServerConfig          `mapstructure:"server"`
	Database        DatabaseConfig        `mapstructure:"database"`
	Redis           RedisConfig           `mapstructure:"redis"`
	Auth            AuthConfig            `mapstructure:"auth"`
	Binding         BindingConfig         `mapstructure:"binding"`
	RatesOwnerRoles RatesOwnerRolesConfig `mapstructure:"rates_owner_roles"`
	Scheduler       SchedulerConfig       `mapstructure:"scheduler"`
	Alert           AlertConfig           `mapstructure:"alert"`
}

// SchedulerConfig 结算调度器开关（多实例部署时只允许一个实例开启，或全部开启依赖 DB 锁互斥）
type SchedulerConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// AlertConfig 告警通道配置
type AlertConfig struct {
	FeishuWebhookURL string `mapstructure:"feishu_webhook_url"`
}

type ServerConfig struct {
	Port     int                  `mapstructure:"port"`
	TLS      ServerTLSConfig      `mapstructure:"tls"`
	Security ServerSecurityConfig `mapstructure:"security"`
}

type ServerTLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type ServerSecurityConfig struct {
	CORS           CORSConfig           `mapstructure:"cors"`
	LoginRateLimit LoginRateLimitConfig `mapstructure:"login_rate_limit"`
}

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

type LoginRateLimitConfig struct {
	Enabled     bool `mapstructure:"enabled"`
	MaxAttempts int  `mapstructure:"max_attempts"`
	WindowSecs  int  `mapstructure:"window_secs"`
	BlockSecs   int  `mapstructure:"block_secs"`
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
	Secret                 string `mapstructure:"secret"`
	AccessTokenTTLMinutes  int    `mapstructure:"access_token_ttl_minutes"`
	RefreshTokenTTLMinutes int    `mapstructure:"refresh_token_ttl_minutes"`
}

type BindingConfig struct {
	// 新字段：客户费归属（销售）可选的系统用户角色名
	AllowedSalesRoles []string `mapstructure:"allowed_sales_roles"`
	// 新字段：线路费归属可选的系统用户角色名
	AllowedLineRoles    []string `mapstructure:"allowed_line_roles"`
	AllowedNodeRoles    []string `mapstructure:"allowed_node_roles"`
	AllowedChannelRoles []string `mapstructure:"allowed_channel_roles"`
}

// RatesOwnerRolesConfig 控制费率页面“归属”下拉可选角色
type RatesOwnerRolesConfig struct {
	CustomerFee    []string `mapstructure:"customer_fee"`
	NetworkLineFee []string `mapstructure:"network_line_fee"`
}

var AppConfig Config

func LoadConfig() {
	viper.SetConfigType("yaml")
	if cfg := os.Getenv("APP_CONFIG"); cfg != "" {
		viper.SetConfigFile(cfg)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("./config")
	}

	viper.SetDefault("server.port", 8081)
	viper.SetDefault("scheduler.enabled", true)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	_ = viper.BindEnv("server.port", "APP_PORT", "NFA_SERVER_PORT")
	_ = viper.BindEnv("database.host", "DB_HOST", "NFA_DB_HOST")
	_ = viper.BindEnv("database.port", "DB_PORT", "NFA_DB_PORT")
	_ = viper.BindEnv("database.username", "DB_USER", "NFA_DB_USER")
	_ = viper.BindEnv("database.password", "DB_PASS", "NFA_DB_PASS")
	_ = viper.BindEnv("database.dbname", "DB_NAME", "NFA_DB_NAME")
	_ = viper.BindEnv("server.tls.enabled", "SERVER_TLS_ENABLED")
	_ = viper.BindEnv("server.tls.cert_file", "SERVER_TLS_CERT_FILE")
	_ = viper.BindEnv("server.tls.key_file", "SERVER_TLS_KEY_FILE")
	_ = viper.BindEnv("server.security.cors.allowed_origins", "CORS_ALLOWED_ORIGINS")
	_ = viper.BindEnv("server.security.cors.allow_credentials", "CORS_ALLOW_CREDENTIALS")
	_ = viper.BindEnv("server.security.login_rate_limit.enabled", "LOGIN_RATE_LIMIT_ENABLED")
	_ = viper.BindEnv("server.security.login_rate_limit.max_attempts", "LOGIN_RATE_LIMIT_MAX_ATTEMPTS")
	_ = viper.BindEnv("server.security.login_rate_limit.window_secs", "LOGIN_RATE_LIMIT_WINDOW_SECS")
	_ = viper.BindEnv("server.security.login_rate_limit.block_secs", "LOGIN_RATE_LIMIT_BLOCK_SECS")
	// Auth via env
	_ = viper.BindEnv("auth.secret", "AUTH_SECRET")
	_ = viper.BindEnv("auth.access_token_ttl_minutes", "AUTH_ACCESS_TOKEN_TTL_MINUTES")
	_ = viper.BindEnv("auth.refresh_token_ttl_minutes", "AUTH_REFRESH_TOKEN_TTL_MINUTES")
	_ = viper.BindEnv("scheduler.enabled", "SCHEDULER_ENABLED")
	_ = viper.BindEnv("alert.feishu_webhook_url", "ALERT_FEISHU_WEBHOOK_URL")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("config file not found, using env only: %v", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode config into struct: %s", err)
	}

	// Apply list-type env overrides (comma-separated)
	applyEnvOverridesFromEnv()

	if err := validateAndSetDefaults(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
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

// validateAndSetDefaults validates essential configuration and applies sane defaults.
func validateAndSetDefaults() error {
	// Default port safeguard (in case env binding/unmarshal didn't set it)
	if AppConfig.Server.Port == 0 {
		AppConfig.Server.Port = 8081
	}
	if len(AppConfig.Server.Security.CORS.AllowedOrigins) == 0 {
		AppConfig.Server.Security.CORS.AllowedOrigins = []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
	}
	if AppConfig.Server.Security.LoginRateLimit.MaxAttempts <= 0 {
		AppConfig.Server.Security.LoginRateLimit.MaxAttempts = 8
	}
	if AppConfig.Server.Security.LoginRateLimit.WindowSecs <= 0 {
		AppConfig.Server.Security.LoginRateLimit.WindowSecs = 600
	}
	if AppConfig.Server.Security.LoginRateLimit.BlockSecs <= 0 {
		AppConfig.Server.Security.LoginRateLimit.BlockSecs = 900
	}
	if !AppConfig.Server.Security.LoginRateLimit.Enabled {
		AppConfig.Server.Security.LoginRateLimit.Enabled = true
	}
	if AppConfig.Server.TLS.Enabled {
		if strings.TrimSpace(AppConfig.Server.TLS.CertFile) == "" || strings.TrimSpace(AppConfig.Server.TLS.KeyFile) == "" {
			return fmt.Errorf("server.tls is enabled but cert_file/key_file is missing")
		}
	}

	// Database required fields
	db := AppConfig.Database
	if db.Host == "" || db.Port == 0 || db.Username == "" || db.Password == "" || db.DBName == "" {
		return fmt.Errorf("incomplete database config")
	}
	return nil
}

func GetCORSAllowedOrigins() []string {
	return AppConfig.Server.Security.CORS.AllowedOrigins
}

func GetCORSAllowCredentials() bool {
	return AppConfig.Server.Security.CORS.AllowCredentials
}

func IsLoginRateLimitEnabled() bool {
	return AppConfig.Server.Security.LoginRateLimit.Enabled
}

func GetLoginRateLimitMaxAttempts() int {
	return AppConfig.Server.Security.LoginRateLimit.MaxAttempts
}

func GetLoginRateLimitWindowSecs() int {
	return AppConfig.Server.Security.LoginRateLimit.WindowSecs
}

func GetLoginRateLimitBlockSecs() int {
	return AppConfig.Server.Security.LoginRateLimit.BlockSecs
}

// applyEnvOverridesFromEnv overrides list-type configs from comma-separated env vars.
func applyEnvOverridesFromEnv() {
	if v := strings.TrimSpace(os.Getenv("BINDING_ALLOWED_SALES_ROLES")); v != "" {
		AppConfig.Binding.AllowedSalesRoles = splitCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("BINDING_ALLOWED_LINE_ROLES")); v != "" {
		AppConfig.Binding.AllowedLineRoles = splitCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("BINDING_ALLOWED_NODE_ROLES")); v != "" {
		AppConfig.Binding.AllowedNodeRoles = splitCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("BINDING_ALLOWED_CHANNEL_ROLES")); v != "" {
		AppConfig.Binding.AllowedChannelRoles = splitCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("RATES_OWNER_ROLES_CUSTOMER_FEE")); v != "" {
		AppConfig.RatesOwnerRoles.CustomerFee = splitCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("RATES_OWNER_ROLES_NETWORK_LINE_FEE")); v != "" {
		AppConfig.RatesOwnerRoles.NetworkLineFee = splitCSV(v)
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// 新增：分别获取销售与线路的角色白名单
func GetAllowedSalesRoles() []string   { return AppConfig.Binding.AllowedSalesRoles }
func GetAllowedLineRoles() []string    { return AppConfig.Binding.AllowedLineRoles }
func GetAllowedNodeRoles() []string    { return AppConfig.Binding.AllowedNodeRoles }
func GetAllowedChannelRoles() []string { return AppConfig.Binding.AllowedChannelRoles }

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

// IsSchedulerEnabled 是否在本实例启动结算调度器
func IsSchedulerEnabled() bool {
	return AppConfig.Scheduler.Enabled
}

// GetFeishuWebhookURL 飞书告警 webhook 地址，空串表示未配置
func GetFeishuWebhookURL() string {
	return AppConfig.Alert.FeishuWebhookURL
}
