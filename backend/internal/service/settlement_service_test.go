package service

import "testing"

func TestShouldFailCustomerInitOnZeroAffected(t *testing.T) {
	cases := []struct {
		name     string
		srcCount int64
		affected int64
		want     bool
	}{
		{name: "source positive affected zero", srcCount: 10, affected: 0, want: true},
		{name: "source zero affected zero", srcCount: 0, affected: 0, want: false},
		{name: "source positive affected positive", srcCount: 10, affected: 5, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldFailCustomerInitOnZeroAffected(tc.srcCount, tc.affected)
			if got != tc.want {
				t.Fatalf("shouldFailCustomerInitOnZeroAffected(%d,%d)=%v, want=%v", tc.srcCount, tc.affected, got, tc.want)
			}
		})
	}
}
