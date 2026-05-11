package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"release-notes-gen/internal/webhook"
)

func (s *Store) SaveEvents(events []webhook.Event) error {
	if len(events) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use the first event's date or today
	datePrefix := events[0].MergedAt.Format("2006-01-02")
	if events[0].MergedAt.IsZero() {
		datePrefix = time.Now().UTC().Format("2006-01-02")
	}

	eventsDir := filepath.Join(s.outputDir, "events", datePrefix)
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		return fmt.Errorf("create events dir: %w", err)
	}

	// Create a unique filename for the consolidated events
	// Using project name (slugified) if possible
	projectName := "consolidated"
	if len(events) > 0 && events[0].Repository != "" {
		projectName = slugify(events[0].Repository)
	}

	filename := fmt.Sprintf("%s_sync_%s.json", projectName, time.Now().Format("150405"))
	fullPath := filepath.Join(eventsDir, filename)

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return fmt.Errorf("write events file: %w", err)
	}

	return nil
}

func (s *Store) ListEvents(date string) ([]webhook.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventsDir := filepath.Join(s.outputDir, "events")
	if _, err := os.Stat(eventsDir); os.IsNotExist(err) {
		return []webhook.Event{}, nil
	}

	var events []webhook.Event
	err := filepath.WalkDir(eventsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // skip unreadable
		}

		var item json.RawMessage
		if err := json.Unmarshal(data, &item); err != nil {
			return nil // skip invalid
		}

		var single webhook.Event
		if err := json.Unmarshal(item, &single); err == nil && single.Number != 0 {
			eventDate := single.MergedAt.Format("2006-01-02")
			if date == "" || eventDate == date {
				events = append(events, single)
			}
		} else {
			var multiple []webhook.Event
			if err := json.Unmarshal(item, &multiple); err == nil {
				for _, e := range multiple {
					eventDate := e.MergedAt.Format("2006-01-02")
					if date == "" || eventDate == date {
						events = append(events, e)
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk events dir: %w", err)
	}

	return events, nil
}
