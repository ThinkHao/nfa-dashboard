package repository

import (
	"strings"
	"testing"

	"nfa-dashboard/internal/model"
)

func TestComparisonSchoolScopeClausesUsesExactSchoolKeys(t *testing.T) {
	clause, args := comparisonSchoolScopeClauses([]model.TrafficScopeSchoolKey{
		{SchoolID: "1001", Region: "北京市", CP: "bilibili"},
		{SchoolID: "1002", Region: "北京市", CP: "bilibili"},
	})
	if strings.Count(clause, "s.school_id = ?") != 2 {
		t.Fatalf("clause=%q, want two exact school predicates", clause)
	}
	if len(args) != 6 || args[0] != "1001" || args[3] != "1002" {
		t.Fatalf("args=%v, want two school keys", args)
	}
}

func TestComparisonSchoolScopeClausesEmptyDoesNotFilter(t *testing.T) {
	clause, args := comparisonSchoolScopeClauses(nil)
	if clause != "" || args != nil {
		t.Fatalf("clause=%q args=%v, want empty scope", clause, args)
	}
}

func TestFilterComparisonSchoolsForGroupUsesSourceRegionAndExactSchoolScope(t *testing.T) {
	bj := "北京市"
	sh := "上海市"
	schools := []model.School{
		{SchoolID: "bj-2", Region: "河北省", SrcRegion: &bj, CP: "bilibili", SchoolName: "学校 A"},
		{SchoolID: "bj-1", Region: "吉林省", SrcRegion: &bj, CP: "bilibili", SchoolName: "学校 B"},
		{SchoolID: "sh-1", Region: "上海市", SrcRegion: &sh, CP: "jinshan", SchoolName: "学校 C"},
		{SchoolID: "bj-other-cp", Region: "北京市", SrcRegion: &bj, CP: "jinshan", SchoolName: "学校 D"},
	}
	filtered := filterComparisonSchoolsForGroup(schools, []model.TrafficScopeSchoolKey{
		{SchoolID: "bj-2", Region: "河北省", SrcRegion: &bj, CP: "bilibili"},
		{SchoolID: "sh-1", Region: "上海市", SrcRegion: &sh, CP: "jinshan"},
		{SchoolID: "bj-1", Region: "北京市", SrcRegion: &bj, CP: "bilibili"},
	}, "北京市", "bilibili")
	if len(filtered) != 1 || filtered[0].SchoolID != "bj-2" || filtered[0].Region != "河北省" {
		t.Fatalf("filtered=%+v, want the Beijing-source school owned by Hebei within exact scope", filtered)
	}
}

func TestFilterComparisonSchoolsForGroupEmptyScopeIsUnrestricted(t *testing.T) {
	bj := "北京市"
	schools := []model.School{{SchoolID: "bj-1", Region: "吉林省", SrcRegion: &bj, CP: "bilibili", SchoolName: "学校 A"}}
	filtered := filterComparisonSchoolsForGroup(schools, nil, "北京市", "bilibili")
	if len(filtered) != 1 || filtered[0].SchoolID != "bj-1" {
		t.Fatalf("filtered=%+v, want unfiltered source-area schools", filtered)
	}
}

func TestComparisonNFASchoolClausesUseIndexableExactDimensions(t *testing.T) {
	clause, args := comparisonNFASchoolClauses([]model.School{
		{SchoolID: "1001", Region: "北京市", SchoolName: "学校 A"},
		{SchoolID: "1002", Region: "河北省", SchoolName: "学校 B"},
	})
	if strings.Count(clause, "s.school_name = ?") != 2 || strings.Count(clause, "s.region = ?") != 2 {
		t.Fatalf("clause=%q, want exact region/name predicates for each school", clause)
	}
	if len(args) != 6 || args[0] != "1001" || args[3] != "1002" {
		t.Fatalf("args=%v, want exact school keys", args)
	}
}
