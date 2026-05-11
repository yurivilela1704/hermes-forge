package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"release-notes-gen/internal/webhook"
)

func TestSaveListReadNote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	meta, err := store.SaveNote(webhook.Event{
		Provider:   "github",
		Repository: "acme/hermes",
		Title:      "Add test",
		Author:     "yuri",
		Branch:     "feature/storage",
		Number:     12,
		URL:        "https://github.com/acme/hermes/pull/12",
	}, "### Summary\nStored note")
	if err != nil {
		t.Fatalf("SaveNote returned error: %v", err)
	}
	if !strings.HasSuffix(meta.Filename, ".md") {
		t.Fatalf("expected markdown file, got: %s", meta.Filename)
	}

	items, err := store.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 note in index, got: %d", len(items))
	}

	content, err := store.ReadNote(meta.Filename)
	if err != nil {
		t.Fatalf("ReadNote returned error: %v", err)
	}
	if !strings.Contains(content, "Stored note") {
		t.Fatalf("expected saved content, got: %s", content)
	}
}

func TestSaveListEvents(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	evt := webhook.Event{
		Provider:   "github",
		Repository: "acme/hermes",
		Title:      "Add test",
		Author:     "yuri",
		Branch:     "feature/storage",
		Number:     12,
		URL:        "https://github.com/acme/hermes/pull/12",
	}

	if err := store.SaveEvent(evt); err != nil {
		t.Fatalf("SaveEvent returned error: %v", err)
	}

	events, err := store.ListEvents("")
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got: %d", len(events))
	}

	if events[0].Title != evt.Title {
		t.Fatalf("expected title %q, got: %q", evt.Title, events[0].Title)
	}
}

func TestSaveEventCorrectDatePrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	mergedAt, _ := time.Parse("2006-01-02", "2026-04-17")
	evt := webhook.Event{
		Provider:   "gitlab",
		Repository: "acme/hermes",
		Title:      "Past event",
		Number:     42,
		MergedAt:   mergedAt,
	}

	if err := store.SaveEvent(evt); err != nil {
		t.Fatalf("SaveEvent returned error: %v", err)
	}

	// Check if the file is in the right place with the right prefix
	eventsDir := filepath.Join(dir, "events")
	files, err := os.ReadDir(eventsDir)
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}

	found := false
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "2026-04-17") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected event file with prefix 2026-04-17, but none found in %v", files)
	}

	// Also check ListEvents
	events, err := store.ListEvents("2026-04-17")
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event for 2026-04-17, got %d", len(events))
	}
}
