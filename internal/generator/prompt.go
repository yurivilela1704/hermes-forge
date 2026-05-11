package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"release-notes-gen/internal/webhook"
)

// BuildPrompt renders the configured prompt template with normalized event data.
func BuildPrompt(templateName string, event webhook.Event) (string, error) {
	return build(templateName, event)
}

// BuildDailyPrompt renders the configured prompt template with multiple events.
func BuildDailyPrompt(templateName string, events []webhook.Event) (string, error) {
	if len(events) == 0 {
		return "", fmt.Errorf("no events to build prompt from")
	}

	data := struct {
		Date    string
		Events  []webhook.Event
		Authors map[string][]webhook.Event
	}{
		Date:   time.Now().Format("2006-01-02"),
		Events: events,
	}

	authors := make(map[string][]webhook.Event)
	for _, e := range events {
		authors[e.Author] = append(authors[e.Author], e)
	}
	data.Authors = authors

	return build(templateName, data)
}

func build(templateName string, data any) (string, error) {
	if templateName == "" {
		templateName = "default"
	}
	templatePath, err := resolveTemplatePath(templateName)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("read prompt template: %w", err)
	}

	tmpl, err := template.New(templateName).Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse prompt template: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("execute prompt template: %w", err)
	}
	return out.String(), nil
}

func resolveTemplatePath(templateName string) (string, error) {
	fileName := templateName + ".tmpl"
	candidates := []string{
		filepath.Join("templates", fileName),
		filepath.Join("..", "..", "templates", fileName),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("prompt template %q not found", fileName)
}
