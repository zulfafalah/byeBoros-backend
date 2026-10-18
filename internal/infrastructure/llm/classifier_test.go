package llm

import (
	"context"
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

func TestClassifyMatchesCandidate(t *testing.T) {
	c := newTestClassifier(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"category\":\"makan\"}"}}]}`))
	})

	candidates := []string{"Makan", "Transport", "Belanja"}
	got, err := c.Classify(context.Background(), "beli kopi", "beli kopi 15 ribu", candidates)
	if err != nil {
		t.Fatalf("Classify returned error: %v", err)
	}
	if got != "Makan" {
		t.Fatalf("got %q, want %q", got, "Makan")
	}
}

func TestClassifyExtractsJSONFromProse(t *testing.T) {
	c := newTestClassifier(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"Berikut hasilnya:\\n```json\\n{\\\"category\\\":\\\"Transport\\\"}\\n```\"}}]}"))
	})

	got, err := c.Classify(context.Background(), "naik ojek", "naik ojek 20 ribu", []string{"Makan", "Transport"})
	if err != nil {
		t.Fatalf("Classify returned error: %v", err)
	}
	if got != "Transport" {
		t.Fatalf("got %q, want %q", got, "Transport")
	}
}

func TestClassifyErrorStatus(t *testing.T) {
	c := newTestClassifier(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	})

	if _, err := c.Classify(context.Background(), "x", "x", []string{"Makan"}); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestClassifyUnknownCategory(t *testing.T) {
	c := newTestClassifier(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"category\":\"Ngasal\"}"}}]}`))
	})

	if _, err := c.Classify(context.Background(), "x", "x", []string{"Makan", "Transport"}); err == nil {
		t.Fatal("expected error when model category is not a candidate")
	}
}

func TestClassifyDisabled(t *testing.T) {
	c := NewCategoryClassifier("", "", "", 0)
	if c.Enabled() {
		t.Fatal("classifier should be disabled without url/model")
	}
	if _, err := c.Classify(context.Background(), "x", "x", []string{"Makan"}); err == nil {
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
