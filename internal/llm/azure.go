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

type AzureProvider struct {
	endpoint   string
	apiKey     string
	deployment string
	client     *http.Client
}

func (p *AzureProvider) Generate(ctx context.Context, prompt string) (Generation, error) {
	payload := map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Generation{}, fmt.Errorf("marshal azure request: %w", err)
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-15-preview", p.endpoint, p.deployment)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Generation{}, fmt.Errorf("create azure request: %w", err)
	}
	req.Header.Set("api-key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Generation{}, fmt.Errorf("execute azure request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Generation{}, fmt.Errorf("read azure response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return Generation{}, fmt.Errorf("azure openai error status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Generation{}, fmt.Errorf("decode azure response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return Generation{}, fmt.Errorf("azure response has no choices")
	}
	return Generation{
		Text: strings.TrimSpace(parsed.Choices[0].Message.Content),
		Usage: Usage{
			InputTokens:  parsed.Usage.PromptTokens,
			OutputTokens: parsed.Usage.CompletionTokens,
		},
	}, nil
}

func (p *AzureProvider) Model() string {
	return p.deployment
}
