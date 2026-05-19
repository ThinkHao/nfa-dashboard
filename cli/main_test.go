package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runTestCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("NFA_DASHBOARD_TOKEN", "")
	t.Setenv("NFA_DASHBOARD_REFRESH_TOKEN", "")
	t.Setenv("NFA_DASHBOARD_BASE_URL", "")
	cfg := filepath.Join(t.TempDir(), "config.json")
	args = append([]string{"--config", cfg}, args...)
	var stdout, stderr strings.Builder
	code := run(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

func TestHelpFlagsExitSuccessfully(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"-h"},
		{"auth", "--help"},
		{"traffic", "data", "--help"},
	} {
		out, errOut, code := runTestCLI(t, args...)
		if code != 0 {
			t.Fatalf("args=%v code=%d stderr=%s", args, code, errOut)
		}
		if !strings.Contains(out, "Usage:") {
			t.Fatalf("args=%v missing usage: %s", args, out)
		}
	}
}

func TestVersionCommandDoesNotRequireBaseURL(t *testing.T) {
	out, errOut, code := runTestCLI(t, "version")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "nfa-dashboard-cli") || !strings.Contains(out, "commit:") || !strings.Contains(out, "date:") {
		t.Fatalf("unexpected version output: %s", out)
	}
}

func TestLoginFailureExplainsCredentialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid username or password"})
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "auth", "login", "--username", "admin", "--password", "bad")
	if code == 0 {
		t.Fatalf("expected login failure, stdout=%s", out)
	}
	if !strings.Contains(errOut, "invalid username or password") || !strings.Contains(errOut, "check credentials") {
		t.Fatalf("missing actionable login failure: %s", errOut)
	}
}

func TestLoginRequiresPasswordBeforeSendingRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("login without password should not send HTTP request")
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "auth", "login", "--username", "admin")
	if code == 0 {
		t.Fatalf("expected local validation failure, stdout=%s", out)
	}
	if !strings.Contains(errOut, "password is required") {
		t.Fatalf("missing password validation: %s", errOut)
	}
}

func TestInsecureSkipVerifyAllowsSelfSignedHTTPS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/profile" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"user": map[string]any{"username": "admin"}})
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "--insecure-skip-verify", "auth", "profile")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "items: 1") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestDryRunDoesNotSendRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Fatalf("dry-run should not send HTTP request")
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--dry-run", "api", "post", "--path", "/api/v1/system/permissions/sync", "--body", "{}")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if called {
		t.Fatal("server was called")
	}
	if !strings.Contains(out, `"method": "POST"`) || !strings.Contains(out, `"path": "/api/v1/system/permissions/sync"`) {
		t.Fatalf("unexpected dry-run output: %s", out)
	}
}

func TestDefaultCommandSavesJSONAndPrintsSummary(t *testing.T) {
	outDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
			map[string]any{"id": 1, "name": "admin"},
			map[string]any{"id": 2, "name": "viewer"},
		}})
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "--out-dir", outDir, "system", "roles", "list")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "items: 2") || !strings.Contains(out, ".json") {
		t.Fatalf("missing summary: %s", out)
	}
	files, err := filepath.Glob(filepath.Join(outDir, "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("json file not saved files=%v err=%v", files, err)
	}
}

func TestRawAPIInjectsTokenAndReportsPermissionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-1" {
			t.Fatalf("Authorization=%q", got)
		}
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "permission denied", "missing": "system.user.manage"})
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "api", "get", "--path", "/api/v1/system/users")
	if code == 0 {
		t.Fatalf("expected non-zero code, stdout=%s", out)
	}
	if !strings.Contains(errOut, "403") || !strings.Contains(errOut, "system.user.manage") {
		t.Fatalf("missing useful error details: %s", errOut)
	}
}

func TestRefreshesTokenAfterUnauthorizedAndRetries(t *testing.T) {
	var profileCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/profile":
			profileCalls++
			if profileCalls == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid token"})
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer access-2" {
				t.Fatalf("retry Authorization=%q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"user": map[string]any{"username": "admin"}, "permissions": []string{"system.user.manage"}})
		case "/api/v1/auth/refresh":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "access-2", "refresh_token": "refresh-2"})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "--refresh-token", "refresh-1", "--print-body", "auth", "profile")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if profileCalls != 2 {
		t.Fatalf("profile calls=%d", profileCalls)
	}
	if !strings.Contains(out, `"username": "admin"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestLoginSavesTokensWithoutPrintingThem(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":         "secret-access",
			"refresh_token": "secret-refresh",
			"user":          map[string]any{"username": "admin"},
		})
	}))
	defer srv.Close()

	var stdout, stderr strings.Builder
	code := run([]string{"--config", cfg, "--base-url", srv.URL, "auth", "login", "--username", "admin", "--password", "pass"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "secret-access") || strings.Contains(stdout.String(), "secret-refresh") {
		t.Fatalf("login leaked token: %s", stdout.String())
	}
	raw, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "secret-access") || !strings.Contains(string(raw), srv.URL) {
		t.Fatalf("config not saved correctly: %s", raw)
	}
}

func TestTypedTrafficDataUsesV2EndpointAndQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/traffic" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("region") != "湖北" || r.URL.Query().Get("granularity") != "5m" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{map[string]any{"school_name": "A"}}})
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "--print-body", "traffic", "data", "--query", "region=湖北", "--query", "granularity=5m")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, `"school_name": "A"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestTrafficDataSavesJSONSVGAndPrints95Summary(t *testing.T) {
	outDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/traffic" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": []any{
			map[string]any{"create_time": "2026-05-06T00:00:00+08:00", "total": 30000000000, "total_recv": 29000000000, "total_send": 1000000000},
			map[string]any{"create_time": "2026-05-06T00:05:00+08:00", "total": 60000000000, "total_recv": 58000000000, "total_send": 2000000000},
			map[string]any{"create_time": "2026-05-06T00:10:00+08:00", "total": 90000000000, "total_recv": 87000000000, "total_send": 3000000000},
		}})
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "--out-dir", outDir, "traffic", "data", "--query", "school_name=北京航空航天大学", "--query", "cp=bilibili", "--svg")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "points: 3") || !strings.Contains(out, "p95_total_mbps") || !strings.Contains(out, ".svg") {
		t.Fatalf("missing traffic summary: %s", out)
	}
	jsonFiles, _ := filepath.Glob(filepath.Join(outDir, "*.json"))
	svgFiles, _ := filepath.Glob(filepath.Join(outDir, "*.svg"))
	if len(jsonFiles) != 1 || len(svgFiles) != 1 {
		t.Fatalf("files json=%v svg=%v", jsonFiles, svgFiles)
	}
	svg, err := os.ReadFile(svgFiles[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svg), "服务流速") || !strings.Contains(string(svg), "回源流速") || strings.Contains(string(svg), "total Mbps") {
		t.Fatalf("traffic SVG should mirror web two-series semantics, got:\n%s", svg)
	}
}

func TestTrafficSummaryUsesNFA60SecondDivisor(t *testing.T) {
	body := []byte(`{"data":[{"create_time":"2026-05-06T00:00:00+08:00","total":60000000,"total_recv":60000000,"total_send":0}]}`)
	summary, err := trafficSummary(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "avg_total_mbps: 8.00") || !strings.Contains(summary, "p95_total_mbps: 8.00") {
		t.Fatalf("expected 60 second NFA divisor summary, got:\n%s", summary)
	}
}

func TestTypedCommandReplacesIDTemplate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != "/api/v1/settlement/rates/customer/import/tasks/42/continue" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"task_id": 42, "status": "running"})
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "--print-body", "rates", "customer", "import-task", "continue", "--id", "42")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, `"task_id": 42`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSettlementUserPanelFetchesPanelEndpointsAndAggregates(t *testing.T) {
	outDir := t.TempDir()
	var monthlySeen, dailySeen bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/settlement/data/customer/monthly":
			if r.URL.Query().Get("channel_owner_user_id") != "5" ||
				r.URL.Query().Get("region") != "北京市" ||
				r.URL.Query().Get("cp") != "bilibili" ||
				r.URL.Query().Get("start_service_date") != "2026-04-01 00:00:00" ||
				r.URL.Query().Get("end_service_date") != "2026-04-30 23:59:59" {
				t.Fatalf("unexpected query for %s: %s", r.URL.Path, r.URL.RawQuery)
			}
			monthlySeen = true
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"items": []any{
				map[string]any{"school_name": "A", "region": "北京市", "cp": "bilibili", "service_date": "2026-04", "settlement_value": 60000000000, "customer_bill": 10.0},
				map[string]any{"school_name": "B", "region": "北京市", "cp": "bilibili", "service_date": "2026-04", "settlement_value": 120000000000, "network_line_bill": 20.0},
			}, "total": 2}})
		case "/api/v1/settlement/data/customer":
			if r.URL.Query().Get("channel_owner_user_id") != "5" ||
				r.URL.Query().Get("region") != "北京市" ||
				r.URL.Query().Get("cp") != "bilibili" ||
				r.URL.Query().Get("start_service_date") != "2026-04-01 00:00:00" ||
				r.URL.Query().Get("end_service_date") != "2026-04-30 23:59:59" {
				t.Fatalf("unexpected query for %s: %s", r.URL.Path, r.URL.RawQuery)
			}
			dailySeen = true
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"items": []any{
				map[string]any{"school_name": "A", "region": "北京市", "cp": "bilibili", "service_date": "2026-04-01", "settlement_value": 60000000000},
				map[string]any{"school_name": "B", "region": "北京市", "cp": "bilibili", "service_date": "2026-04-01", "settlement_value": 120000000000},
			}, "total": 2}})
		case "/api/v1/system/settings/traffic":
			_ = json.NewEncoder(w).Encode(map[string]any{"settlement_single_user_rate_unit": "Gbps", "settlement_result_unit_base": 1024, "traffic_byte_unit_base": 1000})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t,
		"--base-url", srv.URL,
		"--token", "access-1",
		"--out-dir", outDir,
		"settlement", "user-panel",
		"--query", "channel_owner_user_id=5",
		"--query", "region=北京市",
		"--query", "cp=bilibili",
		"--query", "start_service_date=2026-04-01 00:00:00",
		"--query", "end_service_date=2026-04-30 23:59:59",
	)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !monthlySeen || !dailySeen {
		t.Fatalf("monthlySeen=%v dailySeen=%v", monthlySeen, dailySeen)
	}
	if !strings.Contains(out, "summary: settlement user-panel") ||
		!strings.Contains(out, "monthly_rows: 2") ||
		!strings.Contains(out, "daily_rows: 2") ||
		!strings.Contains(out, "amount_total: 30.00") {
		t.Fatalf("missing user panel summary: %s", out)
	}
	files, _ := filepath.Glob(filepath.Join(outDir, "*.json"))
	if len(files) != 1 {
		t.Fatalf("expected one JSON result, got %v", files)
	}
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"panel_rows"`) || !strings.Contains(string(raw), `"amount_total": 30`) {
		t.Fatalf("combined panel JSON missing expected fields: %s", raw)
	}
}

func TestDownloadWritesFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "out.csv")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("a,b\n1,2\n"))
	}))
	defer srv.Close()

	out, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "logs", "export", "--download", target)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a,b\n1,2\n" || !strings.Contains(out, target) {
		t.Fatalf("download mismatch stdout=%s data=%q", out, data)
	}
}

func TestTableAndCSVOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
			map[string]any{"id": 1, "name": "admin"},
			map[string]any{"id": 2, "name": "viewer"},
		}})
	}))
	defer srv.Close()

	table, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "--output", "table", "system", "roles", "list")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(table, "id") || !strings.Contains(table, "admin") {
		t.Fatalf("bad table output: %s", table)
	}

	csvOut, errOut, code := runTestCLI(t, "--base-url", srv.URL, "--token", "access-1", "--output", "csv", "system", "roles", "list")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(csvOut, "id,name") || !strings.Contains(csvOut, "1,admin") {
		t.Fatalf("bad csv output: %s", csvOut)
	}
}
