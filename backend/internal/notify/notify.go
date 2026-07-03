package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Notifier 告警通知接口
type Notifier interface {
	Send(title, text string) error
}

// NewFromConfig 根据 webhook URL 创建通知器；URL 为空时返回 no-op 实现
func NewFromConfig(webhookURL string) Notifier {
	if webhookURL == "" {
		return noopNotifier{}
	}
	return &feishuNotifier{url: webhookURL, client: &http.Client{Timeout: 5 * time.Second}}
}

type noopNotifier struct{}

func (noopNotifier) Send(title, text string) error { return nil }

type feishuNotifier struct {
	url    string
	client *http.Client
}

func (n *feishuNotifier) Send(title, text string) error {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content":  map[string]string{"text": fmt.Sprintf("【NFA结算告警】%s\n%s", title, text)},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := n.client.Post(n.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook 返回状态码 %d", resp.StatusCode)
	}
	return nil
}

// SendAsync 异步发送并记录错误，避免阻塞结算主流程
func SendAsync(n Notifier, title, text string) {
	if n == nil {
		return
	}
	go func() {
		if err := n.Send(title, text); err != nil {
			log.Printf("发送告警失败: title=%s err=%v", title, err)
		}
	}()
}
