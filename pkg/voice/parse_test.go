package voice

import (
	"testing"
	"time"
)

func TestParseUserExample(t *testing.T) {
	now := time.Date(2026, 3, 10, 14, 0, 0, 0, time.UTC)
	got := Parse("beli warteg 10000 kemarin prioritas tinggi sama Aulia", now)

	if got.Amount != 10000 {
		t.Errorf("amount = %d, want 10000", got.Amount)
	}
	if got.Type != TypeExpense {
		t.Errorf("type = %q, want expense", got.Type)
	}
	if got.Description != "warteg" {
		t.Errorf("description = %q, want warteg", got.Description)
	}
	if got.Priority != PriorityHigh {
		t.Errorf("priority = %q, want high", got.Priority)
	}
	if got.Notes != "aulia" {
		t.Errorf("notes = %q, want aulia", got.Notes)
	}
	if !got.DateSet || got.Date.Format("2006-01-02") != "2026-03-09" {
		t.Errorf("date = %v (set=%v), want 2026-03-09", got.Date, got.DateSet)
	}
}

func TestParseAmounts(t *testing.T) {
	now := time.Now()
	cases := []struct {
		transcript string
		want       int64
	}{
		{"catat pengeluaran makan siang lima puluh ribu", 50000},
		{"kopi 25.000", 25000},
		{"nasi goreng 50rb", 50000},
		{"langganan 5jt", 5000000},
		{"parkir seribu", 1000},
		{"belanja seratus lima puluh ribu", 150000},
		{"dapat bonus sejuta", 1000000},
		{"bayar listrik dua ratus ribu", 200000},
	}
	for _, tc := range cases {
		if got := Parse(tc.transcript, now).Amount; got != tc.want {
			t.Errorf("Parse(%q).Amount = %d, want %d", tc.transcript, got, tc.want)
		}
	}
}

func TestParseType(t *testing.T) {
	now := time.Now()
	if got := Parse("terima gaji 5000000", now).Type; got != TypeIncome {
		t.Errorf("type = %q, want income", got)
	}
	if got := Parse("bayar kos 1500000", now).Type; got != TypeExpense {
		t.Errorf("type = %q, want expense", got)
	}
}

func TestParsePriority(t *testing.T) {
	now := time.Now()
	cases := map[string]string{
		"beli obat 20000 mendesak":        PriorityHigh,
		"jajan 10000 santai":              PriorityLow,
		"beli buku 30000 prioritas biasa": PriorityMedium,
		"beli air 5000":                   "",
	}
	for transcript, want := range cases {
		if got := Parse(transcript, now).Priority; got != want {
			t.Errorf("Parse(%q).Priority = %q, want %q", transcript, got, want)
		}
	}
}

func TestMatchCategory(t *testing.T) {
	candidates := []string{"Makanan", "Transportasi", "Tagihan", "Hiburan"}
	if got, ok := MatchCategory("makan siang warteg", candidates); !ok || got != "Makanan" {
		t.Errorf("MatchCategory = (%q,%v), want (Makanan,true)", got, ok)
	}
	if _, ok := MatchCategory("warteg", candidates); ok {
		t.Error("MatchCategory should not match unrelated description")
	}
}
