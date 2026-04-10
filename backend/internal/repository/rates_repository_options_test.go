package repository

import (
	"reflect"
	"testing"
)

func TestNormalizeDistinctOptionValues(t *testing.T) {
	got := normalizeDistinctOptionValues([]string{" 华东 ", "", "CMCC", "华东", "  ", "CMCC", "华南"})
	want := []string{"华东", "CMCC", "华南"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeDistinctOptionValues() = %#v, want %#v", got, want)
	}
}
