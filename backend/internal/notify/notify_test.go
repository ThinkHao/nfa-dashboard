package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFeishuNotifierSendsPayload(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
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

func TestFeishuNotifierRejectsBusinessError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":19024,"msg":"key words not found"}`))
	}))
	defer srv.Close()
	err := NewFromConfig(srv.URL).Send("a", "b")
	if err == nil {
		t.Fatal("expected error on feishu business error body")
	}
	if !strings.Contains(err.Error(), "19024") || !strings.Contains(err.Error(), "key words not found") {
		t.Fatalf("error should contain code and msg: %v", err)
	}
}

type fakeNotifier struct {
	done chan struct{}
}

func (f *fakeNotifier) Send(title, text string) error {
	close(f.done)
	return nil
}

func TestSendAsyncNilNotifier(t *testing.T) {
	SendAsync(nil, "a", "b") // 不应 panic
}

func TestSendAsyncDelivers(t *testing.T) {
	f := &fakeNotifier{done: make(chan struct{})}
	SendAsync(f, "a", "b")
	select {
	case <-f.done:
	case <-time.After(2 * time.Second):
		t.Fatal("SendAsync did not deliver within 2s")
	}
}
