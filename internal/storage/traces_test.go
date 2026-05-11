package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveWebhookTrace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	payload := []byte(`{"object_kind":"merge_request"}`)
	respBody := map[string]any{"accepted": true}
	if err := store.SaveWebhookTrace("gitlab", payload, 202, respBody); err != nil {
		t.Fatalf("SaveWebhookTrace returned error: %v", err)
	}

	traceDir := filepath.Join(dir, "traces", "webhooks")
	files, err := os.ReadDir(traceDir)
	if err != nil {
		t.Fatalf("ReadDir traces returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 trace file, got: %d", len(files))
	}
	if !strings.HasSuffix(files[0].Name(), ".json") {
		t.Fatalf("expected json file, got: %s", files[0].Name())
	}
}

func TestSaveGenerationRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	run := &GenerationRun{
		Timestamp: time.Now().UTC(),
		Type:      "daily",
		Prompt:    "Generate daily report",
		FinalNote: "### Report",
	}
	if err := store.SaveGenerationRun(run); err != nil {
		t.Fatalf("SaveGenerationRun returned error: %v", err)
	}

	genDir := filepath.Join(dir, "traces", "generations")
	files, err := os.ReadDir(genDir)
	if err != nil {
		t.Fatalf("ReadDir generations returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 run file, got: %d", len(files))
	}
}
