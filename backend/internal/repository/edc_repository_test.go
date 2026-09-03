package repository

import (
	"database/sql"
	"strings"
	"testing"

	"nfa-dashboard/internal/model"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestBuildEDCTrafficOrderByOnlyOrdersGroupedDimensions(t *testing.T) {
	tests := []struct {
		name                 string
		filter               model.EDCTrafficFilter
		includeEntityDetails bool
		want                 string
	}{
		{
			name:   "entity type only",
			filter: model.EDCTrafficFilter{EntityType: model.EDCEntityTypeNode},
			want:   "t.bucket_5m ASC, t.entity_type ASC",
		},
		{
			name:   "entity and cp",
			filter: model.EDCTrafficFilter{EntityIDs: []uint64{1}, CP: "ali"},
			want:   "t.bucket_5m ASC, t.cp ASC",
		},
		{
			name:   "source and destination regions",
			filter: model.EDCTrafficFilter{SrcRegion: "北京市", DstRegion: "河北省"},
			want:   "t.bucket_5m ASC, t.src_region ASC, t.dst_region ASC",
		},
		{
			name:                 "entity details",
			includeEntityDetails: true,
			want:                 "t.bucket_5m ASC, t.region ASC, t.cp ASC, t.display_name ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildEDCTrafficOrderBy(tt.filter, tt.includeEntityDetails); got != tt.want {
				t.Fatalf("buildEDCTrafficOrderBy() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyEDCEntityFilterAlwaysExcludesDisabledAndBackups(t *testing.T) {
	sqlDB, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:1)/nfa_test")
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}

	var entities []model.EDCEntity
	query := applyEDCEntityFilter(db.Model(&model.EDCEntity{}), model.EDCEntityFilter{})
	query.Find(&entities)
	sql := query.Statement.SQL.String()
	if !strings.Contains(sql, "enabled = ?") || !strings.Contains(sql, "is_backup = ?") {
		t.Fatalf("entity filter SQL = %q, want enabled and backup predicates", sql)
	}
}
