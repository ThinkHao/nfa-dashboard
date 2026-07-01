package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"
)

type SystemTrafficScopeController struct {
	svc       service.TrafficScopeService
	userSvc   service.UserService
	schoolSvc service.SchoolService
}

func NewSystemTrafficScopeController(svc service.TrafficScopeService, userSvc service.UserService, schoolSvc service.SchoolService) *SystemTrafficScopeController {
	return &SystemTrafficScopeController{svc: svc, userSvc: userSvc, schoolSvc: schoolSvc}
}

func (ctl *SystemTrafficScopeController) ListUsers(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 20)
	username := c.Query("username")
	items, total, err := ctl.userSvc.List(username, nil, nil, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	type userResp struct {
		ID          uint64 `json:"id"`
		Username    string `json:"username"`
		Alias       string `json:"alias,omitempty"`
		DisplayName string `json:"display_name"`
		Status      int8   `json:"status"`
	}
	resp := make([]userResp, 0, len(items))
	for _, item := range items {
		displayName := item.Username
		alias := ""
		if item.Alias != nil {
			alias = *item.Alias
		}
		if alias != "" {
			displayName = alias
		}
		resp = append(resp, userResp{
			ID:          item.ID,
			Username:    item.Username,
			Alias:       alias,
			DisplayName: displayName,
			Status:      item.Status,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (ctl *SystemTrafficScopeController) ListRules(c *gin.Context) {
	userID, ok := parseTrafficScopeUserID(c)
	if !ok {
		return
	}
	items, err := ctl.svc.ListRules(userID)
	if err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (ctl *SystemTrafficScopeController) ReplaceRules(c *gin.Context) {
	userID, ok := parseTrafficScopeUserID(c)
	if !ok {
		return
	}
	type reqT struct {
		Rules []model.TrafficScopeRuleGroup `json:"rules"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := ctl.svc.ReplaceRules(userID, req.Rules); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctl *SystemTrafficScopeController) Preview(c *gin.Context) {
	userID, ok := parseTrafficScopeUserID(c)
	if !ok {
		return
	}
	preview, err := ctl.svc.Preview(userID)
	if err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (ctl *SystemTrafficScopeController) ListOptions(c *gin.Context) {
	dimension := strings.TrimSpace(c.Query("dimension"))
	keyword := strings.TrimSpace(c.Query("q"))
	limit := parseIntDefault(c.Query("limit"), 50)
	if limit <= 0 {
		limit = 50
	}

	type optionItem struct {
		Value      string `json:"value"`
		Label      string `json:"label"`
		Dimension  string `json:"dimension"`
		SchoolID   string `json:"school_id,omitempty"`
		SchoolName string `json:"school_name,omitempty"`
		Region     string `json:"region,omitempty"`
		CP         string `json:"cp,omitempty"`
	}

	switch dimension {
	case "region":
		items, err := ctl.schoolSvc.GetAllRegions()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		resp := make([]optionItem, 0, len(items))
		for _, item := range items {
			if keyword != "" && !strings.Contains(strings.ToLower(item), strings.ToLower(keyword)) {
				continue
			}
			resp = append(resp, optionItem{Value: item, Label: item, Dimension: "region"})
			if len(resp) >= limit {
				break
			}
		}
		c.JSON(http.StatusOK, gin.H{"items": resp, "total": len(resp)})
		return
	case "cp":
		items, err := ctl.schoolSvc.GetAllCPs()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		resp := make([]optionItem, 0, len(items))
		for _, item := range items {
			if keyword != "" && !strings.Contains(strings.ToLower(item), strings.ToLower(keyword)) {
				continue
			}
			resp = append(resp, optionItem{Value: item, Label: item, Dimension: "cp"})
			if len(resp) >= limit {
				break
			}
		}
		c.JSON(http.StatusOK, gin.H{"items": resp, "total": len(resp)})
		return
	case "school":
		region := strings.TrimSpace(c.Query("region"))
		cp := strings.TrimSpace(c.Query("cp"))
		schools, total, err := ctl.schoolSvc.GetAllSchools(keyword, region, cp, "", limit, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		resp := make([]optionItem, 0, len(schools))
		for _, item := range schools {
			resp = append(resp, optionItem{
				Value:      item.SchoolID,
				Label:      item.SchoolName + " (" + item.SchoolID + ")",
				Dimension:  "school",
				SchoolID:   item.SchoolID,
				SchoolName: item.SchoolName,
				Region:     item.Region,
				CP:         item.CP,
			})
		}
		c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid dimension"})
		return
	}
}

func parseTrafficScopeUserID(c *gin.Context) (uint64, bool) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil || userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid user id"})
		return 0, false
	}
	return userID, true
}
