package service

import (
	"testing"

	"nfa-dashboard/internal/model"
)

func TestNormalizeDiscountRuleDraftsAllowsProvinceDefaultAndSchoolOverride(t *testing.T) {
	region := "陕西"
	draft := []model.RateDiscountRuleDraft{
		{
			Rule:   model.RateDiscountRule{Name: "省级默认", ScopeType: "region", Enabled: true},
			Scopes: []model.RateDiscountRuleScope{{ScopeType: "region", ScopeKey: &region}},
			Items:  []model.RateDiscountRuleItem{{FromYear: 1, DiscountRate: 0.8}},
		},
		{
			Rule: model.RateDiscountRule{Name: "院校覆盖", ScopeType: "school", Enabled: true},
			Scopes: []model.RateDiscountRuleScope{{
				ScopeType:  "school",
				Region:     stringPointer("陕西"),
				CP:         stringPointer("运营商A"),
				SchoolName: stringPointer("示例大学"),
			}},
			Items: []model.RateDiscountRuleItem{{FromYear: 1, DiscountRate: 0.6}},
		},
	}

	normalized, err := normalizeDiscountRuleDrafts(draft)
	if err != nil {
		t.Fatalf("expected province default and school override to be valid, got %v", err)
	}
	if got := *normalized[0].Rule.ScopeKey; got != "陕西" {
		t.Fatalf("province rule scope key = %q, want 陕西", got)
	}
	if got := discountSchoolKey(normalized[1].Scopes[0]); got != "陕西|运营商A|示例大学" {
		t.Fatalf("school scope key = %q, want province/CP/school tuple", got)
	}
}

func TestNormalizeDiscountRuleDraftsRejectsDuplicateSchoolTarget(t *testing.T) {
	target := model.RateDiscountRuleScope{
		ScopeType:  "school",
		Region:     stringPointer("河北"),
		CP:         stringPointer("运营商A"),
		SchoolName: stringPointer("示例大学"),
	}
	drafts := []model.RateDiscountRuleDraft{
		{Rule: model.RateDiscountRule{Name: "规则一", ScopeType: "school"}, Scopes: []model.RateDiscountRuleScope{target}},
		{Rule: model.RateDiscountRule{Name: "规则二", ScopeType: "school"}, Scopes: []model.RateDiscountRuleScope{target}},
	}

	if _, err := normalizeDiscountRuleDrafts(drafts); err == nil {
		t.Fatal("expected duplicate school assignment to be rejected")
	}
}

func TestDiscountScopesConflictUsesQualifiedSchoolIdentity(t *testing.T) {
	base := model.RateDiscountRuleScope{ScopeType: "region", ScopeKey: stringPointer("辽宁")}
	schoolA := model.RateDiscountRuleScope{
		ScopeType:  "school",
		Region:     stringPointer("辽宁"),
		CP:         stringPointer("运营商A"),
		SchoolName: stringPointer("同名院校"),
	}
	schoolSameNameOtherCP := model.RateDiscountRuleScope{
		ScopeType:  "school",
		Region:     stringPointer("辽宁"),
		CP:         stringPointer("运营商B"),
		SchoolName: stringPointer("同名院校"),
	}
	legacySchool := model.RateDiscountRuleScope{ScopeType: "school", SchoolName: stringPointer("同名院校")}

	if discountScopesConflict(base, schoolA) {
		t.Fatal("school override must coexist with the province default")
	}
	if discountScopesConflict(schoolA, schoolSameNameOtherCP) {
		t.Fatal("same-name schools under different CPs must not conflict")
	}
	if !discountScopesConflict(schoolA, schoolA) {
		t.Fatal("identical qualified school targets must conflict")
	}
	if !discountScopesConflict(legacySchool, schoolA) {
		t.Fatal("legacy name-wide school rules must conflict with matching qualified targets")
	}
}

func stringPointer(value string) *string { return &value }
