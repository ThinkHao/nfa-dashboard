package model

import "testing"

func TestRateCustomerHasCompleteCustomerPrices(t *testing.T) {
	zero := 0.0
	value := 1.25
	allZeroAndValue := RateCustomer{CustomerFee: &zero, NetworkLineFee: &value, GeneralFee: &zero}
	if !allZeroAndValue.HasCompleteCustomerPrices() {
		t.Fatal("expected zero-valued prices to count as configured")
	}

	missingGeneral := RateCustomer{CustomerFee: &value, NetworkLineFee: &zero}
	if missingGeneral.HasCompleteCustomerPrices() {
		t.Fatal("expected nil general fee to remain incomplete")
	}
	if got := missingGeneral.MissingCustomerPriceFields(); len(got) != 1 || got[0] != "general_fee" {
		t.Fatalf("unexpected missing fields: %#v", got)
	}
}
