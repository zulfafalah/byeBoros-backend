package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClassifier(t *testing.T, handler http.HandlerFunc) *CategoryClassifier {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewCategoryClassifier(srv.URL, "test-key", "test-model", 5*time.Second)
}

// refNow is the reference time stamped into prompts during tests.
var refNow = time.Date(2026, time.April, 6, 14, 30, 0, 0, time.UTC)

func writeContent(w http.ResponseWriter, content string) {
	body, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]string{"content": content}},
		},
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func TestAnalyzeFull(t *testing.T) {
	c := newTestClassifier(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization header = %q", got)
		}
		writeContent(w, `{"title":"Belanja susu dan rinso di Indomart","notes":"Belanja susu dan rinso di Indomart untuk stok bulan ini.","amount":44000,"transaction_at":"5/4/2026 09:15:00","category":"Belanja","priority":"high","type":"expense"}`)
	})

	got, err := c.Analyze(context.Background(), refNow, "belanja susu dan rinso ke indomart habis 44ribu", "susu dan rinso ke indomart", "expense", []string{"Makan", "Belanja", "Transport"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got.Title != "Belanja susu dan rinso di Indomart" {
		t.Fatalf("title = %q", got.Title)
	}
	if got.Notes != "Belanja susu dan rinso di Indomart untuk stok bulan ini." {
		t.Fatalf("notes = %q", got.Notes)
	}
	if got.Amount != 44000 {
		t.Fatalf("amount = %d, want 44000", got.Amount)
	}
	if got.TransactionAt != "5/4/2026 09:15:00" {
		t.Fatalf("transaction_at = %q", got.TransactionAt)
	}
	if got.Category != "Belanja" {
		t.Fatalf("category = %q", got.Category)
	}
	if got.Priority != "high" {
		t.Fatalf("priority = %q", got.Priority)
	}
	if got.Type != "expense" {
		t.Fatalf("type = %q", got.Type)
	}
}

func TestAnalyzeExtractsJSONFromProse(t *testing.T) {
	c := newTestClassifier(t, func(w http.ResponseWriter, r *http.Request) {
		writeContent(w, "Berikut hasilnya:\n```json\n{\"title\":\"Naik ojek\",\"category\":\"transport\",\"priority\":\"low\",\"type\":\"expense\"}\n```")
	})

	got, err := c.Analyze(context.Background(), refNow, "naik ojek 20 ribu", "ojek", "expense", []string{"Makan", "Transport"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got.Category != "Transport" {
		t.Fatalf("category = %q, want Transport", got.Category)
	}
	if got.Priority != "low" {
		t.Fatalf("priority = %q, want low", got.Priority)
	}
}

func TestAnalyzeDropsInvalidFields(t *testing.T) {
	c := newTestClassifier(t, func(w http.ResponseWriter, r *http.Request) {
		writeContent(w, `{"title":"Makan siang","amount":-5,"category":"Ngasal","priority":"super","type":"transfer"}`)
	})

	got, err := c.Analyze(context.Background(), refNow, "makan siang", "makan siang", "expense", []string{"Makan", "Transport"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got.Title != "Makan siang" {
		t.Fatalf("title = %q", got.Title)
	}
	if got.Amount != 0 {
		t.Fatalf("non-positive amount should be dropped, got %d", got.Amount)
	}
	if got.Category != "" {
		t.Fatalf("category should be dropped, got %q", got.Category)
	}
	if got.Priority != "" {
		t.Fatalf("priority should be dropped, got %q", got.Priority)
	}
	if got.Type != "" {
		t.Fatalf("type should be dropped, got %q", got.Type)
	}
}

func TestAnalyzeErrorStatus(t *testing.T) {
	c := newTestClassifier(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	})

	if _, err := c.Analyze(context.Background(), refNow, "x", "x", "expense", []string{"Makan"}); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestAnalyzeDisabled(t *testing.T) {
	c := NewCategoryClassifier("", "", "", 0)
	if c.Enabled() {
		t.Fatal("classifier should be disabled without url/model")
	}
	if _, err := c.Analyze(context.Background(), refNow, "x", "x", "expense", []string{"Makan"}); err == nil {
		t.Fatal("expected error when disabled")
	}
}

func TestMatchCandidateSubstring(t *testing.T) {
	got, err := matchCandidate("Makanan", []string{"Makan & Minum", "Transport"})
	if err != nil {
		t.Fatalf("matchCandidate returned error: %v", err)
	}
	if got != "Makan & Minum" {
		t.Fatalf("got %q, want %q", got, "Makan & Minum")
	}
}
