package controller

import (
	"testing"

	"nfa-dashboard/internal/model"
)

func TestIntersectTrafficScopeSchoolKeys(t *testing.T) {
	left := []model.TrafficScopeSchoolKey{
		{SchoolID: "a", Region: "华东", CP: "cmcc"},
		{SchoolID: "b", Region: "华东", CP: "ctcc"},
	}
	right := []model.TrafficScopeSchoolKey{
		{SchoolID: "a", Region: "华东", CP: "cmcc"},
		{SchoolID: "c", Region: "华南", CP: "cmcc"},
	}
	out := intersectTrafficScopeSchoolKeys(left, right)
	if len(out) != 1 || out[0].SchoolID != "a" || out[0].CP != "cmcc" {
		t.Fatalf("expected only school a to remain, got %#v", out)
	}
}

func TestIntersectTrafficScopeSchoolKeys_EmptyIntersection(t *testing.T) {
	left := []model.TrafficScopeSchoolKey{
		{SchoolID: "a", Region: "华东", CP: "cmcc"},
	}
	right := []model.TrafficScopeSchoolKey{
		{SchoolID: "b", Region: "华东", CP: "ctcc"},
	}
	out := intersectTrafficScopeSchoolKeys(left, right)
	if len(out) != 0 {
		t.Fatalf("expected empty intersection, got %#v", out)
	}
}
