package generator

import (
	"strings"
	"testing"

	"release-notes-gen/internal/webhook"
)

func TestBuildPromptDefaultTemplate(t *testing.T) {
	t.Parallel()

	event := webhook.Event{
		Title:       "Add auth",
		Author:      "yuri",
		Branch:      "feature/auth",
		Description: "Adds auth middleware",
		Labels:      "auth, feature",
		GeneratedAt: "2024-05-20 10:00:00",
		Commits: []webhook.Commit{
			{Message: "feat: add middleware", Author: "yuri"},
		},
	}

	prompt, err := BuildPrompt("default", event)
	if err != nil {
		t.Fatalf("BuildPrompt returned error: %v", err)
	}
	if !strings.Contains(prompt, "Title: Add auth") {
		t.Fatalf("expected title in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "- feat: add middleware (yuri)") {
		t.Fatalf("expected commit entry in prompt, got: %s", prompt)
	}
}

func TestBuildDailyPrompt(t *testing.T) {
	t.Parallel()

	events := []webhook.Event{
		{
			Number: 1,
			Title:  "Feature A",
			Author: "alice",
			Commits: []webhook.Commit{
				{Message: "feat a", Author: "alice"},
			},
		},
		{
			Number: 2,
			Title:  "Bugfix B",
			Author: "bob",
			Commits: []webhook.Commit{
				{Message: "fix b", Author: "bob"},
			},
		},
	}

	prompt, err := BuildDailyPrompt("daily", events)
	if err != nil {
		t.Fatalf("BuildDailyPrompt returned error: %v", err)
	}

	if !strings.Contains(prompt, "MR #1: Feature A") {
		t.Fatalf("expected MR 1 title in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "MR #2: Bugfix B") {
		t.Fatalf("expected MR 2 title in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "MRs included: 2") {
		t.Fatalf("expected MR count in comment, got: %s", prompt)
	}
}
