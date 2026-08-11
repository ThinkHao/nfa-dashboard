package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseEDCTrafficFilterParsesEntityIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/api/v2/edc/traffic?entity_ids=1,2,3&entity_type=transmission&src_region=北京市&dst_region=天津市", nil)
	ctx.Request = req

	filter, ok := parseEDCTrafficFilter(ctx)
	if !ok {
		t.Fatalf("parseEDCTrafficFilter() ok=false")
	}
	if len(filter.EntityIDs) != 3 || filter.EntityIDs[0] != 1 || filter.EntityIDs[1] != 2 || filter.EntityIDs[2] != 3 {
		t.Fatalf("EntityIDs=%v, want [1 2 3]", filter.EntityIDs)
	}
	if filter.EntityType != "transmission" || filter.SrcRegion != "北京市" || filter.DstRegion != "天津市" {
		t.Fatalf("dimensions=%q/%q/%q", filter.EntityType, filter.SrcRegion, filter.DstRegion)
	}
}

func TestParseEDCTrafficFilterRejectsInvalidEntityIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v2/edc/traffic?entity_ids=1,abc", nil)
	ctx.Request = req

	_, ok := parseEDCTrafficFilter(ctx)
	if ok {
		t.Fatalf("parseEDCTrafficFilter() ok=true, want false")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestParseEDCTrafficFilterRejectsInvalidEntityType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v2/edc/traffic?entity_type=other", nil)

	_, ok := parseEDCTrafficFilter(ctx)
	if ok {
		t.Fatalf("parseEDCTrafficFilter() ok=true, want false")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}
