package service

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"nfa-dashboard/internal/model"
)

func ptrFloat64(v float64) *float64 { return &v }
func ptrUint64(v uint64) *uint64    { return &v }

type edcNodeSettlementRepoStub struct {
	entities           []model.EDCEntity
	points             []model.EDCNodeTrafficPoint
	trafficPointExists bool
	trafficPointErr    error
	existsCalled       bool
}

func (s *edcNodeSettlementRepoStub) ListEnabledEntities() ([]model.EDCEntity, error) {
	return s.entities, nil
}

func (s *edcNodeSettlementRepoStub) ListTrafficPoints(entityID uint64, start, end time.Time) ([]model.EDCTraffic5m, error) {
	return nil, nil
}

func (s *edcNodeSettlementRepoStub) ListTrafficPointsByDisplayNode(start, end time.Time) ([]model.EDCNodeTrafficPoint, error) {
	return s.points, nil
}

func (s *edcNodeSettlementRepoStub) ExistsTrafficPointByDisplayNode(start, end time.Time) (bool, error) {
	s.existsCalled = true
	return s.trafficPointExists, s.trafficPointErr
}

func (s *edcNodeSettlementRepoStub) DeleteDailySettlements(start, end time.Time) error  { return nil }
func (s *edcNodeSettlementRepoStub) DeleteMonthlySettlements(serviceMonth string) error { return nil }
func (s *edcNodeSettlementRepoStub) UpsertDailySettlements(rows []model.SettlementNodeDaily95) error {
	return nil
}
func (s *edcNodeSettlementRepoStub) UpsertMonthlySettlements(rows []model.SettlementNodeMonthly95) error {
	return nil
}
func (s *edcNodeSettlementRepoStub) ListDailySettlements(filter map[string]interface{}, limit, offset int) ([]model.SettlementNodeDaily95, int64, error) {
	return nil, 0, nil
}
func (s *edcNodeSettlementRepoStub) ListMonthlySettlements(filter map[string]interface{}, limit, offset int) ([]model.SettlementNodeMonthly95, int64, error) {
	return nil, 0, nil
}

func TestEDCNodeRatePrefersEntitySpecificRate(t *testing.T) {
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	rates := []model.RateFinalNode{
		{Region: "天津", CP: "bilibili", SettlementMode: EDCSettlementModeDaily95Avg, UnitBase: 1000, FinalFee: ptrFloat64(3), RackFee: ptrFloat64(100)},
		{EntityID: ptrUint64(12), DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili", SettlementMode: EDCSettlementModeRange95, UnitBase: 1024, FinalFee: ptrFloat64(8), RackFee: ptrFloat64(200)},
	}

	got, ok := selectFinalNodeRate(entity, rates)
	if !ok {
		t.Fatalf("selectFinalNodeRate() did not find a rate")
	}
	if got.EntityID == nil || *got.EntityID != 12 {
		t.Fatalf("expected entity-specific rate, got entity_id=%v", got.EntityID)
	}
	if got.UnitBase != 1024 || got.SettlementMode != EDCSettlementModeRange95 {
		t.Fatalf("got unit_base=%d mode=%s, want 1024/%s", got.UnitBase, got.SettlementMode, EDCSettlementModeRange95)
	}
}

func TestEDCNodeRateFallsBackToRegionCP(t *testing.T) {
	entity := model.EDCEntity{ID: 99, DisplayName: "TJ-Node-B", Region: "天津", CP: "bilibili"}
	rates := []model.RateFinalNode{
		{Region: "天津", CP: "bilibili", SettlementMode: EDCSettlementModeDaily95Avg, UnitBase: 1000, FinalFee: ptrFloat64(3), RackFee: ptrFloat64(100)},
	}

	got, ok := selectFinalNodeRate(entity, rates)
	if !ok {
		t.Fatalf("selectFinalNodeRate() did not fall back to region+cp")
	}
	if got.EntityID != nil {
		t.Fatalf("fallback rate should not be entity-specific, got entity_id=%v", got.EntityID)
	}
}

func TestEDCNodeRateSelectionMatchesSettlementMode(t *testing.T) {
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	rates := []model.RateFinalNode{
		{EntityID: ptrUint64(12), DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili", SettlementMode: EDCSettlementModeDaily95Avg, UnitBase: 1000, FinalFee: ptrFloat64(3)},
		{EntityID: ptrUint64(12), DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili", SettlementMode: EDCSettlementModeRange95, UnitBase: 1000, FinalFee: ptrFloat64(8)},
	}

	got, ok := selectFinalNodeRateForSettlement(entity, rates, EDCSettlementModeRange95)
	if !ok {
		t.Fatalf("selectFinalNodeRateForSettlement() did not find monthly rate")
	}
	if got.FinalFee == nil || *got.FinalFee != 8 {
		t.Fatalf("expected monthly final fee 8, got %+v", got.FinalFee)
	}
}

func TestEDCNodeRateSelectionSkipsEmptyTrafficUnitPrice(t *testing.T) {
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	rates := []model.RateFinalNode{
		{EntityID: ptrUint64(12), DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili", SettlementMode: EDCSettlementModeRange95, UnitBase: 1000},
	}

	if _, ok := selectFinalNodeRateForSettlement(entity, rates, EDCSettlementModeRange95); ok {
		t.Fatalf("expected empty traffic unit price to be skipped")
	}
}

func TestEDCServiceSizeToMbpsSupports1000And1024(t *testing.T) {
	raw := int64(300 * 1000 * 1000)
	got1000 := edcRawToMbps(raw, 1000)
	got1024 := edcRawToMbps(raw, 1024)
	if math.Abs(got1000-8) > 0.000001 {
		t.Fatalf("edcRawToMbps(1000)=%f, want 8", got1000)
	}
	if got1000 <= got1024 {
		t.Fatalf("expected 1000-base Mbps > 1024-base Mbps, got %f <= %f", got1000, got1024)
	}
}

func TestBuildEDCNodeMonthlyTrafficSettlementDoesNotRequireRate(t *testing.T) {
	month := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}

	row := buildEDCNodeMonthlyTrafficSettlement(entity, month, EDCSettlementModeRange95, 1000, 300000000, 8)
	if row.EntityID != entity.ID || row.DisplayName != entity.DisplayName {
		t.Fatalf("row entity mismatch: %+v", row)
	}
	if row.SettlementMode != EDCSettlementModeRange95 || row.UnitBase != 1000 {
		t.Fatalf("got mode/base=%s/%d, want %s/1000", row.SettlementMode, row.UnitBase, EDCSettlementModeRange95)
	}
	if row.Raw95 != 300000000 || row.Mbps95 != 8 || row.SettlementValue != 8 {
		t.Fatalf("unexpected traffic values: raw=%f mbps=%f value=%f", row.Raw95, row.Mbps95, row.SettlementValue)
	}
	if row.Monthly95Fee != nil || row.TrafficBill != nil || row.TotalBill != nil {
		t.Fatalf("traffic-only settlement should not populate fee fields: %+v", row)
	}
}

func TestHasSettlementTrafficUsesRepositoryExistenceCheck(t *testing.T) {
	repo := &edcNodeSettlementRepoStub{trafficPointExists: true}
	svc := &edcNodeSettlementService{repo: repo}
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)

	ok, err := svc.HasSettlementTraffic(start, end)
	if err != nil {
		t.Fatalf("HasSettlementTraffic() error=%v", err)
	}
	if !ok {
		t.Fatalf("HasSettlementTraffic()=false, want true")
	}
	if !repo.existsCalled {
		t.Fatalf("HasSettlementTraffic() did not use existence check")
	}
}

func TestHasSettlementTrafficReturnsFalseWithoutPoints(t *testing.T) {
	svc := &edcNodeSettlementService{repo: &edcNodeSettlementRepoStub{trafficPointExists: false}}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)

	ok, err := svc.HasSettlementTraffic(start, end)
	if err != nil {
		t.Fatalf("HasSettlementTraffic() error=%v", err)
	}
	if ok {
		t.Fatalf("HasSettlementTraffic()=true, want false")
	}
}

func TestHasSettlementTrafficReturnsRepositoryError(t *testing.T) {
	wantErr := errors.New("exists query failed")
	svc := &edcNodeSettlementService{repo: &edcNodeSettlementRepoStub{trafficPointErr: wantErr}}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)

	ok, err := svc.HasSettlementTraffic(start, end)
	if !errors.Is(err, wantErr) {
		t.Fatalf("HasSettlementTraffic() error=%v, want %v", err, wantErr)
	}
	if ok {
		t.Fatalf("HasSettlementTraffic()=true, want false when repository errors")
	}
}

func TestEDCNodeTrafficGroupingUsesVisibleNode(t *testing.T) {
	points := []model.EDCNodeTrafficPoint{
		{EntityID: 1, Region: "BJ", CP: "ali", DisplayName: "BJ-ali-01", Bucket5m: time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local), ServiceSize: 100},
		{EntityID: 1, Region: "BJ", CP: "ali", DisplayName: "BJ-ali-01", Bucket5m: time.Date(2026, 4, 1, 0, 5, 0, 0, time.Local), ServiceSize: 200},
	}

	grouped := groupEDCNodeTrafficPoints(points)
	if len(grouped) != 1 {
		t.Fatalf("group count=%d, want 1", len(grouped))
	}
	keys := sortedEDCNodeKeys(grouped)
	if len(keys) != 1 || edcNodeGroupLabel(keys[0]) != "BJ/ali/BJ-ali-01" {
		t.Fatalf("unexpected node key: %+v", keys)
	}
	raw, ok := computeNodeRange95Raw(grouped[keys[0]])
	if !ok || raw != 100 {
		t.Fatalf("raw95=%f ok=%v, want 100/true", raw, ok)
	}
}

func TestEDCDaily95AvgAndRange95Calculations(t *testing.T) {
	baseDay := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	points := []model.EDCTraffic5m{
		{Bucket5m: baseDay.Add(time.Minute), ServiceSize: 100},
		{Bucket5m: baseDay.Add(2 * time.Minute), ServiceSize: 200},
		{Bucket5m: baseDay.Add(3 * time.Minute), ServiceSize: 300},
		{Bucket5m: baseDay.AddDate(0, 0, 1).Add(time.Minute), ServiceSize: 1000},
		{Bucket5m: baseDay.AddDate(0, 0, 1).Add(2 * time.Minute), ServiceSize: 2000},
		{Bucket5m: baseDay.AddDate(0, 0, 1).Add(3 * time.Minute), ServiceSize: 3000},
	}

	daily, ok := computeDaily95AvgRaw(points)
	if !ok {
		t.Fatalf("computeDaily95AvgRaw() returned no value")
	}
	if daily != 1100 {
		t.Fatalf("daily average raw=%f, want 1100", daily)
	}

	ranged, ok := computeRange95Raw(points)
	if !ok {
		t.Fatalf("computeRange95Raw() returned no value")
	}
	if ranged != 2000 {
		t.Fatalf("range95 raw=%f, want 2000", ranged)
	}
}

func TestBuildEDCNodeMonthlySettlementAddsRackFeeOnce(t *testing.T) {
	month := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	rate := model.RateFinalNode{
		EntityID:       ptrUint64(12),
		DisplayName:    "TJ-Node-A",
		Region:         "天津",
		CP:             "bilibili",
		SettlementMode: EDCSettlementModeRange95,
		UnitBase:       1000,
		FinalFee:       ptrFloat64(2),
		RackFee:        ptrFloat64(50),
	}

	row := buildEDCNodeMonthlySettlement(entity, rate, month, 80, 10, 1000)
	if row.TrafficBill == nil || *row.TrafficBill != 20 {
		t.Fatalf("traffic_bill=%v, want 20", row.TrafficBill)
	}
	if row.RackBill == nil || *row.RackBill != 50 {
		t.Fatalf("rack_bill=%v, want 50", row.RackBill)
	}
	if row.TotalBill == nil || *row.TotalBill != 70 {
		t.Fatalf("total_bill=%v, want 70", row.TotalBill)
	}
}

func TestBuildEDCNodeDailySettlementAddsAllFeeItems(t *testing.T) {
	day := time.Date(2026, 5, 2, 0, 0, 0, 0, time.Local)
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	rate := model.RateFinalNode{
		EntityID:       ptrUint64(12),
		DisplayName:    "TJ-Node-A",
		Region:         "天津",
		CP:             "bilibili",
		SettlementMode: EDCSettlementModeDaily95Avg,
		FinalFee:       ptrFloat64(2),
		CPFee:          ptrFloat64(10),
		RackFee:        ptrFloat64(50),
		OtherFee:       ptrFloat64(5),
	}

	row := buildEDCNodeDailySettlement(entity, rate, day, 80, 10, 1000)
	if row.UnitBase != 1000 {
		t.Fatalf("unit_base=%d, want 1000", row.UnitBase)
	}
	if row.TrafficBill == nil || *row.TrafficBill != 20 {
		t.Fatalf("traffic_bill=%v, want 20", row.TrafficBill)
	}
	if row.CPBill == nil || *row.CPBill != 10 {
		t.Fatalf("cp_bill=%v, want 10", row.CPBill)
	}
	if row.RackBill == nil || *row.RackBill != 50 {
		t.Fatalf("rack_bill=%v, want 50", row.RackBill)
	}
	if row.OtherBill == nil || *row.OtherBill != 5 {
		t.Fatalf("other_bill=%v, want 5", row.OtherBill)
	}
	if row.TotalBill == nil || *row.TotalBill != 85 {
		t.Fatalf("total_bill=%v, want 85", row.TotalBill)
	}
}

func TestBuildEDCNodeMonthlySettlementAddsAllFeeItems(t *testing.T) {
	month := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	rate := model.RateFinalNode{
		EntityID:       ptrUint64(12),
		DisplayName:    "TJ-Node-A",
		Region:         "天津",
		CP:             "bilibili",
		SettlementMode: EDCSettlementModeRange95,
		FinalFee:       ptrFloat64(3),
		CPFee:          ptrFloat64(10),
		RackFee:        ptrFloat64(50),
		OtherFee:       ptrFloat64(5),
	}

	row := buildEDCNodeMonthlySettlement(entity, rate, month, 80, 10, 1024)
	if row.UnitBase != 1024 {
		t.Fatalf("unit_base=%d, want 1024", row.UnitBase)
	}
	if row.TrafficBill == nil || *row.TrafficBill != 30 {
		t.Fatalf("traffic_bill=%v, want 30", row.TrafficBill)
	}
	if row.CPBill == nil || *row.CPBill != 10 {
		t.Fatalf("cp_bill=%v, want 10", row.CPBill)
	}
	if row.RackBill == nil || *row.RackBill != 50 {
		t.Fatalf("rack_bill=%v, want 50", row.RackBill)
	}
	if row.OtherBill == nil || *row.OtherBill != 5 {
		t.Fatalf("other_bill=%v, want 5", row.OtherBill)
	}
	if row.TotalBill == nil || *row.TotalBill != 95 {
		t.Fatalf("total_bill=%v, want 95", row.TotalBill)
	}
}

func TestBuildEDCNodeSettlementRowsForBothUnitBases(t *testing.T) {
	day := time.Date(2026, 5, 2, 0, 0, 0, 0, time.Local)
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	rate := model.RateFinalNode{
		EntityID:       ptrUint64(12),
		DisplayName:    "TJ-Node-A",
		Region:         "天津",
		CP:             "bilibili",
		SettlementMode: EDCSettlementModeDaily95Avg,
		FinalFee:       ptrFloat64(2),
	}

	rows := buildEDCNodeDailySettlementRows(entity, rate, day, 300*1000*1000)
	if len(rows) != 2 {
		t.Fatalf("row count=%d, want 2", len(rows))
	}
	if rows[0].UnitBase != 1000 || rows[1].UnitBase != 1024 {
		t.Fatalf("unit bases=%d/%d, want 1000/1024", rows[0].UnitBase, rows[1].UnitBase)
	}
	if rows[0].Raw95 != rows[1].Raw95 {
		t.Fatalf("raw95 should match, got %f/%f", rows[0].Raw95, rows[1].Raw95)
	}
	if math.Abs(rows[0].Mbps95-8) > 0.000001 {
		t.Fatalf("1000-base mbps=%f, want 8", rows[0].Mbps95)
	}
	if rows[0].Mbps95 <= rows[1].Mbps95 {
		t.Fatalf("expected 1000-base Mbps > 1024-base Mbps, got %f <= %f", rows[0].Mbps95, rows[1].Mbps95)
	}
}

func TestCalculateDailyGeneratesBothUnitBasesAndSkipsMissingRate(t *testing.T) {
	day := time.Date(2026, 5, 2, 0, 0, 0, 0, time.Local)
	entityWithRate := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	entityWithoutRate := model.EDCEntity{ID: 13, DisplayName: "TJ-Node-B", Region: "天津", CP: "bilibili"}
	svc := &edcNodeSettlementService{
		repo: &edcNodeSettlementRepoStub{
			entities: []model.EDCEntity{entityWithRate, entityWithoutRate},
			points: []model.EDCNodeTrafficPoint{
				{EntityID: 12, Region: "天津", CP: "bilibili", DisplayName: "TJ-Node-A", Bucket5m: day, ServiceSize: 300 * 1000 * 1000},
				{EntityID: 13, Region: "天津", CP: "bilibili", DisplayName: "TJ-Node-B", Bucket5m: day, ServiceSize: 300 * 1000 * 1000},
			},
		},
		ratesRepo: &ratesServiceRatesRepoStub{
			finalNodeRates: []model.RateFinalNode{
				{EntityID: ptrUint64(12), DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili", SettlementMode: EDCSettlementModeDaily95Avg, FinalFee: ptrFloat64(2)},
			},
		},
	}

	rows, processed, err := svc.calculateDaily(day)
	if err != nil {
		t.Fatalf("calculateDaily() error=%v", err)
	}
	if processed != 2 || len(rows) != 2 {
		t.Fatalf("processed/rows=%d/%d, want 2/2", processed, len(rows))
	}
	if rows[0].EntityID != 12 || rows[1].EntityID != 12 {
		t.Fatalf("unexpected entity ids: %d/%d", rows[0].EntityID, rows[1].EntityID)
	}
	if rows[0].UnitBase != 1000 || rows[1].UnitBase != 1024 {
		t.Fatalf("unit bases=%d/%d, want 1000/1024", rows[0].UnitBase, rows[1].UnitBase)
	}
}

func TestCalculateMonthlyFailsWhenAllTrafficNodesMissRate(t *testing.T) {
	month := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	entity := model.EDCEntity{ID: 12, DisplayName: "TJ-Node-A", Region: "天津", CP: "bilibili"}
	svc := &edcNodeSettlementService{
		repo: &edcNodeSettlementRepoStub{
			entities: []model.EDCEntity{entity},
			points: []model.EDCNodeTrafficPoint{
				{EntityID: 12, Region: "天津", CP: "bilibili", DisplayName: "TJ-Node-A", Bucket5m: month, ServiceSize: 300 * 1000 * 1000},
			},
		},
		ratesRepo: &ratesServiceRatesRepoStub{},
	}

	rows, processed, err := svc.calculateMonthly(month)
	if err == nil {
		t.Fatalf("calculateMonthly() expected error")
	}
	if processed != 0 || len(rows) != 0 {
		t.Fatalf("processed/rows=%d/%d, want 0/0", processed, len(rows))
	}
	if !strings.Contains(err.Error(), "没有可用的 EDC 节点月95费率或流量单价") {
		t.Fatalf("unexpected error: %v", err)
	}
}
