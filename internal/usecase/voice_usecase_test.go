package usecase

import "testing"

func TestParseTransactionAt(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"5/4/2026 09:15:00", true},
		{"05/04/2026 09:15:00", true},
		{"5/4/2026 9:15", true},
		{"2026-04-05 09:15:00", true},
		{"2026-04-05T09:15:00", true},
		{"2026-04-05", true},
		{"", false},
		{"kemarin", false},
	}

	for _, tc := range cases {
		_, ok := parseTransactionAt(tc.in)
		if ok != tc.ok {
			t.Fatalf("parseTransactionAt(%q) ok = %v, want %v", tc.in, ok, tc.ok)
		}
	}
}
