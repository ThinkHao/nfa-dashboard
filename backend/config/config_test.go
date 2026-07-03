package config

import "testing"

func TestSchedulerAndAlertGetters(t *testing.T) {
	AppConfig = Config{}
	AppConfig.Scheduler.Enabled = true
	AppConfig.Alert.FeishuWebhookURL = "https://example.com/hook"
	if !IsSchedulerEnabled() {
		t.Fatal("IsSchedulerEnabled should be true")
	}
	if GetFeishuWebhookURL() != "https://example.com/hook" {
		t.Fatal("GetFeishuWebhookURL mismatch")
	}
}
