package service

import (
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"time"
)

// SchoolService 学校服务接口
type SchoolService interface {
	// 获取所有学校
	GetAllSchools(schoolName, region, cp string, limit, offset int) ([]model.School, int64, error)
	// 获取所有地区
	GetAllRegions() ([]string, error)
	// 获取所有运营商
	GetAllCPs() ([]string, error)
	// 根据过滤条件获取流量数据
	GetTrafficData(filter model.TrafficFilter) ([]model.TrafficResponse, error)
	// 获取流量汇总数据
	GetTrafficSummary(filter model.TrafficFilter) (model.TrafficResponse, error)
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
func (s *schoolService) GetAllSchools(schoolName, region, cp string, limit, offset int) ([]model.School, int64, error) {
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

// GetTrafficData 根据过滤条件获取流量数据
func (s *schoolService) GetTrafficData(filter model.TrafficFilter) ([]model.TrafficResponse, error) {
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

	return s.repo.GetTrafficData(filter)
}

// GetTrafficSummary 获取流量汇总数据
func (s *schoolService) GetTrafficSummary(filter model.TrafficFilter) (model.TrafficResponse, error) {
	// 设置默认时间范围（如果未指定）
	if filter.StartTime.IsZero() {
		filter.StartTime = time.Now().AddDate(0, 0, -7) // 默认过去7天
	}
	if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}

	return s.repo.GetTrafficSummary(filter)
}
