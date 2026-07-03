package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// writeMinimalConfigYAML 写入仅含合法 database 段的临时配置文件
// （LoadConfig 对不完整的 database 配置会 log.Fatalf，因此必须提供）
func writeMinimalConfigYAML(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `database:
  host: 127.0.0.1
  port: 3306
  username: test
  password: test
  dbname: test
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

// snapshotGlobals 保存并在测试结束后恢复包级全局状态（AppConfig 与 viper 单例），
// 避免测试之间互相泄漏
func snapshotGlobals(t *testing.T) {
	t.Helper()
	saved := AppConfig
	t.Cleanup(func() {
		AppConfig = saved
		viper.Reset()
	})
	viper.Reset()
}

func TestLoadConfigSchedulerAlertDefaults(t *testing.T) {
	snapshotGlobals(t)
	t.Setenv("APP_CONFIG", writeMinimalConfigYAML(t))
	// 显式置空，防止外部环境残留影响断言（viper 默认忽略空值 env）
	t.Setenv("SCHEDULER_ENABLED", "")
	t.Setenv("ALERT_FEISHU_WEBHOOK_URL", "")

	LoadConfig()

	if !IsSchedulerEnabled() {
		t.Fatal("IsSchedulerEnabled should default to true when yaml lacks scheduler section")
	}
	if got := GetFeishuWebhookURL(); got != "" {
		t.Fatalf("GetFeishuWebhookURL should default to empty, got %q", got)
	}
}

func TestLoadConfigSchedulerAlertEnvOverrides(t *testing.T) {
	snapshotGlobals(t)
	t.Setenv("APP_CONFIG", writeMinimalConfigYAML(t))
	t.Setenv("SCHEDULER_ENABLED", "false")
	t.Setenv("ALERT_FEISHU_WEBHOOK_URL", "https://example.com/hook")

	LoadConfig()

	if IsSchedulerEnabled() {
		t.Fatal("IsSchedulerEnabled should be false when SCHEDULER_ENABLED=false")
	}
	if got := GetFeishuWebhookURL(); got != "https://example.com/hook" {
		t.Fatalf("GetFeishuWebhookURL mismatch, got %q", got)
	}
}
