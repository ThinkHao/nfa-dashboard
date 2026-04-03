package service

import (
	"testing"

	"nfa-dashboard/internal/model"
)

func TestParseActionsSet_TemplateFinalFeeMapsToChannelRate(t *testing.T) {
	data := []byte(`{"type":"template","values":{"final_fee":0.12}}`)

	got, err := parseActionsSet(data)
	if err != nil {
		t.Fatalf("parseActionsSet returned error: %v", err)
	}

	if _, ok := got["final_fee"]; ok {
		t.Fatalf("expected final_fee to be normalized away, got map: %#v", got)
	}
	if got["channel_rate"] != 0.12 {
		t.Fatalf("expected channel_rate=0.12, got %#v", got["channel_rate"])
	}
}

func TestParseActionsSet_TemplateChannelRatePreferredOverFinalFee(t *testing.T) {
	data := []byte(`{"type":"template","values":{"channel_rate":0.23,"final_fee":0.11}}`)

	got, err := parseActionsSet(data)
	if err != nil {
		t.Fatalf("parseActionsSet returned error: %v", err)
	}

	if _, ok := got["final_fee"]; ok {
		t.Fatalf("expected final_fee to be normalized away, got map: %#v", got)
	}
	if got["channel_rate"] != 0.23 {
		t.Fatalf("expected channel_rate to keep explicit value 0.23, got %#v", got["channel_rate"])
	}
}

func TestApplyRuleToCustomer_ChannelRateAlwaysUpdatesTopField(t *testing.T) {
	svc := &ratesSyncService{}
	rc := &model.RateCustomer{}
	rule := model.RateCustomerSyncRule{OverwriteStrategy: "always"}

	updated, updates, err := svc.applyRuleToCustomer(rc, rule, nil, map[string]interface{}{"channel_rate": 0.66}, "r", "c", "s")
	if err != nil {
		t.Fatalf("applyRuleToCustomer returned error: %v", err)
	}
	if !updated {
		t.Fatalf("expected update=true")
	}
	if rc.ChannelRate == nil || *rc.ChannelRate != 0.66 {
		t.Fatalf("expected rc.ChannelRate=0.66, got %+v", rc.ChannelRate)
	}
	if v, ok := updates["channel_rate"]; !ok || v != 0.66 {
		t.Fatalf("expected updates[channel_rate]=0.66, got %#v", updates)
	}
}

func TestApplyRuleToCustomer_ChannelRateIfEmptyRespectsExistingValue(t *testing.T) {
	svc := &ratesSyncService{}
	current := 0.31
	rc := &model.RateCustomer{ChannelRate: &current}
	rule := model.RateCustomerSyncRule{OverwriteStrategy: "if_empty"}

	updated, _, err := svc.applyRuleToCustomer(rc, rule, nil, map[string]interface{}{"channel_rate": 0.99}, "r", "c", "s")
	if err != nil {
		t.Fatalf("applyRuleToCustomer returned error: %v", err)
	}
	if updated {
		t.Fatalf("expected update=false when channel_rate already exists")
	}
	if rc.ChannelRate == nil || *rc.ChannelRate != 0.31 {
		t.Fatalf("expected rc.ChannelRate unchanged=0.31, got %+v", rc.ChannelRate)
	}
}
