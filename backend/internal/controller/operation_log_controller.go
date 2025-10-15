package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/service"
)

type OperationLogController struct { svc service.OperationLogService }

func NewOperationLogController(svc service.OperationLogService) *OperationLogController { return &OperationLogController{svc: svc} }

// GET /api/v1/system/operation-logs
func (ctl *OperationLogController) List(c *gin.Context) {
	var (
		userIDPtr *uint64
		statusCodePtr *int
		successPtr *int8
		methodPtr, pathPtr, keywordPtr *string
		startPtr, endPtr *time.Time
	)

	if v := c.Query("user_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil && id > 0 { userID := id; userIDPtr = &userID }
	}
	if v := c.Query("status_code"); v != "" {
		if sc, err := strconv.Atoi(v); err == nil { statusCode := sc; statusCodePtr = &statusCode }
	}
	if v := c.Query("success"); v != "" {
		if s, err := strconv.Atoi(v); err == nil { ss := int8(s); successPtr = &ss }
	}
	if v := c.Query("method"); v != "" { vv := v; methodPtr = &vv }
	if v := c.Query("path"); v != "" { vv := v; pathPtr = &vv }
	if v := c.Query("keyword"); v != "" { vv := v; keywordPtr = &vv }
	if v := c.Query("start_at"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil { start := t; startPtr = &start }
	}
	if v := c.Query("end_at"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil { end := t; endPtr = &end }
	}
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)

	items, total, err := ctl.svc.List(userIDPtr, methodPtr, pathPtr, keywordPtr, statusCodePtr, successPtr, startPtr, endPtr, page, pageSize)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// GET /api/v1/system/operation-logs/export
// Stream CSV export using pagination via service.List to avoid loading all rows at once
func (ctl *OperationLogController) Export(c *gin.Context) {
	var (
		userIDPtr *uint64
		statusCodePtr *int
		successPtr *int8
		methodPtr, pathPtr, keywordPtr *string
		startPtr, endPtr *time.Time
	)

	if v := c.Query("user_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil && id > 0 { userID := id; userIDPtr = &userID }
	}
	if v := c.Query("status_code"); v != "" {
		if sc, err := strconv.Atoi(v); err == nil { statusCode := sc; statusCodePtr = &statusCode }
	}
	if v := c.Query("success"); v != "" {
		if s, err := strconv.Atoi(v); err == nil { ss := int8(s); successPtr = &ss }
	}
	if v := c.Query("method"); v != "" { vv := v; methodPtr = &vv }
	if v := c.Query("path"); v != "" { vv := v; pathPtr = &vv }
	if v := c.Query("keyword"); v != "" { vv := v; keywordPtr = &vv }
	if v := c.Query("start_at"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil { start := t; startPtr = &start }
	}
	if v := c.Query("end_at"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil { end := t; endPtr = &end }
	}

	// headers
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=operation-logs.csv")
	// BOM for Excel
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	// header line
	_, _ = c.Writer.Write([]byte("时间,用户ID,方法,路径,状态码,成功,耗时(ms),IP,错误信息\n"))

	// paginate and stream
	page := 1
	pageSize := 1000
	exported := 0
	const maxExport = 50000
	for exported < maxExport {
		items, total, err := ctl.svc.List(userIDPtr, methodPtr, pathPtr, keywordPtr, statusCodePtr, successPtr, startPtr, endPtr, page, pageSize)
		if err != nil { c.Status(http.StatusInternalServerError); return }
		for _, r := range items {
			created := r.CreatedAt.Format(time.RFC3339)
			var uid string
			if r.UserID != nil { uid = strconv.FormatUint(*r.UserID, 10) }
			success := "否"
			if r.Success == 1 { success = "是" }
			var lat string
			if r.LatencyMS != nil { lat = strconv.Itoa(*r.LatencyMS) }
			var ip string
			if r.IP != nil { ip = *r.IP }
			var errMsg string
			if r.ErrorMessage != nil { errMsg = *r.ErrorMessage }

			line := csvJoin([]string{
				created, uid, r.Method, r.Path, strconv.Itoa(r.StatusCode), success, lat, ip, errMsg,
			}) + "\n"
			_, _ = c.Writer.Write([]byte(line))
			exported++
			if exported >= maxExport { break }
		}
		if page*pageSize >= int(total) { break }
		if exported >= maxExport { break }
		page++
	}
}

// csvJoin escapes fields and joins by comma
func csvJoin(fields []string) string {
	out := make([]byte, 0, 256)
	for i, f := range fields {
		// escape quotes
		needsQuote := false
		if len(f) == 0 {
			// allow empty
		}
		for j := 0; j < len(f); j++ {
			if f[j] == '"' || f[j] == ',' || f[j] == '\n' || f[j] == '\r' { needsQuote = true; break }
		}
		if i > 0 { out = append(out, ',') }
		if needsQuote {
			out = append(out, '"')
			for j := 0; j < len(f); j++ {
				if f[j] == '"' { out = append(out, '"', '"') } else { out = append(out, f[j]) }
			}
			out = append(out, '"')
		} else {
			out = append(out, f...)
		}
	}
	return string(out)
}
