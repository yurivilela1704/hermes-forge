package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WebhookTrace records a webhook interaction.
type WebhookTrace struct {
	Timestamp    time.Time `json:"timestamp"`
	Provider     string    `json:"provider"`
	RawPayload   string    `json:"raw_payload"`
	ResponseCode int       `json:"response_code"`
	ResponseBody any       `json:"response_body"`
}

// GenerationRun records an LLM generation attempt.
type GenerationRun struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // "daily", "manual"
	Events      any       `json:"events,omitempty"`
	Prompt      string    `json:"prompt"`
	RawResponse any       `json:"raw_response,omitempty"`
	FinalNote   string    `json:"final_note,omitempty"`
	Error       string    `json:"error,omitempty"`
	IsDryRun    bool      `json:"is_dry_run"`
}

func (s *Store) SaveWebhookTrace(provider string, payload []byte, respCode int, respBody any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.outputDir, "traces", "webhooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create traces dir: %w", err)
	}

	trace := WebhookTrace{
		Timestamp:    time.Now().UTC(),
		Provider:     provider,
		RawPayload:   string(payload),
		ResponseCode: respCode,
		ResponseBody: respBody,
	}

	filename := fmt.Sprintf("%s_%s_%d.json", trace.Timestamp.Format("2006-01-02_15-04-05"), provider, time.Now().UnixNano()%1000)
	fullPath := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal trace: %w", err)
	}

	return os.WriteFile(fullPath, data, 0o644)
}

func (s *Store) SaveGenerationRun(run *GenerationRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.outputDir, "traces", "generations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generations dir: %w", err)
	}

	if run.Timestamp.IsZero() {
		run.Timestamp = time.Now().UTC()
	}

	filename := fmt.Sprintf("%s_%s.json", run.Timestamp.Format("2006-01-02_15-04-05"), run.Type)
	fullPath := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}

	return os.WriteFile(fullPath, data, 0o644)
}
