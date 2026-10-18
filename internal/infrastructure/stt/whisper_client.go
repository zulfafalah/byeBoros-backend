// Package stt is a small client for OpenAI-compatible speech-to-text
// endpoints. It works with a self-hosted whisper.cpp server (/inference or
// /v1/audio/transcriptions) as well as hosted providers such as Groq.
package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

// Client posts audio to an OpenAI-compatible transcription endpoint.
type Client struct {
	url      string
	apiKey   string
	model    string
	language string
	prompt   string
	beamSize int
	http     *http.Client
}

// NewClient builds a client. url is the full endpoint (e.g.
// http://stt:8080/inference for whisper.cpp or
// https://api.groq.com/openai/v1/audio/transcriptions). model, apiKey, prompt
// and beamSize are optional; beamSize <= 0 omits the field (for providers that
// don't support it, e.g. Groq).
func NewClient(url, apiKey, model, language, prompt string, beamSize int, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		url:      url,
		apiKey:   apiKey,
		model:    model,
		language: language,
		prompt:   prompt,
		beamSize: beamSize,
		http:     &http.Client{Timeout: timeout},
	}
}

type transcriptionResponse struct {
	Text string `json:"text"`
}

// Transcribe uploads audio and returns the transcribed text.
func (c *Client) Transcribe(ctx context.Context, filename string, audio []byte) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("empty audio payload")
	}
	if filename == "" {
		filename = "audio.wav"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("write audio: %w", err)
	}
	_ = writer.WriteField("response_format", "json")
	_ = writer.WriteField("temperature", "0")
	if c.model != "" {
		_ = writer.WriteField("model", c.model)
	}
	if c.language != "" {
		_ = writer.WriteField("language", c.language)
	}
	if c.beamSize > 0 {
		_ = writer.WriteField("beam_size", strconv.Itoa(c.beamSize))
	}
	if c.prompt != "" {
		_ = writer.WriteField("prompt", c.prompt)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, &body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("stt request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read stt response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("stt returned %d: %s", resp.StatusCode, string(payload))
	}

	var parsed transcriptionResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", fmt.Errorf("decode stt response: %w", err)
	}
	return parsed.Text, nil
}
