package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"release-notes-gen/internal/webhook"
)

// NoteMetadata is the index entry used by UI/API listing.
type NoteMetadata struct {
	Filename   string    `json:"filename"`
	Provider   string    `json:"provider"`
	Repository string    `json:"repository"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	Branch     string    `json:"branch"`
	Number     int       `json:"number"`
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"created_at"`
}

// Store manages generated note files and index metadata.
type Store struct {
	outputDir string
	indexPath string
	mu        sync.Mutex
}

func New(outputDir string) (*Store, error) {
	if strings.TrimSpace(outputDir) == "" {
		outputDir = "./output"
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	s := &Store{
		outputDir: outputDir,
		indexPath: filepath.Join(outputDir, "index.json"),
	}
	if err := s.ensureIndexFile(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) SaveNote(event webhook.Event, content string) (NoteMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	base := slugify(event.Branch)
	if base == "" {
		base = slugify(event.Title)
	}
	if base == "" {
		base = "release-note"
	}

	datePrefix := now.Format("2006-01-02")
	filename := fmt.Sprintf("%s_%s.md", datePrefix, base)
	filename = s.nextAvailableFilename(filename)

	fullPath := filepath.Join(s.outputDir, filename)
	if err := os.WriteFile(fullPath, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		return NoteMetadata{}, fmt.Errorf("write note file: %w", err)
	}

	meta := NoteMetadata{
		Filename:   filename,
		Provider:   event.Provider,
		Repository: event.Repository,
		Title:      event.Title,
		Author:     event.Author,
		Branch:     event.Branch,
		Number:     event.Number,
		URL:        event.URL,
		CreatedAt:  now,
	}

	index, err := s.loadIndex()
	if err != nil {
		return NoteMetadata{}, err
	}
	index = append([]NoteMetadata{meta}, index...)
	if err := s.saveIndex(index); err != nil {
		return NoteMetadata{}, err
	}

	return meta, nil
}

func (s *Store) SaveDailyNote(date string, content string) (NoteMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if date == "" {
		date = now.Format("2006-01-02")
	}

	filename := fmt.Sprintf("%s_daily-release-note.md", date)
	filename = s.nextAvailableFilename(filename)

	fullPath := filepath.Join(s.outputDir, filename)
	if err := os.WriteFile(fullPath, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		return NoteMetadata{}, fmt.Errorf("write daily note file: %w", err)
	}

	meta := NoteMetadata{
		Filename:   filename,
		Provider:   "multiple",
		Repository: "multiple",
		Title:      "Daily Release Note - " + date,
		Author:     "system",
		Branch:     "main",
		Number:     0,
		URL:        "",
		CreatedAt:  now,
	}

	index, err := s.loadIndex()
	if err != nil {
		return NoteMetadata{}, err
	}
	index = append([]NoteMetadata{meta}, index...)
	if err := s.saveIndex(index); err != nil {
		return NoteMetadata{}, err
	}

	return meta, nil
}

func (s *Store) ListNotes() ([]NoteMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadIndex()
}

func (s *Store) ReadNote(filename string) (string, error) {
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", fmt.Errorf("invalid filename")
	}
	fullPath := filepath.Join(s.outputDir, filename)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read note file: %w", err)
	}
	return string(data), nil
}

func (s *Store) nextAvailableFilename(filename string) string {
	if _, err := os.Stat(filepath.Join(s.outputDir, filename)); os.IsNotExist(err) {
		return filename
	}

	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", name, i, ext)
		if _, err := os.Stat(filepath.Join(s.outputDir, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

func slugify(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range v {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}

func (s *Store) ensureIndexFile() error {
	if _, err := os.Stat(s.indexPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check index file: %w", err)
	}
	empty, err := json.Marshal([]NoteMetadata{})
	if err != nil {
		return fmt.Errorf("marshal empty index: %w", err)
	}
	if err := os.WriteFile(s.indexPath, empty, 0o644); err != nil {
		return fmt.Errorf("create index file: %w", err)
	}
	return nil
}
