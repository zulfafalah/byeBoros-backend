// Package llm classifies free-form text with an OpenAI-compatible chat
// completions endpoint. It is used to pick the best transaction category for a
// spoken transaction.
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

// CategoryClassifier asks a chat model to choose one of the candidate
// categories for a transaction description.
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

const systemPrompt = `Kamu mengklasifikasikan transaksi keuangan berbahasa Indonesia ke salah satu kategori yang disediakan. Balas HANYA JSON valid dengan format {"category":"<kategori>"}. Nilai category harus persis salah satu dari daftar kategori yang diberikan. Jika tidak yakin, pilih yang paling mendekati. Jangan menambahkan penjelasan.`

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

type categoryPayload struct {
	Category string `json:"category"`
}

// Classify returns the candidate category the model matches best. The returned
// value is the original candidate string.
func (c *CategoryClassifier) Classify(ctx context.Context, description, transcript string, candidates []string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("classifier is not enabled")
	}
	candidates = trimCandidates(candidates)
	if len(candidates) == 0 {
		return "", fmt.Errorf("no candidate categories")
	}

	user := fmt.Sprintf("Deskripsi: %s\nTranskrip: %s\nKategori yang tersedia: %s",
		description, transcript, strings.Join(candidates, ", "))

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
		return "", fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("llm error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	category, err := parseCategory(content)
	if err != nil {
		return "", err
	}
	return matchCandidate(category, candidates)
}

// parseCategory extracts the category value from the model content, which is
// expected to be a JSON object but may include surrounding prose or code fences.
func parseCategory(content string) (string, error) {
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "`")
	if start := strings.Index(content, "{"); start >= 0 {
		if end := strings.LastIndex(content, "}"); end > start {
			var payload categoryPayload
			if err := json.Unmarshal([]byte(content[start:end+1]), &payload); err == nil && strings.TrimSpace(payload.Category) != "" {
				return strings.TrimSpace(payload.Category), nil
			}
		}
	}
	if content == "" {
		return "", fmt.Errorf("empty llm content")
	}
	return content, nil
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
// tolerating case, whitespace and simple word-form differences.
func matchCandidate(answer string, candidates []string) (string, error) {
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
