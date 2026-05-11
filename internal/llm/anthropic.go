package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func (p *AnthropicProvider) Generate(ctx context.Context, prompt string) (Generation, error) {
	payload := map[string]any{
		"model":      p.model,
		"max_tokens": 1200,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Generation{}, fmt.Errorf("marshal anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return Generation{}, fmt.Errorf("create anthropic request: %w", err)
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Generation{}, fmt.Errorf("execute anthropic request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Generation{}, fmt.Errorf("read anthropic response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return Generation{}, fmt.Errorf("anthropic error status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Generation{}, fmt.Errorf("decode anthropic response: %w", err)
	}
	for _, part := range parsed.Content {
		if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
			return Generation{
				Text: strings.TrimSpace(part.Text),
				Usage: Usage{
					InputTokens:  parsed.Usage.InputTokens,
					OutputTokens: parsed.Usage.OutputTokens,
				},
			}, nil
		}
	}
	return Generation{}, fmt.Errorf("anthropic response has no text content")
}

func (p *AnthropicProvider) Model() string {
	return p.model
}
