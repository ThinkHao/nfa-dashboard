package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

type configFile struct {
	BaseURL      string `json:"base_url,omitempty"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type options struct {
	ConfigPath   string
	BaseURL      string
	Token        string
	RefreshToken string
	Output       string
	DryRun       bool
	OutDir       string
	JSONFile     string
	PrintBody    bool
	InsecureTLS  bool
}

type requestSpec struct {
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Query    map[string]string `json:"query,omitempty"`
	Body     string            `json:"body,omitempty"`
	FilePath string            `json:"file,omitempty"`
	Download string            `json:"download,omitempty"`
	SVG      bool              `json:"svg,omitempty"`
	SVGFile  string            `json:"svg_file,omitempty"`
	Command  string            `json:"-"`
}

type apiError struct {
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
	Missing string `json:"missing,omitempty"`
	Body    string `json:"body,omitempty"`
}

func (e *apiError) Error() string {
	parts := []string{fmt.Sprintf("HTTP %d", e.Status)}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	if e.Missing != "" {
		parts = append(parts, "missing="+e.Missing)
	}
	if e.Body != "" && e.Message == "" {
		parts = append(parts, e.Body)
	}
	return strings.Join(parts, ": ")
}

type client struct {
	httpClient *http.Client
	opts       options
	config     configFile
}

type responseData struct {
	Body        []byte
	ContentType string
	Spec        requestSpec
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		printHelp(stdout, args)
		return 0
	}
	opts, rest, err := parseGlobalFlags(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(stdout, args)
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 2
	}
	if len(rest) == 0 {
		printHelp(stdout, args)
		return 0
	}

	cfg, _ := loadConfig(opts.ConfigPath)
	opts = mergeOptions(opts, cfg)
	c := &client{httpClient: newHTTPClient(opts), opts: opts, config: cfg}

	data, err := dispatch(c, rest, stdout)
	if err != nil {
		if isLoginCredentialError(rest, err) {
			fmt.Fprintln(stderr, err.Error()+": check credentials for the target dashboard; seed SQL passwords may differ from the live database")
			return 1
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	if data == nil {
		return 0
	}
	if err := writeResult(stdout, opts, data); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func parseGlobalFlags(args []string) (options, []string, error) {
	var opts options
	opts.Output = "json"
	fs := flag.NewFlagSet("nfa-dashboard-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.ConfigPath, "config", defaultConfigPath(), "config file")
	fs.StringVar(&opts.BaseURL, "base-url", "", "dashboard base URL")
	fs.StringVar(&opts.Token, "token", "", "access token")
	fs.StringVar(&opts.RefreshToken, "refresh-token", "", "refresh token")
	fs.StringVar(&opts.Output, "output", "json", "json, table, or csv")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "print request without sending it")
	fs.StringVar(&opts.OutDir, "out-dir", "", "directory for saved response files")
	fs.StringVar(&opts.JSONFile, "json-file", "", "JSON response file")
	fs.BoolVar(&opts.PrintBody, "print-body", false, "print response body instead of summary")
	fs.BoolVar(&opts.InsecureTLS, "insecure-skip-verify", false, "skip TLS certificate verification for self-signed HTTPS targets")
	if err := fs.Parse(args); err != nil {
		return opts, nil, err
	}
	opts.Output = strings.ToLower(strings.TrimSpace(opts.Output))
	if opts.Output == "" {
		opts.Output = "json"
	}
	if opts.Output != "json" && opts.Output != "table" && opts.Output != "csv" {
		return opts, nil, fmt.Errorf("unsupported output %q", opts.Output)
	}
	return opts, fs.Args(), nil
}

func newHTTPClient(opts options) *http.Client {
	if !opts.InsecureTLS {
		return &http.Client{Timeout: 60 * time.Second}
	}
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			return true
		}
	}
	return false
}

func defaultConfigPath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "nfa-dashboard-cli", "config.json")
	}
	return filepath.Join(".", ".nfa-dashboard-cli.json")
}

func loadConfig(path string) (configFile, error) {
	var cfg configFile
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	return cfg, json.Unmarshal(data, &cfg)
}

func saveConfig(path string, cfg configFile) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func mergeOptions(opts options, cfg configFile) options {
	if opts.BaseURL == "" {
		opts.BaseURL = firstNonEmpty(os.Getenv("NFA_DASHBOARD_BASE_URL"), cfg.BaseURL)
	}
	if opts.Token == "" {
		opts.Token = firstNonEmpty(os.Getenv("NFA_DASHBOARD_TOKEN"), cfg.Token)
	}
	if opts.RefreshToken == "" {
		opts.RefreshToken = firstNonEmpty(os.Getenv("NFA_DASHBOARD_REFRESH_TOKEN"), cfg.RefreshToken)
	}
	opts.BaseURL = strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join(os.TempDir(), "nfa-dashboard-cli")
	}
	return opts
}

func isLoginCredentialError(args []string, err error) bool {
	if len(args) < 2 || args[0] != "auth" || args[1] != "login" {
		return false
	}
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func dispatch(c *client, args []string, stdout io.Writer) (*responseData, error) {
	switch args[0] {
	case "version":
		_, _ = fmt.Fprintf(stdout, "nfa-dashboard-cli %s\ncommit: %s\ndate: %s\n", version, commit, date)
		return nil, nil
	case "auth":
		return dispatchAuth(c, args[1:], stdout)
	case "api":
		return dispatchRawAPI(c, args[1:])
	case "traffic", "settlement", "rates", "system", "logs":
		if args[0] == "settlement" && len(args) > 1 && args[1] == "user-panel" {
			return dispatchSettlementUserPanel(c, args[2:])
		}
		spec, err := typedSpec(args)
		if err != nil {
			return nil, err
		}
		return c.do(spec)
	case "help", "-h", "--help":
		printHelp(stdout, args)
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown command %q", args[0])
	}
}

func dispatchAuth(c *client, args []string, stdout io.Writer) (*responseData, error) {
	if len(args) == 0 {
		return nil, errors.New("auth subcommand required")
	}
	switch args[0] {
	case "login", "change-password":
		spec, err := authBodySpec(args)
		if err != nil {
			return nil, err
		}
		data, err := c.do(spec)
		if err != nil {
			return nil, err
		}
		if args[0] == "login" {
			if err := c.saveLogin(data.Body); err != nil {
				return nil, err
			}
			_, _ = fmt.Fprintln(stdout, `{"message": "login successful"}`)
			return nil, nil
		}
		return data, nil
	case "profile":
		return c.do(requestSpec{Method: http.MethodGet, Path: "/api/v1/auth/profile"})
	case "refresh":
		data, err := c.refreshToken()
		if err != nil {
			return nil, err
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unknown auth subcommand %q", args[0])
	}
}

func authBodySpec(args []string) (requestSpec, error) {
	cmd := args[0]
	fs := flag.NewFlagSet("auth "+cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	username := fs.String("username", "", "username")
	password := fs.String("password", "", "password")
	passwordEnv := fs.String("password-env", "", "environment variable containing password")
	oldPassword := fs.String("old-password", "", "old password")
	newPassword := fs.String("new-password", "", "new password")
	body := fs.String("body", "", "raw JSON body")
	if err := fs.Parse(args[1:]); err != nil {
		return requestSpec{}, err
	}
	if *body != "" {
		path := "/api/v1/auth/login"
		if cmd == "change-password" {
			path = "/api/v1/auth/change-password"
		}
		return requestSpec{Method: http.MethodPost, Path: path, Body: *body}, nil
	}
	payload := map[string]string{}
	path := "/api/v1/auth/login"
	if cmd == "login" {
		if *password == "" && *passwordEnv != "" {
			*password = os.Getenv(*passwordEnv)
		}
		if strings.TrimSpace(*username) == "" {
			return requestSpec{}, errors.New("username is required for login")
		}
		if *password == "" {
			return requestSpec{}, errors.New("password is required for login; pass --password or --password-env")
		}
		payload["username"] = *username
		payload["password"] = *password
	} else {
		path = "/api/v1/auth/change-password"
		payload["old_password"] = *oldPassword
		payload["new_password"] = *newPassword
	}
	raw, _ := json.Marshal(payload)
	return requestSpec{Method: http.MethodPost, Path: path, Body: string(raw)}, nil
}

func dispatchRawAPI(c *client, args []string) (*responseData, error) {
	if len(args) == 0 {
		return nil, errors.New("api method required")
	}
	method := strings.ToUpper(args[0])
	if method != http.MethodGet && method != http.MethodPost && method != http.MethodPut && method != http.MethodDelete {
		return nil, fmt.Errorf("unsupported api method %q", args[0])
	}
	spec, err := parseRequestFlags(method, "", args[1:])
	if err != nil {
		return nil, err
	}
	if spec.Path == "" {
		return nil, errors.New("--path is required")
	}
	return c.do(spec)
}

func dispatchSettlementUserPanel(c *client, args []string) (*responseData, error) {
	fs := flag.NewFlagSet("settlement user-panel", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	query := multiFlag{}
	view := fs.String("view", "monthly-columns", "detail or monthly-columns")
	rateUnit := fs.String("rate-unit", "", "Mbps or Gbps; defaults to system setting")
	unitBase := fs.Int("unit-base", 0, "1000 or 1024; defaults to system setting")
	fs.Var(&query, "query", "query key=value")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	filter := parseQueryFlags(query)
	normalizedView := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(*view), "_", "-"))
	if normalizedView == "" {
		normalizedView = "monthly-columns"
	}
	if normalizedView != "monthly-columns" && normalizedView != "detail" {
		return nil, fmt.Errorf("unsupported user-panel view %q", *view)
	}

	monthlySpec := requestSpec{Method: http.MethodGet, Path: "/api/v1/settlement/data/customer/monthly", Query: copyQuery(filter), Command: "settlement user-panel"}
	dailySpec := requestSpec{Method: http.MethodGet, Path: "/api/v1/settlement/data/customer", Query: copyQuery(filter), Command: "settlement user-panel"}
	settingsSpec := requestSpec{Method: http.MethodGet, Path: "/api/v1/system/settings/traffic", Command: "settlement user-panel"}
	if c.opts.DryRun {
		raw, _ := json.MarshalIndent(map[string]any{
			"command":  "settlement user-panel",
			"view":     normalizedView,
			"requests": []requestSpec{monthlySpec, dailySpec, settingsSpec},
		}, "", "  ")
		return &responseData{Body: raw, ContentType: "application/json", Spec: requestSpec{Method: http.MethodGet, Path: "/api/v1/settlement/data/customer/monthly + /api/v1/settlement/data/customer", Query: filter, Command: "settlement user-panel"}}, nil
	}

	settings := c.singleUserPanelSettings()
	if *rateUnit != "" {
		settings.RateUnit = normalizeRateUnit(*rateUnit, settings.RateUnit)
	}
	if *unitBase == 1000 || *unitBase == 1024 {
		settings.UnitBase = *unitBase
	}

	monthlyRows, monthlyTotal, err := c.fetchAllRows(monthlySpec.Path, filter)
	if err != nil {
		return nil, err
	}
	dailyRows, dailyTotal, err := c.fetchAllRows(dailySpec.Path, filter)
	if err != nil {
		return nil, err
	}
	panel := buildUserPanelResponse(filter, normalizedView, settings, monthlyRows, dailyRows, monthlyTotal, dailyTotal)
	raw, err := json.Marshal(panel)
	if err != nil {
		return nil, err
	}
	spec := requestSpec{Method: http.MethodGet, Path: "/api/v1/settlement/data/customer/monthly + /api/v1/settlement/data/customer", Query: filter, Command: "settlement user-panel"}
	return &responseData{Body: raw, ContentType: "application/json", Spec: spec}, nil
}

func parseRequestFlags(method, path string, args []string) (requestSpec, error) {
	fs := flag.NewFlagSet("request", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	query := multiFlag{}
	body := fs.String("body", "", "JSON body")
	bodyFile := fs.String("body-file", "", "JSON body file")
	filePath := fs.String("file", "", "multipart file")
	download := fs.String("download", "", "download target")
	pathFlag := fs.String("path", path, "API path")
	id := fs.String("id", "", "replace {id} or {user_id} in typed paths")
	userID := fs.String("user-id", "", "replace {user_id} in typed paths")
	svg := fs.Bool("svg", false, "also write SVG chart when supported")
	svgFile := fs.String("svg-file", "", "SVG output file")
	fs.Var(&query, "query", "query key=value")
	if err := fs.Parse(args); err != nil {
		return requestSpec{}, err
	}
	bodyValue := *body
	if bodyValue == "" && *bodyFile != "" {
		raw, err := os.ReadFile(*bodyFile)
		if err != nil {
			return requestSpec{}, err
		}
		bodyValue = string(raw)
	}
	resolvedPath := *pathFlag
	if *id != "" {
		resolvedPath = strings.ReplaceAll(resolvedPath, "{id}", *id)
		resolvedPath = strings.ReplaceAll(resolvedPath, "{user_id}", *id)
	}
	if *userID != "" {
		resolvedPath = strings.ReplaceAll(resolvedPath, "{user_id}", *userID)
	}
	if strings.Contains(resolvedPath, "{") {
		return requestSpec{}, fmt.Errorf("path %q requires --id or --user-id", resolvedPath)
	}
	return requestSpec{
		Method:   method,
		Path:     resolvedPath,
		Query:    parseQueryFlags(query),
		Body:     bodyValue,
		FilePath: *filePath,
		Download: *download,
		SVG:      *svg,
		SVGFile:  *svgFile,
	}, nil
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func parseQueryFlags(values []string) map[string]string {
	out := map[string]string{}
	for _, item := range values {
		if item == "" {
			continue
		}
		key, val, found := strings.Cut(item, "=")
		if !found {
			out[item] = ""
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func typedSpec(args []string) (requestSpec, error) {
	keyArgs, rest := splitBeforeFlags(args)
	key := strings.Join(keyArgs, " ")
	route, ok := typedRoutes()[key]
	if !ok {
		return requestSpec{}, fmt.Errorf("unknown typed command %q", key)
	}
	spec, err := parseRequestFlags(route.Method, route.Path, rest)
	spec.Command = key
	return spec, err
}

func splitBeforeFlags(args []string) ([]string, []string) {
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return args[:i], args[i:]
		}
	}
	return args, nil
}

type route struct {
	Method string
	Path   string
}

func typedRoutes() map[string]route {
	return map[string]route{
		"traffic schools": {http.MethodGet, "/api/v2/schools"},
		"traffic regions": {http.MethodGet, "/api/v2/regions"},
		"traffic cps":     {http.MethodGet, "/api/v2/cps"},
		"traffic data":    {http.MethodGet, "/api/v2/traffic"},
		"traffic summary": {http.MethodGet, "/api/v2/traffic/summary"},

		"settlement config get":                    {http.MethodGet, "/api/v1/settlement/config"},
		"settlement config update":                 {http.MethodPut, "/api/v1/settlement/config"},
		"settlement tasks list":                    {http.MethodGet, "/api/v1/settlement/tasks"},
		"settlement tasks get":                     {http.MethodGet, "/api/v1/settlement/tasks/{id}"},
		"settlement tasks create-daily":            {http.MethodPost, "/api/v1/settlement/tasks/daily"},
		"settlement tasks create-weekly":           {http.MethodPost, "/api/v1/settlement/tasks/weekly"},
		"settlement tasks delete":                  {http.MethodDelete, "/api/v1/settlement/tasks/{id}"},
		"settlement data list":                     {http.MethodGet, "/api/v1/settlement/data"},
		"settlement daily-details list":            {http.MethodGet, "/api/v1/settlement/daily-details"},
		"settlement results list":                  {http.MethodGet, "/api/v1/settlement/results"},
		"settlement channel-results list":          {http.MethodGet, "/api/v1/settlement/results/channels"},
		"settlement data customer":                 {http.MethodGet, "/api/v1/settlement/data/customer"},
		"settlement data monthly":                  {http.MethodGet, "/api/v1/settlement/data/customer/monthly"},
		"settlement owner-subjects":                {http.MethodGet, "/api/v1/settlement/data/customer/owner-subjects"},
		"settlement data recalculate":              {http.MethodPost, "/api/v1/settlement/data/customer/recalculate"},
		"settlement data rebuild-monthly":          {http.MethodPost, "/api/v1/settlement/data/customer/monthly/rebuild"},
		"settlement formulas list":                 {http.MethodGet, "/api/v1/settlement/formulas"},
		"settlement formulas get":                  {http.MethodGet, "/api/v1/settlement/formulas/{id}"},
		"settlement formulas create":               {http.MethodPost, "/api/v1/settlement/formulas"},
		"settlement formulas update":               {http.MethodPut, "/api/v1/settlement/formulas/{id}"},
		"settlement formulas delete":               {http.MethodDelete, "/api/v1/settlement/formulas/{id}"},
		"settlement entities list":                 {http.MethodGet, "/api/v1/settlement/entities"},
		"settlement entities create":               {http.MethodPost, "/api/v1/settlement/entities"},
		"settlement entities update":               {http.MethodPut, "/api/v1/settlement/entities/{id}"},
		"settlement entities delete":               {http.MethodDelete, "/api/v1/settlement/entities/{id}"},
		"settlement business-types list":           {http.MethodGet, "/api/v1/settlement/business-types"},
		"settlement business-types create":         {http.MethodPost, "/api/v1/settlement/business-types"},
		"settlement business-types update":         {http.MethodPut, "/api/v1/settlement/business-types/{id}"},
		"settlement business-types delete":         {http.MethodDelete, "/api/v1/settlement/business-types/{id}"},
		"rates customer list":                      {http.MethodGet, "/api/v1/settlement/rates/customer"},
		"rates customer upsert":                    {http.MethodPost, "/api/v1/settlement/rates/customer"},
		"rates customer export":                    {http.MethodGet, "/api/v1/settlement/rates/customer/export"},
		"rates customer export-xlsx":               {http.MethodGet, "/api/v1/settlement/rates/customer/export-xlsx"},
		"rates customer import-template":           {http.MethodGet, "/api/v1/settlement/rates/customer/import-template"},
		"rates customer import":                    {http.MethodPost, "/api/v1/settlement/rates/customer/import"},
		"rates customer import-task":               {http.MethodPost, "/api/v1/settlement/rates/customer/import/tasks"},
		"rates customer import-task get":           {http.MethodGet, "/api/v1/settlement/rates/customer/import/tasks/{id}"},
		"rates customer import-task continue":      {http.MethodPost, "/api/v1/settlement/rates/customer/import/tasks/{id}/continue"},
		"rates customer import-task errors":        {http.MethodGet, "/api/v1/settlement/rates/customer/import/tasks/{id}/errors.csv"},
		"rates customer import-task created-users": {http.MethodGet, "/api/v1/settlement/rates/customer/import/tasks/{id}/created-users.csv"},
		"rates node list":                          {http.MethodGet, "/api/v1/settlement/rates/node"},
		"rates node upsert":                        {http.MethodPost, "/api/v1/settlement/rates/node"},
		"rates final list":                         {http.MethodGet, "/api/v1/settlement/rates/final"},
		"rates final discounted":                   {http.MethodGet, "/api/v1/settlement/rates/final-discounted"},
		"rates final upsert":                       {http.MethodPost, "/api/v1/settlement/rates/final"},
		"rates final refresh":                      {http.MethodPost, "/api/v1/settlement/rates/final/refresh"},
		"rates final init-from-customer":           {http.MethodPost, "/api/v1/settlement/rates/final/init-from-customer"},
		"rates final cleanup-invalid":              {http.MethodPost, "/api/v1/settlement/rates/final/cleanup-invalid"},
		"rates sync execute":                       {http.MethodPost, "/api/v1/settlement/rates/sync/execute"},
		"rates sync-rules options":                 {http.MethodGet, "/api/v1/settlement/rates/sync-rules/options"},
		"rates sync-rules list":                    {http.MethodGet, "/api/v1/settlement/rates/sync-rules"},
		"rates sync-rules create":                  {http.MethodPost, "/api/v1/settlement/rates/sync-rules"},
		"rates sync-rules update":                  {http.MethodPut, "/api/v1/settlement/rates/sync-rules/{id}"},
		"rates sync-rules delete":                  {http.MethodDelete, "/api/v1/settlement/rates/sync-rules/{id}"},
		"rates sync-rules priority":                {http.MethodPut, "/api/v1/settlement/rates/sync-rules/{id}/priority"},
		"rates sync-rules enabled":                 {http.MethodPut, "/api/v1/settlement/rates/sync-rules/{id}/enabled"},
		"rates filter-rules options":               {http.MethodGet, "/api/v1/settlement/rates/filter-rules/options"},
		"rates filter-rules list":                  {http.MethodGet, "/api/v1/settlement/rates/filter-rules"},
		"rates filter-rules create":                {http.MethodPost, "/api/v1/settlement/rates/filter-rules"},
		"rates filter-rules update":                {http.MethodPut, "/api/v1/settlement/rates/filter-rules/{id}"},
		"rates filter-rules delete":                {http.MethodDelete, "/api/v1/settlement/rates/filter-rules/{id}"},
		"rates filter-rules priority":              {http.MethodPut, "/api/v1/settlement/rates/filter-rules/{id}/priority"},
		"rates filter-rules enabled":               {http.MethodPut, "/api/v1/settlement/rates/filter-rules/{id}/enabled"},
		"rates discount-rules list":                {http.MethodGet, "/api/v1/settlement/rates/discount-rules"},
		"rates discount-rules get":                 {http.MethodGet, "/api/v1/settlement/rates/discount-rules/{id}"},
		"rates discount-rules create":              {http.MethodPost, "/api/v1/settlement/rates/discount-rules"},
		"rates discount-rules update":              {http.MethodPut, "/api/v1/settlement/rates/discount-rules/{id}"},
		"rates discount-rules delete":              {http.MethodDelete, "/api/v1/settlement/rates/discount-rules/{id}"},
		"rates discount-rules replace-items":       {http.MethodPut, "/api/v1/settlement/rates/discount-rules/{id}/items"},
		"system users list":                        {http.MethodGet, "/api/v1/system/users"},
		"system users create":                      {http.MethodPost, "/api/v1/system/users"},
		"system users status":                      {http.MethodPut, "/api/v1/system/users/{id}/status"},
		"system users roles":                       {http.MethodPut, "/api/v1/system/users/{id}/roles"},
		"system users alias":                       {http.MethodPut, "/api/v1/system/users/{id}/alias"},
		"system roles list":                        {http.MethodGet, "/api/v1/system/roles"},
		"system roles create":                      {http.MethodPost, "/api/v1/system/roles"},
		"system roles update":                      {http.MethodPut, "/api/v1/system/roles/{id}"},
		"system roles delete":                      {http.MethodDelete, "/api/v1/system/roles/{id}"},
		"system roles permissions":                 {http.MethodGet, "/api/v1/system/roles/{id}/permissions"},
		"system roles set-permissions":             {http.MethodPut, "/api/v1/system/roles/{id}/permissions"},
		"system permissions list":                  {http.MethodGet, "/api/v1/system/permissions"},
		"system permissions create":                {http.MethodPost, "/api/v1/system/permissions"},
		"system permissions get":                   {http.MethodGet, "/api/v1/system/permissions/{id}"},
		"system permissions update":                {http.MethodPut, "/api/v1/system/permissions/{id}"},
		"system permissions delete":                {http.MethodDelete, "/api/v1/system/permissions/{id}"},
		"system permissions sync":                  {http.MethodPost, "/api/v1/system/permissions/sync"},
		"system user-schools owner":                {http.MethodPost, "/api/v1/system/user-schools/owner"},
		"system traffic-scopes users":              {http.MethodGet, "/api/v1/system/traffic-scopes/users"},
		"system traffic-scopes options":            {http.MethodGet, "/api/v1/system/traffic-scopes/options"},
		"system traffic-scopes list":               {http.MethodGet, "/api/v1/system/traffic-scopes/{user_id}"},
		"system traffic-scopes replace":            {http.MethodPut, "/api/v1/system/traffic-scopes/{user_id}"},
		"system traffic-scopes preview":            {http.MethodGet, "/api/v1/system/traffic-scopes/{user_id}/preview"},
		"system settings traffic get":              {http.MethodGet, "/api/v1/system/settings/traffic"},
		"system settings traffic update":           {http.MethodPut, "/api/v1/system/settings/traffic"},
		"logs list":                                {http.MethodGet, "/api/v1/system/operation-logs"},
		"logs export":                              {http.MethodGet, "/api/v1/system/operation-logs/export"},
	}
}

func (c *client) do(spec requestSpec) (*responseData, error) {
	if c.opts.DryRun {
		raw, _ := json.MarshalIndent(spec, "", "  ")
		return &responseData{Body: raw, ContentType: "application/json", Spec: spec}, nil
	}
	data, err := c.send(spec)
	if apiErr := (*apiError)(nil); errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized && c.opts.RefreshToken != "" && spec.Path != "/api/v1/auth/login" && spec.Path != "/api/v1/auth/refresh" {
		if _, refreshErr := c.refreshToken(); refreshErr != nil {
			return nil, err
		}
		return c.send(spec)
	}
	return data, err
}

func (c *client) send(spec requestSpec) (*responseData, error) {
	if c.opts.BaseURL == "" {
		return nil, errors.New("base URL is required; pass --base-url or set NFA_DASHBOARD_BASE_URL")
	}
	fullURL, err := buildURL(c.opts.BaseURL, spec.Path, spec.Query)
	if err != nil {
		return nil, err
	}
	body, contentType, err := requestBody(spec)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(spec.Method, fullURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.opts.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.opts.Token)
	}
	req.Header.Set("User-Agent", "nfa-dashboard-cli/1.0")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeAPIError(resp.StatusCode, data)
	}
	if spec.Download != "" {
		if err := os.WriteFile(spec.Download, data, 0o600); err != nil {
			return nil, err
		}
		return &responseData{Body: []byte(fmt.Sprintf("downloaded %s\n", spec.Download)), ContentType: "text/plain", Spec: spec}, nil
	}
	return &responseData{Body: data, ContentType: resp.Header.Get("Content-Type"), Spec: spec}, nil
}

func buildURL(baseURL, path string, query map[string]string) (string, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/"))
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func requestBody(spec requestSpec) (io.Reader, string, error) {
	if spec.FilePath != "" {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, err := writer.CreateFormFile("file", filepath.Base(spec.FilePath))
		if err != nil {
			return nil, "", err
		}
		f, err := os.Open(spec.FilePath)
		if err != nil {
			return nil, "", err
		}
		defer f.Close()
		if _, err := io.Copy(part, f); err != nil {
			return nil, "", err
		}
		if spec.Body != "" {
			_ = writer.WriteField("payload", spec.Body)
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return &buf, writer.FormDataContentType(), nil
	}
	if spec.Body != "" {
		return strings.NewReader(spec.Body), "application/json", nil
	}
	return nil, "", nil
}

func decodeAPIError(status int, data []byte) error {
	errObj := &apiError{Status: status}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err == nil {
		if v, ok := m["message"].(string); ok {
			errObj.Message = v
		}
		if v, ok := m["missing"].(string); ok {
			errObj.Missing = v
		}
	}
	if errObj.Message == "" && errObj.Missing == "" {
		errObj.Body = strings.TrimSpace(string(data))
	}
	return errObj
}

func (c *client) refreshToken() (*responseData, error) {
	refresh := c.opts.RefreshToken
	if refresh == "" {
		return nil, errors.New("refresh token is required")
	}
	payload, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	spec := requestSpec{Method: http.MethodPost, Path: "/api/v1/auth/refresh", Body: string(payload)}
	oldToken := c.opts.Token
	c.opts.Token = ""
	data, err := c.send(spec)
	c.opts.Token = oldToken
	if err != nil {
		return nil, err
	}
	if err := c.saveLogin(data.Body); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *client) saveLogin(data []byte) error {
	var res struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}
	if res.Token == "" {
		return errors.New("login response did not include token")
	}
	cfg := c.config
	cfg.BaseURL = c.opts.BaseURL
	cfg.Token = res.Token
	cfg.RefreshToken = res.RefreshToken
	c.config = cfg
	c.opts.Token = cfg.Token
	c.opts.RefreshToken = cfg.RefreshToken
	return saveConfig(c.opts.ConfigPath, cfg)
}

type userPanelSettings struct {
	RateUnit string `json:"rate_unit"`
	UnitBase int    `json:"unit_base"`
}

type userPanelRow struct {
	SchoolName        string  `json:"school_name"`
	Region            string  `json:"region,omitempty"`
	CP                string  `json:"cp,omitempty"`
	ServiceMonth      string  `json:"service_month"`
	MonthlyDaily95    float64 `json:"monthly_daily95"`
	Amount            float64 `json:"amount"`
	CustomerBill      float64 `json:"customer_bill"`
	NetworkLineBill   float64 `json:"network_line_bill"`
	NodeDeductionBill float64 `json:"node_deduction_bill"`
	ChannelBill       float64 `json:"channel_bill"`
	DataSource        string  `json:"data_source,omitempty"`
	StockStartAt      string  `json:"stock_start_at,omitempty"`
	IncrementStartAt  string  `json:"increment_start_at,omitempty"`
}

type userPanelSummary struct {
	View                   string  `json:"view"`
	MonthlyRows            int     `json:"monthly_rows"`
	DailyRows              int     `json:"daily_rows"`
	MonthlyTotal           int     `json:"monthly_total"`
	DailyTotal             int     `json:"daily_total"`
	AmountTotal            float64 `json:"amount_total"`
	CustomerBillTotal      float64 `json:"customer_bill_total"`
	NetworkLineBillTotal   float64 `json:"network_line_bill_total"`
	NodeDeductionBillTotal float64 `json:"node_deduction_bill_total"`
	ChannelBillTotal       float64 `json:"channel_bill_total"`
	MonthlyDaily95Total    float64 `json:"monthly_daily95_total"`
	RateUnit               string  `json:"rate_unit"`
	UnitBase               int     `json:"unit_base"`
}

func (c *client) singleUserPanelSettings() userPanelSettings {
	settings := userPanelSettings{RateUnit: "Gbps", UnitBase: 1024}
	data, err := c.do(requestSpec{Method: http.MethodGet, Path: "/api/v1/system/settings/traffic"})
	if err != nil || data == nil {
		return settings
	}
	var m map[string]any
	if err := json.Unmarshal(data.Body, &m); err != nil {
		return settings
	}
	if v, ok := m["settlement_single_user_rate_unit"].(string); ok {
		settings.RateUnit = normalizeRateUnit(v, settings.RateUnit)
	}
	if v := intNumber(m["settlement_result_unit_base"]); v == 1000 || v == 1024 {
		settings.UnitBase = v
	}
	return settings
}

func (c *client) fetchAllRows(path string, filter map[string]string) ([]map[string]any, int, error) {
	const pageSize = 1000
	var all []map[string]any
	total := 0
	for page := 1; ; page++ {
		q := copyQuery(filter)
		q["page"] = strconv.Itoa(page)
		q["page_size"] = strconv.Itoa(pageSize)
		data, err := c.do(requestSpec{Method: http.MethodGet, Path: path, Query: q})
		if err != nil {
			return nil, 0, err
		}
		items, gotTotal, err := rowsAndTotal(data.Body)
		if err != nil {
			return nil, 0, err
		}
		if gotTotal > total {
			total = gotTotal
		}
		all = append(all, items...)
		if len(items) < pageSize {
			break
		}
		if total > 0 && page*pageSize >= total {
			break
		}
	}
	if total == 0 {
		total = len(all)
	}
	return all, total, nil
}

func rowsAndTotal(data []byte) ([]map[string]any, int, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, 0, err
	}
	var items []any
	total := 0
	if m, ok := v.(map[string]any); ok {
		if dataObj, ok := m["data"].(map[string]any); ok {
			if arr, ok := dataObj["items"].([]any); ok {
				items = arr
			}
			total = intNumber(dataObj["total"])
		}
		if items == nil {
			if arr, ok := m["items"].([]any); ok {
				items = arr
			}
			total = intNumber(m["total"])
		}
	} else if arr, ok := v.([]any); ok {
		items = arr
		total = len(arr)
	}
	return normalizeRows(items), total, nil
}

func buildUserPanelResponse(filter map[string]string, view string, settings userPanelSettings, monthlyRows, dailyRows []map[string]any, monthlyTotal, dailyTotal int) map[string]any {
	daily95 := monthlyDaily95BySchool(dailyRows, settings)
	panelRows := make([]userPanelRow, 0, len(monthlyRows))
	summary := userPanelSummary{
		View:         view,
		MonthlyRows:  len(monthlyRows),
		DailyRows:    len(dailyRows),
		MonthlyTotal: monthlyTotal,
		DailyTotal:   dailyTotal,
		RateUnit:     settings.RateUnit,
		UnitBase:     settings.UnitBase,
	}
	for _, row := range monthlyRows {
		school := stringValue(row["school_name"])
		month := serviceMonth(row["service_date"])
		out := userPanelRow{
			SchoolName:        school,
			Region:            stringValue(row["region"]),
			CP:                stringValue(row["cp"]),
			ServiceMonth:      month,
			MonthlyDaily95:    round2(daily95[school][month]),
			CustomerBill:      round2(numberValue(row["customer_bill"])),
			NetworkLineBill:   round2(numberValue(row["network_line_bill"])),
			NodeDeductionBill: round2(numberValue(row["node_deduction_bill"])),
			ChannelBill:       round2(numberValue(row["channel_bill"])),
			DataSource:        stringValue(row["data_source"]),
			StockStartAt:      stringValue(row["stock_start_at"]),
			IncrementStartAt:  stringValue(row["increment_start_at"]),
		}
		out.Amount = round2(out.CustomerBill + out.NetworkLineBill + out.NodeDeductionBill + out.ChannelBill)
		panelRows = append(panelRows, out)
		summary.AmountTotal += out.Amount
		summary.CustomerBillTotal += out.CustomerBill
		summary.NetworkLineBillTotal += out.NetworkLineBill
		summary.NodeDeductionBillTotal += out.NodeDeductionBill
		summary.ChannelBillTotal += out.ChannelBill
		summary.MonthlyDaily95Total += out.MonthlyDaily95
	}
	sort.Slice(panelRows, func(i, j int) bool {
		return panelRows[i].SchoolName < panelRows[j].SchoolName
	})
	summary.AmountTotal = round2(summary.AmountTotal)
	summary.CustomerBillTotal = round2(summary.CustomerBillTotal)
	summary.NetworkLineBillTotal = round2(summary.NetworkLineBillTotal)
	summary.NodeDeductionBillTotal = round2(summary.NodeDeductionBillTotal)
	summary.ChannelBillTotal = round2(summary.ChannelBillTotal)
	summary.MonthlyDaily95Total = round2(summary.MonthlyDaily95Total)
	return map[string]any{
		"command":      "settlement user-panel",
		"filters":      filter,
		"summary":      summary,
		"panel_rows":   panelRows,
		"monthly_rows": monthlyRows,
		"daily_rows":   dailyRows,
	}
}

func monthlyDaily95BySchool(dailyRows []map[string]any, settings userPanelSettings) map[string]map[string]float64 {
	grouped := map[string]map[string]float64{}
	for _, row := range dailyRows {
		school := stringValue(row["school_name"])
		date := normalizeDateOnly(row["service_date"])
		if school == "" || date == "" {
			continue
		}
		if grouped[school] == nil {
			grouped[school] = map[string]float64{}
		}
		grouped[school][date] += numberValue(row["settlement_value"])
	}
	result := map[string]map[string]float64{}
	for school, byDay := range grouped {
		monthAgg := map[string]struct {
			Sum   float64
			Count int
		}{}
		for date, raw := range byDay {
			month := serviceMonth(date)
			agg := monthAgg[month]
			agg.Sum += settlementValueToRate(raw, settings)
			agg.Count++
			monthAgg[month] = agg
		}
		result[school] = map[string]float64{}
		for month, agg := range monthAgg {
			if agg.Count > 0 {
				result[school][month] = agg.Sum / float64(agg.Count)
			}
		}
	}
	return result
}

func settlementValueToRate(raw float64, settings userPanelSettings) float64 {
	bitsPerSecond := raw * 8 / 60
	div := float64(settings.UnitBase * settings.UnitBase)
	if settings.RateUnit == "Gbps" {
		div *= float64(settings.UnitBase)
	}
	return bitsPerSecond / div
}

func normalizeRateUnit(v, fallback string) string {
	if strings.EqualFold(strings.TrimSpace(v), "Mbps") {
		return "Mbps"
	}
	if strings.EqualFold(strings.TrimSpace(v), "Gbps") {
		return "Gbps"
	}
	return fallback
}

func copyQuery(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func intNumber(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		n, _ := strconv.Atoi(x)
		return n
	default:
		return 0
	}
}

func serviceMonth(v any) string {
	s := normalizeDateOnly(v)
	if len(s) >= 7 {
		return s[:7]
	}
	return s
}

func normalizeDateOnly(v any) string {
	s := stringValue(v)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "T") {
		s = strings.SplitN(s, "T", 2)[0]
	}
	if strings.Contains(s, " ") {
		s = strings.SplitN(s, " ", 2)[0]
	}
	return s
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func writeResult(w io.Writer, opts options, data *responseData) error {
	if data == nil || len(bytes.TrimSpace(data.Body)) == 0 {
		return nil
	}
	if opts.DryRun || opts.PrintBody || opts.Output != "json" {
		return writeOutput(w, opts.Output, data.Body)
	}
	if !looksJSON(data.Body) {
		_, err := w.Write(data.Body)
		return err
	}
	jsonFile, err := saveJSONResult(opts, data)
	if err != nil {
		return err
	}
	var svgFile string
	if data.Spec.SVG {
		svgFile, err = saveSVGResult(opts, data)
		if err != nil {
			return err
		}
	}
	summary := buildSummary(data, jsonFile, svgFile)
	_, err = fmt.Fprint(w, summary)
	return err
}

func looksJSON(data []byte) bool {
	s := bytes.TrimSpace(data)
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

func saveJSONResult(opts options, data *responseData) (string, error) {
	target := opts.JSONFile
	if target == "" {
		target = filepath.Join(opts.OutDir, defaultResultName(data.Spec, "json"))
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", err
	}
	var v any
	if err := json.Unmarshal(data.Body, &v); err != nil {
		return "", err
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, append(pretty, '\n'), 0o600); err != nil {
		return "", err
	}
	return target, nil
}

func saveSVGResult(opts options, data *responseData) (string, error) {
	target := data.Spec.SVGFile
	if target == "" {
		target = filepath.Join(opts.OutDir, defaultResultName(data.Spec, "svg"))
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", err
	}
	rows, err := trafficRows(data.Body)
	if err != nil {
		return "", err
	}
	svg := renderTrafficSVG(rows, data.Spec)
	if err := os.WriteFile(target, []byte(svg), 0o600); err != nil {
		return "", err
	}
	return target, nil
}

func defaultResultName(spec requestSpec, ext string) string {
	base := spec.Command
	if base == "" {
		base = strings.Trim(spec.Path, "/")
	}
	if base == "" {
		base = "response"
	}
	re := regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	base = strings.Trim(re.ReplaceAllString(base, "-"), "-")
	if base == "" {
		base = "response"
	}
	return fmt.Sprintf("%s-%s.%s", time.Now().Format("20060102-150405-000"), base, ext)
}

func buildSummary(data *responseData, jsonFile, svgFile string) string {
	var b strings.Builder
	title := data.Spec.Command
	if title == "" {
		title = data.Spec.Method + " " + data.Spec.Path
	}
	fmt.Fprintf(&b, "summary: %s\n", title)
	fmt.Fprintf(&b, "method: %s\npath: %s\n", data.Spec.Method, data.Spec.Path)
	if jsonFile != "" {
		fmt.Fprintf(&b, "json: %s\n", jsonFile)
	}
	if svgFile != "" {
		fmt.Fprintf(&b, "svg: %s\n", svgFile)
	}
	if data.Spec.Command == "traffic data" {
		if s, err := trafficSummary(data.Body); err == nil {
			b.WriteString(s)
			return b.String()
		}
	}
	if data.Spec.Command == "settlement user-panel" {
		if s, err := userPanelSummaryText(data.Body); err == nil {
			b.WriteString(s)
			return b.String()
		}
	}
	count := responseItemCount(data.Body)
	if count >= 0 {
		fmt.Fprintf(&b, "items: %d\n", count)
	}
	return b.String()
}

func userPanelSummaryText(data []byte) (string, error) {
	var payload struct {
		Summary userPanelSummary `json:"summary"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "view: %s\n", payload.Summary.View)
	fmt.Fprintf(&b, "monthly_rows: %d\n", payload.Summary.MonthlyRows)
	fmt.Fprintf(&b, "daily_rows: %d\n", payload.Summary.DailyRows)
	fmt.Fprintf(&b, "rate_unit: %s\nunit_base: %d\n", payload.Summary.RateUnit, payload.Summary.UnitBase)
	fmt.Fprintf(&b, "monthly_daily95_total: %.2f\n", payload.Summary.MonthlyDaily95Total)
	fmt.Fprintf(&b, "amount_total: %.2f\n", payload.Summary.AmountTotal)
	fmt.Fprintf(&b, "customer_bill_total: %.2f\n", payload.Summary.CustomerBillTotal)
	fmt.Fprintf(&b, "network_line_bill_total: %.2f\n", payload.Summary.NetworkLineBillTotal)
	fmt.Fprintf(&b, "node_deduction_bill_total: %.2f\n", payload.Summary.NodeDeductionBillTotal)
	fmt.Fprintf(&b, "channel_bill_total: %.2f\n", payload.Summary.ChannelBillTotal)
	return b.String(), nil
}

func responseItemCount(data []byte) int {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return -1
	}
	switch x := v.(type) {
	case []any:
		return len(x)
	case map[string]any:
		if items, ok := x["items"].([]any); ok {
			return len(items)
		}
		if d, ok := x["data"].([]any); ok {
			return len(d)
		}
		if d, ok := x["data"].(map[string]any); ok {
			if items, ok := d["items"].([]any); ok {
				return len(items)
			}
		}
		return 1
	default:
		return 1
	}
}

type trafficPoint struct {
	Time      string
	Total     float64
	Recv      float64
	Send      float64
	TotalMbps float64
	RecvMbps  float64
	SendMbps  float64
}

func trafficRows(data []byte) ([]trafficPoint, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	var items []any
	switch x := v.(type) {
	case []any:
		items = x
	case map[string]any:
		if d, ok := x["data"].([]any); ok {
			items = d
		} else if d, ok := x["data"].(map[string]any); ok {
			if arr, ok := d["items"].([]any); ok {
				items = arr
			}
		} else if arr, ok := x["items"].([]any); ok {
			items = arr
		}
	}
	rows := make([]trafficPoint, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		p := trafficPoint{
			Time:  stringValue(m["create_time"]),
			Total: numberValue(m["total"]),
			Recv:  numberValue(m["total_recv"]),
			Send:  numberValue(m["total_send"]),
		}
		p.TotalMbps = bytesToMbps(p.Total)
		p.RecvMbps = bytesToMbps(p.Recv)
		p.SendMbps = bytesToMbps(p.Send)
		rows = append(rows, p)
	}
	return rows, nil
}

func trafficSummary(data []byte) (string, error) {
	rows, err := trafficRows(data)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "points: %d\n", len(rows))
	if len(rows) == 0 {
		return b.String(), nil
	}
	fmt.Fprintf(&b, "first: %s\nlast: %s\n", rows[0].Time, rows[len(rows)-1].Time)
	total := make([]float64, 0, len(rows))
	recv := make([]float64, 0, len(rows))
	send := make([]float64, 0, len(rows))
	maxTotal := rows[0].TotalMbps
	maxTime := rows[0].Time
	var sumTotal, sumRecv, sumSend float64
	for _, row := range rows {
		total = append(total, row.TotalMbps)
		recv = append(recv, row.RecvMbps)
		send = append(send, row.SendMbps)
		sumTotal += row.TotalMbps
		sumRecv += row.RecvMbps
		sumSend += row.SendMbps
		if row.TotalMbps > maxTotal {
			maxTotal = row.TotalMbps
			maxTime = row.Time
		}
	}
	fmt.Fprintf(&b, "avg_total_mbps: %.2f\n", sumTotal/float64(len(rows)))
	fmt.Fprintf(&b, "avg_recv_mbps: %.2f\n", sumRecv/float64(len(rows)))
	fmt.Fprintf(&b, "avg_send_mbps: %.2f\n", sumSend/float64(len(rows)))
	fmt.Fprintf(&b, "p95_total_mbps: %.2f\n", percentile(total, 95))
	fmt.Fprintf(&b, "p95_recv_mbps: %.2f\n", percentile(recv, 95))
	fmt.Fprintf(&b, "p95_send_mbps: %.2f\n", percentile(send, 95))
	fmt.Fprintf(&b, "max_total_mbps: %.2f\nmax_total_time: %s\n", maxTotal, maxTime)
	return b.String(), nil
}

func bytesToMbps(v float64) float64 {
	// Match TrafficView.formatBitRate(): bit-rate units use decimal 1000.
	// traffic_byte_unit_base only affects byte-size display such as B/KB/MB/GB.
	return v * 8 / 60 / 1_000_000
}

func percentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	idx := (p*len(cp)+99)/100 - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func renderTrafficSVG(rows []trafficPoint, spec requestSpec) string {
	const width = 1200
	const height = 520
	const left = 70
	const right = 30
	const top = 38
	const bottom = 70
	plotW := float64(width - left - right)
	plotH := float64(height - top - bottom)
	maxY := 1.0
	for _, row := range rows {
		if row.RecvMbps > maxY {
			maxY = row.RecvMbps
		}
		if row.SendMbps > maxY {
			maxY = row.SendMbps
		}
	}
	maxY = float64((int(maxY/100) + 1) * 100)
	x := func(i int) float64 {
		if len(rows) <= 1 {
			return left
		}
		return left + float64(i)/float64(len(rows)-1)*plotW
	}
	y := func(v float64) float64 {
		return top + (1-v/maxY)*plotH
	}
	pathFor := func(kind string) string {
		if len(rows) == 0 {
			return ""
		}
		value := func(p trafficPoint) float64 {
			switch kind {
			case "recv":
				return p.RecvMbps
			case "send":
				return p.SendMbps
			default:
				return p.TotalMbps
			}
		}
		step := len(rows) / 900
		if step < 1 {
			step = 1
		}
		var parts []string
		parts = append(parts, fmt.Sprintf("M %.2f %.2f", x(0), y(value(rows[0]))))
		for i := step; i < len(rows); i += step {
			parts = append(parts, fmt.Sprintf("L %.2f %.2f", x(i), y(value(rows[i]))))
		}
		if (len(rows)-1)%step != 0 {
			i := len(rows) - 1
			parts = append(parts, fmt.Sprintf("L %.2f %.2f", x(i), y(value(rows[i]))))
		}
		return strings.Join(parts, " ")
	}
	title := "traffic data"
	if school := spec.Query["school_name"]; school != "" {
		title = school
	}
	if cp := spec.Query["cp"]; cp != "" {
		title += " / " + cp
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height)
	b.WriteString(`<rect width="100%" height="100%" fill="white"/><style>text{font-family:Arial,Microsoft YaHei,sans-serif;fill:#111827}.axis{stroke:#374151}.grid{stroke:#e5e7eb}.label{font-size:13px}.title{font-size:20px;font-weight:700}</style>`)
	fmt.Fprintf(&b, `<text class="title" x="%d" y="26">%s 流速图</text>`, left, escapeXML(title))
	for i := 0; i <= 5; i++ {
		val := maxY * float64(i) / 5
		yy := y(val)
		fmt.Fprintf(&b, `<line class="grid" x1="%d" y1="%.2f" x2="%d" y2="%.2f"/><text class="label" x="10" y="%.2f">%.0f</text>`, left, yy, width-right, yy, yy+4, val)
	}
	fmt.Fprintf(&b, `<line class="axis" x1="%d" y1="%d" x2="%d" y2="%d"/><line class="axis" x1="%d" y1="%d" x2="%d" y2="%d"/>`, left, top, left, height-bottom, left, height-bottom, width-right, height-bottom)
	if len(rows) > 0 {
		ticks := []int{0, len(rows) / 4, len(rows) / 2, len(rows) * 3 / 4, len(rows) - 1}
		for _, i := range ticks {
			fmt.Fprintf(&b, `<text class="label" transform="translate(%.2f,%d) rotate(-25)" text-anchor="end">%s</text>`, x(i), height-35, shortTime(rows[i].Time))
		}
	}
	b.WriteString(`<text class="label" x="16" y="54">Mbps</text>`)
	fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#2563eb" stroke-width="2"/>`, pathFor("recv"))
	fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#16a34a" stroke-width="1.5" opacity="0.85"/>`, pathFor("send"))
	b.WriteString(`<line x1="890" y1="45" x2="925" y2="45" stroke="#2563eb" stroke-width="3"/><text class="label" x="935" y="50">服务流速</text>`)
	b.WriteString(`<line x1="890" y1="68" x2="925" y2="68" stroke="#16a34a" stroke-width="3"/><text class="label" x="935" y="73">回源流速</text>`)
	b.WriteString(`</svg>`)
	return b.String()
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func numberValue(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	default:
		return 0
	}
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func shortTime(s string) string {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("01-02 15:04")
	}
	if len(s) >= 16 {
		return s[5:16]
	}
	return s
}

func writeOutput(w io.Writer, format string, data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if format == "json" {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			_, err = w.Write(data)
			return err
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(v)
	}
	rows, err := rowsFromJSON(data)
	if err != nil {
		return err
	}
	switch format {
	case "table":
		return writeTable(w, rows)
	case "csv":
		return writeCSV(w, rows)
	default:
		return fmt.Errorf("unsupported output %q", format)
	}
}

func rowsFromJSON(data []byte) ([]map[string]any, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	switch x := v.(type) {
	case []any:
		return normalizeRows(x), nil
	case map[string]any:
		if items, ok := x["items"].([]any); ok {
			return normalizeRows(items), nil
		}
		if dataObj, ok := x["data"].(map[string]any); ok {
			if items, ok := dataObj["items"].([]any); ok {
				return normalizeRows(items), nil
			}
		}
		return []map[string]any{x}, nil
	default:
		return []map[string]any{{"value": x}}, nil
	}
}

func normalizeRows(items []any) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			rows = append(rows, m)
		} else {
			rows = append(rows, map[string]any{"value": item})
		}
	}
	return rows
}

func writeTable(w io.Writer, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}
	keys := tableKeys(rows)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, strings.Join(keys, "\t"))
	for _, row := range rows {
		vals := make([]string, 0, len(keys))
		for _, key := range keys {
			vals = append(vals, scalar(row[key]))
		}
		_, _ = fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	return tw.Flush()
}

func writeCSV(w io.Writer, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}
	keys := tableKeys(rows)
	cw := csv.NewWriter(w)
	if err := cw.Write(keys); err != nil {
		return err
	}
	for _, row := range rows {
		vals := make([]string, 0, len(keys))
		for _, key := range keys {
			vals = append(vals, scalar(row[key]))
		}
		if err := cw.Write(vals); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func tableKeys(rows []map[string]any) []string {
	seen := map[string]bool{}
	var keys []string
	for _, row := range rows {
		for key := range row {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

func scalar(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64, bool:
		return fmt.Sprint(x)
	default:
		raw, _ := json.Marshal(x)
		return string(raw)
	}
}

func printHelp(w io.Writer, args []string) {
	topic := helpTopic(args)
	switch topic {
	case "auth":
		printAuthHelp(w)
	case "api":
		printAPIHelp(w)
	case "traffic":
		printTypedHelp(w, "traffic", []string{
			"traffic schools --query school_name=四川大学 --query cp=bilibili --query limit=20",
			"traffic data --query region=北京市 --query school_name=北京航空航天大学 --query cp=bilibili --query start_time=\"2026-05-08 00:00:00\" --query end_time=\"2026-05-14 23:59:59\" --query granularity=5m --svg",
			"traffic summary --query region=北京市 --query cp=bilibili",
		})
	case "settlement":
		if hasHelpToken(args, "user-panel") {
			printSettlementUserPanelHelp(w)
			return
		}
		printTypedHelp(w, "settlement", []string{
			"settlement owner-subjects --query region=北京市 --query cp=bilibili --query start_service_date=\"2026-04-01 00:00:00\" --query end_service_date=\"2026-04-30 23:59:59\"",
			"settlement user-panel --query channel_owner_user_id=9 --query region=北京市 --query cp=bilibili --query start_service_date=\"2026-04-01 00:00:00\" --query end_service_date=\"2026-04-30 23:59:59\"",
			"settlement tasks list --query limit=20",
			"settlement data rebuild-monthly --body \"{}\" --dry-run",
		})
	case "rates":
		printTypedHelp(w, "rates", []string{
			"rates customer list --query cp=bilibili --query limit=20",
			"rates customer export --download .\\customer-rates.csv",
			"rates customer import --file .\\customer-rates.xlsx --dry-run",
		})
	case "system":
		printTypedHelp(w, "system", []string{
			"system users list --query username=liuxy",
			"system traffic-scopes preview --user-id 9",
			"system permissions sync --body \"{}\" --dry-run",
		})
	case "logs":
		printTypedHelp(w, "logs", []string{
			"logs list --query username=admin --query limit=20",
			"logs export --download .\\operation-logs.csv",
		})
	default:
		printUsage(w)
	}
}

func helpTopic(args []string) string {
	for _, arg := range args {
		switch arg {
		case "auth", "api", "traffic", "settlement", "rates", "system", "logs":
			return arg
		}
	}
	return ""
}

func hasHelpToken(args []string, token string) bool {
	for _, arg := range args {
		if arg == token {
			return true
		}
	}
	return false
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `nfa-dashboard-cli

HTTP CLI for nfa-dashboard. It always calls backend APIs, so JWT auth, RBAC,
traffic scopes, and operation-log auditing are inherited from the server.

Usage:
  nfa-dashboard-cli [global flags] <command> [subcommand] [flags]
  nfa-dashboard-cli help <command>
  nfa-dashboard-cli <command> --help

Commands:
  auth        login, profile, refresh, change-password
  api         raw API get/post/put/delete for endpoints without typed commands
  traffic     schools, regions, cps, data, summary
  settlement  config, tasks, data, user-panel, formulas, entities
  rates       customer, node, final, sync/filter/discount rules
  system      users, roles, permissions, traffic scopes, settings
  logs        operation-log list/export
  version     print CLI version, commit, and build date

Global flags:
  --config PATH              user config path for saved base URL and tokens
  --base-url URL             dashboard URL, or NFA_DASHBOARD_BASE_URL
  --token TOKEN              access token, or NFA_DASHBOARD_TOKEN
  --refresh-token TOKEN      refresh token, or NFA_DASHBOARD_REFRESH_TOKEN
  --output json|table|csv    json summary is default; table/csv print rows
  --out-dir DIR              directory for saved JSON/SVG responses
  --json-file PATH           exact JSON output path
  --print-body               print full JSON body to stdout
  --insecure-skip-verify     allow self-signed HTTPS targets
  --dry-run                  print request plan and do not send writes

Examples:
  nfa-dashboard-cli --base-url http://localhost:8081 auth login --username admin --password-env NFA_DASHBOARD_PASSWORD
  nfa-dashboard-cli auth profile
  nfa-dashboard-cli version
  nfa-dashboard-cli traffic data --query region=北京市 --query cp=bilibili --query granularity=5m --svg
  nfa-dashboard-cli settlement user-panel --query channel_owner_user_id=9 --query cp=bilibili
  nfa-dashboard-cli api get --path /api/v1/auth/profile --print-body

Run "nfa-dashboard-cli <command> --help" for command-specific examples.`)
}

func printAuthHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, `nfa-dashboard-cli auth

Authentication commands. Tokens are saved to the user config file and are not
printed. Automation can use NFA_DASHBOARD_BASE_URL, NFA_DASHBOARD_TOKEN, and
NFA_DASHBOARD_REFRESH_TOKEN.

Usage:
  nfa-dashboard-cli [global flags] auth login --username USER --password PASS
  nfa-dashboard-cli [global flags] auth login --username USER --password-env ENV
  nfa-dashboard-cli [global flags] auth profile
  nfa-dashboard-cli [global flags] auth refresh
  nfa-dashboard-cli [global flags] auth change-password --old-password OLD --new-password NEW

Examples:
  nfa-dashboard-cli --base-url https://192.168.9.104:8090 --insecure-skip-verify auth login --username admin --password-env NFA_DASHBOARD_PASSWORD
  nfa-dashboard-cli auth profile
  nfa-dashboard-cli --print-body auth profile`)
}

func printAPIHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, `nfa-dashboard-cli api

Raw API command for full endpoint coverage. Use typed commands when available;
use api get/post/put/delete for new or rare endpoints.

Usage:
  nfa-dashboard-cli [global flags] api get    --path /api/v1/path [--query k=v]
  nfa-dashboard-cli [global flags] api post   --path /api/v1/path [--body JSON|--body-file FILE]
  nfa-dashboard-cli [global flags] api put    --path /api/v1/path [--body JSON|--body-file FILE]
  nfa-dashboard-cli [global flags] api delete --path /api/v1/path

Request flags:
  --path PATH          backend API path
  --query KEY=VALUE    repeatable query parameter
  --body JSON          raw JSON request body
  --body-file PATH     load JSON body from a file
  --file PATH          multipart file upload field named "file"
  --download PATH      write response bytes to a file

Examples:
  nfa-dashboard-cli api get --path /api/v1/system/permissions
  nfa-dashboard-cli api post --path /api/v1/system/permissions/sync --body "{}" --dry-run
  nfa-dashboard-cli api get --path /api/v1/system/operation-logs/export --download .\operation-logs.csv`)
}

func printTypedHelp(w io.Writer, group string, examples []string) {
	_, _ = fmt.Fprintf(w, "nfa-dashboard-cli %s\n\n", group)
	_, _ = fmt.Fprintln(w, `Typed HTTP API commands. All commands share the same request flags:
Usage:
  nfa-dashboard-cli [global flags] <typed command> [request flags]

Request flags:
  --query KEY=VALUE    repeatable query parameter
  --body JSON          JSON body for POST/PUT commands
  --body-file PATH     load JSON body from a file
  --file PATH          multipart file upload field named "file"
  --download PATH      write response bytes to a file
  --id ID              fill {id} path parameters
  --user-id ID         fill {user_id} path parameters
  --svg               also write SVG chart when supported
  --svg-file PATH      exact SVG output path

Examples:`)
	for _, example := range examples {
		_, _ = fmt.Fprintf(w, "  nfa-dashboard-cli %s\n", example)
	}
	_, _ = fmt.Fprintln(w, "\nCommands:")
	for _, line := range typedRouteLines(group) {
		_, _ = fmt.Fprintln(w, "  "+line)
	}
}

func typedRouteLines(group string) []string {
	prefix := group + " "
	var keys []string
	for key := range typedRoutes() {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	if group == "settlement" {
		keys = append(keys, "settlement user-panel")
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "settlement user-panel" {
			lines = append(lines, fmt.Sprintf("%-45s %s", key, "GET /api/v1/settlement/data/customer/monthly + /api/v1/settlement/data/customer"))
			continue
		}
		route := typedRoutes()[key]
		lines = append(lines, fmt.Sprintf("%-45s %s %s", key, route.Method, route.Path))
	}
	return lines
}

func printSettlementUserPanelHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, `nfa-dashboard-cli settlement user-panel

Single-user settlement panel query. This mirrors the frontend page: monthly
amount rows come from /api/v1/settlement/data/customer/monthly, and monthly 95
columns are rebuilt from /api/v1/settlement/data/customer using the single-user
unit settings.

Usage:
  nfa-dashboard-cli [global flags] settlement user-panel --query KEY=VALUE...

Common query keys:
  channel_owner_user_id   owner subject id from settlement owner-subjects
  region                  region filter, for example 北京市
  cp                      business filter, for example bilibili
  school_name             optional school filter
  start_service_date      for example "2026-04-01 00:00:00"
  end_service_date        for example "2026-04-30 23:59:59"

Flags:
  --view monthly-columns|detail   output panel row shape
  --rate-unit Mbps|Gbps           override display unit
  --unit-base 1000|1024           override settlement unit base
  --dry-run                       show the three backend requests

Examples:
  nfa-dashboard-cli settlement owner-subjects --query region=北京市 --query cp=bilibili --query start_service_date="2026-04-01 00:00:00" --query end_service_date="2026-04-30 23:59:59"
  nfa-dashboard-cli settlement user-panel --query channel_owner_user_id=9 --query region=北京市 --query cp=bilibili --query start_service_date="2026-04-01 00:00:00" --query end_service_date="2026-04-30 23:59:59"
  nfa-dashboard-cli settlement user-panel --query channel_owner_user_id=9 --query cp=bilibili --dry-run`)
}
