package storage

import (
	"encoding/json"
	"fmt"
	"os"
)

func (s *Store) loadIndex() ([]NoteMetadata, error) {
	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		return nil, fmt.Errorf("read index file: %w", err)
	}
	var items []NoteMetadata
	if len(data) == 0 {
		return []NoteMetadata{}, nil
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("decode index file: %w", err)
	}
	return items, nil
}

func (s *Store) saveIndex(items []NoteMetadata) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("encode index file: %w", err)
	}

	tmpPath := s.indexPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp index file: %w", err)
	}
	if err := os.Rename(tmpPath, s.indexPath); err != nil {
		return fmt.Errorf("replace index file: %w", err)
	}
	return nil
}
