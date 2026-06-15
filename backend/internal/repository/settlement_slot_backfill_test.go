package repository

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"nfa-dashboard/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestBackfillSlot_DailyIncrementalKeepsAllDays reproduces the production data-loss
// bug where consecutive single-day (nightly) backfills drop alternating service
// dates from the active slot because of the partial "hit-key" slot copy.
//
// Run with a reachable MySQL:
//
//	NFA_TEST_MYSQL_DSN="root:pass@tcp(host:3306)/nfa?parseTime=true&loc=Local&charset=utf8mb4" \
//	  go test ./internal/repository -run TestBackfillSlot_DailyIncrementalKeepsAllDays -v
func TestBackfillSlot_DailyIncrementalKeepsAllDays(t *testing.T) {
	dsn := os.Getenv("NFA_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NFA_TEST_MYSQL_DSN not set; skipping MySQL integration test")
	}

	srcSchema, testDSN := deriveTestDSN(dsn, "nfa_slot_test_bug")

	// Bootstrap an isolated test schema by cloning the real table definitions.
	admin, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect source: %v", err)
	}
	exec(t, admin, "DROP DATABASE IF EXISTS nfa_slot_test_bug")
	exec(t, admin, "CREATE DATABASE nfa_slot_test_bug DEFAULT CHARSET utf8mb4")
	clone := []string{
		"settlement_customer_v",
		"settlement_customer_monthly_v",
		"settlement_month_slot_pointer",
		"nfa_school_settlement",
		"rate_customer",
		"rate_discount_rule",
		"rate_discount_rule_item",
	}
	for _, tbl := range clone {
		exec(t, admin, fmt.Sprintf("CREATE TABLE nfa_slot_test_bug.%s LIKE %s.%s", tbl, srcSchema, tbl))
	}
	t.Cleanup(func() { exec(t, admin, "DROP DATABASE IF EXISTS nfa_slot_test_bug") })

	db, err := gorm.Open(mysql.Open(testDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test schema: %v", err)
	}
	prevDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = prevDB })

	const region, cp, school, schoolID = "TestRegion", "testcp", "TestSchool", "S1"

	// Seed the source table with one daily 95 row per day for June 1..14.
	loc := time.Local
	const days = 14
	for d := 1; d <= days; d++ {
		date := time.Date(2026, 6, d, 0, 0, 0, 0, loc)
		row := model.SchoolSettlement{
			SchoolID:        schoolID,
			SchoolName:      school,
			Region:          region,
			CP:              cp,
			SettlementValue: int64(1000 + d),
			SettlementTime:  date.Add(2 * time.Hour),
			SettlementDate:  date,
			CreateTime:      time.Now(),
			UpdateTime:      time.Now(),
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed source day %d: %v", d, err)
		}
	}

	// Simulate the nightly job: one single-day backfill per day, in order.
	repo := NewSettlementDataRepository()
	for d := 1; d <= days; d++ {
		date := time.Date(2026, 6, d, 0, 0, 0, 0, loc)
		if _, err := repo.BackfillFromSchoolSettlement(region, cp, school, date, date, false, nil); err != nil {
			t.Fatalf("backfill day %d: %v", d, err)
		}
	}

	// Read what an end user would see: rows in the currently active slot.
	var activeSlot int8
	if err := db.Raw("SELECT active_slot FROM settlement_month_slot_pointer WHERE service_month = ?", "2026-06").
		Scan(&activeSlot).Error; err != nil {
		t.Fatalf("read pointer: %v", err)
	}
	var visibleDays []int
	if err := db.Raw(
		"SELECT DAY(service_date) FROM settlement_customer_v WHERE service_month = ? AND slot = ? ORDER BY service_date",
		"2026-06", activeSlot,
	).Scan(&visibleDays).Error; err != nil {
		t.Fatalf("read active slot: %v", err)
	}

	if len(visibleDays) != days {
		t.Fatalf("active slot must contain all %d days, got %d: %v", days, len(visibleDays), visibleDays)
	}
}

func exec(t *testing.T, db *gorm.DB, sql string) {
	t.Helper()
	if err := db.Exec(sql).Error; err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

// deriveTestDSN extracts the source schema from a DSN and returns a DSN pointing
// at the given test schema (keeping the same connection params).
func deriveTestDSN(dsn, testSchema string) (srcSchema, testDSN string) {
	slash := strings.LastIndex(dsn, "/")
	rest := dsn[slash+1:]
	if q := strings.Index(rest, "?"); q >= 0 {
		return rest[:q], dsn[:slash+1] + testSchema + rest[q:]
	}
	return rest, dsn[:slash+1] + testSchema
}
