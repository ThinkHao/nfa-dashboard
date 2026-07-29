package service

import (
	"context"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"time"
)

// SchoolService 学校服务接口
type SchoolService interface {
	// 获取所有学校
	GetAllSchools(schoolName, region, cp, isKeySchool string, limit, offset int) ([]model.School, int64, error)
	// v2：按院校范围过滤的学校列表（为空时不过滤）
	GetAllSchoolsWithScope(schoolName, region, cp, sort, isKeySchool string, allowedSchoolKeys []model.TrafficScopeSchoolKey, limit, offset int) ([]model.School, int64, error)
	// 获取所有地区
	GetAllRegions() ([]string, error)
	// 获取所有运营商
	GetAllCPs() ([]string, error)
	// v2：按院校范围过滤的地区列表
	GetRegionsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error)
	// v2：按院校范围过滤的运营商列表
	GetCPsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error)
	// 根据过滤条件获取流量数据
	GetTrafficData(ctx context.Context, filter model.TrafficFilter) ([]model.TrafficResponse, error)
	// 获取流量汇总数据
	GetTrafficSummary(ctx context.Context, filter model.TrafficFilter) (model.TrafficResponse, error)
	// 获取院校自然日服务流量
	GetDailyTrafficVolume(ctx context.Context, filter model.TrafficFilter) ([]model.DailyTrafficVolumeResponse, error)
}

// schoolService 学校服务实现
type schoolService struct {
	repo repository.SchoolRepository
}

// NewSchoolService 创建学校服务实例
func NewSchoolService(repo repository.SchoolRepository) SchoolService {
	return &schoolService{
		repo: repo,
	}
}

// GetAllSchools 获取所有学校
func (s *schoolService) GetAllSchools(schoolName, region, cp, isKeySchool string, limit, offset int) ([]model.School, int64, error) {
	// 构建过滤条件
	filter := make(map[string]interface{})
	if schoolName != "" {
		filter["school_name"] = schoolName
	}
	if region != "" {
		filter["region"] = region
	}
	if cp != "" {
		filter["cp"] = cp
	}
	if isKeySchool != "" {
		filter["is_key_school"] = isKeySchool
	}

	return s.repo.GetAllSchools(filter, limit, offset)
}

// GetAllSchoolsWithScope v2：支持按院校范围过滤
func (s *schoolService) GetAllSchoolsWithScope(schoolName, region, cp, sort, isKeySchool string, allowedSchoolKeys []model.TrafficScopeSchoolKey, limit, offset int) ([]model.School, int64, error) {
	filter := make(map[string]interface{})
	if schoolName != "" {
		filter["school_name"] = schoolName
	}
	if region != "" {
		filter["region"] = region
	}
	if cp != "" {
		filter["cp"] = cp
	}
	if sort != "" {
		filter["sort"] = sort
	}
	if isKeySchool != "" {
		filter["is_key_school"] = isKeySchool
	}
	if len(allowedSchoolKeys) > 0 {
		filter["allowed_school_keys"] = allowedSchoolKeys
	}
	return s.repo.GetAllSchools(filter, limit, offset)
}

// GetAllRegions 获取所有地区
func (s *schoolService) GetAllRegions() ([]string, error) {
	return s.repo.GetAllRegions()
}

// GetAllCPs 获取所有运营商
func (s *schoolService) GetAllCPs() ([]string, error) {
	return s.repo.GetAllCPs()
}

// GetRegionsWithScope v2：按院校范围过滤
func (s *schoolService) GetRegionsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error) {
	return s.repo.GetRegionsWithScope(allowedSchoolKeys)
}

// GetCPsWithScope v2：按院校范围过滤
func (s *schoolService) GetCPsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error) {
	return s.repo.GetCPsWithScope(allowedSchoolKeys)
}

// GetTrafficData 根据过滤条件获取流量数据
func (s *schoolService) GetTrafficData(ctx context.Context, filter model.TrafficFilter) ([]model.TrafficResponse, error) {
	// 设置默认时间范围（如果未指定）
	if filter.StartTime.IsZero() {
		filter.StartTime = time.Now().AddDate(0, 0, -7) // 默认过去7天
	}
	if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}

	// 设置默认时间间隔
	if filter.Interval == "" {
		filter.Interval = "hour"
	}

	// 设置默认分页
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	return s.repo.GetTrafficData(ctx, filter)
}

func (s *schoolService) GetDailyTrafficVolume(ctx context.Context, filter model.TrafficFilter) ([]model.DailyTrafficVolumeResponse, error) {
	return s.repo.GetDailyTrafficVolume(ctx, filter)
}

// GetTrafficSummary 获取流量汇总数据
func (s *schoolService) GetTrafficSummary(ctx context.Context, filter model.TrafficFilter) (model.TrafficResponse, error) {
	// 设置默认时间范围（如果未指定）
	if filter.StartTime.IsZero() {
		filter.StartTime = time.Now().AddDate(0, 0, -7) // 默认过去7天
	}
	if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}

	return s.repo.GetTrafficSummary(ctx, filter)
}
