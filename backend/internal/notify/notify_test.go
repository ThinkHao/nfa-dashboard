package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeishuNotifierSendsPayload(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewFromConfig(srv.URL)
	if err := n.Send("测试标题", "测试正文"); err != nil {
		t.Fatal(err)
	}
	if got["msg_type"] != "text" {
		t.Fatalf("unexpected payload: %+v", got)
	}
	content, _ := got["content"].(map[string]interface{})
	text, _ := content["text"].(string)
	if !strings.Contains(text, "测试标题") || !strings.Contains(text, "测试正文") {
		t.Fatalf("text missing title/body: %s", text)
	}
}

func TestNoopNotifierWhenURLEmpty(t *testing.T) {
	if err := NewFromConfig("").Send("a", "b"); err != nil {
		t.Fatalf("noop notifier should never fail: %v", err)
	}
}

func TestFeishuNotifierRejectsBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	if err := NewFromConfig(srv.URL).Send("a", "b"); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}
