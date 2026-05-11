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

type OpenAIProvider struct {
	apiKey       string
	model        string
	organization string
	baseURI      string
	client       *http.Client
}

func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (Generation, error) {
	payload := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Generation{}, fmt.Errorf("marshal openai request: %w", err)
	}

	baseURI := p.baseURI
	if baseURI == "" {
		baseURI = "https://api.openai.com/v1"
	}
	baseURI = strings.TrimSuffix(baseURI, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURI+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Generation{}, fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	if p.organization != "" {
		req.Header.Set("OpenAI-Organization", p.organization)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Generation{}, fmt.Errorf("execute openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Generation{}, fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return Generation{}, fmt.Errorf("openai error status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
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
		return Generation{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return Generation{}, fmt.Errorf("openai response has no choices")
	}
	return Generation{
		Text: strings.TrimSpace(parsed.Choices[0].Message.Content),
		Usage: Usage{
			InputTokens:  parsed.Usage.PromptTokens,
			OutputTokens: parsed.Usage.CompletionTokens,
		},
	}, nil
}

func (p *OpenAIProvider) Model() string {
	return p.model
}
