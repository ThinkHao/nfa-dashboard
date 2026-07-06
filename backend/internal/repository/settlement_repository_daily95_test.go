package repository

import (
	"sort"
	"testing"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/settlement95"
)

func TestPick95PointMatchesDescendingIndex(t *testing.T) {
	values := []int64{10, 50, 30, 90, 70, 20, 60, 40, 80, 100}
	times := make([]time.Time, len(values))
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := range times {
		times[i] = base.Add(time.Duration(i) * time.Minute)
	}

	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a] > sorted[b] })
	wantVal := sorted[settlement95.DescendingIndex(len(sorted))]

	gotVal, gotTime := pick95Point(values, times)
	if gotVal != wantVal {
		t.Fatalf("95 值不一致: got %d want %d", gotVal, wantVal)
	}
	found := false
	for i, v := range values {
		if v == gotVal && times[i].Equal(gotTime) {
			found = true
		}
	}
	if !found {
		t.Fatal("95 时间点必须来自值等于 95 值的采样点")
	}
}

func TestPick95PointSinglePoint(t *testing.T) {
	v, at := pick95Point([]int64{7}, []time.Time{time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)})
	if v != 7 || at.Hour() != 8 {
		t.Fatalf("单点应原样返回: v=%d at=%v", v, at)
	}
}

func TestCalculateDaily95ForCombosMatchesLegacy(t *testing.T) {
	openLockTestDB(t)
	repo := NewSettlementRepository()
	date := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)

	// 插入学校行：旧方法 CalculateDaily95WithRegionAndCP 依赖 nfa_school 表存在该组合
	school := model.School{
		SchoolID:        "TEST95",
		SchoolName:      "测试校",
		Region:          "浙江",
		CP:              "CT",
		HashUUIDs:       "t95-hash",
		PrimaryHashUUID: "t95-hash",
		HashCount:       1,
		DataHash:        "t95-datahash",
	}
	if err := model.DB.Create(&school).Error; err != nil {
		t.Fatal(err)
	}
	defer model.DB.Where("school_id = ?", "TEST95").Delete(&model.School{})

	// 插入 3 个测试采样点（聚合方法只依赖流量行）
	points := []model.SchoolTraffic{
		{CreateTime: date.Add(1 * time.Minute), SchoolID: "TEST95", SchoolName: "测试校", Region: "浙江", CP: "CT", HashUUID: "t95-1", TotalRecv: 100},
		{CreateTime: date.Add(2 * time.Minute), SchoolID: "TEST95", SchoolName: "测试校", Region: "浙江", CP: "CT", HashUUID: "t95-2", TotalRecv: 300},
		{CreateTime: date.Add(3 * time.Minute), SchoolID: "TEST95", SchoolName: "测试校", Region: "浙江", CP: "CT", HashUUID: "t95-3", TotalRecv: 200},
	}
	if err := model.DB.Create(&points).Error; err != nil {
		t.Fatal(err)
	}
	defer model.DB.Where("school_id = ?", "TEST95").Delete(&model.SchoolTraffic{})

	combos := []model.SchoolRegionCP{{SchoolID: "TEST95", SchoolName: "测试校", Region: "浙江", CP: "CT"}}
	got, err := repo.CalculateDaily95ForCombos(date, combos)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("应命中 1 个组合, got %d", len(got))
	}
	// 独立断言：n=3 时 DescendingIndex 的期望取值，用于固化预期数学口径
	sorted := []int64{300, 200, 100}
	want := sorted[settlement95.DescendingIndex(3)]
	if got[0].SettlementValue != want {
		t.Fatalf("95 值: got %d want %d", got[0].SettlementValue, want)
	}

	// 真实新旧对比：直接调用旧方法逐字段核对（值 100/300/200 互不相同，无并列点，不受排序稳定性影响）
	legacy, err := repo.CalculateDaily95WithRegionAndCP(date, "TEST95", "浙江", "CT")
	if err != nil {
		t.Fatalf("旧方法计算失败: %v", err)
	}
	if got[0].SettlementValue != legacy.SettlementValue {
		t.Fatalf("新旧 95 值不一致: new %d legacy %d", got[0].SettlementValue, legacy.SettlementValue)
	}
	if !got[0].SettlementTime.Equal(legacy.SettlementTime) {
		t.Fatalf("新旧 95 时间不一致: new %v legacy %v", got[0].SettlementTime, legacy.SettlementTime)
	}
}
