// Package llm analyzes spoken transactions with an OpenAI-compatible chat
// completions endpoint: it produces a clean title, the best matching category,
// the priority and the transaction type.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Analysis is the structured result of analyzing a transcript.
type Analysis struct {
	Title    string `json:"title"`
	Notes    string `json:"notes"`
	Category string `json:"category"`
	Priority string `json:"priority"`
	Type     string `json:"type"`
}

// CategoryClassifier asks a chat model to summarize a transaction and choose
// one of the candidate categories.
type CategoryClassifier struct {
	url    string
	apiKey string
	model  string
	http   *http.Client
}

// NewCategoryClassifier builds a classifier. url must be a full chat
// completions endpoint (e.g. https://api.groq.com/openai/v1/chat/completions).
func NewCategoryClassifier(url, apiKey, model string, timeout time.Duration) *CategoryClassifier {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &CategoryClassifier{
		url:    url,
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{Timeout: timeout},
	}
}

// Enabled reports whether the classifier is configured.
func (c *CategoryClassifier) Enabled() bool {
	return c != nil && c.url != "" && c.model != ""
}

const systemPrompt = `Kamu membantu mencatat transaksi keuangan berbahasa Indonesia.
Balas HANYA JSON valid dengan format:
{"title":"...","notes":"...","category":"...","priority":"high|medium|low","type":"expense|income"}
Aturan:
- "title": judul singkat transaksi (kira-kira maksimal 6 kata), huruf awal kapital. Rapikan nama barang/tempat (contoh: "indomart" -> "Indomart"). Jangan sertakan nominal uang.
- "notes": jika ucapan berisi keterangan/detail tambahan atau kalimat panjang, tulis ulang menjadi satu kalimat yang rapi dan jelas (huruf awal kapital, ejaan dibetulkan). Jika tidak ada keterangan tambahan, gunakan string kosong.
- "category": WAJIB persis salah satu dari daftar kategori yang diberikan. Jika tidak ada yang cocok, gunakan string kosong.
- "priority": "high", "medium", atau "low" sesuai kata prioritas pada ucapan (mis. "prioritas tinggi" -> "high"); default "medium".
- "type": "expense" untuk pengeluaran, "income" untuk pemasukan.
Jangan menambahkan penjelasan apa pun.`

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	Temperature    float64           `json:"temperature"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Analyze summarizes the transcript and fills in the category, priority and
// type. transport/parse failures return an error; individual fields that cannot
// be validated are left empty.
func (c *CategoryClassifier) Analyze(ctx context.Context, transcript, description, typeHint string, candidates []string) (Analysis, error) {
	if !c.Enabled() {
		return Analysis{}, fmt.Errorf("classifier is not enabled")
	}
	candidates = trimCandidates(candidates)

	user := fmt.Sprintf("Ucapan: %s\nDeskripsi terdeteksi: %s\nJenis terdeteksi: %s\nKategori yang tersedia: %s",
		transcript, description, typeHint, strings.Join(candidates, ", "))

	body, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: user},
		},
		Temperature:    0,
		ResponseFormat: map[string]string{"type": "json_object"},
	})
	if err != nil {
		return Analysis{}, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return Analysis{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Analysis{}, fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Analysis{}, fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Analysis{}, fmt.Errorf("llm returned %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return Analysis{}, fmt.Errorf("decode llm response: %w", err)
	}
	if parsed.Error != nil {
		return Analysis{}, fmt.Errorf("llm error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return Analysis{}, fmt.Errorf("llm returned no choices")
	}

	raw, err := parseAnalysis(parsed.Choices[0].Message.Content)
	if err != nil {
		return Analysis{}, err
	}
	return c.validate(raw, candidates), nil
}

// parseAnalysis extracts the JSON object from the model content, which may be
// wrapped in prose or code fences.
func parseAnalysis(content string) (Analysis, error) {
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "`")
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return Analysis{}, fmt.Errorf("llm content is not a JSON object")
	}
	var a Analysis
	if err := json.Unmarshal([]byte(content[start:end+1]), &a); err != nil {
		return Analysis{}, fmt.Errorf("decode llm content: %w", err)
	}
	a.Title = strings.TrimSpace(a.Title)
	a.Notes = strings.TrimSpace(a.Notes)
	a.Category = strings.TrimSpace(a.Category)
	a.Priority = strings.TrimSpace(a.Priority)
	a.Type = strings.TrimSpace(a.Type)
	return a, nil
}

// validate keeps only the fields that match the allowed values and maps the
// category back to the original candidate string.
func (c *CategoryClassifier) validate(a Analysis, candidates []string) Analysis {
	out := Analysis{Title: a.Title, Notes: a.Notes}
	if a.Category != "" {
		if match, err := matchCandidate(a.Category, candidates); err == nil {
			out.Category = match
		}
	}
	switch strings.ToLower(a.Priority) {
	case "high", "medium", "low":
		out.Priority = strings.ToLower(a.Priority)
	}
	switch strings.ToLower(a.Type) {
	case "income", "expense":
		out.Type = strings.ToLower(a.Type)
	}
	return out
}

func trimCandidates(candidates []string) []string {
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if s := strings.TrimSpace(c); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// matchCandidate maps the model answer back to the original candidate string,
// tolerating case and simple word-form differences. An empty candidate list
// yields an error so callers fall back to keyword matching.
func matchCandidate(answer string, candidates []string) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("no candidate categories")
	}
	answer = strings.TrimSpace(answer)
	for _, c := range candidates {
		if strings.EqualFold(c, answer) {
			return c, nil
		}
	}
	for _, c := range candidates {
		if tokensPrefixRelated(c, answer) {
			return c, nil
		}
	}
	return "", fmt.Errorf("llm category %q not in candidate list", answer)
}

// tokensPrefixRelated reports whether two phrases share a word that is equal or
// has a common prefix of at least four characters (makan/makanan).
func tokensPrefixRelated(a, b string) bool {
	aTokens := strings.FieldsFunc(strings.ToLower(a), isSeparator)
	bTokens := strings.FieldsFunc(strings.ToLower(b), isSeparator)
	const minPrefix = 4
	for _, at := range aTokens {
		for _, bt := range bTokens {
			if at == bt {
				return true
			}
			if len(at) >= minPrefix && len(bt) >= minPrefix && at[:minPrefix] == bt[:minPrefix] {
				return true
			}
		}
	}
	return false
}

func isSeparator(r rune) bool {
	return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r < 0x80
}
