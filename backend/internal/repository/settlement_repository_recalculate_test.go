package repository

import (
	"testing"

	"nfa-dashboard/internal/model"
)

func TestFilterSchoolCombos(t *testing.T) {
	combos := []model.SchoolRegionCP{
		{SchoolID: "1", SchoolName: "天津大学城", Region: "天津市", CP: "ali"},
		{SchoolID: "2", SchoolName: "天津大学城", Region: "天津市", CP: "bilibili"},
		{SchoolID: "3", SchoolName: "其他学校", Region: "天津市", CP: "ali"},
		{SchoolID: "4", SchoolName: "天津大学城", Region: "北京市", CP: "ali"},
	}

	got := filterSchoolCombos(combos, "天津市", "ali", "天津大学城")
	if len(got) != 1 || got[0].SchoolID != "1" {
		t.Fatalf("filterSchoolCombos() = %#v, want only school 1", got)
	}
}
