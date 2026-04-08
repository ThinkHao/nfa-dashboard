package service

import (
	"testing"

	"gorm.io/datatypes"
	"nfa-dashboard/internal/model"
)

func TestMatchRateCustomerFilterRule_ExactMatchRequiresAllNonEmptyConditions(t *testing.T) {
	rule := model.RateCustomerFilterRule{
		ScopeRegion:         datatypes.JSON([]byte(`["华东"]`)),
		ScopeCP:             datatypes.JSON([]byte(`["CMCC"]`)),
		SchoolNameMatchType: "exact",
		SchoolNameValues:    datatypes.JSON([]byte(`["示例大学"]`)),
	}
	customer := model.RateCustomer{
		Region: "华东",
		CP:     "CMCC",
		SchoolName: func() *string {
			v := "示例大学"
			return &v
		}(),
	}

	matched, err := matchRateCustomerFilterRule(customer, rule)
	if err != nil {
		t.Fatalf("matchRateCustomerFilterRule returned error: %v", err)
	}
	if !matched {
		t.Fatalf("expected customer to match exact rule")
	}

	customer.CP = "CT"
	matched, err = matchRateCustomerFilterRule(customer, rule)
	if err != nil {
		t.Fatalf("matchRateCustomerFilterRule returned error after cp change: %v", err)
	}
	if matched {
		t.Fatalf("expected cp mismatch to make the rule fail")
	}
}

func TestMatchRateCustomerFilterRule_ContainsMatchUsesAnySchoolKeyword(t *testing.T) {
	rule := model.RateCustomerFilterRule{
		SchoolNameMatchType: "contains",
		SchoolNameValues:    datatypes.JSON([]byte(`["附属","测试学院"]`)),
	}
	customer := model.RateCustomer{
		Region: "华北",
		CP:     "CT",
		SchoolName: func() *string {
			v := "北京附属实验学校"
			return &v
		}(),
	}

	matched, err := matchRateCustomerFilterRule(customer, rule)
	if err != nil {
		t.Fatalf("matchRateCustomerFilterRule returned error: %v", err)
	}
	if !matched {
		t.Fatalf("expected customer to match contains rule")
	}
}

func TestMatchRateCustomerFilterRule_EmptyScopeMeansUnrestricted(t *testing.T) {
	rule := model.RateCustomerFilterRule{
		SchoolNameMatchType: "contains",
		SchoolNameValues:    datatypes.JSON([]byte(`["国际"]`)),
	}
	customer := model.RateCustomer{
		Region: "西南",
		CP:     "CU",
		SchoolName: func() *string {
			v := "国际学院"
			return &v
		}(),
	}

	matched, err := matchRateCustomerFilterRule(customer, rule)
	if err != nil {
		t.Fatalf("matchRateCustomerFilterRule returned error: %v", err)
	}
	if !matched {
		t.Fatalf("expected empty region/cp scope to allow match")
	}
}

func TestSummarizeMatchedSchoolNames_TruncatesAndCountsRemainder(t *testing.T) {
	got := summarizeMatchedSchoolNames([]string{"一中", "二中", "三中", "四中"}, 3)
	want := "一中、二中、三中 等 1 所"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
